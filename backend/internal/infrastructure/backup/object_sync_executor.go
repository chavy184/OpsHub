package backup

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	appBackup "ops-hub/internal/application/backup"
	domainBackup "ops-hub/internal/domain/backup"
	"path"
	"sort"
	"strings"
	"time"
)

// ObjectSyncExecutor 手动对象存储同步执行器
type ObjectSyncExecutor struct {
	backupUC *appBackup.UseCase
	client   *http.Client
}

func NewObjectSyncExecutor(backupUC *appBackup.UseCase) *ObjectSyncExecutor {
	return &ObjectSyncExecutor{
		backupUC: backupUC,
		client:   &http.Client{Timeout: 0},
	}
}

func (e *ObjectSyncExecutor) ExecuteNow(ctx context.Context, taskID string, confirmOverwrite bool) error {
	task, err := e.backupUC.GetObjectSyncTaskEntity(ctx, taskID)
	if err != nil {
		return domainBackup.ErrObjectSyncTaskNotFound
	}
	if task.Mode == domainBackup.ObjectSyncModeOverwrite && !confirmOverwrite {
		return domainBackup.ErrOverwriteConfirmRequired
	}

	go e.executeTask(task)
	return nil
}

func (e *ObjectSyncExecutor) executeTask(task *domainBackup.ObjectSyncTask) {
	ctx := context.Background()
	start := time.Now()
	record := &domainBackup.ObjectSyncRecord{
		TaskID:       task.ID,
		TaskName:     task.Name,
		Mode:         task.Mode,
		Status:       domainBackup.ObjectSyncStatusRunning,
		SourceBucket: task.SourceBucket,
		SourcePath:   task.SourcePath,
		TargetBucket: task.TargetBucket,
		TargetPath:   task.TargetPath,
		StartedAt:    start,
	}
	if err := e.backupUC.CreateObjectSyncRecord(ctx, record); err != nil {
		log.Printf("[ObjectSyncExecutor] create record failed: %v", err)
		return
	}

	sourceAccessKey, err := e.backupUC.DecryptObjectSyncSecret(task.SourceAccessKey)
	if err != nil {
		e.finishRecord(ctx, task, record, "解密源访问用户名失败: "+err.Error())
		return
	}
	sourceSecretKey, err := e.backupUC.DecryptObjectSyncSecret(task.SourceSecretKey)
	if err != nil {
		e.finishRecord(ctx, task, record, "解密源访问密码失败: "+err.Error())
		return
	}
	targetAccessKey, err := e.backupUC.DecryptObjectSyncSecret(task.TargetAccessKey)
	if err != nil {
		e.finishRecord(ctx, task, record, "解密目标访问用户名失败: "+err.Error())
		return
	}
	targetSecretKey, err := e.backupUC.DecryptObjectSyncSecret(task.TargetSecretKey)
	if err != nil {
		e.finishRecord(ctx, task, record, "解密目标访问密码失败: "+err.Error())
		return
	}

	src := s3Client{
		httpClient: e.client,
		endpoint:   task.SourceEndpoint,
		region:     task.SourceRegion,
		accessKey:  sourceAccessKey,
		secretKey:  sourceSecretKey,
		useSSL:     task.SourceUseSSL,
	}
	dst := s3Client{
		httpClient: e.client,
		endpoint:   task.TargetEndpoint,
		region:     task.TargetRegion,
		accessKey:  targetAccessKey,
		secretKey:  targetSecretKey,
		useSSL:     task.TargetUseSSL,
	}

	if task.Mode == domainBackup.ObjectSyncModeOverwrite && sameObjectRoot(task) {
		e.finishRecord(ctx, task, record, "覆盖模式不允许源与目标指向同一路径")
		return
	}

	objects, sourcePrefix, err := e.discoverObjects(ctx, src, task.SourceBucket, task.SourcePath)
	if err != nil {
		e.finishRecord(ctx, task, record, err.Error())
		return
	}
	if len(objects) == 0 {
		e.finishRecord(ctx, task, record, "源路径没有可同步对象")
		return
	}

	failureReasons := make([]string, 0)
	for _, obj := range objects {
		item := e.syncObject(ctx, src, dst, task, record.ID, sourcePrefix, obj)
		record.ObjectCount++
		record.BytesTotal += item.Size
		switch item.Status {
		case domainBackup.ObjectSyncStatusSuccess:
			record.SuccessCount++
		case domainBackup.ObjectSyncStatusSkipped:
			record.SkippedCount++
		default:
			record.FailedCount++
			if len(failureReasons) < 10 {
				failureReasons = append(failureReasons, objectSyncFailureReason(item))
			}
		}
		if err := e.backupUC.CreateObjectSyncRecordItem(ctx, item); err != nil {
			log.Printf("[ObjectSyncExecutor] create item failed: %v", err)
		}
	}

	e.finishRecord(ctx, task, record, objectSyncFailureSummary(failureReasons, record.FailedCount))
}

