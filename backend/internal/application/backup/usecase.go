package backup

import (
	"context"
	"fmt"
	"ops-hub/internal/domain/backup"
	"ops-hub/internal/infrastructure/crypto"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

// ============================
// DTO
// ============================

type CreateTaskCmd struct {
	Name          string `json:"name" binding:"required"`
	BackupType    string `json:"backup_type" binding:"required"`
	CronExpr      string `json:"cron_expr" binding:"required"`
	Enabled       bool   `json:"enabled"`
	DBHost        string `json:"db_host" binding:"required"`
	DBPort        int    `json:"db_port" binding:"required"`
	DBUser        string `json:"db_user" binding:"required"`
	DBPassword    string `json:"db_password" binding:"required"`
	DBName        string `json:"db_name"`
	TargetHostID  string `json:"target_host_id" binding:"required"`
	TargetPath    string `json:"target_path" binding:"required"`
	RetentionDays int    `json:"retention_days"`
	Description   string `json:"description"`
}

type UpdateTaskCmd struct {
	ID            string  `json:"-"`
	Name          string  `json:"name"`
	CronExpr      string  `json:"cron_expr"`
	Enabled       *bool   `json:"enabled"`
	DBHost        string  `json:"db_host"`
	DBPort        int     `json:"db_port"`
	DBUser        string  `json:"db_user"`
	DBPassword    string  `json:"db_password"`
	DBName        *string `json:"db_name"`
	TargetHostID  string  `json:"target_host_id"`
	TargetPath    string  `json:"target_path"`
	RetentionDays int     `json:"retention_days"`
	Description   string  `json:"description"`
}

type TaskQueryCmd struct {
	Keyword  string `form:"keyword"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type RecordQueryCmd struct {
	TaskID   string `form:"task_id"`
	Status   string `form:"status"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type CreateMigrationTaskCmd struct {
	Name           string `json:"name" binding:"required"`
	DBType         string `json:"db_type" binding:"required"`
	SourceHost     string `json:"source_host" binding:"required"`
	SourcePort     int    `json:"source_port" binding:"required"`
	SourceUser     string `json:"source_user" binding:"required"`
	SourcePassword string `json:"source_password" binding:"required"`
	DBNames        string `json:"db_names" binding:"required"`
	TargetHost     string `json:"target_host" binding:"required"`
	TargetPort     int    `json:"target_port" binding:"required"`
	TargetUser     string `json:"target_user" binding:"required"`
	TargetPassword string `json:"target_password" binding:"required"`
	Mode           string `json:"mode" binding:"required"`
	Description    string `json:"description"`
}

type UpdateMigrationTaskCmd struct {
	ID             string `json:"-"`
	Name           string `json:"name"`
	DBType         string `json:"db_type"`
	SourceHost     string `json:"source_host"`
	SourcePort     int    `json:"source_port"`
	SourceUser     string `json:"source_user"`
	SourcePassword string `json:"source_password"`
	DBNames        string `json:"db_names"`
	TargetHost     string `json:"target_host"`
	TargetPort     int    `json:"target_port"`
	TargetUser     string `json:"target_user"`
	TargetPassword string `json:"target_password"`
	Mode           string `json:"mode"`
	Description    string `json:"description"`
}

type MigrationTaskQueryCmd struct {
	Keyword  string `form:"keyword"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type MigrationRecordQueryCmd struct {
	TaskID   string `form:"task_id"`
	Status   string `form:"status"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type ExecuteMigrationCmd struct {
	ConfirmOverwrite bool `json:"confirm_overwrite"`
}

type CreateObjectSyncTaskCmd struct {
	Name            string `json:"name" binding:"required"`
	SourceEndpoint  string `json:"source_endpoint" binding:"required"`
	SourceRegion    string `json:"source_region"`
	SourceBucket    string `json:"source_bucket" binding:"required"`
	SourcePath      string `json:"source_path"`
	SourceAccessKey string `json:"source_access_key" binding:"required"`
	SourceSecretKey string `json:"source_secret_key" binding:"required"`
	SourceUseSSL    bool   `json:"source_use_ssl"`
	TargetEndpoint  string `json:"target_endpoint" binding:"required"`
	TargetRegion    string `json:"target_region"`
	TargetBucket    string `json:"target_bucket" binding:"required"`
	TargetPath      string `json:"target_path"`
	TargetAccessKey string `json:"target_access_key" binding:"required"`
	TargetSecretKey string `json:"target_secret_key" binding:"required"`
	TargetUseSSL    bool   `json:"target_use_ssl"`
	Mode            string `json:"mode" binding:"required"`
	Description     string `json:"description"`
}

type UpdateObjectSyncTaskCmd struct {
	ID              string `json:"-"`
	Name            string `json:"name"`
	SourceEndpoint  string `json:"source_endpoint"`
	SourceRegion    string `json:"source_region"`
	SourceBucket    string `json:"source_bucket"`
	SourcePath      string `json:"source_path"`
	SourceAccessKey string `json:"source_access_key"`
	SourceSecretKey string `json:"source_secret_key"`
	SourceUseSSL    *bool  `json:"source_use_ssl"`
	TargetEndpoint  string `json:"target_endpoint"`
	TargetRegion    string `json:"target_region"`
	TargetBucket    string `json:"target_bucket"`
	TargetPath      string `json:"target_path"`
	TargetAccessKey string `json:"target_access_key"`
	TargetSecretKey string `json:"target_secret_key"`
	TargetUseSSL    *bool  `json:"target_use_ssl"`
	Mode            string `json:"mode"`
	Description     string `json:"description"`
}

type ObjectSyncTaskQueryCmd struct {
	Keyword  string `form:"keyword"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type ObjectSyncRecordQueryCmd struct {
	TaskID   string `form:"task_id"`
	Status   string `form:"status"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type ExecuteObjectSyncCmd struct {
	ConfirmOverwrite bool `json:"confirm_overwrite"`
}

type TaskDTO struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	BackupType    string  `json:"backup_type"`
	CronExpr      string  `json:"cron_expr"`
	Enabled       bool    `json:"enabled"`
	DBHost        string  `json:"db_host"`
	DBPort        int     `json:"db_port"`
	DBUser        string  `json:"db_user"`
	DBName        string  `json:"db_name"`
	TargetHostID  string  `json:"target_host_id"`
	TargetPath    string  `json:"target_path"`
	RetentionDays int     `json:"retention_days"`
	Description   string  `json:"description"`
	LastRunAt     *string `json:"last_run_at"`
	LastRunStatus string  `json:"last_run_status"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type RecordDTO struct {
	ID         string  `json:"id"`
	TaskID     string  `json:"task_id"`
	TaskName   string  `json:"task_name"`
	Status     string  `json:"status"`
	FileName   string  `json:"file_name"`
	FileSize   int64   `json:"file_size"`
	Duration   int64   `json:"duration"`
	Error      string  `json:"error"`
	StartedAt  string  `json:"started_at"`
	FinishedAt *string `json:"finished_at"`
}

type MigrationTaskDTO struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	DBType        string  `json:"db_type"`
	SourceHost    string  `json:"source_host"`
	SourcePort    int     `json:"source_port"`
	SourceUser    string  `json:"source_user"`
	DBNames       string  `json:"db_names"`
	TargetHost    string  `json:"target_host"`
	TargetPort    int     `json:"target_port"`
	TargetUser    string  `json:"target_user"`
	Mode          string  `json:"mode"`
	Description   string  `json:"description"`
	LastRunAt     *string `json:"last_run_at"`
	LastRunStatus string  `json:"last_run_status"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type MigrationRecordDTO struct {
	ID         string                    `json:"id"`
	TaskID     string                    `json:"task_id"`
	TaskName   string                    `json:"task_name"`
	DBType     string                    `json:"db_type"`
	Mode       string                    `json:"mode"`
	Status     string                    `json:"status"`
	SourceHost string                    `json:"source_host"`
	TargetHost string                    `json:"target_host"`
	DBNames    string                    `json:"db_names"`
	Summary    string                    `json:"summary"`
	Error      string                    `json:"error"`
	StartedAt  string                    `json:"started_at"`
	FinishedAt *string                   `json:"finished_at"`
	Duration   int64                     `json:"duration"`
	Items      []*MigrationRecordItemDTO `json:"items,omitempty"`
}

type MigrationRecordItemDTO struct {
	ID         string  `json:"id"`
	RecordID   string  `json:"record_id"`
	DBName     string  `json:"db_name"`
	Action     string  `json:"action"`
	Status     string  `json:"status"`
	Message    string  `json:"message"`
	StartedAt  string  `json:"started_at"`
	FinishedAt *string `json:"finished_at"`
	Duration   int64   `json:"duration"`
}

type ObjectSyncTaskDTO struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	SourceEndpoint string  `json:"source_endpoint"`
	SourceRegion   string  `json:"source_region"`
	SourceBucket   string  `json:"source_bucket"`
	SourcePath     string  `json:"source_path"`
	SourceUseSSL   bool    `json:"source_use_ssl"`
	TargetEndpoint string  `json:"target_endpoint"`
	TargetRegion   string  `json:"target_region"`
	TargetBucket   string  `json:"target_bucket"`
	TargetPath     string  `json:"target_path"`
	TargetUseSSL   bool    `json:"target_use_ssl"`
	Mode           string  `json:"mode"`
	Description    string  `json:"description"`
	LastRunAt      *string `json:"last_run_at"`
	LastRunStatus  string  `json:"last_run_status"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

type ObjectSyncRecordDTO struct {
	ID           string                     `json:"id"`
	TaskID       string                     `json:"task_id"`
	TaskName     string                     `json:"task_name"`
	Mode         string                     `json:"mode"`
	Status       string                     `json:"status"`
	SourceBucket string                     `json:"source_bucket"`
	SourcePath   string                     `json:"source_path"`
	TargetBucket string                     `json:"target_bucket"`
	TargetPath   string                     `json:"target_path"`
	ObjectCount  int                        `json:"object_count"`
	SuccessCount int                        `json:"success_count"`
	SkippedCount int                        `json:"skipped_count"`
	FailedCount  int                        `json:"failed_count"`
	BytesTotal   int64                      `json:"bytes_total"`
	Summary      string                     `json:"summary"`
	Error        string                     `json:"error"`
	StartedAt    string                     `json:"started_at"`
	FinishedAt   *string                    `json:"finished_at"`
	Duration     int64                      `json:"duration"`
	Items        []*ObjectSyncRecordItemDTO `json:"items,omitempty"`
}

type ObjectSyncRecordItemDTO struct {
	ID         string  `json:"id"`
	RecordID   string  `json:"record_id"`
	SourceKey  string  `json:"source_key"`
	TargetKey  string  `json:"target_key"`
	Size       int64   `json:"size"`
	ETag       string  `json:"etag"`
	Action     string  `json:"action"`
	Status     string  `json:"status"`
	Message    string  `json:"message"`
	StartedAt  string  `json:"started_at"`
	FinishedAt *string `json:"finished_at"`
	Duration   int64   `json:"duration"`
}

func toTaskDTO(e *backup.BackupTask) *TaskDTO {
	dto := &TaskDTO{
		ID:            e.ID,
		Name:          e.Name,
		BackupType:    string(e.BackupType),
		CronExpr:      e.CronExpr,
		Enabled:       e.Enabled,
		DBHost:        e.DBHost,
		DBPort:        e.DBPort,
		DBUser:        e.DBUser,
		DBName:        e.DBName,
		TargetHostID:  e.TargetHostID,
		TargetPath:    e.TargetPath,
		RetentionDays: e.RetentionDays,
		Description:   e.Description,
		LastRunStatus: string(e.LastRunStatus),
		CreatedAt:     e.CreatedAt.Format(time.DateTime),
		UpdatedAt:     e.UpdatedAt.Format(time.DateTime),
	}
	if e.LastRunAt != nil {
		s := e.LastRunAt.Format(time.DateTime)
		dto.LastRunAt = &s
	}
	return dto
}

func toRecordDTO(e *backup.BackupRecord) *RecordDTO {
	dto := &RecordDTO{
		ID:        e.ID,
		TaskID:    e.TaskID,
		TaskName:  e.TaskName,
		Status:    string(e.Status),
		FileName:  e.FileName,
		FileSize:  e.FileSize,
		Duration:  e.Duration,
		Error:     e.Error,
		StartedAt: e.StartedAt.Format(time.DateTime),
	}
	if e.FinishedAt != nil {
		s := e.FinishedAt.Format(time.DateTime)
		dto.FinishedAt = &s
	}
	return dto
}

func toMigrationTaskDTO(e *backup.MigrationTask) *MigrationTaskDTO {
	dto := &MigrationTaskDTO{
		ID:            e.ID,
		Name:          e.Name,
		DBType:        string(e.DBType),
		SourceHost:    e.SourceHost,
		SourcePort:    e.SourcePort,
		SourceUser:    e.SourceUser,
		DBNames:       e.DBNames,
		TargetHost:    e.TargetHost,
		TargetPort:    e.TargetPort,
		TargetUser:    e.TargetUser,
		Mode:          string(e.Mode),
		Description:   e.Description,
		LastRunStatus: string(e.LastRunStatus),
		CreatedAt:     e.CreatedAt.Format(time.DateTime),
		UpdatedAt:     e.UpdatedAt.Format(time.DateTime),
	}
	if e.LastRunAt != nil {
		s := e.LastRunAt.Format(time.DateTime)
		dto.LastRunAt = &s
	}
	return dto
}

func toMigrationRecordDTO(e *backup.MigrationRecord) *MigrationRecordDTO {
	dto := &MigrationRecordDTO{
		ID:         e.ID,
		TaskID:     e.TaskID,
		TaskName:   e.TaskName,
		DBType:     string(e.DBType),
		Mode:       string(e.Mode),
		Status:     string(e.Status),
		SourceHost: e.SourceHost,
		TargetHost: e.TargetHost,
		DBNames:    e.DBNames,
		Summary:    e.Summary,
		Error:      e.Error,
		StartedAt:  e.StartedAt.Format(time.DateTime),
		Duration:   e.Duration,
	}
	if e.FinishedAt != nil {
		s := e.FinishedAt.Format(time.DateTime)
		dto.FinishedAt = &s
	}
	if len(e.Items) > 0 {
		dto.Items = make([]*MigrationRecordItemDTO, len(e.Items))
		for i, item := range e.Items {
			dto.Items[i] = toMigrationRecordItemDTO(item)
		}
	}
	return dto
}

func toMigrationRecordItemDTO(e *backup.MigrationRecordItem) *MigrationRecordItemDTO {
	dto := &MigrationRecordItemDTO{
		ID:        e.ID,
		RecordID:  e.RecordID,
		DBName:    e.DBName,
		Action:    string(e.Action),
		Status:    string(e.Status),
		Message:   e.Message,
		StartedAt: e.StartedAt.Format(time.DateTime),
		Duration:  e.Duration,
	}
	if e.FinishedAt != nil {
		s := e.FinishedAt.Format(time.DateTime)
		dto.FinishedAt = &s
	}
	return dto
}

func toObjectSyncTaskDTO(e *backup.ObjectSyncTask) *ObjectSyncTaskDTO {
	dto := &ObjectSyncTaskDTO{
		ID:             e.ID,
		Name:           e.Name,
		SourceEndpoint: e.SourceEndpoint,
		SourceRegion:   e.SourceRegion,
		SourceBucket:   e.SourceBucket,
		SourcePath:     e.SourcePath,
		SourceUseSSL:   e.SourceUseSSL,
		TargetEndpoint: e.TargetEndpoint,
		TargetRegion:   e.TargetRegion,
		TargetBucket:   e.TargetBucket,
		TargetPath:     e.TargetPath,
		TargetUseSSL:   e.TargetUseSSL,
		Mode:           string(e.Mode),
		Description:    e.Description,
		LastRunStatus:  string(e.LastRunStatus),
		CreatedAt:      e.CreatedAt.Format(time.DateTime),
		UpdatedAt:      e.UpdatedAt.Format(time.DateTime),
	}
	if e.LastRunAt != nil {
		s := e.LastRunAt.Format(time.DateTime)
		dto.LastRunAt = &s
	}
	return dto
}

func toObjectSyncRecordDTO(e *backup.ObjectSyncRecord) *ObjectSyncRecordDTO {
	dto := &ObjectSyncRecordDTO{
		ID:           e.ID,
		TaskID:       e.TaskID,
		TaskName:     e.TaskName,
		Mode:         string(e.Mode),
		Status:       string(e.Status),
		SourceBucket: e.SourceBucket,
		SourcePath:   e.SourcePath,
		TargetBucket: e.TargetBucket,
		TargetPath:   e.TargetPath,
		ObjectCount:  e.ObjectCount,
		SuccessCount: e.SuccessCount,
		SkippedCount: e.SkippedCount,
		FailedCount:  e.FailedCount,
		BytesTotal:   e.BytesTotal,
		Summary:      e.Summary,
		Error:        e.Error,
		StartedAt:    e.StartedAt.Format(time.DateTime),
		Duration:     e.Duration,
	}
	if e.FinishedAt != nil {
		s := e.FinishedAt.Format(time.DateTime)
		dto.FinishedAt = &s
	}
	if len(e.Items) > 0 {
		dto.Items = make([]*ObjectSyncRecordItemDTO, len(e.Items))
		for i, item := range e.Items {
			dto.Items[i] = toObjectSyncRecordItemDTO(item)
		}
	}
	return dto
}

func toObjectSyncRecordItemDTO(e *backup.ObjectSyncRecordItem) *ObjectSyncRecordItemDTO {
	dto := &ObjectSyncRecordItemDTO{
		ID:        e.ID,
		RecordID:  e.RecordID,
		SourceKey: e.SourceKey,
		TargetKey: e.TargetKey,
		Size:      e.Size,
		ETag:      e.ETag,
		Action:    string(e.Action),
		Status:    string(e.Status),
		Message:   e.Message,
		StartedAt: e.StartedAt.Format(time.DateTime),
		Duration:  e.Duration,
	}
	if e.FinishedAt != nil {
		s := e.FinishedAt.Format(time.DateTime)
		dto.FinishedAt = &s
	}
	return dto
}

// ============================
// UseCase
// ============================

type UseCase struct {
	taskRepo                backup.BackupTaskRepository
	recordRepo              backup.BackupRecordRepository
	migrationTaskRepo       backup.MigrationTaskRepository
	migrationRecordRepo     backup.MigrationRecordRepository
	migrationRecordItemRepo backup.MigrationRecordItemRepository
	objectSyncTaskRepo      backup.ObjectSyncTaskRepository
	objectSyncRecordRepo    backup.ObjectSyncRecordRepository
	objectSyncItemRepo      backup.ObjectSyncRecordItemRepository
	encryptor               crypto.Encryptor
}

func NewUseCase(
	taskRepo backup.BackupTaskRepository,
	recordRepo backup.BackupRecordRepository,
	migrationTaskRepo backup.MigrationTaskRepository,
	migrationRecordRepo backup.MigrationRecordRepository,
	migrationRecordItemRepo backup.MigrationRecordItemRepository,
	objectSyncTaskRepo backup.ObjectSyncTaskRepository,
	objectSyncRecordRepo backup.ObjectSyncRecordRepository,
	objectSyncItemRepo backup.ObjectSyncRecordItemRepository,
	encryptor crypto.Encryptor,
) *UseCase {
	return &UseCase{
		taskRepo:                taskRepo,
		recordRepo:              recordRepo,
		migrationTaskRepo:       migrationTaskRepo,
		migrationRecordRepo:     migrationRecordRepo,
		migrationRecordItemRepo: migrationRecordItemRepo,
		objectSyncTaskRepo:      objectSyncTaskRepo,
		objectSyncRecordRepo:    objectSyncRecordRepo,
		objectSyncItemRepo:      objectSyncItemRepo,
		encryptor:               encryptor,
	}
}

// CreateTask 创建备份任务
func (uc *UseCase) CreateTask(ctx context.Context, cmd CreateTaskCmd) (*TaskDTO, error) {
	// 校验 cron 表达式
	if _, err := cron.ParseStandard(cmd.CronExpr); err != nil {
		return nil, backup.ErrInvalidCronExpr
	}

	retentionDays := cmd.RetentionDays
	if retentionDays <= 0 {
		retentionDays = 10
	}

	// 加密数据库密码
	encPwd, err := uc.encryptor.Encrypt(cmd.DBPassword)
	if err != nil {
		return nil, fmt.Errorf("加密数据库密码失败: %w", err)
	}

	e := &backup.BackupTask{
		Name:          cmd.Name,
		BackupType:    backup.BackupType(cmd.BackupType),
		CronExpr:      cmd.CronExpr,
		Enabled:       cmd.Enabled,
		DBHost:        cmd.DBHost,
		DBPort:        cmd.DBPort,
		DBUser:        cmd.DBUser,
		DBPassword:    encPwd,
		DBName:        cmd.DBName,
		TargetHostID:  cmd.TargetHostID,
		TargetPath:    cmd.TargetPath,
		RetentionDays: retentionDays,
		Description:   cmd.Description,
	}

	if err := uc.taskRepo.Save(ctx, e); err != nil {
		return nil, fmt.Errorf("保存备份任务失败: %w", err)
	}
	return toTaskDTO(e), nil
}

// UpdateTask 更新备份任务
func (uc *UseCase) UpdateTask(ctx context.Context, cmd UpdateTaskCmd) (*TaskDTO, error) {
	e, err := uc.taskRepo.FindByID(ctx, cmd.ID)
	if err != nil {
		return nil, backup.ErrTaskNotFound
	}

	if cmd.Name != "" {
		e.Name = cmd.Name
	}
	if cmd.CronExpr != "" {
		if _, err := cron.ParseStandard(cmd.CronExpr); err != nil {
			return nil, backup.ErrInvalidCronExpr
		}
		e.CronExpr = cmd.CronExpr
	}
	if cmd.Enabled != nil {
		e.Enabled = *cmd.Enabled
	}
	if cmd.DBHost != "" {
		e.DBHost = cmd.DBHost
	}
	if cmd.DBPort > 0 {
		e.DBPort = cmd.DBPort
	}
	if cmd.DBUser != "" {
		e.DBUser = cmd.DBUser
	}
	if cmd.DBPassword != "" {
		encPwd, encErr := uc.encryptor.Encrypt(cmd.DBPassword)
		if encErr != nil {
			return nil, fmt.Errorf("加密数据库密码失败: %w", encErr)
		}
		e.DBPassword = encPwd
	}
	if cmd.DBName != nil {
		e.DBName = *cmd.DBName
	}
	if cmd.TargetHostID != "" {
		e.TargetHostID = cmd.TargetHostID
	}
	if cmd.TargetPath != "" {
		e.TargetPath = cmd.TargetPath
	}
	if cmd.RetentionDays > 0 {
		e.RetentionDays = cmd.RetentionDays
	}
	if cmd.Description != "" {
		e.Description = cmd.Description
	}

	if err := uc.taskRepo.Update(ctx, e); err != nil {
		return nil, fmt.Errorf("更新备份任务失败: %w", err)
	}
	return toTaskDTO(e), nil
}

// GetTask 获取单个任务
func (uc *UseCase) GetTask(ctx context.Context, id string) (*TaskDTO, error) {
	e, err := uc.taskRepo.FindByID(ctx, id)
	if err != nil {
		return nil, backup.ErrTaskNotFound
	}
	return toTaskDTO(e), nil
}

// ListTasks 查询任务列表
func (uc *UseCase) ListTasks(ctx context.Context, cmd TaskQueryCmd) ([]*TaskDTO, int64, error) {
	if cmd.Page < 1 {
		cmd.Page = 1
	}
	if cmd.PageSize < 1 || cmd.PageSize > 100 {
		cmd.PageSize = 20
	}

	entities, total, err := uc.taskRepo.Find(ctx, backup.TaskQuery{
		Keyword:  cmd.Keyword,
		Page:     cmd.Page,
		PageSize: cmd.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]*TaskDTO, len(entities))
	for i, e := range entities {
		dtos[i] = toTaskDTO(e)
	}
	return dtos, total, nil
}

// DeleteTask 删除备份任务及其记录
func (uc *UseCase) DeleteTask(ctx context.Context, id string) error {
	if _, err := uc.taskRepo.FindByID(ctx, id); err != nil {
		return backup.ErrTaskNotFound
	}
	_ = uc.recordRepo.DeleteByTaskID(ctx, id)
	return uc.taskRepo.Delete(ctx, id)
}

// ListRecords 查询备份记录
func (uc *UseCase) ListRecords(ctx context.Context, cmd RecordQueryCmd) ([]*RecordDTO, int64, error) {
	if cmd.Page < 1 {
		cmd.Page = 1
	}
	if cmd.PageSize < 1 || cmd.PageSize > 100 {
		cmd.PageSize = 20
	}

	entities, total, err := uc.recordRepo.Find(ctx, backup.RecordQuery{
		TaskID:   cmd.TaskID,
		Status:   cmd.Status,
		Page:     cmd.Page,
		PageSize: cmd.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]*RecordDTO, len(entities))
	for i, e := range entities {
		dtos[i] = toRecordDTO(e)
	}
	return dtos, total, nil
}

// GetAllEnabledTasks 获取所有启用的任务（供调度器使用）
func (uc *UseCase) GetAllEnabledTasks(ctx context.Context) ([]*backup.BackupTask, error) {
	return uc.taskRepo.FindAllEnabled(ctx)
}

// CreateRecord 创建备份记录
func (uc *UseCase) CreateRecord(ctx context.Context, record *backup.BackupRecord) error {
	return uc.recordRepo.Save(ctx, record)
}

// UpdateRecord 更新备份记录
func (uc *UseCase) UpdateRecord(ctx context.Context, record *backup.BackupRecord) error {
	return uc.recordRepo.Update(ctx, record)
}

// UpdateTaskLastRun 更新任务最后运行信息
func (uc *UseCase) UpdateTaskLastRun(ctx context.Context, taskID string, status backup.BackupStatus) error {
	e, err := uc.taskRepo.FindByID(ctx, taskID)
	if err != nil {
		return err
	}
	now := time.Now()
	e.LastRunAt = &now
	e.LastRunStatus = status
	return uc.taskRepo.Update(ctx, e)
}

// GetTaskEntity 获取任务实体（供调度器手动触发使用，不受 enabled 限制）
func (uc *UseCase) GetTaskEntity(ctx context.Context, id string) (*backup.BackupTask, error) {
	return uc.taskRepo.FindByID(ctx, id)
}

// DecryptDBPassword 解密任务的数据库密码（供执行器使用）
func (uc *UseCase) DecryptDBPassword(encPwd string) (string, error) {
	return uc.encryptor.Decrypt(encPwd)
}

// CreateMigrationTask 创建数据库迁移任务
func (uc *UseCase) CreateMigrationTask(ctx context.Context, cmd CreateMigrationTaskCmd) (*MigrationTaskDTO, error) {
	dbType := backup.BackupType(cmd.DBType)
	if !isSupportedMigrationDBType(dbType) {
		return nil, backup.ErrInvalidMigrationDBType
	}
	mode := backup.MigrationMode(cmd.Mode)
	if !isSupportedMigrationMode(mode) {
		return nil, backup.ErrInvalidMigrationMode
	}
	if strings.TrimSpace(cmd.DBNames) == "" {
		return nil, backup.ErrMigrationDBNamesRequired
	}

	sourcePassword, err := uc.encryptor.Encrypt(cmd.SourcePassword)
	if err != nil {
		return nil, fmt.Errorf("加密源数据库密码失败: %w", err)
	}
	targetPassword, err := uc.encryptor.Encrypt(cmd.TargetPassword)
	if err != nil {
		return nil, fmt.Errorf("加密目标数据库密码失败: %w", err)
	}

	e := &backup.MigrationTask{
		Name:           cmd.Name,
		DBType:         dbType,
		SourceHost:     cmd.SourceHost,
		SourcePort:     cmd.SourcePort,
		SourceUser:     cmd.SourceUser,
		SourcePassword: sourcePassword,
		DBNames:        cmd.DBNames,
		TargetHost:     cmd.TargetHost,
		TargetPort:     cmd.TargetPort,
		TargetUser:     cmd.TargetUser,
		TargetPassword: targetPassword,
		Mode:           mode,
		Description:    cmd.Description,
	}
	if err := uc.migrationTaskRepo.Save(ctx, e); err != nil {
		return nil, fmt.Errorf("保存迁移任务失败: %w", err)
	}
	return toMigrationTaskDTO(e), nil
}

// UpdateMigrationTask 更新数据库迁移任务
func (uc *UseCase) UpdateMigrationTask(ctx context.Context, cmd UpdateMigrationTaskCmd) (*MigrationTaskDTO, error) {
	e, err := uc.migrationTaskRepo.FindByID(ctx, cmd.ID)
	if err != nil {
		return nil, backup.ErrMigrationTaskNotFound
	}
	if cmd.Name != "" {
		e.Name = cmd.Name
	}
	if cmd.DBType != "" {
		dbType := backup.BackupType(cmd.DBType)
		if !isSupportedMigrationDBType(dbType) {
			return nil, backup.ErrInvalidMigrationDBType
		}
		e.DBType = dbType
	}
	if cmd.SourceHost != "" {
		e.SourceHost = cmd.SourceHost
	}
	if cmd.SourcePort > 0 {
		e.SourcePort = cmd.SourcePort
	}
	if cmd.SourceUser != "" {
		e.SourceUser = cmd.SourceUser
	}
	if cmd.SourcePassword != "" {
		encPwd, encErr := uc.encryptor.Encrypt(cmd.SourcePassword)
		if encErr != nil {
			return nil, fmt.Errorf("加密源数据库密码失败: %w", encErr)
		}
		e.SourcePassword = encPwd
	}
	if cmd.DBNames != "" {
		e.DBNames = cmd.DBNames
	}
	if cmd.TargetHost != "" {
		e.TargetHost = cmd.TargetHost
	}
	if cmd.TargetPort > 0 {
		e.TargetPort = cmd.TargetPort
	}
	if cmd.TargetUser != "" {
		e.TargetUser = cmd.TargetUser
	}
	if cmd.TargetPassword != "" {
		encPwd, encErr := uc.encryptor.Encrypt(cmd.TargetPassword)
		if encErr != nil {
			return nil, fmt.Errorf("加密目标数据库密码失败: %w", encErr)
		}
		e.TargetPassword = encPwd
	}
	if cmd.Mode != "" {
		mode := backup.MigrationMode(cmd.Mode)
		if !isSupportedMigrationMode(mode) {
			return nil, backup.ErrInvalidMigrationMode
		}
		e.Mode = mode
	}
	if cmd.Description != "" {
		e.Description = cmd.Description
	}
	if strings.TrimSpace(e.DBNames) == "" {
		return nil, backup.ErrMigrationDBNamesRequired
	}

	if err := uc.migrationTaskRepo.Update(ctx, e); err != nil {
		return nil, fmt.Errorf("更新迁移任务失败: %w", err)
	}
	return toMigrationTaskDTO(e), nil
}

func (uc *UseCase) ListMigrationTasks(ctx context.Context, cmd MigrationTaskQueryCmd) ([]*MigrationTaskDTO, int64, error) {
	if cmd.Page < 1 {
		cmd.Page = 1
	}
	if cmd.PageSize < 1 || cmd.PageSize > 100 {
		cmd.PageSize = 20
	}
	entities, total, err := uc.migrationTaskRepo.Find(ctx, backup.MigrationTaskQuery{
		Keyword:  cmd.Keyword,
		Page:     cmd.Page,
		PageSize: cmd.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}
	dtos := make([]*MigrationTaskDTO, len(entities))
	for i, e := range entities {
		dtos[i] = toMigrationTaskDTO(e)
	}
	return dtos, total, nil
}

func (uc *UseCase) GetMigrationTask(ctx context.Context, id string) (*MigrationTaskDTO, error) {
	e, err := uc.migrationTaskRepo.FindByID(ctx, id)
	if err != nil {
		return nil, backup.ErrMigrationTaskNotFound
	}
	return toMigrationTaskDTO(e), nil
}

func (uc *UseCase) DeleteMigrationTask(ctx context.Context, id string) error {
	if _, err := uc.migrationTaskRepo.FindByID(ctx, id); err != nil {
		return backup.ErrMigrationTaskNotFound
	}
	return uc.migrationTaskRepo.Delete(ctx, id)
}

func (uc *UseCase) GetMigrationTaskEntity(ctx context.Context, id string) (*backup.MigrationTask, error) {
	return uc.migrationTaskRepo.FindByID(ctx, id)
}

func (uc *UseCase) ListMigrationRecords(ctx context.Context, cmd MigrationRecordQueryCmd) ([]*MigrationRecordDTO, int64, error) {
	if cmd.Page < 1 {
		cmd.Page = 1
	}
	if cmd.PageSize < 1 || cmd.PageSize > 100 {
		cmd.PageSize = 20
	}
	entities, total, err := uc.migrationRecordRepo.Find(ctx, backup.MigrationRecordQuery{
		TaskID:   cmd.TaskID,
		Status:   cmd.Status,
		Page:     cmd.Page,
		PageSize: cmd.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}
	dtos := make([]*MigrationRecordDTO, len(entities))
	for i, e := range entities {
		dtos[i] = toMigrationRecordDTO(e)
	}
	return dtos, total, nil
}

func (uc *UseCase) GetMigrationRecord(ctx context.Context, id string) (*MigrationRecordDTO, error) {
	e, err := uc.migrationRecordRepo.FindByID(ctx, id)
	if err != nil {
		return nil, backup.ErrMigrationRecordNotFound
	}
	items, err := uc.migrationRecordItemRepo.FindByRecordID(ctx, id)
	if err != nil {
		return nil, err
	}
	e.Items = items
	return toMigrationRecordDTO(e), nil
}

func (uc *UseCase) CreateMigrationRecord(ctx context.Context, record *backup.MigrationRecord) error {
	return uc.migrationRecordRepo.Save(ctx, record)
}

func (uc *UseCase) UpdateMigrationRecord(ctx context.Context, record *backup.MigrationRecord) error {
	return uc.migrationRecordRepo.Update(ctx, record)
}

func (uc *UseCase) CreateMigrationRecordItem(ctx context.Context, item *backup.MigrationRecordItem) error {
	return uc.migrationRecordItemRepo.Save(ctx, item)
}

func (uc *UseCase) UpdateMigrationTaskLastRun(ctx context.Context, taskID string, status backup.MigrationStatus) error {
	e, err := uc.migrationTaskRepo.FindByID(ctx, taskID)
	if err != nil {
		return err
	}
	now := time.Now()
	e.LastRunAt = &now
	e.LastRunStatus = status
	return uc.migrationTaskRepo.Update(ctx, e)
}

func (uc *UseCase) DecryptMigrationPassword(encPwd string) (string, error) {
	return uc.encryptor.Decrypt(encPwd)
}

func isSupportedMigrationDBType(dbType backup.BackupType) bool {
	return dbType == backup.BackupTypePostgres || dbType == backup.BackupTypeMySQL
}

func isSupportedMigrationMode(mode backup.MigrationMode) bool {
	return mode == backup.MigrationModeCreateIfMissing || mode == backup.MigrationModeOverwrite
}

func (uc *UseCase) CreateObjectSyncTask(ctx context.Context, cmd CreateObjectSyncTaskCmd) (*ObjectSyncTaskDTO, error) {
	mode := backup.ObjectSyncMode(cmd.Mode)
	if !isSupportedObjectSyncMode(mode) {
		return nil, backup.ErrInvalidObjectSyncMode
	}
	sourceAccessKey, err := uc.encryptor.Encrypt(cmd.SourceAccessKey)
	if err != nil {
		return nil, fmt.Errorf("加密源访问用户名失败: %w", err)
	}
	sourceSecretKey, err := uc.encryptor.Encrypt(cmd.SourceSecretKey)
	if err != nil {
		return nil, fmt.Errorf("加密源访问密码失败: %w", err)
	}
	targetAccessKey, err := uc.encryptor.Encrypt(cmd.TargetAccessKey)
	if err != nil {
		return nil, fmt.Errorf("加密目标访问用户名失败: %w", err)
	}
	targetSecretKey, err := uc.encryptor.Encrypt(cmd.TargetSecretKey)
	if err != nil {
		return nil, fmt.Errorf("加密目标访问密码失败: %w", err)
	}

	e := &backup.ObjectSyncTask{
		Name:            cmd.Name,
		SourceEndpoint:  cmd.SourceEndpoint,
		SourceRegion:    defaultRegion(cmd.SourceRegion),
		SourceBucket:    cmd.SourceBucket,
		SourcePath:      cmd.SourcePath,
		SourceAccessKey: sourceAccessKey,
		SourceSecretKey: sourceSecretKey,
		SourceUseSSL:    cmd.SourceUseSSL,
		TargetEndpoint:  cmd.TargetEndpoint,
		TargetRegion:    defaultRegion(cmd.TargetRegion),
		TargetBucket:    cmd.TargetBucket,
		TargetPath:      cmd.TargetPath,
		TargetAccessKey: targetAccessKey,
		TargetSecretKey: targetSecretKey,
		TargetUseSSL:    cmd.TargetUseSSL,
		Mode:            mode,
		Description:     cmd.Description,
	}
	if err := uc.objectSyncTaskRepo.Save(ctx, e); err != nil {
		return nil, fmt.Errorf("保存对象同步任务失败: %w", err)
	}
	return toObjectSyncTaskDTO(e), nil
}

func (uc *UseCase) UpdateObjectSyncTask(ctx context.Context, cmd UpdateObjectSyncTaskCmd) (*ObjectSyncTaskDTO, error) {
	e, err := uc.objectSyncTaskRepo.FindByID(ctx, cmd.ID)
	if err != nil {
		return nil, backup.ErrObjectSyncTaskNotFound
	}
	if cmd.Name != "" {
		e.Name = cmd.Name
	}
	if cmd.SourceEndpoint != "" {
		e.SourceEndpoint = cmd.SourceEndpoint
	}
	if cmd.SourceRegion != "" {
		e.SourceRegion = defaultRegion(cmd.SourceRegion)
	}
	if cmd.SourceBucket != "" {
		e.SourceBucket = cmd.SourceBucket
	}
	if cmd.SourcePath != "" {
		e.SourcePath = cmd.SourcePath
	}
	if cmd.SourceAccessKey != "" {
		enc, encErr := uc.encryptor.Encrypt(cmd.SourceAccessKey)
		if encErr != nil {
			return nil, fmt.Errorf("加密源访问用户名失败: %w", encErr)
		}
		e.SourceAccessKey = enc
	}
	if cmd.SourceSecretKey != "" {
		enc, encErr := uc.encryptor.Encrypt(cmd.SourceSecretKey)
		if encErr != nil {
			return nil, fmt.Errorf("加密源访问密码失败: %w", encErr)
		}
		e.SourceSecretKey = enc
	}
	if cmd.SourceUseSSL != nil {
		e.SourceUseSSL = *cmd.SourceUseSSL
	}
	if cmd.TargetEndpoint != "" {
		e.TargetEndpoint = cmd.TargetEndpoint
	}
	if cmd.TargetRegion != "" {
		e.TargetRegion = defaultRegion(cmd.TargetRegion)
	}
	if cmd.TargetBucket != "" {
		e.TargetBucket = cmd.TargetBucket
	}
	if cmd.TargetPath != "" {
		e.TargetPath = cmd.TargetPath
	}
	if cmd.TargetAccessKey != "" {
		enc, encErr := uc.encryptor.Encrypt(cmd.TargetAccessKey)
		if encErr != nil {
			return nil, fmt.Errorf("加密目标访问用户名失败: %w", encErr)
		}
		e.TargetAccessKey = enc
	}
	if cmd.TargetSecretKey != "" {
		enc, encErr := uc.encryptor.Encrypt(cmd.TargetSecretKey)
		if encErr != nil {
			return nil, fmt.Errorf("加密目标访问密码失败: %w", encErr)
		}
		e.TargetSecretKey = enc
	}
	if cmd.TargetUseSSL != nil {
		e.TargetUseSSL = *cmd.TargetUseSSL
	}
	if cmd.Mode != "" {
		mode := backup.ObjectSyncMode(cmd.Mode)
		if !isSupportedObjectSyncMode(mode) {
			return nil, backup.ErrInvalidObjectSyncMode
		}
		e.Mode = mode
	}
	if cmd.Description != "" {
		e.Description = cmd.Description
	}
	if err := uc.objectSyncTaskRepo.Update(ctx, e); err != nil {
		return nil, fmt.Errorf("更新对象同步任务失败: %w", err)
	}
	return toObjectSyncTaskDTO(e), nil
}

func (uc *UseCase) ListObjectSyncTasks(ctx context.Context, cmd ObjectSyncTaskQueryCmd) ([]*ObjectSyncTaskDTO, int64, error) {
	if cmd.Page < 1 {
		cmd.Page = 1
	}
	if cmd.PageSize < 1 || cmd.PageSize > 100 {
		cmd.PageSize = 20
	}
	entities, total, err := uc.objectSyncTaskRepo.Find(ctx, backup.ObjectSyncTaskQuery{
		Keyword:  cmd.Keyword,
		Page:     cmd.Page,
		PageSize: cmd.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}
	dtos := make([]*ObjectSyncTaskDTO, len(entities))
	for i, e := range entities {
		dtos[i] = toObjectSyncTaskDTO(e)
	}
	return dtos, total, nil
}

func (uc *UseCase) GetObjectSyncTask(ctx context.Context, id string) (*ObjectSyncTaskDTO, error) {
	e, err := uc.objectSyncTaskRepo.FindByID(ctx, id)
	if err != nil {
		return nil, backup.ErrObjectSyncTaskNotFound
	}
	return toObjectSyncTaskDTO(e), nil
}

func (uc *UseCase) DeleteObjectSyncTask(ctx context.Context, id string) error {
	if _, err := uc.objectSyncTaskRepo.FindByID(ctx, id); err != nil {
		return backup.ErrObjectSyncTaskNotFound
	}
	return uc.objectSyncTaskRepo.Delete(ctx, id)
}

func (uc *UseCase) GetObjectSyncTaskEntity(ctx context.Context, id string) (*backup.ObjectSyncTask, error) {
	return uc.objectSyncTaskRepo.FindByID(ctx, id)
}

func (uc *UseCase) ListObjectSyncRecords(ctx context.Context, cmd ObjectSyncRecordQueryCmd) ([]*ObjectSyncRecordDTO, int64, error) {
	if cmd.Page < 1 {
		cmd.Page = 1
	}
	if cmd.PageSize < 1 || cmd.PageSize > 100 {
		cmd.PageSize = 20
	}
	entities, total, err := uc.objectSyncRecordRepo.Find(ctx, backup.ObjectSyncRecordQuery{
		TaskID:   cmd.TaskID,
		Status:   cmd.Status,
		Page:     cmd.Page,
		PageSize: cmd.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}
	dtos := make([]*ObjectSyncRecordDTO, len(entities))
	for i, e := range entities {
		dtos[i] = toObjectSyncRecordDTO(e)
	}
	return dtos, total, nil
}

func (uc *UseCase) GetObjectSyncRecord(ctx context.Context, id string) (*ObjectSyncRecordDTO, error) {
	e, err := uc.objectSyncRecordRepo.FindByID(ctx, id)
	if err != nil {
		return nil, backup.ErrObjectSyncRecordNotFound
	}
	items, err := uc.objectSyncItemRepo.FindByRecordID(ctx, id)
	if err != nil {
		return nil, err
	}
	e.Items = items
	return toObjectSyncRecordDTO(e), nil
}

func (uc *UseCase) CreateObjectSyncRecord(ctx context.Context, record *backup.ObjectSyncRecord) error {
	return uc.objectSyncRecordRepo.Save(ctx, record)
}

func (uc *UseCase) UpdateObjectSyncRecord(ctx context.Context, record *backup.ObjectSyncRecord) error {
	return uc.objectSyncRecordRepo.Update(ctx, record)
}

func (uc *UseCase) CreateObjectSyncRecordItem(ctx context.Context, item *backup.ObjectSyncRecordItem) error {
	return uc.objectSyncItemRepo.Save(ctx, item)
}

func (uc *UseCase) UpdateObjectSyncTaskLastRun(ctx context.Context, taskID string, status backup.ObjectSyncStatus) error {
	e, err := uc.objectSyncTaskRepo.FindByID(ctx, taskID)
	if err != nil {
		return err
	}
	now := time.Now()
	e.LastRunAt = &now
	e.LastRunStatus = status
	return uc.objectSyncTaskRepo.Update(ctx, e)
}

func (uc *UseCase) DecryptObjectSyncSecret(enc string) (string, error) {
	return uc.encryptor.Decrypt(enc)
}

func isSupportedObjectSyncMode(mode backup.ObjectSyncMode) bool {
	return mode == backup.ObjectSyncModeCopyIfMissing ||
		mode == backup.ObjectSyncModeOverwrite ||
		mode == backup.ObjectSyncModeChecksumSkip
}

func defaultRegion(region string) string {
	if strings.TrimSpace(region) == "" {
		return "us-east-1"
	}
	return strings.TrimSpace(region)
}
