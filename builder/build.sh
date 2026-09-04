#!/bin/bash -ilex
#
# OpsHub 前后端源码构建 + Docker 容器部署脚本（Jenkins 使用）
#
# Jenkins 参数:
#   server   - 1: 编译后端并更新镜像; 0: 跳过后端编译
#   frontend - 1: 编译前端并更新镜像; 0: 跳过前端编译
#
# 环境变量:
#   WORKSPACE - Jenkins 工作区路径（本地调试时可省略，自动推断为仓库根目录）
#
set -euo pipefail

server=${server:-1}
frontend=${frontend:-1}
WORKSPACE=${WORKSPACE:-$(cd "$(dirname "$0")/.." && pwd)}

BUILDER_DIR="$WORKSPACE/builder"
SERVER_DIR="$BUILDER_DIR/server"
CONTAINER_NAME="ops-hub-server"
COMPOSE_FILE="$BUILDER_DIR/docker-compose.yml"

if command -v docker-compose >/dev/null 2>&1; then
  DOCKER_COMPOSE="docker-compose"
else
  DOCKER_COMPOSE="docker compose"
fi

echo "========== OpsHub 构建开始 =========="
echo "WORKSPACE=$WORKSPACE"
echo "server=$server, frontend=$frontend"
echo "COMPOSE_FILE=$COMPOSE_FILE"
echo "说明: 前端在宿主机 pnpm build，Docker 构建 ops-hub-server（ubuntu:24.04），包含后端多阶段 Go 构建"
echo "若日志出现 [frontend internal] node:18.x，说明 Jenkins 未执行本脚本或仍在使用旧的 docker-compose"

# ---------------------------------------------------------------------------
# 1. 准备配置文件（统一使用 config.yaml）
# ---------------------------------------------------------------------------
mkdir -p "$SERVER_DIR/configs"

if [ -f "$WORKSPACE/configs/config.yaml" ]; then
  echo "使用部署配置: $WORKSPACE/configs/config.yaml"
  cp "$WORKSPACE/configs/config.yaml" "$SERVER_DIR/configs/config.yaml"
elif [ -f "$WORKSPACE/backend/configs/config.yaml" ]; then
  echo "使用默认配置: backend/configs/config.yaml"
  cp "$WORKSPACE/backend/configs/config.yaml" "$SERVER_DIR/configs/config.yaml"
else
  echo "错误: 未找到 config.yaml，请在以下位置之一提供配置:"
  echo "  - \$WORKSPACE/configs/config.yaml（Jenkins 部署推荐）"
  echo "  - \$WORKSPACE/backend/configs/config.yaml"
  echo "可参考: backend/configs/config.yaml.example"
  exit 1
fi

# ---------------------------------------------------------------------------
# 2. 后端同步与依赖准备（Go 编译现已移至 Docker 容器内进行）
# ---------------------------------------------------------------------------
if [ "$server" = "1" ]; then
  echo "========== 准备后端构建依赖 =========="
  echo "后端 Go 编译已配置为在 Docker 容器内使用 multi-stage 自动构建，无需宿主机 Go 环境。"

  echo "同步数据库迁移文件..."
  mkdir -p "$SERVER_DIR/migrations"
  cp -r "$WORKSPACE/backend/migrations/." "$SERVER_DIR/migrations/"
else
  echo "server=0，跳过后端依赖准备"
fi

# ---------------------------------------------------------------------------
# 3. 前端同步与依赖准备（前端编译现已移至 Docker 容器内进行）
# ---------------------------------------------------------------------------
if [ "$frontend" = "1" ]; then
  echo "========== 准备前端构建依赖 =========="
  echo "前端编译已配置为在 Docker 容器内使用 node:24 自动构建，无需宿主机 Node 环境。"
else
  echo "frontend=0，跳过前端依赖准备"
fi