func (e *ObjectSyncExecutor) discoverObjects(ctx context.Context, src s3Client, bucket, sourcePath string) ([]s3Object, string, error) {
	sourceKey := normalizeObjectKey(sourcePath)
	if sourceKey != "" && !strings.HasSuffix(sourceKey, "/") {
		if head, ok, err := src.headObject(ctx, bucket, sourceKey); err != nil {
			return nil, "", err
		} else if ok {
			return []s3Object{{Key: sourceKey, Size: head.Size, ETag: head.ETag}}, path.Dir(sourceKey), nil
		}
	}

	prefix := sourceKey
	objects, err := src.listObjects(ctx, bucket, prefix)
	return objects, strings.TrimSuffix(prefix, "/"), err
}

func (e *ObjectSyncExecutor) syncObject(ctx context.Context, src, dst s3Client, task *domainBackup.ObjectSyncTask, recordID, sourcePrefix string, obj s3Object) *domainBackup.ObjectSyncRecordItem {
	start := time.Now()
	item := &domainBackup.ObjectSyncRecordItem{
		RecordID:  recordID,
		SourceKey: obj.Key,
		TargetKey: buildTargetKey(task.TargetPath, sourcePrefix, obj.Key),
		Size:      obj.Size,
		ETag:      obj.ETag,
		StartedAt: start,
	}
	targetHead, exists, err := dst.headObject(ctx, task.TargetBucket, item.TargetKey)
	if err != nil {
		return finishObjectItem(item, domainBackup.ObjectSyncStatusFailed, "", "检查目标对象失败: "+err.Error())
	}

	if exists {
		switch task.Mode {
		case domainBackup.ObjectSyncModeCopyIfMissing:
			return finishObjectItem(item, domainBackup.ObjectSyncStatusSkipped, domainBackup.ObjectSyncActionSkipped, "目标对象已存在，跳过")
		case domainBackup.ObjectSyncModeChecksumSkip:
			if obj.Size == targetHead.Size && cleanETag(obj.ETag) == cleanETag(targetHead.ETag) {
				return finishObjectItem(item, domainBackup.ObjectSyncStatusSkipped, domainBackup.ObjectSyncActionSkipped, "目标对象校验一致，跳过")
			}
			item.Action = domainBackup.ObjectSyncActionOverwritten
		case domainBackup.ObjectSyncModeOverwrite:
			item.Action = domainBackup.ObjectSyncActionOverwritten
		}
	} else {
		item.Action = domainBackup.ObjectSyncActionCopied
	}

	body, err := src.getObject(ctx, task.SourceBucket, obj.Key)
	if err != nil {
		return finishObjectItem(item, domainBackup.ObjectSyncStatusFailed, item.Action, "读取源对象失败: "+err.Error())
	}
	defer body.Close()
	if err := dst.putObject(ctx, task.TargetBucket, item.TargetKey, body, obj.Size); err != nil {
		return finishObjectItem(item, domainBackup.ObjectSyncStatusFailed, item.Action, "写入目标对象失败: "+err.Error())
	}
	return finishObjectItem(item, domainBackup.ObjectSyncStatusSuccess, item.Action, "同步成功")
}

func (e *ObjectSyncExecutor) finishRecord(ctx context.Context, task *domainBackup.ObjectSyncTask, record *domainBackup.ObjectSyncRecord, errMessage string) {
	now := time.Now()
	record.FinishedAt = &now
	record.Duration = int64(now.Sub(record.StartedAt).Seconds())
	record.Error = errMessage
	record.Summary = fmt.Sprintf("对象 %d 个，成功 %d，跳过 %d，失败 %d，数据 %d bytes", record.ObjectCount, record.SuccessCount, record.SkippedCount, record.FailedCount, record.BytesTotal)

	if errMessage != "" && record.ObjectCount == 0 {
		record.Status = domainBackup.ObjectSyncStatusFailed
	} else if record.FailedCount > 0 && record.SuccessCount+record.SkippedCount > 0 {
		record.Status = domainBackup.ObjectSyncStatusPartialSuccess
	} else if record.FailedCount > 0 {
		record.Status = domainBackup.ObjectSyncStatusFailed
	} else if record.SuccessCount == 0 && record.SkippedCount > 0 {
		record.Status = domainBackup.ObjectSyncStatusSkipped
	} else {
		record.Status = domainBackup.ObjectSyncStatusSuccess
	}

	if err := e.backupUC.UpdateObjectSyncRecord(ctx, record); err != nil {
		log.Printf("[ObjectSyncExecutor] update record failed: %v", err)
	}
	if err := e.backupUC.UpdateObjectSyncTaskLastRun(ctx, task.ID, record.Status); err != nil {
		log.Printf("[ObjectSyncExecutor] update task last run failed: %v", err)
	}
}

