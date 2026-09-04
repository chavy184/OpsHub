package alert

import (
	"context"
	"errors"
	"time"
)

// ============================
// 值对象 (Value Objects)
// ============================

type Severity string

const (
	SeverityP1 Severity = "P1"
	SeverityP2 Severity = "P2"
	SeverityP3 Severity = "P3"
	SeverityP4 Severity = "P4"
)

type AlertStatus string

const (
	AlertStatusOpen       AlertStatus = "open"
	AlertStatusAcked      AlertStatus = "acked"
	AlertStatusClosed     AlertStatus = "closed"
	AlertStatusSuppressed AlertStatus = "suppressed"
)

// ============================
// 领域实体 (Domain Entities)
// ============================

// AlertEvent 告警事件聚合根
type AlertEvent struct {
	ID               string
	ServiceID        string
	EnvID            string
	TenantID         string
	AlertSource      string
	AlertFingerprint string
	Severity         Severity
	Title            string
	Content          string
	Status           AlertStatus
	FirstSeenAt      time.Time
	LastSeenAt       time.Time
	AssigneeUserID   string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// ============================
// 领域方法 (Domain Methods)
// ============================

func (a *AlertEvent) Ack(userID string) error {
	if a.Status != AlertStatusOpen {
		return ErrInvalidAlertState
	}
	a.Status = AlertStatusAcked
	a.AssigneeUserID = userID
	return nil
}

func (a *AlertEvent) Close() error {
	if a.Status == AlertStatusClosed {
		return ErrInvalidAlertState
	}
	a.Status = AlertStatusClosed
	return nil
}

func (a *AlertEvent) Suppress() {
	a.Status = AlertStatusSuppressed
}

// ============================
// 日志检索接口 (LogSearcher Port)
// Domain 层定义接口，Infrastructure 层实现
// MVP 适配器: LokiAdapter (Loki API), FileAdapter (SSH + grep)
// ============================

// LogSearcher 日志检索接口
type LogSearcher interface {
	Search(ctx context.Context, query LogSearchQuery) (*LogSearchResult, error)
}

type LogSearchQuery struct {
	ServiceID string
	EnvID     string
	TenantID  string
	Keyword   string
	Level     string
	TraceID   string
	StartTime time.Time
	EndTime   time.Time
	Page      int
	PageSize  int
}

type LogEntry struct {
	Timestamp time.Time         `json:"timestamp"`
	Level     string            `json:"level"`
	Message   string            `json:"message"`
	Service   string            `json:"service"`
	Env       string            `json:"env"`
	TraceID   string            `json:"trace_id"`
	Fields    map[string]string `json:"fields"`
}

type LogSearchResult struct {
	Entries []*LogEntry `json:"entries"`
	Total   int64       `json:"total"`
}

// ============================
// 领域错误 (Domain Errors)
// ============================

var (
	ErrAlertNotFound     = errors.New("告警事件不存在")
	ErrInvalidAlertState = errors.New("告警状态不允许该操作")
)
