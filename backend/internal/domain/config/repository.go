package config

import "context"

// ============================
// 仓储接口 (Repository Ports)
// ============================

type ConfigItemRepository interface {
	Save(ctx context.Context, entity *ConfigItem) error
	FindByID(ctx context.Context, id string) (*ConfigItem, error)
	FindByServiceID(ctx context.Context, serviceID string, scope ConfigScope) ([]*ConfigItem, error)
	Update(ctx context.Context, entity *ConfigItem) error
	Delete(ctx context.Context, id string) error
}

type ConfigOverrideRepository interface {
	Save(ctx context.Context, entity *ConfigOverride) error
	FindByTenantAndService(ctx context.Context, tenantID, serviceID string) ([]*ConfigOverride, error)
	Update(ctx context.Context, entity *ConfigOverride) error
	Delete(ctx context.Context, id string) error
}
