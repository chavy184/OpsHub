// Package jenkins 封装 Jenkins REST API 客户端
package jenkins

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ============================
// 数据模型
// ============================

// JobInfo Jenkins Job 信息
type JobInfo struct {
	Description          string      `json:"description"`
	Buildable            bool        `json:"buildable"`
	FullName             string      `json:"fullName"`
	URL                  string      `json:"url"`
	LastBuild            *BuildRef   `json:"lastBuild"`
	LastSuccessfulBuild  *BuildRef   `json:"lastSuccessfulBuild"`
	LastFailedBuild      *BuildRef   `json:"lastFailedBuild"`
	ParameterDefinitions []ParamDef  `json:"-"` // 从 actions 中提取
	Builds               []BuildRef  `json:"builds"`
	RawActions           []RawAction `json:"actions"`
}

type BuildRef struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
}

type RawAction struct {
	Class                string     `json:"_class"`
	ParameterDefinitions []ParamDef `json:"parameterDefinitions,omitempty"`
}

// ParamDef Jenkins 参数定义
type ParamDef struct {
	Name                  string      `json:"name"`
	Type                  string      `json:"type"`
	Description           string      `json:"description"`
	DefaultParameterValue *ParamValue `json:"defaultParameterValue"`
	Choices               []string    `json:"choices,omitempty"`
}

type ParamValue struct {
	Value interface{} `json:"value"`
}

// BuildInfo Jenkins 构建详情
type BuildInfo struct {
	Number    int    `json:"number"`
	Result    string `json:"result"` // SUCCESS / FAILURE / ABORTED / null
	Building  bool   `json:"building"`
	Timestamp int64  `json:"timestamp"`
	Duration  int64  `json:"duration"` // 毫秒
	URL       string `json:"url"`
}

// QueueItem Jenkins 队列项
type QueueItem struct {
	ID         int       `json:"id"`
	Blocked    bool      `json:"blocked"`
	Buildable  bool      `json:"buildable"`
	Executable *BuildRef `json:"executable"`
	Why        string    `json:"why"`
}

// ============================
// Client
// ============================

// Client Jenkins REST API 客户端
type Client struct {
	baseURL string
	user    string
	token   string
	http    *http.Client
}

// NewClient 创建 Jenkins 客户端
func NewClient(baseURL, user, token string) *Client {
	baseURL = strings.TrimRight(baseURL, "/")
	return &Client{
		baseURL: baseURL,
		user:    user,
		token:   token,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetJobInfo 获取 Job 信息
func (c *Client) GetJobInfo(ctx context.Context, jobPath string) (*JobInfo, error) {
	apiURL := c.jobURL(jobPath) + "/api/json?tree=description,buildable,fullName,url,lastBuild[number,url],lastSuccessfulBuild[number,url],lastFailedBuild[number,url],builds[number,url]{0,10},actions[_class,parameterDefinitions[name,type,description,defaultParameterValue[value],choices]]"
	body, err := c.doGet(ctx, apiURL)
	if err != nil {
		return nil, fmt.Errorf("获取 Job 信息失败: %w", err)
	}

	var info JobInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("解析 Job 信息失败: %w", err)
	}

	// 从 actions 提取 parameterDefinitions
	for _, action := range info.RawActions {
		if len(action.ParameterDefinitions) > 0 {
			info.ParameterDefinitions = action.ParameterDefinitions
			break
		}
	}

	return &info, nil
}

// GetBuildInfo 获取构建详情
func (c *Client) GetBuildInfo(ctx context.Context, jobPath string, buildNumber int) (*BuildInfo, error) {
	apiURL := fmt.Sprintf("%s/%d/api/json", c.jobURL(jobPath), buildNumber)

	body, err := c.doGet(ctx, apiURL)
	if err != nil {
		return nil, fmt.Errorf("获取构建信息失败: %w", err)
	}

	var info BuildInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("解析构建信息失败: %w", err)
	}

	return &info, nil
}

