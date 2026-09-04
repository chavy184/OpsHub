package http

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	appHost "ops-hub/internal/application/host"
	appService "ops-hub/internal/application/service"
	"ops-hub/pkg/apperr"
	"ops-hub/pkg/response"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/ssh"
)

type LogHandler struct {
	uc     *appService.UseCase
	hostUC *appHost.UseCase
}

func NewLogHandler(uc *appService.UseCase, hostUC *appHost.UseCase) *LogHandler {
	return &LogHandler{uc: uc, hostUC: hostUC}
}

func (h *LogHandler) RegisterRoutes(api *gin.RouterGroup) {
	api.POST("/logs/search", apperr.Handle(h.Search))
	api.GET("/logs/loki-labels", apperr.Handle(h.LokiLabels))
	api.GET("/logs/loki-label-values", apperr.Handle(h.LokiLabelValues))
}

type LogSearchReq struct {
	ServiceID string `json:"serviceId"`
	EnvID     string `json:"envId"`
	Container string `json:"container"` // docker 多容器时指定目标容器
	File      string `json:"file"`      // file 多文件时指定目标文件路径
	LabelSet  string `json:"labelSet"`  // loki 多标签集时指定目标集名称
	Keyword   string `json:"keyword"`
	Level     string `json:"level"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Limit     int    `json:"limit"`
}

// lokiStreamsResp 统一返回格式（兼容 Loki query_range response），file/docker 也封装成此格式
type lokiStreamsResp struct {
	Status string          `json:"status"`
	Data   lokiStreamsData `json:"data"`
}

type lokiStreamsData struct {
	ResultType string          `json:"resultType"`
	Result     []lokiStreamRow `json:"result"`
}

type lokiStreamRow struct {
	Stream map[string]string `json:"stream"`
	Values [][]string        `json:"values"` // [[ts_ns_str, line], ...]
}

// Search 根据 log_source_type 分发到 loki / file / docker 三种实现
func (h *LogHandler) Search(c *gin.Context) error {
	var req LogSearchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return apperr.BadRequest(err.Error())
	}
	if req.ServiceID == "" || req.EnvID == "" {
		return apperr.BadRequest("serviceId and envId are required")
	}

	envs, err := h.uc.ListEnvs(c.Request.Context(), req.ServiceID)
	if err != nil {
		return apperr.Internal(err)
	}
	var target *appService.ServiceEnvDTO
	for _, e := range envs {
		if e.ID == req.EnvID {
			target = e
			break
		}
	}
	if target == nil {
		return apperr.NotFound("env not found")
	}

	// 解析时间范围（默认最近 7 天）
	now := time.Now()
	startT := now.Add(-7 * 24 * time.Hour)
	endT := now
	if req.StartTime != "" {
		if t, err2 := parseTimeFlexible(req.StartTime); err2 == nil {
			startT = t
		}
	}
	if req.EndTime != "" {
		if t, err2 := parseTimeFlexible(req.EndTime); err2 == nil {
			endT = t
		}
	}

	limit := 200
	if req.Limit > 0 && req.Limit <= 2000 {
		limit = req.Limit
	}

	var cfg map[string]interface{}
	if target.LogSourceConfig != "" {
		_ = json.Unmarshal([]byte(target.LogSourceConfig), &cfg)
	}
	if cfg == nil {
		cfg = map[string]interface{}{}
	}

	srcType := strings.ToLower(target.LogSourceType)
	switch srcType {
	case "loki":
		return h.searchLoki(c, req, cfg, startT, endT, limit)
	case "file":
		return h.searchSSH(c, "file", req, target, cfg, startT, endT, limit)
	case "docker":
		return h.searchSSH(c, "docker", req, target, cfg, startT, endT, limit)
	default:
		return apperr.BadRequest(fmt.Sprintf("unsupported log_source_type: %s", srcType))
	}
}

// ─── Loki ────────────────────────────────────────────────────────────────────

func (h *LogHandler) searchLoki(c *gin.Context, req LogSearchReq, cfg map[string]interface{}, startT, endT time.Time, limit int) error {
	endpointVal, ok := cfg["endpoint"]
	if !ok {
		return apperr.BadRequest("loki endpoint not configured")
	}
	endpoint, ok := endpointVal.(string)
	if !ok || endpoint == "" {
		return apperr.BadRequest("invalid loki endpoint")
	}

	var labelPairs []string
	labelKey, labelValues := parseLokiLabels(cfg)
	if labelKey != "" && len(labelValues) > 0 {
		// 确定目标值（前端传入选中的 label value）
		targetValue := labelValues[0]
		if req.LabelSet != "" {
			for _, v := range labelValues {
				if v == req.LabelSet {
					targetValue = v
					break
				}
			}
		}
		labelPairs = append(labelPairs, fmt.Sprintf(`%s="%s"`, labelKey, targetValue))
	} else if labelsRaw, ok := cfg["labels"]; ok {
		if labelsMap, ok := labelsRaw.(map[string]interface{}); ok {
			for k, v := range labelsMap {
				labelPairs = append(labelPairs, fmt.Sprintf(`%s="%v"`, k, v))
			}
		}
	}
	queryBuilder := fmt.Sprintf("{%s}", strings.Join(labelPairs, ","))
	if req.Keyword != "" {
		queryBuilder += ` |= "` + req.Keyword + `"`
	}
	if req.Level != "" {
		queryBuilder += ` | json | level="` + req.Level + `"`
	}

	startStr := startT.Format(time.RFC3339Nano)
	endStr := endT.Format(time.RFC3339Nano)
	u := fmt.Sprintf("%s/loki/api/v1/query_range?query=%s&start=%s&end=%s&limit=%d",
		strings.TrimRight(endpoint, "/"),
		url.QueryEscape(queryBuilder),
		url.QueryEscape(startStr),
		url.QueryEscape(endStr),
		limit,
	)
	log.Printf("Querying Loki: %s", u)

	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get(u)
	if err != nil {
		return apperr.Internal(err)
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return apperr.Internal(err)
	}

	var data interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		response.OK(c, string(body))
		return nil
	}

	// 若命中为空，用纯文本 grep 做 fallback（可能日志非 JSON 格式导致 | json 管道失败）
	hit := lokiHit(data)
	if !hit && len(labelPairs) > 0 {
		fbQB := fmt.Sprintf("{%s}", strings.Join(labelPairs, ","))
		if req.Keyword != "" {
			fbQB += ` |= "` + req.Keyword + `"`
		}
		if req.Level != "" {
			fbQB += ` |~ "(?i)` + req.Level + `"`
		}
		fbURL := fmt.Sprintf("%s/loki/api/v1/query_range?query=%s&start=%s&end=%s&limit=%d",
			strings.TrimRight(endpoint, "/"),
			url.QueryEscape(fbQB),
			url.QueryEscape(startStr),
			url.QueryEscape(endStr),
			limit,
		)
		log.Printf("Loki fallback: %s", fbURL)
		if resp2, err2 := client.Get(fbURL); err2 == nil {
			defer resp2.Body.Close()
			if body2, err2 := ioutil.ReadAll(resp2.Body); err2 == nil {
				var d2 interface{}
				if err3 := json.Unmarshal(body2, &d2); err3 == nil {
					response.OK(c, d2)
					return nil
				}
			}
		}
	}

	response.OK(c, data)
	return nil
}

func lokiHit(data interface{}) bool {
	m, ok := data.(map[string]interface{})
	if !ok {
		return false
	}
	dd, ok := m["data"].(map[string]interface{})
	if !ok {
		return false
	}
	res, ok := dd["result"].([]interface{})
	return ok && len(res) > 0
}

// ─── SSH (file / docker) ─────────────────────────────────────────────────────

func (h *LogHandler) searchSSH(c *gin.Context, srcType string, req LogSearchReq, env *appService.ServiceEnvDTO, cfg map[string]interface{}, startT, endT time.Time, limit int) error {
	if env.HostID == "" {
		return apperr.BadRequest("该环境未关联主机，无法执行 SSH 日志查询")
	}

	sshClient, err := h.hostUC.GetSSH(c.Request.Context(), env.HostID)
	if err != nil {
		return apperr.Internal(fmt.Errorf("SSH 连接失败: %w", err))
	}
	// 池化连接：不在这里 Close；session 创建失败时 evict 让下次重拨

	var cmd string
	streamLabels := map[string]string{"type": srcType}

	switch srcType {
	case "file":
		fileSources := parseFileSources(cfg)
		if len(fileSources) == 0 {
			return apperr.BadRequest("file 类型需要配置 path/files 字段")
		}
		targetFile := fileSources[0]
		if req.File != "" {
			for _, f := range fileSources {
				if f.Path == req.File || f.Name == req.File {
					targetFile = f
					break
				}
			}
		}
		streamLabels["path"] = targetFile.Path
		cmd = buildFileGrepCmd(targetFile.Path, req.Keyword, req.Level, limit)

	case "docker":
		containers := parseDockerContainers(cfg)
		if len(containers) == 0 {
			return apperr.BadRequest("docker 类型需要配置 container(s) 字段")
		}
		// 确定目标容器
		targetCt := containers[0]
		if req.Container != "" {
			for _, ct := range containers {
				if ct.Container == req.Container || ct.Name == req.Container {
					targetCt = ct
					break
				}
			}
		}
		streamLabels["container"] = targetCt.Container
		cmd = buildDockerLogsCmd(targetCt.Container, req.Keyword, req.Level, startT, endT, limit)
	}

	log.Printf("SSH log command [%s]: %s", srcType, cmd)
	output, cmdErr := runSSHCommand(sshClient, cmd)
	if cmdErr != nil {
		// exit 1 = grep 无匹配，是正常情况；其他错误打印但不 abort（output 可能有部分内容）
		log.Printf("SSH command finished with: %v | output_len=%d", cmdErr, len(output))
		// 连接层错误（如 NewSession 失败）踢出重建； grep exit 1 不会出现这里
		if strings.Contains(cmdErr.Error(), "NewSession") || strings.Contains(cmdErr.Error(), "connection") || strings.Contains(cmdErr.Error(), "closed") {
			h.hostUC.EvictSSH(env.HostID)
		}
	}

	lines := parseOutputLines(output)
	now := time.Now().UnixNano()
	var values [][]string
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		ts := fmt.Sprintf("%d", now-int64(len(lines)-i)*int64(time.Millisecond))
		values = append(values, []string{ts, line})
	}

	result := lokiStreamsResp{
		Status: "success",
		Data: lokiStreamsData{
			ResultType: "streams",
			Result: []lokiStreamRow{
				{Stream: streamLabels, Values: values},
			},
		},
	}
	response.OK(c, result)
	return nil
}

// ─── Docker 多容器解析 ───────────────────────────────────────────────────────

type dockerContainer struct {
	Name      string `json:"name"`
	Container string `json:"container"`
}

// parseDockerContainers 解析 docker 日志源配置，兼容新旧格式
// 新格式: { "containers": [{"name":"web","container":"my-app-web"}, ...] }
// 旧格式: { "container": "my-app" }
func parseDockerContainers(cfg map[string]interface{}) []dockerContainer {
	// 新格式
	if raw, ok := cfg["containers"]; ok {
		data, _ := json.Marshal(raw)
		var list []dockerContainer
		if json.Unmarshal(data, &list) == nil && len(list) > 0 {
			return list
		}
	}
	// 旧格式兼容
	if c, ok := cfg["container"].(string); ok && c != "" {
		return []dockerContainer{{Name: "default", Container: c}}
	}
	return nil
}

// buildFileGrepCmd 构造 file 日志搜索命令
// 使用 tail 取最近 N*10 行 + grep，避免 cat 大文件；无关键词时直接 tail
func buildFileGrepCmd(path, keyword, level string, limit int) string {
	escPath := shellEscape(path)
	hasFilter := keyword != "" || level != ""
	if !hasFilter {
		return fmt.Sprintf("tail -n %d %s", limit, escPath)
	}
	// 先 tail 更大范围，再 grep，最后截取
	scanLines := limit * 20
	if scanLines < 2000 {
		scanLines = 2000
	}
	parts := []string{fmt.Sprintf("tail -n %d %s", scanLines, escPath)}
	if keyword != "" {
		parts = append(parts, "grep -i "+shellEscape(keyword))
	}
	if level != "" {
		parts = append(parts, "grep -i "+shellEscape(level))
	}
	parts = append(parts, fmt.Sprintf("tail -n %d", limit))
	return strings.Join(parts, " | ")
}

// buildDockerLogsCmd 构造 docker logs 命令
// 改用相对时长 --since（如 168h），避免绝对时间时区解析差异
// 去掉 --until，避免当前时刻边界导致最新日志被截断
// 用 2>&1 在管道前合并 stderr（docker 日志默认写 stderr）
func buildDockerLogsCmd(container, keyword, level string, startT, endT time.Time, limit int) string {
	_ = endT // 不再使用 --until
	duration := time.Since(startT)
	sinceHours := int(duration.Hours())
	if sinceHours < 1 {
		sinceHours = 1
	}
	sinceArg := fmt.Sprintf("%dh", sinceHours)

	// 2>&1 在管道前合并 stderr -> stdout，确保日志行进入管道
	dockerCmd := fmt.Sprintf("docker logs --since %s --tail %d %s 2>&1",
		sinceArg, limit, shellEscape(container))
	if keyword != "" {
		dockerCmd += " | grep -i " + shellEscape(keyword)
	}
	if level != "" {
		dockerCmd += " | grep -i " + shellEscape(level)
	}
	return dockerCmd
}

// runSSHCommand 执行远程命令并返回合并输出
// 显式用 sh -c 包裹，确保管道、重定向等 shell 特性在任何 sshd 实现下都能工作
func runSSHCommand(client *ssh.Client, command string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建 SSH session 失败: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	// 显式 sh -c 包裹，保证 pipe/2>&1 等 shell 特性在任何 sshd 下都能正确运行
	err = session.Run("sh -c " + shellEscape(command))
	result := stdout.String()
	if result == "" {
		// stdout 无内容时附上 stderr 内容（方便调试）
		result = stderr.String()
	}
	return result, err
}

// parseOutputLines 将命令输出分割为行
func parseOutputLines(output string) []string {
	var lines []string
	for _, line := range strings.Split(output, "\n") {
		if l := strings.TrimRight(line, "\r"); l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

// shellEscape 对命令参数做简单转义（用单引号包裹，转义内部单引号）
func shellEscape(s string) string {
	escaped := strings.ReplaceAll(s, "'", "'\\''")
	return "'" + escaped + "'"
}

// parseTimeFlexible 解析前端传来的时间字符串
func parseTimeFlexible(s string) (time.Time, error) {
	layouts := []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04:05", "2006-01-02T15:04"}
	var lastErr error
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t, nil
		} else {
			lastErr = err
		}
	}
	return time.Time{}, lastErr
}

// ─── Loki Label 代理 ─────────────────────────────────────────────────────────

// LokiLabels 代理 Loki /loki/api/v1/labels
// GET /api/v1/logs/loki-labels?endpoint=http://...
func (h *LogHandler) LokiLabels(c *gin.Context) error {
	endpoint := strings.TrimRight(c.Query("endpoint"), "/")
	if endpoint == "" {
		return apperr.BadRequest("endpoint is required")
	}

	u := fmt.Sprintf("%s/loki/api/v1/labels", endpoint)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(u)
	if err != nil {
		return apperr.Internal(fmt.Errorf("请求 Loki 失败: %w", err))
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	var data struct {
		Data []string `json:"data"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return apperr.Internal(fmt.Errorf("解析 Loki 响应失败: %w", err))
	}
	response.OK(c, data.Data)
	return nil
}

