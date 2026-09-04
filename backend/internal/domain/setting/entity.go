package setting

import (
	"context"
	"errors"
	"time"
)

// SystemSetting 系统设置实体
type SystemSetting struct {
	ID          string
	SettingKey  string
	Value       string
	ValueType   string // string / int / bool / json
	Category    string // general / notification / security / deploy
	Description string
	UpdatedBy   string
	UpdatedAt   time.Time
}

// SystemSettingRepository 系统设置仓储接口
type SystemSettingRepository interface {
	FindAll(ctx context.Context) ([]*SystemSetting, error)
	FindByKey(ctx context.Context, key string) (*SystemSetting, error)
	FindByCategory(ctx context.Context, category string) ([]*SystemSetting, error)
	Upsert(ctx context.Context, setting *SystemSetting) error
	BatchUpsert(ctx context.Context, settings []*SystemSetting) error
}

// 领域错误
var ErrSettingNotFound = errors.New("设置项不存在")
