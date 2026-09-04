package logsearch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"ops-hub/internal/domain/alert"
	"time"
)

// LokiAdapter 通过 Loki HTTP API 检索日志
// 使用 LogQL 查询语法
type LokiAdapter struct {
	endpoint   string
	httpClient *http.Client
}

func NewLokiAdapter(endpoint string) *LokiAdapter {
	return &LokiAdapter{
		endpoint:   endpoint,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (a *LokiAdapter) Search(ctx context.Context, query alert.LogSearchQuery) (*alert.LogSearchResult, error) {
	logql := a.buildLogQL(query)

	params := url.Values{}
	params.Set("query", logql)
	params.Set("limit", fmt.Sprintf("%d", query.PageSize))
	if !query.StartTime.IsZero() {
		params.Set("start", fmt.Sprintf("%d", query.StartTime.UnixNano()))
	}
	if !query.EndTime.IsZero() {
		params.Set("end", fmt.Sprintf("%d", query.EndTime.UnixNano()))
	}

	reqURL := fmt.Sprintf("%s/loki/api/v1/query_range?%s", a.endpoint, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("构建 Loki 请求失败: %w", err)
	}

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Loki 请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取 Loki 响应失败: %w", err)
	}

	return a.parseLokiResponse(body)
}

// buildLogQL 构建 LogQL 查询
func (a *LokiAdapter) buildLogQL(query alert.LogSearchQuery) string {
	// 基础流选择器
	logql := fmt.Sprintf(`{service="%s"`, query.ServiceID)
	if query.EnvID != "" {
		logql += fmt.Sprintf(`, env="%s"`, query.EnvID)
	}
	logql += "}"

	// 管道操作
	if query.Keyword != "" {
		logql += fmt.Sprintf(` |~ "%s"`, query.Keyword)
	}
	if query.Level != "" {
		logql += fmt.Sprintf(` | level="%s"`, query.Level)
	}
	if query.TraceID != "" {
		logql += fmt.Sprintf(` | trace_id="%s"`, query.TraceID)
	}

	return logql
}

// parseLokiResponse 解析 Loki 响应为统一日志格式
func (a *LokiAdapter) parseLokiResponse(body []byte) (*alert.LogSearchResult, error) {
	var lokiResp struct {
		Data struct {
			Result []struct {
				Values [][]string `json:"values"`
			} `json:"result"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &lokiResp); err != nil {
		return nil, fmt.Errorf("解析 Loki 响应失败: %w", err)
	}

	var entries []*alert.LogEntry
	for _, stream := range lokiResp.Data.Result {
		for _, v := range stream.Values {
			if len(v) < 2 {
				continue
			}
			entry := &alert.LogEntry{
				Message: v[1],
				Fields:  make(map[string]string),
			}
			// 尝试解析 JSON 日志行
			var jsonLog map[string]interface{}
			if err := json.Unmarshal([]byte(v[1]), &jsonLog); err == nil {
				if ts, ok := jsonLog["timestamp"].(string); ok {
					if t, err := time.Parse(time.RFC3339, ts); err == nil {
						entry.Timestamp = t
					}
				}
				if lvl, ok := jsonLog["level"].(string); ok {
					entry.Level = lvl
				}
				if msg, ok := jsonLog["message"].(string); ok {
					entry.Message = msg
				}
				if tid, ok := jsonLog["trace_id"].(string); ok {
					entry.TraceID = tid
				}
			}
			entries = append(entries, entry)
		}
	}

	return &alert.LogSearchResult{
		Entries: entries,
		Total:   int64(len(entries)),
	}, nil
}