// LokiLabelValues 代理 Loki /loki/api/v1/label/{name}/values
// GET /api/v1/logs/loki-label-values?endpoint=http://...&label=app
func (h *LogHandler) LokiLabelValues(c *gin.Context) error {
	endpoint := strings.TrimRight(c.Query("endpoint"), "/")
	label := c.Query("label")
	if endpoint == "" || label == "" {
		return apperr.BadRequest("endpoint and label are required")
	}

	u := fmt.Sprintf("%s/loki/api/v1/label/%s/values", endpoint, url.PathEscape(label))
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(u)
	if err != nil {
		return apperr.Internal(fmt.Errorf("请求 Loki 失败: %w", err))
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	var data struct {
		Data []string `json:"data"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return apperr.Internal(fmt.Errorf("解析 Loki 响应失败: %w", err))
	}
	response.OK(c, data.Data)
	return nil
}

// ─── Loki 标签解析 ──────────────────────────────────────────────────────

// parseLokiLabels 解析 loki 日志源配置，返回 label_key 和 label_values
// 新格式: { "endpoint": "...", "label_key": "app", "label_values": ["web", "api"] }
// 旧格式: { "endpoint": "...", "labels": {"app":"my-app"} } → label_key="app", values=["my-app"]
func parseLokiLabels(cfg map[string]interface{}) (string, []string) {
	if keyRaw, ok := cfg["label_key"]; ok {
		key, _ := keyRaw.(string)
		if key != "" {
			if valsRaw, ok := cfg["label_values"]; ok {
				data, _ := json.Marshal(valsRaw)
				var vals []string
				if json.Unmarshal(data, &vals) == nil && len(vals) > 0 {
					return key, vals
				}
			}
		}
	}
	// 旧格式兼容
	if labelsRaw, ok := cfg["labels"]; ok {
		if labelsMap, ok := labelsRaw.(map[string]interface{}); ok && len(labelsMap) > 0 {
			for k, v := range labelsMap {
				return k, []string{fmt.Sprintf("%v", v)}
			}
		}
	}
	return "", nil
}

// ─── File 多文件解析 ─────────────────────────────────────────────────────────

type fileSource struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// parseFileSources 解析 file 日志源配置，兼容新旧格式
// 新格式: { "files": [{"name":"App","path":"/var/log/app.log"}, ...] }
// 旧格式: { "path": "/var/log/app.log" }
func parseFileSources(cfg map[string]interface{}) []fileSource {
	if raw, ok := cfg["files"]; ok {
		data, _ := json.Marshal(raw)
		var list []fileSource
		if json.Unmarshal(data, &list) == nil && len(list) > 0 {
			return list
		}
	}
	if p, ok := cfg["path"].(string); ok && p != "" {
		return []fileSource{{Name: "default", Path: p}}
	}
	return nil
}
