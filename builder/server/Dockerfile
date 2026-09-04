# Stage 1: Build the frontend
FROM node:24 AS frontend-builder
WORKDIR /app

# Enable corepack for pnpm support
RUN corepack enable pnpm

# Set Aliyun pnpm registry
RUN pnpm config set registry https://registry.npmmirror.com

# Copy package manifests and lockfile
COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile --ignore-scripts

# Copy source code and build
COPY frontend/ ./
RUN pnpm run build

# Stage 2: Build the Go backend
FROM golang:1.25 AS backend-builder
WORKDIR /build

# Set Aliyun Go proxy
ENV GOPROXY=https://goproxy.cn,direct

# Copy dependency manifests and download Go modules
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy the rest of the backend source code and compile
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o ops-hub ./cmd/server

# Stage 3: Final runtime image
FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive \
    TZ=Asia/Shanghai

# Configure apt sources to use Aliyun mirrors for Ubuntu 24.04 (Noble Numbat)
RUN sed -i 's|archive.ubuntu.com|mirrors.aliyun.com|g; s|security.ubuntu.com|mirrors.aliyun.com|g' /etc/apt/sources.list.d/ubuntu.sources 2>/dev/null || true \
    && sed -i 's|archive.ubuntu.com|mirrors.aliyun.com|g; s|security.ubuntu.com|mirrors.aliyun.com|g' /etc/apt/sources.list 2>/dev/null || true \
    && apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        tzdata \
        mysql-client \
        postgresql-client \
        openssh-client \
        wget \
    && ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy binary from backend-builder stage
COPY --from=backend-builder /build/ops-hub ./ops-hub

# Copy static assets from frontend-builder stage
COPY --from=frontend-builder /app/dist ./static

# Copy migrations and a public configuration example from build context.
# The real config is mounted or injected at runtime and is excluded by .dockerignore.
COPY builder/server/migrations ./migrations
COPY builder/server/configs/config.yaml.example ./configs/config.yaml.example

ENV OPSHUB_CONFIG=./configs/config.yaml

RUN chmod +x ./ops-hub

EXPOSE 48884

ENV GIN_MODE=release

ENTRYPOINT ["./ops-hub"]