// GetConsoleOutput 获取构建日志
func (c *Client) GetConsoleOutput(ctx context.Context, jobPath string, buildNumber int) (string, error) {
	apiURL := fmt.Sprintf("%s/%d/consoleText", c.jobURL(jobPath), buildNumber)

	body, err := c.doGet(ctx, apiURL)
	if err != nil {
		return "", fmt.Errorf("获取构建日志失败: %w", err)
	}

	return string(body), nil
}

// TriggerBuild 触发参数化构建，返回 queue item URL
func (c *Client) TriggerBuild(ctx context.Context, jobPath string, params map[string]string) (string, error) {
	var apiURL string
	var reqBody io.Reader

	if len(params) > 0 {
		apiURL = c.jobURL(jobPath) + "/buildWithParameters"
		form := url.Values{}
		for k, v := range params {
			form.Set(k, v)
		}
		reqBody = strings.NewReader(form.Encode())
	} else {
		apiURL = c.jobURL(jobPath) + "/build"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, reqBody)
	if err != nil {
		return "", err
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	c.setAuth(req)

	// Jenkins CSRF crumb
	crumbField, crumbValue, crumbErr := c.getCrumb(ctx)
	if crumbErr == nil && crumbField != "" {
		req.Header.Set(crumbField, crumbValue)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("触发构建请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("触发构建失败 HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	// Location header 包含 queue item URL
	queueURL := resp.Header.Get("Location")
	return queueURL, nil
}

// GetQueueItem 获取队列项信息（等待 build number 分配）
func (c *Client) GetQueueItem(ctx context.Context, queueURL string) (*QueueItem, error) {
	apiURL := queueURL + "api/json"
	if !strings.Contains(queueURL, "/api/json") {
		if !strings.HasSuffix(queueURL, "/") {
			apiURL = queueURL + "/api/json"
		}
	}

	body, err := c.doGet(ctx, apiURL)
	if err != nil {
		return nil, fmt.Errorf("获取队列项失败: %w", err)
	}

	var item QueueItem
	if err := json.Unmarshal(body, &item); err != nil {
		return nil, fmt.Errorf("解析队列项失败: %w", err)
	}

	return &item, nil
}

// GetRecentBuilds 获取最近 N 次构建
func (c *Client) GetRecentBuilds(ctx context.Context, jobPath string, count int) ([]BuildInfo, error) {
	apiURL := fmt.Sprintf("%s/api/json?tree=builds[number,result,building,timestamp,duration,url]{0,%d}", c.jobURL(jobPath), count)
	body, err := c.doGet(ctx, apiURL)
	if err != nil {
		return nil, fmt.Errorf("获取构建历史失败: %w", err)
	}

	var result struct {
		Builds []BuildInfo `json:"builds"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析构建历史失败: %w", err)
	}

	return result.Builds, nil
}

// ============================
// 内部方法
// ============================

func (c *Client) jobURL(jobPath string) string {
	// 支持 folder/job 路径格式：my-folder/my-job → job/my-folder/job/my-job
	parts := strings.Split(jobPath, "/")
	var sb strings.Builder
	for _, p := range parts {
		sb.WriteString("/job/")
		sb.WriteString(p)
	}
	return c.baseURL + sb.String()
}

func (c *Client) setAuth(req *http.Request) {
	if c.user != "" && c.token != "" {
		req.SetBasicAuth(c.user, c.token)
	}
}

// getCrumb 获取 Jenkins CSRF crumb token
func (c *Client) getCrumb(ctx context.Context) (field, value string, err error) {
	apiURL := c.baseURL + "/crumbIssuer/api/json"
	body, err := c.doGet(ctx, apiURL)
	if err != nil {
		return "", "", err
	}
	var result struct {
		Crumb             string `json:"crumb"`
		CrumbRequestField string `json:"crumbRequestField"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", "", err
	}
	return result.CrumbRequestField, result.Crumb, nil
}

func (c *Client) doGet(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	c.setAuth(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}
