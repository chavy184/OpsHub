package release

import (
	"errors"
	"time"
)

// ============================
// 值对象 (Value Objects)
// ============================

type ReleaseType string

const (
	ReleaseTypeDeploy   ReleaseType = "deploy"
	ReleaseTypeRollback ReleaseType = "rollback"
)

type ReleaseStatus string

const (
	ReleaseStatusPending   ReleaseStatus = "pending"
	ReleaseStatusRunning   ReleaseStatus = "running"
	ReleaseStatusSuccess   ReleaseStatus = "success"
	ReleaseStatusFailed    ReleaseStatus = "failed"
	ReleaseStatusCancelled ReleaseStatus = "cancelled"
)

// ============================
// 领域实体 (Domain Entities)
// ============================

// ReleaseRecord 发布记录聚合根
type ReleaseRecord struct {
	ID              string
	ServiceID       string
	EnvID           string
	TenantID        string // 可为空
	TargetVersionID string
	PrevVersionID   string // 回滚来源
	ReleaseType     ReleaseType
	Strategy        string
	Status          ReleaseStatus
	ErrorMessage    string
	OperatorID      string
	IdempotencyKey  string
	JenkinsParams   string // Jenkins 构建参数 JSON
	JenkinsBuildNo  int    // Jenkins 构建号
	StartedAt       *time.Time
	EndedAt         *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ============================
// 领域方法 (Domain Methods)
// ============================

// CanExecute 是否可以执行
func (r *ReleaseRecord) CanExecute() bool {
	return r.Status == ReleaseStatusPending
}

// MarkRunning 标记为执行中
func (r *ReleaseRecord) MarkRunning() error {
	if !r.CanExecute() {
		return ErrInvalidReleaseState
	}
	r.Status = ReleaseStatusRunning
	now := time.Now()
	r.StartedAt = &now
	return nil
}

// MarkSuccess 标记为成功
func (r *ReleaseRecord) MarkSuccess() {
	r.Status = ReleaseStatusSuccess
	now := time.Now()
	r.EndedAt = &now
}

// MarkFailed 标记为失败
func (r *ReleaseRecord) MarkFailed(errMsg string) {
	r.Status = ReleaseStatusFailed
	r.ErrorMessage = errMsg
	now := time.Now()
	r.EndedAt = &now
}

// CanRollback 是否可以回滚
func (r *ReleaseRecord) CanRollback() bool {
	return r.Status == ReleaseStatusSuccess || r.Status == ReleaseStatusFailed
}

// IsTerminal 是否为终态
func (r *ReleaseRecord) IsTerminal() bool {
	return r.Status == ReleaseStatusSuccess ||
		r.Status == ReleaseStatusFailed ||
		r.Status == ReleaseStatusCancelled
}

// ============================
// 领域错误 (Domain Errors)
// ============================

var (
	ErrReleaseNotFound      = errors.New("发布记录不存在")
	ErrInvalidReleaseState  = errors.New("发布状态不允许该操作")
	ErrIdempotencyConflict  = errors.New("幂等键冲突，重复发布")
	ErrNoPreviousVersion    = errors.New("无可回滚的历史版本")
	ErrReleaseAlreadyActive = errors.New("该服务环境已有进行中的发布")
)
