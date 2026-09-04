package release

import (
	"context"
	"time"
)

// ============================
// 发布执行器接口 (Executor Port)
// Domain 层定义接口，Infrastructure 层实现
// ============================

// ReleaseExecutor 发布执行器接口
// MVP 实现: SSHExecutor (SSH + Shell), DockerExecutor (SSH + docker compose)
// Phase 2:  K8sExecutor (kubectl / ArgoCD API)
type ReleaseExecutor interface {
	// Execute 执行部署
	Execute(ctx context.Context, task *ExecutionTask) (*ExecutionResult, error)

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context, endpoint string) (*HealthResult, error)
}

// ExecutionTask 执行任务
type ExecutionTask struct {
	ReleaseID   string
	ServiceKey  string
	RuntimeType string // systemd | docker | k8s
	EnvCode     string
	ArtifactURI string
	Version     string
	PrevVersion string
	Endpoint    string              // 目标机器或集群地址
	Namespace   string              // k8s 用
	Scripts     map[string]string   // 脚本模板 key=step, value=script content
	Vars        map[string]string   // 模板变量
	OnOutput    func(output string) // 实时日志回调（可选）
}

// ExecutionResult 执行结果
type ExecutionResult struct {
	Success     bool
	Output      string
	Error       string
	Elapsed     time.Duration
	BuildNumber int // Jenkins 构建号
}

// HealthResult 健康检查结果
type HealthResult struct {
	Healthy bool
	Message string
	Latency time.Duration
}
