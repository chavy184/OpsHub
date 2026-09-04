package logsearch

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"ops-hub/internal/domain/alert"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// FileAdapter 通过 SSH + grep 检索远程服务器上的日志文件
// 适用于私有化部署无 Loki/ELK 的场景
// 要求: 日志文件为 JSON Lines 格式
type FileAdapter struct {
	host    string
	logPath string
	sshUser string
	sshPort int
}

func NewFileAdapter(host, logPath string, config map[string]string) *FileAdapter {
	a := &FileAdapter{
		host:    host,
		logPath: logPath,
		sshUser: "deploy",
		sshPort: 22,
	}
	if user, ok := config["ssh_user"]; ok {
		a.sshUser = user
	}
	return a
}

func (a *FileAdapter) Search(ctx context.Context, query alert.LogSearchQuery) (*alert.LogSearchResult, error) {
	command := a.buildGrepCommand(query)

	output, err := a.runSSHCommand(ctx, command)
	if err != nil {
		return nil, fmt.Errorf("远程日志检索失败: %w", err)
	}

	return a.parseOutput(output)
}

// buildGrepCommand 构建 grep 命令
func (a *FileAdapter) buildGrepCommand(query alert.LogSearchQuery) string {
	parts := []string{"cat", a.logPath}

	// 关键字过滤
	if query.Keyword != "" {
		parts = append(parts, "|", "grep", "-i", fmt.Sprintf("'%s'", query.Keyword))
	}

	// 日志级别过滤
	if query.Level != "" {
		parts = append(parts, "|", "grep", fmt.Sprintf("'\"level\":\"%s\"'", strings.ToUpper(query.Level)))
	}

	// TraceID 过滤
	if query.TraceID != "" {
		parts = append(parts, "|", "grep", fmt.Sprintf("'\"trace_id\":\"%s\"'", query.TraceID))
	}

	// 限制行数
	limit := query.PageSize
	if limit <= 0 {
		limit = 100
	}
	parts = append(parts, "|", "tail", "-n", fmt.Sprintf("%d", limit))

	return strings.Join(parts, " ")
}

// runSSHCommand 执行远程命令
func (a *FileAdapter) runSSHCommand(ctx context.Context, command string) (string, error) {
	config := &ssh.ClientConfig{
		User:            a.sshUser,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", a.host, a.sshPort)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return "", fmt.Errorf("SSH 连接失败: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建 SSH session 失败: %w", err)
	}
	defer session.Close()

	var stdout bytes.Buffer
	session.Stdout = &stdout

	if err := session.Run(command); err != nil {
		// grep 无匹配返回 exit 1，不算错误
		if stdout.Len() == 0 {
			return "", nil
		}
	}

	return stdout.String(), nil
}

// parseOutput 解析 JSON Lines 格式的日志输出
func (a *FileAdapter) parseOutput(output string) (*alert.LogSearchResult, error) {
	var entries []*alert.LogEntry

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		entry := &alert.LogEntry{
			Fields: make(map[string]string),
		}

		var jsonLog map[string]interface{}
		if err := json.Unmarshal([]byte(line), &jsonLog); err == nil {
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
			if svc, ok := jsonLog["service"].(string); ok {
				entry.Service = svc
			}
			if env, ok := jsonLog["env"].(string); ok {
				entry.Env = env
			}
		} else {
			// 非 JSON 行，原样保留
			entry.Message = line
			entry.Timestamp = time.Now()
		}

		entries = append(entries, entry)
	}

	return &alert.LogSearchResult{
		Entries: entries,
		Total:   int64(len(entries)),
	}, nil
}
