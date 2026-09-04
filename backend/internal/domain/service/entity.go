package service

import (
	"errors"
	"time"
)

// ============================
// 领域实体 (Domain Entities)
// ============================

// Service 服务聚合根
type Service struct {
	ID          string
	ServiceKey  string
	ServiceName string
	OwnerUserID string
	RepoURL     string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ServiceEnv 服务环境实体
type ServiceEnv struct {
	ID                      string
	ServiceID               string
	EnvCode                 string
	ClusterName             string
	Namespace               string
	AccessEndpoint          string
	HealthcheckURL          string
	LogSourceType           string
	LogSourceConfig         string
	HostID                  string
	DeployPath              string
	EnvVars                 string
	HealthCheckInterval     int
	HealthCheckTimeout      int
	HealthCheckSuccessCodes string
	HealthCheckEnabled      bool
	HealthStatus            string
	HealthLastCheckedAt     *time.Time
	HealthLastMessage       string
	JenkinsJobs             string // JSON array: [{"name":"构建","job":"folder/job"}]
	CreatedAt               time.Time
	UpdatedAt               time.Time
}

// ============================
// 领域错误 (Domain Errors)
// ============================

var (
	ErrServiceNotFound      = errors.New("服务不存在")
	ErrServiceKeyDuplicated = errors.New("服务标识已存在")
	ErrServiceEnvNotFound   = errors.New("服务环境不存在")
)
