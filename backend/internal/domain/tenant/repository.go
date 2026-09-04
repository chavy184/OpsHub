package tenant

import "context"

// ============================
// 仓储接口 (Repository Ports)
// ============================

type TenantRepository interface {
	Save(ctx context.Context, entity *Tenant) error
	FindByID(ctx context.Context, id string) (*Tenant, error)
	FindByCode(ctx context.Context, code string) (*Tenant, error)
	Find(ctx context.Context, query TenantQuery) ([]*Tenant, int64, error)
	Delete(ctx context.Context, id string) error
}

type TenantServiceBindingRepository interface {
	Save(ctx context.Context, entity *TenantServiceBinding) error
	FindByTenantID(ctx context.Context, tenantID string) ([]*TenantServiceBinding, error)
	FindByTenantAndService(ctx context.Context, tenantID, serviceID string) (*TenantServiceBinding, error)
	Update(ctx context.Context, entity *TenantServiceBinding) error
}

type TenantQuery struct {
	Status   string
	Keyword  string
	Page     int
	PageSize int
}

func (q TenantQuery) Offset() int {
	if q.Page < 1 {
		q.Page = 1
	}
	return (q.Page - 1) * q.PageSize
}
