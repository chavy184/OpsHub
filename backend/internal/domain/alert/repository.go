package alert

import "context"

// ============================
// 仓储接口 (Repository Ports)
// ============================

type AlertEventRepository interface {
	Save(ctx context.Context, entity *AlertEvent) error
	FindByID(ctx context.Context, id string) (*AlertEvent, error)
	Find(ctx context.Context, query AlertQuery) ([]*AlertEvent, int64, error)
	FindByFingerprint(ctx context.Context, fingerprint string) (*AlertEvent, error)
	Update(ctx context.Context, entity *AlertEvent) error
}

type AlertQuery struct {
	ServiceID string
	EnvID     string
	TenantID  string
	Severity  string
	Status    string
	Keyword   string
	Page      int
	PageSize  int
}

func (q AlertQuery) Offset() int {
	if q.Page < 1 {
		q.Page = 1
	}
	return (q.Page - 1) * q.PageSize
}
