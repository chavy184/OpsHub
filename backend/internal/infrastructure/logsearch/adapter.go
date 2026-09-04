// Package logsearch 日志检索适配器
// 实现 domain/alert.LogSearcher 接口
package logsearch

import (
	"fmt"
	"ops-hub/internal/domain/alert"
)

// NewLogSearcher 根据 log_source_type 创建对应的日志检索适配器
func NewLogSearcher(sourceType string, config map[string]string) (alert.LogSearcher, error) {
	switch sourceType {
	case "loki":
		endpoint := config["endpoint"]
		if endpoint == "" {
			return nil, fmt.Errorf("Loki endpoint 不能为空")
		}
		return NewLokiAdapter(endpoint), nil
	case "file":
		host := config["host"]
		logPath := config["log_path"]
		if host == "" || logPath == "" {
			return nil, fmt.Errorf("文件日志源需要 host 和 log_path")
		}
		return NewFileAdapter(host, logPath, config), nil
	default:
		return nil, fmt.Errorf("不支持的日志源类型: %s", sourceType)
	}
}