# ---------------------------------------------------------------------------
# 4. 重建 Docker 镜像并启动容器
# ---------------------------------------------------------------------------
need_rebuild=0
if [ "$server" = "1" ] || [ "$frontend" = "1" ]; then
  need_rebuild=1
fi

cd "$BUILDER_DIR"

if [ "$need_rebuild" = "1" ]; then
  # Docker COPY 依赖这些产物，缺失则提前失败并给出明确提示
  for artifact in "$SERVER_DIR/migrations"; do
    if [ ! -e "$artifact" ]; then
      echo "错误: 缺少构建产物 $artifact"
      echo "提示: 首次部署请设置 server=1 且 frontend=1"
      exit 1
    fi
  done
  if [ ! -f "$SERVER_DIR/configs/config.yaml" ]; then
    echo "错误: 缺少 $SERVER_DIR/configs/config.yaml"
    exit 1
  fi

  echo "========== 重建 Docker 容器 =========="
  echo "构建服务: ops-hub-server（FROM ubuntu:24.04）"

  # 清理损坏的 buildkit 缓存（解决 metadata size validation 报错）
  echo "清理 Docker BuildKit 缓存..."
  docker builder prune -af >/dev/null 2>&1 || true
  docker buildx prune -af >/dev/null 2>&1 || true

  running_id=$(docker ps -q --filter "name=^/${CONTAINER_NAME}$" 2>/dev/null || true)
  if [ -n "$running_id" ]; then
    echo "停止旧容器: $running_id"
    docker stop "$running_id"
    docker rm "$running_id"
  fi

  # 清理旧镜像（compose 默认镜像名: builder-ops-hub-server）
  old_images=$(docker images --format '{{.Repository}}:{{.Tag}}' \
    | grep -E '(^builder-ops-hub-server:|^ops-hub-server:)' || true)
  if [ -n "$old_images" ]; then
    echo "删除旧镜像..."
    echo "$old_images" | xargs -r docker rmi -f || true
  fi

  $DOCKER_COMPOSE -f "$COMPOSE_FILE" build --no-cache
  $DOCKER_COMPOSE -f "$COMPOSE_FILE" up -d
else
  echo "server=0 且 frontend=0，仅启动容器（不重建镜像）"
  $DOCKER_COMPOSE -f "$COMPOSE_FILE" up -d
fi

echo "========== 构建完成 =========="

# ---------------------------------------------------------------------------
# 5. 同步配置和迁移到持久化目录（供 volume 映射）
# ---------------------------------------------------------------------------
PERSIST_DIR="${OPSHUB_DATA_DIR:-$BUILDER_DIR/data}"
export OPSHUB_DATA_DIR="$PERSIST_DIR"

echo "当前用户: $(whoami) (uid=$(id -u), gid=$(id -g))"
echo "PERSIST_DIR=$PERSIST_DIR 权限信息:"
ls -ld "$PERSIST_DIR" 2>/dev/null || echo "  目录不存在，将创建"
ls -la "$PERSIST_DIR/" 2>/dev/null || true

sudo mkdir -p "$PERSIST_DIR/configs" "$PERSIST_DIR/migrations" "$PERSIST_DIR/logs" "$PERSIST_DIR/documents" 2>/dev/null \
  || mkdir -p "$PERSIST_DIR/configs" "$PERSIST_DIR/migrations" "$PERSIST_DIR/logs" "$PERSIST_DIR/documents"

# 仅在持久化目录无配置时同步（首次部署）
if [ ! -f "$PERSIST_DIR/configs/config.yaml" ]; then
  cp "$SERVER_DIR/configs/config.yaml" "$PERSIST_DIR/configs/config.yaml"
  echo "已初始化配置文件到 $PERSIST_DIR/configs/config.yaml"
fi
# 迁移脚本始终同步（可能有新增）
cp -r "$SERVER_DIR/migrations/." "$PERSIST_DIR/migrations/"

$DOCKER_COMPOSE -f "$COMPOSE_FILE" ps
