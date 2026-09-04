package host

import (
	"context"
	"errors"
	"time"
)

// AgentStatus 机器 Agent 状态
type AgentStatus string

const (
	AgentStatusUnknown AgentStatus = "unknown"
	AgentStatusOnline  AgentStatus = "online"
	AgentStatusOffline AgentStatus = "offline"
)

// Host 机器聚合根
type Host struct {
	ID            string
	Name          string
	HostAddress   string
	SSHPort       int
	Username      string
	CredentialID  string
	IsProd        bool
	Labels        string // JSON
	OsInfo        string
	AgentStatus   AgentStatus
	LastHeartbeat *time.Time
	Description   string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// HostRepository 机器仓储接口
type HostRepository interface {
	Save(ctx context.Context, entity *Host) error
	Update(ctx context.Context, entity *Host) error
	FindByID(ctx context.Context, id string) (*Host, error)
	FindByIsProd(ctx context.Context, isProd bool) ([]*Host, error)
	Find(ctx context.Context, query HostQuery) ([]*Host, int64, error)
	Delete(ctx context.Context, id string) error
}

// HostMetricRepository 主机指标仓储接口
type HostMetricRepository interface {
	Save(ctx context.Context, snapshot *HostMetricSnapshot) error
	FindLatest(ctx context.Context, hostID string) (*HostMetricSnapshot, error)
	FindHistory(ctx context.Context, query HostMetricQuery) ([]*HostMetricSnapshot, error)
	FindAllLatest(ctx context.Context) ([]*HostMetricSnapshot, error)
	DeleteOlderThan(ctx context.Context, before time.Time) error
}

// HostQuery 查询对象
type HostQuery struct {
	Keyword  string
	Page     int
	PageSize int
}

func (q HostQuery) Offset() int {
	if q.Page < 1 {
		q.Page = 1
	}
	return (q.Page - 1) * q.PageSize
}

// 领域错误
var (
	ErrHostNotFound         = errors.New("机器不存在")
	ErrHostNameExists       = errors.New("机器名称已存在")
	ErrHostUsernameRequired = errors.New("选择凭证时 SSH 用户名不能为空")
)
