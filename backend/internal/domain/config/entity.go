package config

import (
	"errors"
	"time"
)

// ============================
// 值对象 (Value Objects)
// ============================

type ConfigScope string

const (
	ConfigScopeBase     ConfigScope = "base"
	ConfigScopeEnv      ConfigScope = "env"
	ConfigScopeCustomer ConfigScope = "customer"
)

type ValueType string

const (
	ValueTypeString    ValueType = "string"
	ValueTypeInt       ValueType = "int"
	ValueTypeBool      ValueType = "bool"
	ValueTypeJSON      ValueType = "json"
	ValueTypeSecretRef ValueType = "secret_ref"
)

// ============================
// 领域实体 (Domain Entities)
// ============================

// ConfigItem 配置项实体
type ConfigItem struct {
	ID            string
	ServiceID     string
	EnvID         string // 空表示全局 base 配置
	ConfigKey     string
	ConfigScope   ConfigScope
	ValueType     ValueType
	DefaultValue  string
	EncryptedFlag bool
	VersionNo     int // 乐观锁
	CreatedBy     string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ConfigOverride 配置覆盖 (租户级)
type ConfigOverride struct {
	ID            string
	TenantID      string
	ServiceID     string
	EnvID         string
	ConfigItemID  string
	OverrideValue string
	VersionNo     int
	EffectiveFrom *time.Time
	EffectiveTo   *time.Time
	UpdatedBy     string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// ============================
// 领域方法 (Domain Methods)
// ============================

// IsSecret 是否为密钥引用
func (c *ConfigItem) IsSecret() bool {
	return c.ValueType == ValueTypeSecretRef || c.EncryptedFlag
}

// BumpVersion 递增版本号 (乐观锁)
func (c *ConfigItem) BumpVersion() {
	c.VersionNo++
}

// ============================
// 领域错误 (Domain Errors)
// ============================

var (
	ErrConfigItemNotFound    = errors.New("配置项不存在")
	ErrConfigVersionConflict = errors.New("配置版本冲突，请刷新后重试")
	ErrConfigValidationFail  = errors.New("配置校验失败")
)
