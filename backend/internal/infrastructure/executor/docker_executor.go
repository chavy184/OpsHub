package executor

import (
	"context"
	"fmt"
	"ops-hub/internal/domain/release"
	"time"
)

// DockerExecutor 通过 SSH + docker compose 执行部署
// 适用于 docker 类型的服务
// 流程: SSH → 更新 compose 文件中的镜像版本 → docker compose pull → up -d → 健康检查
type DockerExecutor struct {
	cfg ExecutorConfig
	ssh *SSHExecutor // 复用 SSH 基础能力
}

func NewDockerExecutor(cfg ExecutorConfig) *DockerExecutor {
	return &DockerExecutor{
		cfg: cfg,
		ssh: NewSSHExecutor(cfg),
	}
}

func (e *DockerExecutor) Execute(ctx context.Context, task *release.ExecutionTask) (*release.ExecutionResult, error) {
	start := time.Now()

	client, err := e.ssh.connect(task.Endpoint)
	if err != nil {
		return &release.ExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("SSH 连接失败: %v", err),
			Elapsed: time.Since(start),
		}, nil
	}
	defer client.Close()

	script := e.buildDockerScript(task)

	output, err := e.ssh.runCommand(client, script)
	if err != nil {
		return &release.ExecutionResult{
			Success: false,
			Output:  output,
			Error:   fmt.Sprintf("Docker 部署失败: %v", err),
			Elapsed: time.Since(start),
		}, nil
	}

	return &release.ExecutionResult{
		Success: true,
		Output:  output,
		Elapsed: time.Since(start),
	}, nil
}

func (e *DockerExecutor) HealthCheck(ctx context.Context, endpoint string) (*release.HealthResult, error) {
	return e.ssh.HealthCheck(ctx, endpoint)
}

func (e *DockerExecutor) buildDockerScript(task *release.ExecutionTask) string {
	if script, ok := task.Scripts["deploy"]; ok {
		return script
	}

	return fmt.Sprintf(`#!/bin/bash
set -e

SERVICE_KEY="%s"
VERSION="%s"
COMPOSE_DIR="/opt/$SERVICE_KEY"

echo "[OpsHub] Docker 部署 $SERVICE_KEY:$VERSION"

cd $COMPOSE_DIR

# 更新镜像版本 (sed 替换 image tag)
if [ -f "docker-compose.yml" ]; then
    sed -i "s|image:.*${SERVICE_KEY}:.*|image: ${SERVICE_KEY}:${VERSION}|g" docker-compose.yml
fi

# 拉取新镜像
echo "[OpsHub] 拉取镜像..."
docker compose pull

# 滚动更新
echo "[OpsHub] 启动容器..."
docker compose up -d --remove-orphans

# 等待健康检查
sleep 10

# 检查容器状态
if docker compose ps | grep -q "Up"; then
    echo "[OpsHub] Docker 部署成功"
else
    echo "[OpsHub] Docker 部署失败"
    docker compose logs --tail=50
    exit 1
fi
`, task.ServiceKey, task.Version)
}
