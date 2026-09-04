// Package service 服务模块应用层 - 用例编排
package service

import (
	"context"
	"fmt"
	"ops-hub/internal/domain/service"
	"time"

	"github.com/google/uuid"
)

// ============================
// DTO (Data Transfer Objects)
// ============================

type CreateServiceCmd struct {
	ServiceKey  string `json:"service_key" binding:"required"`
	ServiceName string `json:"service_name" binding:"required"`
	OwnerUserID string `json:"owner_user_id"`
	RepoURL     string `json:"repo_url"`
}

type UpdateServiceCmd struct {
	ServiceID   string `json:"-"`
	ServiceName string `json:"service_name"`
	OwnerUserID string `json:"owner_user_id"`
	RepoURL     string `json:"repo_url"`
}

type ServiceQueryCmd struct {
	Keyword  string `form:"keyword"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type ServiceDTO struct {
	ID          string `json:"id"`
	ServiceKey  string `json:"service_key"`
	ServiceName string `json:"service_name"`
	OwnerUserID string `json:"owner_user_id"`
	RepoURL     string `json:"repo_url"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type CreateEnvCmd struct {
	ServiceID               string `json:"-"`
	EnvCode                 string `json:"env_code" binding:"required"`
	ClusterName             string `json:"cluster_name"`
	Namespace               string `json:"namespace"`
	AccessEndpoint          string `json:"access_endpoint"`
	HealthcheckURL          string `json:"healthcheck_url"`
	LogSourceType           string `json:"log_source_type"`
	LogSourceConfig         string `json:"log_source_config"`
	HostID                  string `json:"host_id"`
	DeployPath              string `json:"deploy_path"`
	EnvVars                 string `json:"env_vars"`
	HealthCheckInterval     int    `json:"health_check_interval"`
	HealthCheckTimeout      int    `json:"health_check_timeout"`
	HealthCheckSuccessCodes string `json:"health_check_success_codes"`
	HealthCheckEnabled      *bool  `json:"health_check_enabled"`
	JenkinsJobs             string `json:"jenkins_jobs"`
}

type UpdateEnvCmd struct {
	EnvID                   string `json:"-"`
	HostID                  string `json:"host_id"`
	DeployPath              string `json:"deploy_path"`
	EnvVars                 string `json:"env_vars"`
	AccessEndpoint          string `json:"access_endpoint"`
	HealthcheckURL          string `json:"healthcheck_url"`
	LogSourceType           string `json:"log_source_type"`
	LogSourceConfig         string `json:"log_source_config"`
	HealthCheckInterval     int    `json:"health_check_interval"`
	HealthCheckTimeout      int    `json:"health_check_timeout"`
	HealthCheckSuccessCodes string `json:"health_check_success_codes"`
	HealthCheckEnabled      *bool  `json:"health_check_enabled"`
	JenkinsJobs             string `json:"jenkins_jobs"`
}

type ServiceEnvDTO struct {
	ID                      string  `json:"id"`
	ServiceID               string  `json:"service_id"`
	EnvCode                 string  `json:"env_code"`
	ClusterName             string  `json:"cluster_name"`
	Namespace               string  `json:"namespace"`
	AccessEndpoint          string  `json:"access_endpoint"`
	HealthcheckURL          string  `json:"healthcheck_url"`
	LogSourceType           string  `json:"log_source_type"`
	LogSourceConfig         string  `json:"log_source_config"`
	HostID                  string  `json:"host_id"`
	DeployPath              string  `json:"deploy_path"`
	EnvVars                 string  `json:"env_vars"`
	HealthCheckInterval     int     `json:"health_check_interval"`
	HealthCheckTimeout      int     `json:"health_check_timeout"`
	HealthCheckSuccessCodes string  `json:"health_check_success_codes"`
	HealthCheckEnabled      bool    `json:"health_check_enabled"`
	HealthStatus            string  `json:"health_status"`
	HealthLastCheckedAt     *string `json:"health_last_checked_at"`
	HealthLastMessage       string  `json:"health_last_message"`
	JenkinsJobs             string  `json:"jenkins_jobs"`
	CreatedAt               string  `json:"created_at"`
	UpdatedAt               string  `json:"updated_at"`
}

// ============================
// Assembler
// ============================

func serviceToDTO(e *service.Service) *ServiceDTO {
	return &ServiceDTO{
		ID:          e.ID,
		ServiceKey:  e.ServiceKey,
		ServiceName: e.ServiceName,
		OwnerUserID: e.OwnerUserID,
		RepoURL:     e.RepoURL,
		CreatedAt:   e.CreatedAt.Format(time.DateTime),
		UpdatedAt:   e.UpdatedAt.Format(time.DateTime),
	}
}

func envToDTO(e *service.ServiceEnv) *ServiceEnvDTO {
	dto := &ServiceEnvDTO{
		ID:                      e.ID,
		ServiceID:               e.ServiceID,
		EnvCode:                 e.EnvCode,
		ClusterName:             e.ClusterName,
		Namespace:               e.Namespace,
		AccessEndpoint:          e.AccessEndpoint,
		HealthcheckURL:          e.HealthcheckURL,
		LogSourceType:           e.LogSourceType,
		LogSourceConfig:         e.LogSourceConfig,
		HostID:                  e.HostID,
		DeployPath:              e.DeployPath,
		EnvVars:                 e.EnvVars,
		HealthCheckInterval:     e.HealthCheckInterval,
		HealthCheckTimeout:      e.HealthCheckTimeout,
		HealthCheckSuccessCodes: e.HealthCheckSuccessCodes,
		HealthCheckEnabled:      e.HealthCheckEnabled,
		HealthStatus:            e.HealthStatus,
		HealthLastMessage:       e.HealthLastMessage,
		JenkinsJobs:             e.JenkinsJobs,
		CreatedAt:               e.CreatedAt.Format(time.DateTime),
		UpdatedAt:               e.UpdatedAt.Format(time.DateTime),
	}
	if e.HealthLastCheckedAt != nil {
		s := e.HealthLastCheckedAt.Format(time.DateTime)
		dto.HealthLastCheckedAt = &s
	}
	return dto
}

// ============================
// UseCase
// ============================

type UseCase struct {
	serviceRepo service.ServiceRepository
	envRepo     service.ServiceEnvRepository
}

func NewUseCase(
	serviceRepo service.ServiceRepository,
	envRepo service.ServiceEnvRepository,
) *UseCase {
	return &UseCase{
		serviceRepo: serviceRepo,
		envRepo:     envRepo,
	}
}

// CreateService 创建服务
func (uc *UseCase) CreateService(ctx context.Context, cmd CreateServiceCmd) (*ServiceDTO, error) {
	// 检查 service_key 唯一性
	existing, _ := uc.serviceRepo.FindByKey(ctx, cmd.ServiceKey)
	if existing != nil {
		return nil, service.ErrServiceKeyDuplicated
	}

	entity := &service.Service{
		ID:          uuid.New().String(),
		ServiceKey:  cmd.ServiceKey,
		ServiceName: cmd.ServiceName,
		OwnerUserID: cmd.OwnerUserID,
		RepoURL:     cmd.RepoURL,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := uc.serviceRepo.Save(ctx, entity); err != nil {
		return nil, fmt.Errorf("保存服务失败: %w", err)
	}

	return serviceToDTO(entity), nil
}

// UpdateService 更新服务
func (uc *UseCase) UpdateService(ctx context.Context, cmd UpdateServiceCmd) (*ServiceDTO, error) {
	entity, err := uc.serviceRepo.FindByID(ctx, cmd.ServiceID)
	if err != nil {
		return nil, service.ErrServiceNotFound
	}

	if cmd.ServiceName != "" {
		entity.ServiceName = cmd.ServiceName
	}
	if cmd.OwnerUserID != "" {
		entity.OwnerUserID = cmd.OwnerUserID
	}
	if cmd.RepoURL != "" {
		entity.RepoURL = cmd.RepoURL
	}
	entity.UpdatedAt = time.Now()

	if err := uc.serviceRepo.Save(ctx, entity); err != nil {
		return nil, fmt.Errorf("更新服务失败: %w", err)
	}

	return serviceToDTO(entity), nil
}

// GetService 获取单个服务
func (uc *UseCase) GetService(ctx context.Context, id string) (*ServiceDTO, error) {
	entity, err := uc.serviceRepo.FindByID(ctx, id)
	if err != nil {
		return nil, service.ErrServiceNotFound
	}
	return serviceToDTO(entity), nil
}

// ListServices 服务列表
func (uc *UseCase) ListServices(ctx context.Context, q ServiceQueryCmd) ([]*ServiceDTO, int64, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}

	entities, total, err := uc.serviceRepo.Find(ctx, service.ServiceQuery{
		Keyword:  q.Keyword,
		Page:     q.Page,
		PageSize: q.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]*ServiceDTO, len(entities))
	for i, e := range entities {
		dtos[i] = serviceToDTO(e)
	}
	return dtos, total, nil
}

// CreateEnv 创建服务环境
func (uc *UseCase) CreateEnv(ctx context.Context, cmd CreateEnvCmd) (*ServiceEnvDTO, error) {
	// 验证服务存在
	_, err := uc.serviceRepo.FindByID(ctx, cmd.ServiceID)
	if err != nil {
		return nil, service.ErrServiceNotFound
	}

	// Ensure LogSourceConfig is valid JSON for jsonb column
	if cmd.LogSourceConfig == "" {
		cmd.LogSourceConfig = "{}"
	}
	if cmd.EnvVars == "" {
		cmd.EnvVars = "{}"
	}
	healthCheckEnabled := true
	if cmd.HealthCheckEnabled != nil {
		healthCheckEnabled = *cmd.HealthCheckEnabled
	}
	interval := cmd.HealthCheckInterval
	if interval <= 0 {
		interval = 60
	}
	timeout := cmd.HealthCheckTimeout
	if timeout <= 0 {
		timeout = 10
	}
	successCodes := cmd.HealthCheckSuccessCodes
	if successCodes == "" {
		successCodes = "200"
	}

	entity := &service.ServiceEnv{
		ID:                      uuid.New().String(),
		ServiceID:               cmd.ServiceID,
		EnvCode:                 cmd.EnvCode,
		ClusterName:             cmd.ClusterName,
		Namespace:               cmd.Namespace,
		AccessEndpoint:          cmd.AccessEndpoint,
		HealthcheckURL:          cmd.HealthcheckURL,
		LogSourceType:           cmd.LogSourceType,
		LogSourceConfig:         cmd.LogSourceConfig,
		HostID:                  cmd.HostID,
		DeployPath:              cmd.DeployPath,
		EnvVars:                 cmd.EnvVars,
		HealthCheckInterval:     interval,
		HealthCheckTimeout:      timeout,
		HealthCheckSuccessCodes: successCodes,
		HealthCheckEnabled:      healthCheckEnabled,
		HealthStatus:            "unknown",
		JenkinsJobs:             cmd.JenkinsJobs,
		CreatedAt:               time.Now(),
		UpdatedAt:               time.Now(),
	}

	if err := uc.envRepo.Save(ctx, entity); err != nil {
		return nil, fmt.Errorf("创建环境失败: %w", err)
	}

	return envToDTO(entity), nil
}

// UpdateEnv 更新服务环境
func (uc *UseCase) UpdateEnv(ctx context.Context, cmd UpdateEnvCmd) (*ServiceEnvDTO, error) {
	entity, err := uc.envRepo.FindByID(ctx, cmd.EnvID)
	if err != nil {
		return nil, service.ErrServiceEnvNotFound
	}

	if cmd.HostID != "" {
		entity.HostID = cmd.HostID
	}
	if cmd.DeployPath != "" {
		entity.DeployPath = cmd.DeployPath
	}
	if cmd.EnvVars != "" {
		entity.EnvVars = cmd.EnvVars
	}
	if cmd.AccessEndpoint != "" {
		entity.AccessEndpoint = cmd.AccessEndpoint
	}
	if cmd.HealthcheckURL != "" {
		entity.HealthcheckURL = cmd.HealthcheckURL
	}
	if cmd.LogSourceType != "" {
		entity.LogSourceType = cmd.LogSourceType
	}
	if cmd.LogSourceConfig != "" {
		entity.LogSourceConfig = cmd.LogSourceConfig
	}
	if cmd.HealthCheckInterval > 0 {
		entity.HealthCheckInterval = cmd.HealthCheckInterval
	}
	if cmd.HealthCheckTimeout > 0 {
		entity.HealthCheckTimeout = cmd.HealthCheckTimeout
	}
	if cmd.HealthCheckSuccessCodes != "" {
		entity.HealthCheckSuccessCodes = cmd.HealthCheckSuccessCodes
	}
	if cmd.HealthCheckEnabled != nil {
		entity.HealthCheckEnabled = *cmd.HealthCheckEnabled
	}
	if cmd.JenkinsJobs != "" {
		entity.JenkinsJobs = cmd.JenkinsJobs
	}
	entity.UpdatedAt = time.Now()

	if err := uc.envRepo.Save(ctx, entity); err != nil {
		return nil, fmt.Errorf("更新环境失败: %w", err)
	}
	return envToDTO(entity), nil
}

// DeleteEnv 删除服务环境
func (uc *UseCase) DeleteEnv(ctx context.Context, envID string) error {
	if _, err := uc.envRepo.FindByID(ctx, envID); err != nil {
		return service.ErrServiceEnvNotFound
	}
	return uc.envRepo.Delete(ctx, envID)
}

// ListEnvs 列出服务环境
func (uc *UseCase) ListEnvs(ctx context.Context, serviceID string) ([]*ServiceEnvDTO, error) {
	entities, err := uc.envRepo.FindByServiceID(ctx, serviceID)
	if err != nil {
		return nil, err
	}

	dtos := make([]*ServiceEnvDTO, len(entities))
	for i, e := range entities {
		dtos[i] = envToDTO(e)
	}
	return dtos, nil
}

// DeleteService 删除服务
func (uc *UseCase) DeleteService(ctx context.Context, id string) error {
	if _, err := uc.serviceRepo.FindByID(ctx, id); err != nil {
		return service.ErrServiceNotFound
	}
	return uc.serviceRepo.Delete(ctx, id)
}
