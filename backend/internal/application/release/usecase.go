// Package release 发布模块应用层 - 用例编排
// 核心链路: 创建发布单 → 执行 → 回滚
package release

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"ops-hub/internal/domain/host"
	"ops-hub/internal/domain/notification"
	"ops-hub/internal/domain/release"
	"ops-hub/internal/domain/service"
	"strings"
	"time"

	"github.com/google/uuid"
)

const ErrCodeProdTargetBlocked = 3004

var ErrProdTargetBlocked = errors.New("prod target blocked")

type prodTargetBlockedError struct {
	message string
}

func (e prodTargetBlockedError) Error() string {
	return e.message
}

func (e prodTargetBlockedError) Is(target error) bool {
	return target == ErrProdTargetBlocked
}

// NotificationDispatcher 通知分发接口（避免循环依赖）
type NotificationDispatcher interface {
	Dispatch(ctx context.Context, eventType notification.EventType, fingerprint, title, content string)
}

// ============================
// DTO
// ============================

type CreateReleaseCmd struct {
	ServiceID       string            `json:"service_id" binding:"required"`
	EnvID           string            `json:"env_id"`
	TenantID        string            `json:"tenant_id"`
	TargetVersionID string            `json:"target_version_id"`
	Strategy        string            `json:"strategy"`
	JenkinsParams   map[string]string `json:"jenkins_params"`
	ForceProdTarget bool              `json:"force_prod_target"`
	AdminPassword   string            `json:"admin_password"`
	OperatorID      string            `json:"operator_id"`
	IdempotencyKey  string            `json:"idempotency_key"`
}

type ReleaseQueryCmd struct {
	ServiceID   string `form:"service_id"`
	EnvID       string `form:"env_id"`
	TenantID    string `form:"tenant_id"`
	Status      string `form:"status"`
	ReleaseType string `form:"release_type"`
	Page        int    `form:"page"`
	PageSize    int    `form:"page_size"`
}

type ReleaseDTO struct {
	ID              string  `json:"id"`
	ServiceID       string  `json:"service_id"`
	EnvID           string  `json:"env_id"`
	TenantID        string  `json:"tenant_id"`
	TargetVersionID string  `json:"target_version_id"`
	PrevVersionID   string  `json:"prev_version_id"`
	ReleaseType     string  `json:"release_type"`
	Strategy        string  `json:"strategy"`
	Status          string  `json:"status"`
	ErrorMessage    string  `json:"error_message"`
	OperatorID      string  `json:"operator_id"`
	JenkinsParams   string  `json:"jenkins_params"`
	JenkinsBuildNo  int     `json:"jenkins_build_no"`
	StartedAt       *string `json:"started_at"`
	EndedAt         *string `json:"ended_at"`
	CreatedAt       string  `json:"created_at"`
	EnvCode         string  `json:"env_code"`
}

func releaseToDTO(e *release.ReleaseRecord) *ReleaseDTO {
	dto := &ReleaseDTO{
		ID:              e.ID,
		ServiceID:       e.ServiceID,
		EnvID:           e.EnvID,
		TenantID:        e.TenantID,
		TargetVersionID: e.TargetVersionID,
		PrevVersionID:   e.PrevVersionID,
		ReleaseType:     string(e.ReleaseType),
		Strategy:        e.Strategy,
		Status:          string(e.Status),
		ErrorMessage:    e.ErrorMessage,
		OperatorID:      e.OperatorID,
		JenkinsParams:   e.JenkinsParams,
		JenkinsBuildNo:  e.JenkinsBuildNo,
		CreatedAt:       e.CreatedAt.Format(time.DateTime),
	}
	if e.StartedAt != nil {
		s := e.StartedAt.Format(time.DateTime)
		dto.StartedAt = &s
	}
	if e.EndedAt != nil {
		s := e.EndedAt.Format(time.DateTime)
		dto.EndedAt = &s
	}
	return dto
}

// ReleaseStepDTO 步骤日志 DTO
type ReleaseStepDTO struct {
	ID          string  `json:"id"`
	StepOrder   int     `json:"step_order"`
	StepName    string  `json:"step_name"`
	StepStatus  string  `json:"step_status"`
	StartedAt   *string `json:"started_at"`
	EndedAt     *string `json:"ended_at"`
	DurationMs  int     `json:"duration_ms"`
	Output      string  `json:"output"`
	ErrorOutput string  `json:"error_output"`
}

