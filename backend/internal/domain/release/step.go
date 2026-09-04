package release

import (
	"context"
	"time"
)

// StepStatus 步骤状态
type StepStatus string

const (
	StepPending StepStatus = "pending"
	StepRunning StepStatus = "running"
	StepSuccess StepStatus = "success"
	StepFailed  StepStatus = "failed"
	StepSkipped StepStatus = "skipped"
)

// ReleaseStepLog 发布步骤日志实体
type ReleaseStepLog struct {
	ID          string
	ReleaseID   string
	StepOrder   int
	StepName    string
	StepStatus  StepStatus
	StartedAt   *time.Time
	EndedAt     *time.Time
	DurationMs  int
	Output      string
	ErrorOutput string
	CreatedAt   time.Time
}

// MarkRunning 标记步骤开始
func (s *ReleaseStepLog) MarkRunning() {
	s.StepStatus = StepRunning
	now := time.Now()
	s.StartedAt = &now
}

// MarkSuccess 标记步骤成功
func (s *ReleaseStepLog) MarkSuccess(output string) {
	s.StepStatus = StepSuccess
	s.Output = output
	now := time.Now()
	s.EndedAt = &now
	if s.StartedAt != nil {
		s.DurationMs = int(now.Sub(*s.StartedAt).Milliseconds())
	}
}

// MarkFailed 标记步骤失败
func (s *ReleaseStepLog) MarkFailed(output, errorOutput string) {
	s.StepStatus = StepFailed
	s.Output = output
	s.ErrorOutput = errorOutput
	now := time.Now()
	s.EndedAt = &now
	if s.StartedAt != nil {
		s.DurationMs = int(now.Sub(*s.StartedAt).Milliseconds())
	}
}

// MarkSkipped 标记步骤跳过
func (s *ReleaseStepLog) MarkSkipped(reason string) {
	s.StepStatus = StepSkipped
	s.Output = reason
}

// ReleaseStepLogRepository 步骤日志仓储接口
type ReleaseStepLogRepository interface {
	Save(ctx context.Context, step *ReleaseStepLog) error
	Update(ctx context.Context, step *ReleaseStepLog) error
	FindByReleaseID(ctx context.Context, releaseID string) ([]*ReleaseStepLog, error)
}