func finishObjectItem(item *domainBackup.ObjectSyncRecordItem, status domainBackup.ObjectSyncStatus, action domainBackup.ObjectSyncAction, message string) *domainBackup.ObjectSyncRecordItem {
	now := time.Now()
	item.Status = status
	if action != "" {
		item.Action = action
	}
	item.Message = message
	item.FinishedAt = &now
	item.Duration = int64(now.Sub(item.StartedAt).Seconds())
	return item
}

func objectSyncFailureReason(item *domainBackup.ObjectSyncRecordItem) string {
	message := strings.TrimSpace(item.Message)
	if message == "" {
		message = "同步失败"
	}
	return fmt.Sprintf("%s -> %s: %s", item.SourceKey, item.TargetKey, message)
}

func objectSyncFailureSummary(reasons []string, failedCount int) string {
	if failedCount == 0 {
		return ""
	}
	summary := fmt.Sprintf("%d 个对象同步失败", failedCount)
	if len(reasons) > 0 {
		summary += ": " + strings.Join(reasons, "; ")
	}
	if failedCount > len(reasons) {
		summary += fmt.Sprintf("; 另有 %d 个失败对象未展示", failedCount-len(reasons))
	}
	return summary
}

type s3Client struct {
	httpClient *http.Client
	endpoint   string
	region     string
	accessKey  string
	secretKey  string
	useSSL     bool
}

type s3Object struct {
	Key  string
	Size int64
	ETag string
}

func (c s3Client) listObjects(ctx context.Context, bucket, prefix string) ([]s3Object, error) {
	var result []s3Object
	token := ""
	for {
		query := map[string]string{"list-type": "2"}
		if prefix != "" {
			query["prefix"] = prefix
		}
		if token != "" {
			query["continuation-token"] = token
		}
		resp, err := c.request(ctx, http.MethodGet, bucket, "", query, nil, -1)
		if err != nil {
			return nil, err
		}
		data, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return nil, readErr
		}
		if resp.StatusCode >= 300 {
			return nil, fmt.Errorf("列出对象失败: %s %s", resp.Status, string(data))
		}
		var parsed listBucketResult
		if err := xml.Unmarshal(data, &parsed); err != nil {
			return nil, err
		}
		for _, item := range parsed.Contents {
			if item.Key != "" && !strings.HasSuffix(item.Key, "/") {
				result = append(result, s3Object{Key: item.Key, Size: item.Size, ETag: item.ETag})
			}
		}
		if !parsed.IsTruncated || parsed.NextContinuationToken == "" {
			return result, nil
		}
		token = parsed.NextContinuationToken
	}
}

func (c s3Client) headObject(ctx context.Context, bucket, key string) (s3Object, bool, error) {
	resp, err := c.request(ctx, http.MethodHead, bucket, key, nil, nil, -1)
	if err != nil {
		return s3Object{}, false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return s3Object{}, false, nil
	}
	if resp.StatusCode >= 300 {
		return s3Object{}, false, fmt.Errorf("读取对象元信息失败: %s", resp.Status)
	}
	return s3Object{Key: key, Size: resp.ContentLength, ETag: resp.Header.Get("ETag")}, true, nil
}

func (c s3Client) getObject(ctx context.Context, bucket, key string) (io.ReadCloser, error) {
	resp, err := c.request(ctx, http.MethodGet, bucket, key, nil, nil, -1)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("读取对象失败: %s %s", resp.Status, string(data))
	}
	return resp.Body, nil
}

func (c s3Client) putObject(ctx context.Context, bucket, key string, body io.Reader, size int64) error {
	resp, err := c.request(ctx, http.MethodPut, bucket, key, nil, body, size)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("写入对象失败: %s %s", resp.Status, string(data))
	}
	return nil
}

func (c s3Client) request(ctx context.Context, method, bucket, key string, query map[string]string, body io.Reader, contentLength int64) (*http.Response, error) {
	u, err := c.objectURL(bucket, key, query)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}
	if contentLength >= 0 {
		req.ContentLength = contentLength
	}
	req.Header.Set("x-amz-content-sha256", "UNSIGNED-PAYLOAD")
	req.Header.Set("x-amz-date", time.Now().UTC().Format("20060102T150405Z"))
	c.sign(req)
	return c.httpClient.Do(req)
}

