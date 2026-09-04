package release

import "context"

// ============================
// 仓储接口 (Repository Ports)
// ============================

// ReleaseRecordRepository 发布记录仓储接口
type ReleaseRecordRepository interface {
	Save(ctx context.Context, entity *ReleaseRecord) error
	FindByID(ctx context.Context, id string) (*ReleaseRecord, error)
	Find(ctx context.Context, query ReleaseQuery) ([]*ReleaseRecord, int64, error)
	FindLastSuccess(ctx context.Context, serviceID, envID string) (*ReleaseRecord, error)
	FindActiveByServiceEnv(ctx context.Context, serviceID, envID string) (*ReleaseRecord, error)
	Update(ctx context.Context, entity *ReleaseRecord) error
	Delete(ctx context.Context, id string) error
}

// ============================
// 查询对象 (Query Objects)
// ============================

type ReleaseQuery struct {
	ServiceID   string
	EnvID       string
	TenantID    string
	Status      string
	ReleaseType string
	Keyword     string
	Page        int
	PageSize    int
}

func (q ReleaseQuery) Offset() int {
	if q.Page < 1 {
		q.Page = 1
	}
	return (q.Page - 1) * q.PageSize
}