func stepToDTO(s *release.ReleaseStepLog) *ReleaseStepDTO {
	dto := &ReleaseStepDTO{
		ID:          s.ID,
		StepOrder:   s.StepOrder,
		StepName:    s.StepName,
		StepStatus:  string(s.StepStatus),
		DurationMs:  s.DurationMs,
		Output:      s.Output,
		ErrorOutput: s.ErrorOutput,
	}
	if s.StartedAt != nil {
		t := s.StartedAt.Format(time.DateTime)
		dto.StartedAt = &t
	}
	if s.EndedAt != nil {
		t := s.EndedAt.Format(time.DateTime)
		dto.EndedAt = &t
	}
	return dto
}

// ============================
// UseCase
// ============================

type UseCase struct {
	releaseRepo     release.ReleaseRecordRepository
	stepRepo        release.ReleaseStepLogRepository
	serviceRepo     service.ServiceRepository
	envRepo         service.ServiceEnvRepository
	hostRepo        host.HostRepository
	executor        release.ReleaseExecutor
	jenkinsExecutor release.ReleaseExecutor // Jenkins 执行器（可选）
	notifier        NotificationDispatcher  // 通知分发（可选）
	adminPassword   string
}

func NewUseCase(
	releaseRepo release.ReleaseRecordRepository,
	stepRepo release.ReleaseStepLogRepository,
	serviceRepo service.ServiceRepository,
	envRepo service.ServiceEnvRepository,
	hostRepo host.HostRepository,
	executor release.ReleaseExecutor,
) *UseCase {
	return &UseCase{
		releaseRepo: releaseRepo,
		stepRepo:    stepRepo,
		serviceRepo: serviceRepo,
		envRepo:     envRepo,
		hostRepo:    hostRepo,
		executor:    executor,
	}
}

// SetJenkinsExecutor 设置 Jenkins 执行器
func (uc *UseCase) SetJenkinsExecutor(exec release.ReleaseExecutor) {
	uc.jenkinsExecutor = exec
}

// SetNotifier 设置通知分发器
func (uc *UseCase) SetNotifier(n NotificationDispatcher) {
	uc.notifier = n
}

// SetAdminPassword 设置强制操作的管理员密码。密码只用于请求校验，不写入发布记录。
func (uc *UseCase) SetAdminPassword(password string) {
	uc.adminPassword = password
}