func (c s3Client) objectURL(bucket, key string, query map[string]string) (*url.URL, error) {
	endpoint := strings.TrimRight(c.endpoint, "/")
	if endpoint == "" {
		return nil, errors.New("对象存储 endpoint 不能为空")
	}
	if !strings.HasPrefix(endpoint, "http://") && !strings.HasPrefix(endpoint, "https://") {
		scheme := "http"
		if c.useSSL {
			scheme = "https"
		}
		endpoint = scheme + "://" + endpoint
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	u.Path = "/" + strings.Trim(path.Join(bucket, key), "/")
	q := u.Query()
	for k, v := range query {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	return u, nil
}

func (c s3Client) sign(req *http.Request) {
	now := req.Header.Get("x-amz-date")
	date := now[:8]
	region := c.region
	if strings.TrimSpace(region) == "" {
		region = "us-east-1"
	}
	scope := date + "/" + region + "/s3/aws4_request"
	canonicalHeaders := fmt.Sprintf("host:%s\nx-amz-content-sha256:%s\nx-amz-date:%s\n", req.URL.Host, req.Header.Get("x-amz-content-sha256"), now)
	canonicalRequest := strings.Join([]string{
		req.Method,
		canonicalURI(req.URL.EscapedPath()),
		canonicalQuery(req.URL.Query()),
		canonicalHeaders,
		"host;x-amz-content-sha256;x-amz-date",
		req.Header.Get("x-amz-content-sha256"),
	}, "\n")
	hashedRequest := sha256Hex([]byte(canonicalRequest))
	stringToSign := strings.Join([]string{"AWS4-HMAC-SHA256", now, scope, hashedRequest}, "\n")
	signature := hex.EncodeToString(hmacSHA256(signingKey(c.secretKey, date, region), []byte(stringToSign)))
	req.Header.Set("Authorization", fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=host;x-amz-content-sha256;x-amz-date, Signature=%s", c.accessKey, scope, signature))
}

type listBucketResult struct {
	Contents              []listBucketContent `xml:"Contents"`
	IsTruncated           bool                `xml:"IsTruncated"`
	NextContinuationToken string              `xml:"NextContinuationToken"`
}

type listBucketContent struct {
	Key  string `xml:"Key"`
	Size int64  `xml:"Size"`
	ETag string `xml:"ETag"`
}

func signingKey(secret, date, region string) []byte {
	kDate := hmacSHA256([]byte("AWS4"+secret), []byte(date))
	kRegion := hmacSHA256(kDate, []byte(region))
	kService := hmacSHA256(kRegion, []byte("s3"))
	return hmacSHA256(kService, []byte("aws4_request"))
}

func hmacSHA256(key, data []byte) []byte {
	h := hmac.New(sha256.New, key)
	h.Write(data)
	return h.Sum(nil)
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func canonicalQuery(values url.Values) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		vals := append([]string(nil), values[key]...)
		sort.Strings(vals)
		for _, val := range vals {
			parts = append(parts, url.QueryEscape(key)+"="+url.QueryEscape(val))
		}
	}
	return strings.ReplaceAll(strings.Join(parts, "&"), "+", "%20")
}

func canonicalURI(p string) string {
	if p == "" {
		return "/"
	}
	return p
}

func normalizeObjectKey(key string) string {
	return strings.TrimLeft(strings.TrimSpace(key), "/")
}

func buildTargetKey(targetRoot, sourcePrefix, sourceKey string) string {
	targetRoot = normalizeObjectKey(targetRoot)
	sourceKey = normalizeObjectKey(sourceKey)
	sourcePrefix = strings.Trim(normalizeObjectKey(sourcePrefix), "/")
	if sourcePrefix != "" && strings.HasPrefix(sourceKey, sourcePrefix+"/") {
		sourceKey = strings.TrimPrefix(sourceKey, sourcePrefix+"/")
	} else if sourcePrefix == path.Dir(sourceKey) {
		sourceKey = path.Base(sourceKey)
	}
	if targetRoot == "" {
		return sourceKey
	}
	if strings.HasSuffix(targetRoot, "/") {
		return targetRoot + sourceKey
	}
	if strings.Contains(sourceKey, "/") {
		return path.Join(targetRoot, sourceKey)
	}
	return targetRoot
}

func cleanETag(etag string) string {
	return strings.Trim(strings.TrimSpace(etag), `"`)
}

func sameObjectRoot(task *domainBackup.ObjectSyncTask) bool {
	return strings.EqualFold(strings.TrimRight(task.SourceEndpoint, "/"), strings.TrimRight(task.TargetEndpoint, "/")) &&
		task.SourceBucket == task.TargetBucket &&
		normalizeObjectKey(task.SourcePath) == normalizeObjectKey(task.TargetPath)
}
