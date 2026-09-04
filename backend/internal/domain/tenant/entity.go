package tenant

import (
	"errors"
	"time"
)

// ============================
// 值对象 (Value Objects)
// ============================

type LicenseType string

const (
	LicenseTrial      LicenseType = "trial"
	LicenseStandard   LicenseType = "standard"
	LicenseEnterprise LicenseType = "enterprise"
)

type TenantStatus string

const (
	TenantStatusActive   TenantStatus = "active"
	TenantStatusInactive TenantStatus = "inactive"
)

type CompatStatus string

const (
	CompatUnknown CompatStatus = "unknown"
	CompatPass    CompatStatus = "pass"
	CompatFail    CompatStatus = "fail"
)

// ============================
// 领域实体 (Domain Entities)
// ============================

// Tenant 租户聚合根
type Tenant struct {
	ID            string
	TenantCode    string
	TenantName    string
	LicenseType   LicenseType
	ContractStart *time.Time
	ContractEnd   *time.Time
	SupportLevel  string
	UpgradeWindow string
	Status        TenantStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// TenantServiceBinding 租户-服务绑定实体
type TenantServiceBinding struct {
	ID               string
	TenantID         string
	ServiceID        string
	CurrentVersionID string
	PinnedVersion    bool
	LastUpgradeAt    *time.Time
	CompatStatus     CompatStatus
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// ============================
// 领域方法 (Domain Methods)
// ============================

func (t *Tenant) IsActive() bool {
	return t.Status == TenantStatusActive
}

func (t *Tenant) IsContractValid() bool {
	if t.ContractEnd == nil {
		return true
	}
	return time.Now().Before(*t.ContractEnd)
}

// CanUpgrade 是否在升级窗口内（简化逻辑，Phase 2 细化）
func (b *TenantServiceBinding) CanUpgrade() bool {
	return !b.PinnedVersion
}

// ============================
// 领域错误 (Domain Errors)
// ============================

var (
	ErrTenantNotFound       = errors.New("租户不存在")
	ErrTenantCodeDuplicated = errors.New("租户编码已存在")
	ErrBindingNotFound      = errors.New("租户服务绑定不存在")
	ErrVersionPinned        = errors.New("该服务版本已锁定，无法升级")
	ErrCompatCheckFailed    = errors.New("兼容性检查未通过")
)
