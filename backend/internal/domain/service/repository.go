package service

import "context"

// ============================
// 仓储接口 (Repository Ports)
// ============================

// ServiceRepository 服务聚合根仓储接口
type ServiceRepository interface {
	Save(ctx context.Context, entity *Service) error
	FindByID(ctx context.Context, id string) (*Service, error)
	FindByKey(ctx context.Context, key string) (*Service, error)
	Find(ctx context.Context, query ServiceQuery) ([]*Service, int64, error)
	Delete(ctx context.Context, id string) error
}

// ServiceEnvRepository 服务环境仓储接口
type ServiceEnvRepository interface {
	Save(ctx context.Context, entity *ServiceEnv) error
	FindByID(ctx context.Context, id string) (*ServiceEnv, error)
	FindByServiceID(ctx context.Context, serviceID string) ([]*ServiceEnv, error)
	Delete(ctx context.Context, id string) error
	FindAllHealthCheckEnabled(ctx context.Context) ([]*ServiceEnv, error)
	UpdateHealthStatus(ctx context.Context, env *ServiceEnv) error
}

// ============================
// 查询对象 (Query Objects)
// ============================

type ServiceQuery struct {
	Keyword  string
	Page     int
	PageSize int
}

func (q ServiceQuery) Offset() int {
	if q.Page < 1 {
		q.Page = 1
	}
	return (q.Page - 1) * q.PageSize
}
