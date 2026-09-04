// Package executor 发布执行器工厂
// 根据 service.runtime_type 选择对应的执行器实现
package executor

import (
	"fmt"
	"ops-hub/internal/domain/credential"
	"ops-hub/internal/domain/host"
	"ops-hub/internal/domain/release"
	"ops-hub/internal/infrastructure/crypto"
)

// NewExecutor 根据 runtime_type 创建对应的执行器
func NewExecutor(runtimeType string, cfg ExecutorConfig) (release.ReleaseExecutor, error) {
	switch runtimeType {
	case "systemd", "script", "process":
		return NewSSHExecutor(cfg), nil
	case "docker":
		return NewDockerExecutor(cfg), nil
	default:
		return nil, fmt.Errorf("不支持的部署类型: %s", runtimeType)
	}
}

// NewExecutorFromHost 从 Host + Credential 动态构建执行器
func NewExecutorFromHost(
	runtimeType string,
	h *host.Host,
	cred *credential.Credential,
	encryptor crypto.Encryptor,
) (release.ReleaseExecutor, error) {
	secret, err := encryptor.Decrypt(cred.SecretData)
	if err != nil {
		return nil, fmt.Errorf("凭证解密失败: %w", err)
	}

	cfg := ExecutorConfig{
		SSHUser:        h.Username,
		SSHPort:        h.SSHPort,
		DefaultTimeout: 300,
	}

	switch cred.CredType {
	case "ssh_key":
		cfg.SSHKeyData = secret
		// 解密 passphrase
		if cred.Passphrase != "" {
			passphrase, ppErr := encryptor.Decrypt(cred.Passphrase)
			if ppErr == nil {
				cfg.SSHKeyPassphrase = passphrase
			}
		}
	case "ssh_password":
		cfg.SSHPassword = secret
	default:
		cfg.SSHKeyData = secret
	}

	switch runtimeType {
	case "systemd", "process":
		return NewSSHExecutor(cfg), nil
	case "docker":
		return NewDockerExecutor(cfg), nil
	default:
		return nil, fmt.Errorf("不支持的部署类型: %s", runtimeType)
	}
}

// ExecutorConfig 执行器通用配置
type ExecutorConfig struct {
	SSHUser          string
	SSHKeyPath       string // 保留兼容文件路径
	SSHKeyData       string // 内存中的密钥内容（从 Credential 解密）
	SSHKeyPassphrase string // SSH 私钥密码短语
	SSHPassword      string // 密码认证
	SSHPort          int
	DefaultTimeout   int // 秒
}

// DefaultExecutorConfig 默认配置
func DefaultExecutorConfig() ExecutorConfig {
	return ExecutorConfig{
		SSHUser:        "deploy",
		SSHKeyPath:     "",
		SSHPort:        22,
		DefaultTimeout: 300,
	}
}