// CreateRelease 创建发布单
func (uc *UseCase) CreateRelease(ctx context.Context, cmd CreateReleaseCmd) (*ReleaseDTO, error) {
	// 1. 校验服务存在
	if _, err := uc.serviceRepo.FindByID(ctx, cmd.ServiceID); err != nil {
		return nil, service.ErrServiceNotFound
	}

	var selectedEnv *service.ServiceEnv

	// 判断是否 Jenkins 模式：优先看 strategy 参数，否则看环境是否配置了 jenkins_jobs
	isJenkins := cmd.Strategy == "jenkins"
	if !isJenkins && cmd.EnvID != "" {
		env, envErr := uc.envRepo.FindByID(ctx, cmd.EnvID)
		if envErr == nil {
			selectedEnv = env
			if env.JenkinsJobs != "" && env.JenkinsJobs != "[]" {
				isJenkins = true
			}
		}
	}

	// 2. 非 Jenkins 模式需要校验环境和版本
	if !isJenkins {
		if cmd.EnvID == "" || cmd.TargetVersionID == "" {
			return nil, fmt.Errorf("脚本发布模式必须指定环境和目标版本")
		}
		if selectedEnv == nil {
			env, err := uc.envRepo.FindByID(ctx, cmd.EnvID)
			if err != nil {
				return nil, service.ErrServiceEnvNotFound
			}
			selectedEnv = env
		}
		// 检查是否已有进行中的发布
		active, _ := uc.releaseRepo.FindActiveByServiceEnv(ctx, cmd.ServiceID, cmd.EnvID)
		if active != nil {
			return nil, release.ErrReleaseAlreadyActive
		}
	}
	if selectedEnv == nil && cmd.EnvID != "" {
		env, err := uc.envRepo.FindByID(ctx, cmd.EnvID)
		if err != nil {
			return nil, service.ErrServiceEnvNotFound
		}
		selectedEnv = env
	}

	strategy := cmd.Strategy
	if strategy == "" {
		if isJenkins {
			strategy = "jenkins"
		} else {
			strategy = "default"
		}
	}

	if err := uc.validateProdTarget(ctx, cmd, strategy, selectedEnv); err != nil {
		return nil, err
	}

	// Jenkins 参数序列化
	jenkinsParamsJSON := "{}"
	if len(cmd.JenkinsParams) > 0 {
		if data, e := json.Marshal(cmd.JenkinsParams); e == nil {
			jenkinsParamsJSON = string(data)
		}
	}

	// 空幂等键不做重复校验，自动生成唯一值
	idempotencyKey := cmd.IdempotencyKey
	if idempotencyKey == "" {
		idempotencyKey = uuid.New().String()
	}

	entity := &release.ReleaseRecord{
		ID:              uuid.New().String(),
		ServiceID:       cmd.ServiceID,
		EnvID:           cmd.EnvID,
		TenantID:        cmd.TenantID,
		TargetVersionID: cmd.TargetVersionID,
		ReleaseType:     release.ReleaseTypeDeploy,
		Strategy:        strategy,
		Status:          release.ReleaseStatusPending,
		OperatorID:      cmd.OperatorID,
		IdempotencyKey:  idempotencyKey,
		JenkinsParams:   jenkinsParamsJSON,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := uc.releaseRepo.Save(ctx, entity); err != nil {
		return nil, fmt.Errorf("创建发布单失败: %w", err)
	}

	return releaseToDTO(entity), nil
}

func (uc *UseCase) validateProdTarget(ctx context.Context, cmd CreateReleaseCmd, strategy string, env *service.ServiceEnv) error {
	if uc.hostRepo == nil || !isRiskyDataOperation(strategy, cmd.JenkinsParams, env) {
		return nil
	}
	prodHosts, err := uc.findProdTargets(ctx, cmd.JenkinsParams, env)
	if err != nil {
		return err
	}
	if len(prodHosts) == 0 {
		return nil
	}
	if !cmd.ForceProdTarget {
		return prodTargetBlockedError{
			message: fmt.Sprintf("目标主机 %s 已标记为 Prod，数据库迁移/对象同步不能默认使用线上服务器作为目标；如需强制迁移，请输入 admin 密码", formatHostNames(prodHosts)),
		}
	}
	if strings.TrimSpace(uc.adminPassword) == "" {
		return fmt.Errorf("未配置 OPSHUB_ADMIN_PASSWORD，无法强制使用 Prod 目标")
	}
	if cmd.AdminPassword != uc.adminPassword {
		return fmt.Errorf("admin 密码错误，无法强制使用 Prod 目标")
	}
	return nil
}

func isRiskyDataOperation(strategy string, params map[string]string, env *service.ServiceEnv) bool {
	var parts []string
	parts = append(parts, strategy)
	if env != nil {
		parts = append(parts, env.JenkinsJobs)
	}
	for k, v := range params {
		parts = append(parts, k, v)
	}
	text := strings.ToLower(strings.Join(parts, " "))
	keywords := []string{
		"数据库迁移", "对象同步", "db_migration", "database_migration", "data_migration",
		"db-migration", "database-migration", "data-migration", "object_sync", "object-sync",
		"object sync", "mysql_migration", "postgres_migration",
	}
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func (uc *UseCase) findProdTargets(ctx context.Context, params map[string]string, env *service.ServiceEnv) ([]*host.Host, error) {
	hosts, err := uc.hostRepo.FindByIsProd(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("校验 Prod 目标失败: %w", err)
	}
	targetValues := collectTargetValues(params)
	matched := make([]*host.Host, 0)
	seen := map[string]bool{}
	for _, h := range hosts {
		if env != nil && env.HostID != "" && env.HostID == h.ID {
			matched = append(matched, h)
			seen[h.ID] = true
			continue
		}
		if seen[h.ID] {
			continue
		}
		if matchesTargetHost(targetValues, h) {
			matched = append(matched, h)
			seen[h.ID] = true
		}
	}
	return matched, nil
}

func collectTargetValues(params map[string]string) []string {
	values := make([]string, 0)
	for k, v := range params {
		key := strings.ToLower(k)
		if strings.Contains(key, "target") ||
			strings.Contains(key, "dest") ||
			strings.Contains(key, "dst") ||
			strings.Contains(key, "目标") ||
			strings.Contains(key, "目的") {
			values = append(values, v)
		}
	}
	return values
}

func matchesTargetHost(values []string, h *host.Host) bool {
	for _, value := range values {
		v := strings.ToLower(strings.TrimSpace(value))
		if v == "" {
			continue
		}
		if v == strings.ToLower(h.ID) ||
			v == strings.ToLower(h.Name) ||
			v == strings.ToLower(h.HostAddress) {
			return true
		}
	}
	return false
}

func formatHostNames(hosts []*host.Host) string {
	names := make([]string, 0, len(hosts))
	for _, h := range hosts {
		names = append(names, fmt.Sprintf("%s(%s)", h.Name, h.HostAddress))
	}
	return strings.Join(names, "、")
}

// ExecuteRelease 执行发布（含步骤日志）
func (uc *UseCase) ExecuteRelease(ctx context.Context, releaseID string) (*ReleaseDTO, error) {
	record, err := uc.releaseRepo.FindByID(ctx, releaseID)
	if err != nil {
		return nil, release.ErrReleaseNotFound
	}

	if err := record.MarkRunning(); err != nil {
		return nil, err
	}
	if err := uc.releaseRepo.Update(ctx, record); err != nil {
		return nil, fmt.Errorf("更新发布状态失败: %w", err)
	}

	svc, err := uc.serviceRepo.FindByID(ctx, record.ServiceID)
	if err != nil {
		return nil, service.ErrServiceNotFound
	}

	isJenkins := record.Strategy == "jenkins"

	if isJenkins {
		// Jenkins 模式：异步执行，立即返回 running 状态
		go uc.executeJenkins(record, svc)
		return releaseToDTO(record), nil
	}

	// 脚本发布已下线：仅支持 Jenkins
	record.MarkFailed("当前仅支持 Jenkins 发布，请为环境配置 jenkins_jobs")
	_ = uc.releaseRepo.Update(ctx, record)
	return releaseToDTO(record), nil
}

// executeJenkins 异步执行 Jenkins 构建（在 goroutine 中调用）
func (uc *UseCase) executeJenkins(record *release.ReleaseRecord, svc *service.Service) {
	ctx := context.Background()

	if uc.jenkinsExecutor == nil {
		record.MarkFailed("Jenkins 执行器未配置")
		_ = uc.releaseRepo.Update(ctx, record)
		return
	}

	// 从环境获取 jenkins_job
	jenkinsJob := ""
	if record.EnvID != "" {
		if env, err := uc.envRepo.FindByID(ctx, record.EnvID); err == nil {
			type jj struct {
				Name string `json:"name"`
				Job  string `json:"job"`
			}
			var jobs []jj
			if json.Unmarshal([]byte(env.JenkinsJobs), &jobs) == nil && len(jobs) > 0 {
				jenkinsJob = jobs[0].Job
			}
		}
	}

	step := &release.ReleaseStepLog{
		ReleaseID: record.ID,
		StepOrder: 1,
		StepName:  "触发 Jenkins 构建",
		CreatedAt: time.Now(),
	}
	step.MarkRunning()
	_ = uc.stepRepo.Save(ctx, step)

	task := &release.ExecutionTask{
		ReleaseID:  record.ID,
		ServiceKey: svc.ServiceKey,
		Scripts: map[string]string{
			"jenkins_job":    jenkinsJob,
			"jenkins_params": record.JenkinsParams,
		},
		OnOutput: func(output string) {
			step.Output = output
			_ = uc.stepRepo.Update(ctx, step)
		},
	}

	result, execErr := uc.jenkinsExecutor.Execute(ctx, task)
	if execErr != nil {
		step.MarkFailed("", execErr.Error())
	} else if result != nil && !result.Success {
		step.MarkFailed(result.Output, result.Error)
	} else if result != nil {
		step.MarkSuccess(result.Output)
	} else {
		step.MarkSuccess("Jenkins 构建完成")
	}
	_ = uc.stepRepo.Update(ctx, step)

	// 回写构建号
	if result != nil && result.BuildNumber > 0 {
		record.JenkinsBuildNo = result.BuildNumber
	}

	if execErr != nil {
		record.MarkFailed(execErr.Error())
	} else if result != nil && !result.Success {
		record.MarkFailed(result.Error)
	} else {
		record.MarkSuccess()
	}
	_ = uc.releaseRepo.Update(ctx, record)

	// 发布通知
	uc.dispatchNotification(ctx, record, svc)
}

// executeScript 已下线：当前仅支持 Jenkins 发布。函数保留为空骨架以减少上层 import 影响，
// 待恢复脚本/SSH 部署能力时按上一版本恢复实现。
//
// 历史实现：在主机 host_id 对应凭证下经 SSHExecutor 同步执行部署脚本 + 健康检查。

// GetReleaseSteps 获取发布步骤日志
func (uc *UseCase) GetReleaseSteps(ctx context.Context, releaseID string) ([]*ReleaseStepDTO, error) {
	if _, err := uc.releaseRepo.FindByID(ctx, releaseID); err != nil {
		return nil, release.ErrReleaseNotFound
	}
	steps, err := uc.stepRepo.FindByReleaseID(ctx, releaseID)
	if err != nil {
		return nil, err
	}
	dtos := make([]*ReleaseStepDTO, len(steps))
	for i, s := range steps {
		dtos[i] = stepToDTO(s)
	}
	return dtos, nil
}

// RollbackRelease 回滚发布
func (uc *UseCase) RollbackRelease(ctx context.Context, releaseID, operatorID string) (*ReleaseDTO, error) {
	original, err := uc.releaseRepo.FindByID(ctx, releaseID)
	if err != nil {
		return nil, release.ErrReleaseNotFound
	}

	if !original.CanRollback() {
		return nil, release.ErrInvalidReleaseState
	}

	// 查找上一个成功的发布版本
	lastSuccess, err := uc.releaseRepo.FindLastSuccess(ctx, original.ServiceID, original.EnvID)
	if err != nil || lastSuccess == nil || lastSuccess.ID == original.ID {
		return nil, release.ErrNoPreviousVersion
	}

	// 创建回滚记录
	rollback := &release.ReleaseRecord{
		ID:              uuid.New().String(),
		ServiceID:       original.ServiceID,
		EnvID:           original.EnvID,
		TenantID:        original.TenantID,
		TargetVersionID: lastSuccess.TargetVersionID,
		PrevVersionID:   original.TargetVersionID,
		ReleaseType:     release.ReleaseTypeRollback,
		Strategy:        original.Strategy,
		Status:          release.ReleaseStatusPending,
		OperatorID:      operatorID,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := uc.releaseRepo.Save(ctx, rollback); err != nil {
		return nil, fmt.Errorf("创建回滚记录失败: %w", err)
	}

	return releaseToDTO(rollback), nil
}

// GetRelease 获取发布详情
func (uc *UseCase) GetRelease(ctx context.Context, id string) (*ReleaseDTO, error) {
	entity, err := uc.releaseRepo.FindByID(ctx, id)
	if err != nil {
		return nil, release.ErrReleaseNotFound
	}
	return releaseToDTO(entity), nil
}

// DeleteRelease 删除发布记录
func (uc *UseCase) DeleteRelease(ctx context.Context, id string) error {
	entity, err := uc.releaseRepo.FindByID(ctx, id)
	if err != nil {
		return release.ErrReleaseNotFound
	}
	if entity.Status == release.ReleaseStatusRunning {
		return fmt.Errorf("执行中的发布不能删除")
	}
	return uc.releaseRepo.Delete(ctx, id)
}

// ListReleases 发布记录列表
func (uc *UseCase) ListReleases(ctx context.Context, q ReleaseQueryCmd) ([]*ReleaseDTO, int64, error) {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}

	entities, total, err := uc.releaseRepo.Find(ctx, release.ReleaseQuery{
		ServiceID:   q.ServiceID,
		EnvID:       q.EnvID,
		TenantID:    q.TenantID,
		Status:      q.Status,
		ReleaseType: q.ReleaseType,
		Page:        q.Page,
		PageSize:    q.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]*ReleaseDTO, len(entities))
	for i, e := range entities {
		dtos[i] = releaseToDTO(e)
		if e.EnvID != "" {
			if env, err := uc.envRepo.FindByID(ctx, e.EnvID); err == nil {
				dtos[i].EnvCode = env.EnvCode
			}
		}
	}
	return dtos, total, nil
}

// dispatchNotification 发布完成后分发通知
func (uc *UseCase) dispatchNotification(ctx context.Context, record *release.ReleaseRecord, svc *service.Service) {
	if uc.notifier == nil {
		return
	}

	var eventType notification.EventType
	switch release.ReleaseStatus(record.Status) {
	case release.ReleaseStatusSuccess:
		eventType = notification.EventDeploySuccess
	case release.ReleaseStatusFailed:
		eventType = notification.EventDeployFail
	default:
		return
	}

	fingerprint := record.ServiceID + ":" + record.ID
	title := fmt.Sprintf("[OpsHub发布] %s %s", svc.ServiceName, record.Status)
	content := fmt.Sprintf("服务: %s\n策略: %s\n状态: %s", svc.ServiceName, record.Strategy, record.Status)
	if record.ErrorMessage != "" {
		content += fmt.Sprintf("\n错误: %s", record.ErrorMessage)
	}
	if record.JenkinsBuildNo > 0 {
		content += fmt.Sprintf("\nJenkins Build: #%d", record.JenkinsBuildNo)
	}

	uc.notifier.Dispatch(ctx, eventType, fingerprint, title, content)
}
