package executor

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"ops-hub/internal/domain/release"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHExecutor 通过 SSH 执行部署脚本
// 适用于 systemd / script 类型的服务
// 流程: SSH 连接目标机 → 上传制品(可选) → 执行部署脚本 → 检查健康
type SSHExecutor struct {
	cfg ExecutorConfig
}

func NewSSHExecutor(cfg ExecutorConfig) *SSHExecutor {
	return &SSHExecutor{cfg: cfg}
}

func (e *SSHExecutor) Execute(ctx context.Context, task *release.ExecutionTask) (*release.ExecutionResult, error) {
	start := time.Now()

	// 1. 建立 SSH 连接
	client, err := e.connect(task.Endpoint)
	if err != nil {
		return &release.ExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("SSH 连接失败: %v", err),
			Elapsed: time.Since(start),
		}, nil
	}
	defer client.Close()

	// 2. 构建部署脚本
	script := e.buildDeployScript(task)

	// 3. 执行脚本
	output, err := e.runCommand(client, script)
	if err != nil {
		return &release.ExecutionResult{
			Success: false,
			Output:  output,
			Error:   fmt.Sprintf("脚本执行失败: %v", err),
			Elapsed: time.Since(start),
		}, nil
	}

	return &release.ExecutionResult{
		Success: true,
		Output:  output,
		Elapsed: time.Since(start),
	}, nil
}

func (e *SSHExecutor) HealthCheck(ctx context.Context, endpoint string) (*release.HealthResult, error) {
	start := time.Now()

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(endpoint)
	if err != nil {
		return &release.HealthResult{
			Healthy: false,
			Message: fmt.Sprintf("健康检查失败: %v", err),
			Latency: time.Since(start),
		}, nil
	}
	defer resp.Body.Close()

	healthy := resp.StatusCode >= 200 && resp.StatusCode < 300
	return &release.HealthResult{
		Healthy: healthy,
		Message: fmt.Sprintf("HTTP %d", resp.StatusCode),
		Latency: time.Since(start),
	}, nil
}

// connect 建立 SSH 连接
func (e *SSHExecutor) connect(host string) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User:            e.cfg.SSHUser,
		Auth:            e.authMethods(),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // MVP 简化; Phase 2 使用 known_hosts
		Timeout:         15 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", host, e.cfg.SSHPort)
	return ssh.Dial("tcp", addr, config)
}

// authMethods 构建认证方式
func (e *SSHExecutor) authMethods() []ssh.AuthMethod {
	var methods []ssh.AuthMethod

	// 1. 内存密钥认证（从 Credential 解密）
	if e.cfg.SSHKeyData != "" {
		signer, err := ssh.ParsePrivateKey([]byte(e.cfg.SSHKeyData))
		if err != nil && e.cfg.SSHKeyPassphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(e.cfg.SSHKeyData), []byte(e.cfg.SSHKeyPassphrase))
		}
		if err == nil {
			methods = append(methods, ssh.PublicKeys(signer))
		}
	}

	// 2. 文件密钥认证（兼容旧配置）
	if e.cfg.SSHKeyPath != "" {
		if conn, err := net.Dial("unix", ""); err == nil {
			conn.Close()
		}
	}

	// 3. 密码认证
	if e.cfg.SSHPassword != "" {
		methods = append(methods, ssh.Password(e.cfg.SSHPassword))
	}

	return methods
}

// runCommand 执行远程命令
func (e *SSHExecutor) runCommand(client *ssh.Client, command string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建 SSH session 失败: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	if err := session.Run(command); err != nil {
		return stdout.String() + "\n" + stderr.String(), err
	}

	return stdout.String(), nil
}

// buildDeployScript 根据任务构建部署脚本
func (e *SSHExecutor) buildDeployScript(task *release.ExecutionTask) string {
	// 如果有自定义脚本，直接使用
	if script, ok := task.Scripts["deploy"]; ok {
		return script
	}

	// 默认部署脚本模板 (systemd 类型)
	return fmt.Sprintf(`#!/bin/bash
set -e

SERVICE_KEY="%s"
VERSION="%s"
ARTIFACT_URI="%s"

echo "[OpsHub] 开始部署 $SERVICE_KEY $VERSION"

# 下载制品
if [ -n "$ARTIFACT_URI" ]; then
    echo "[OpsHub] 下载制品: $ARTIFACT_URI"
    cd /opt/$SERVICE_KEY
    wget -q -O "${SERVICE_KEY}-${VERSION}.tar.gz" "$ARTIFACT_URI"
    tar -xzf "${SERVICE_KEY}-${VERSION}.tar.gz"
fi

# 重启服务
echo "[OpsHub] 重启服务: $SERVICE_KEY"
sudo systemctl restart $SERVICE_KEY

# 等待启动
sleep 5

# 检查状态
if systemctl is-active --quiet $SERVICE_KEY; then
    echo "[OpsHub] 部署成功"
else
    echo "[OpsHub] 部署失败: 服务未启动"
    exit 1
fi
`, task.ServiceKey, task.Version, task.ArtifactURI)
}
