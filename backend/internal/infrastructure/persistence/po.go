package persistence

import (
	"time"

	"gorm.io/gorm"
)

// ============================
// 持久化对象 (Persistent Objects)
// PO 与 Domain Entity 解耦，禁止在 Domain 层使用 gorm tag
// ============================

// ServicePO 服务表
type ServicePO struct {
	ID          string         `gorm:"primaryKey;size:36"`
	ServiceKey  string         `gorm:"uniqueIndex;size:64;not null"`
	ServiceName string         `gorm:"size:128;not null"`
	OwnerUserID string         `gorm:"size:36;index"`
	RepoURL     string         `gorm:"size:512"`
	CreatedAt   time.Time      `gorm:"not null"`
	UpdatedAt   time.Time      `gorm:"not null"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (ServicePO) TableName() string { return "services" }

// ServiceEnvPO 服务环境表
type ServiceEnvPO struct {
	ID                      string `gorm:"primaryKey;size:36"`
	ServiceID               string `gorm:"size:36;not null;uniqueIndex:uk_service_envs"`
	EnvCode                 string `gorm:"size:16;not null;uniqueIndex:uk_service_envs"`
	ClusterName             string `gorm:"size:128"`
	Namespace               string `gorm:"size:128"`
	AccessEndpoint          string `gorm:"size:512"`
	HealthcheckURL          string `gorm:"size:512"`
	LogSourceType           string `gorm:"size:16;not null;default:file"`
	LogSourceConfig         string `gorm:"type:jsonb;not null;default:'{}'"`
	HostID                  string `gorm:"size:36"`
	DeployPath              string `gorm:"size:256"`
	EnvVars                 string `gorm:"type:jsonb;not null;default:'{}'"`
	HealthCheckInterval     int    `gorm:"not null;default:60"`
	HealthCheckTimeout      int    `gorm:"not null;default:10"`
	HealthCheckSuccessCodes string `gorm:"size:64;default:200"`
	HealthCheckEnabled      bool   `gorm:"not null;default:true"`
	HealthStatus            string `gorm:"size:16;not null;default:unknown"`
	HealthLastCheckedAt     *time.Time
	HealthLastMessage       string         `gorm:"type:text"`
	JenkinsJobs             string         `gorm:"type:jsonb;not null;default:'[]'"`
	CreatedAt               time.Time      `gorm:"not null"`
	UpdatedAt               time.Time      `gorm:"not null"`
	DeletedAt               gorm.DeletedAt `gorm:"index"`
}

func (ServiceEnvPO) TableName() string { return "service_envs" }

// ReleaseRecordPO 发布记录表
type ReleaseRecordPO struct {
	ID              string `gorm:"primaryKey;size:36"`
	ServiceID       string `gorm:"size:36;not null;index:idx_release_records_service"`
	EnvID           string `gorm:"size:36;not null"`
	TenantID        string `gorm:"size:36"`
	TargetVersionID string `gorm:"size:36;not null"`
	PrevVersionID   string `gorm:"size:36"`
	ReleaseType     string `gorm:"size:16;not null;default:deploy"`
	Strategy        string `gorm:"size:32;not null;default:default"`
	Status          string `gorm:"size:16;not null;default:pending;index:idx_release_records_status"`
	ErrorMessage    string `gorm:"type:text"`
	OperatorID      string `gorm:"size:36"`
	IdempotencyKey  string `gorm:"size:128;uniqueIndex:uk_release_idempotency"`
	JenkinsParams   string `gorm:"type:jsonb;default:'{}'"`
	JenkinsBuildNo  int    `gorm:"default:0"`
	StartedAt       *time.Time
	EndedAt         *time.Time
	CreatedAt       time.Time `gorm:"not null;index:idx_release_records_service"`
	UpdatedAt       time.Time `gorm:"not null"`
}

func (ReleaseRecordPO) TableName() string { return "release_records" }

// TenantPO 租户表
type TenantPO struct {
	ID            string `gorm:"primaryKey;size:36"`
	TenantCode    string `gorm:"uniqueIndex;size:64;not null"`
	TenantName    string `gorm:"size:128;not null"`
	LicenseType   string `gorm:"size:32;not null;default:standard"`
	ContractStart *time.Time
	ContractEnd   *time.Time
	SupportLevel  string         `gorm:"size:16;not null;default:standard"`
	UpgradeWindow string         `gorm:"size:64"`
	Status        string         `gorm:"size:16;not null;default:active"`
	CreatedAt     time.Time      `gorm:"not null"`
	UpdatedAt     time.Time      `gorm:"not null"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

func (TenantPO) TableName() string { return "tenants" }

// TenantServiceBindingPO 租户-服务绑定表
type TenantServiceBindingPO struct {
	ID               string `gorm:"primaryKey;size:36"`
	TenantID         string `gorm:"size:36;not null;uniqueIndex:uk_tenant_service"`
	ServiceID        string `gorm:"size:36;not null;uniqueIndex:uk_tenant_service"`
	CurrentVersionID string `gorm:"size:36"`
	PinnedVersion    bool   `gorm:"not null;default:false"`
	LastUpgradeAt    *time.Time
	CompatStatus     string    `gorm:"size:16;not null;default:unknown"`
	CreatedAt        time.Time `gorm:"not null"`
	UpdatedAt        time.Time `gorm:"not null"`
}

func (TenantServiceBindingPO) TableName() string { return "tenant_service_bindings" }

// ConfigItemPO 配置项表
type ConfigItemPO struct {
	ID            string         `gorm:"primaryKey;size:36"`
	ServiceID     string         `gorm:"size:36;not null;index:idx_config_items_service"`
	EnvID         string         `gorm:"size:36"`
	ConfigKey     string         `gorm:"size:256;not null"`
	ConfigScope   string         `gorm:"size:16;not null;default:base"`
	ValueType     string         `gorm:"size:16;not null;default:string"`
	DefaultValue  string         `gorm:"type:text"`
	EncryptedFlag bool           `gorm:"not null;default:false"`
	VersionNo     int            `gorm:"not null;default:1"`
	CreatedBy     string         `gorm:"size:36"`
	CreatedAt     time.Time      `gorm:"not null"`
	UpdatedAt     time.Time      `gorm:"not null"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

func (ConfigItemPO) TableName() string { return "config_items" }

// ConfigOverridePO 配置覆盖表
type ConfigOverridePO struct {
	ID            string `gorm:"primaryKey;size:36"`
	TenantID      string `gorm:"size:36;not null"`
	ServiceID     string `gorm:"size:36;not null"`
	EnvID         string `gorm:"size:36"`
	ConfigItemID  string `gorm:"size:36;not null"`
	OverrideValue string `gorm:"type:text"`
	VersionNo     int    `gorm:"not null;default:1"`
	EffectiveFrom *time.Time
	EffectiveTo   *time.Time
	UpdatedBy     string    `gorm:"size:36"`
	CreatedAt     time.Time `gorm:"not null"`
	UpdatedAt     time.Time `gorm:"not null"`
}

func (ConfigOverridePO) TableName() string { return "config_overrides" }

// AlertEventPO 告警事件表
type AlertEventPO struct {
	ID               string    `gorm:"primaryKey;size:36"`
	ServiceID        string    `gorm:"size:36;not null;index:idx_alert_events_service"`
	EnvID            string    `gorm:"size:36;not null"`
	TenantID         string    `gorm:"size:36"`
	AlertSource      string    `gorm:"size:32;not null;default:custom"`
	AlertFingerprint string    `gorm:"size:128;uniqueIndex:uk_alert_fingerprint"`
	Severity         string    `gorm:"size:4;not null;default:P3;index:idx_alert_events_severity"`
	Title            string    `gorm:"size:512;not null"`
	Content          string    `gorm:"type:text"`
	Status           string    `gorm:"size:16;not null;default:open;index:idx_alert_events_service"`
	FirstSeenAt      time.Time `gorm:"not null"`
	LastSeenAt       time.Time `gorm:"not null"`
	AssigneeUserID   string    `gorm:"size:36"`
	CreatedAt        time.Time `gorm:"not null"`
	UpdatedAt        time.Time `gorm:"not null"`
}

func (AlertEventPO) TableName() string { return "alert_events" }

// OpAuditLogPO 操作审计日志表
type OpAuditLogPO struct {
	ID              string    `gorm:"primaryKey;size:36"`
	OperatorID      string    `gorm:"size:36"`
	Module          string    `gorm:"size:32;not null;index:idx_op_audit_logs_module"`
	Action          string    `gorm:"size:32;not null"`
	TargetType      string    `gorm:"size:64;index:idx_op_audit_logs_target"`
	TargetID        string    `gorm:"size:36;index:idx_op_audit_logs_target"`
	RequestSnapshot string    `gorm:"type:jsonb;not null;default:'{}'"`
	ResultCode      int       `gorm:"not null;default:0"`
	IP              string    `gorm:"size:45"`
	CreatedAt       time.Time `gorm:"not null;index:idx_op_audit_logs_module"`
}

func (OpAuditLogPO) TableName() string { return "op_audit_logs" }

// CredentialPO 凭证表
type CredentialPO struct {
	ID          string         `gorm:"primaryKey;size:36"`
	Name        string         `gorm:"size:128;not null"`
	CredType    string         `gorm:"size:32;not null;default:ssh_key"`
	SecretData  string         `gorm:"type:text;not null"`
	Passphrase  string         `gorm:"type:text"`
	Fingerprint string         `gorm:"size:128"`
	Description string         `gorm:"type:text"`
	CreatedBy   string         `gorm:"size:36"`
	CreatedAt   time.Time      `gorm:"not null"`
	UpdatedAt   time.Time      `gorm:"not null"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (CredentialPO) TableName() string { return "credentials" }

// HostPO 机器表
type HostPO struct {
	ID            string `gorm:"primaryKey;size:36"`
	Name          string `gorm:"size:128;not null"`
	HostAddress   string `gorm:"size:256;not null"`
	SSHPort       int    `gorm:"not null;default:22"`
	Username      string `gorm:"size:128;not null;default:''"`
	CredentialID  string `gorm:"size:36"`
	IsProd        bool   `gorm:"not null;default:false;index"`
	Labels        string `gorm:"type:jsonb;not null;default:'{}'"`
	OsInfo        string `gorm:"size:256"`
	AgentStatus   string `gorm:"size:16;not null;default:unknown"`
	LastHeartbeat *time.Time
	Description   string         `gorm:"type:text"`
	CreatedAt     time.Time      `gorm:"not null"`
	UpdatedAt     time.Time      `gorm:"not null"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

func (HostPO) TableName() string { return "hosts" }

// ReleaseStepLogPO 发布步骤日志表
type ReleaseStepLogPO struct {
	ID          string `gorm:"primaryKey;size:36"`
	ReleaseID   string `gorm:"size:36;not null;index:idx_release_step_logs_release"`
	StepOrder   int    `gorm:"not null"`
	StepName    string `gorm:"size:64;not null"`
	StepStatus  string `gorm:"size:16;not null;default:pending"`
	StartedAt   *time.Time
	EndedAt     *time.Time
	DurationMs  int
	Output      string    `gorm:"type:text"`
	ErrorOutput string    `gorm:"type:text"`
	CreatedAt   time.Time `gorm:"not null"`
}

func (ReleaseStepLogPO) TableName() string { return "release_step_logs" }

// SystemSettingPO 系统设置表
type SystemSettingPO struct {
	ID          string    `gorm:"primaryKey;size:36"`
	SettingKey  string    `gorm:"size:128;not null;uniqueIndex"`
	Value       string    `gorm:"type:text;not null"`
	ValueType   string    `gorm:"size:16;not null;default:string"`
	Category    string    `gorm:"size:32;not null;default:general"`
	Description string    `gorm:"type:text"`
	UpdatedBy   string    `gorm:"size:36"`
	UpdatedAt   time.Time `gorm:"not null"`
}

func (SystemSettingPO) TableName() string { return "system_settings" }

// HostMetricSnapshotPO 主机指标快照表
type HostMetricSnapshotPO struct {
	ID          string `gorm:"primaryKey;size:36"`
	HostID      string `gorm:"size:36;not null;index:idx_host_metrics_host_time"`
	CPUUsage    float64
	CPUCores    int
	MemTotalMB  int64
	MemUsedMB   int64
	MemUsage    float64
	DiskTotalGB int64
	DiskUsedGB  int64
	DiskUsage   float64
	DisksJSON   string `gorm:"type:jsonb;default:'[]'"`
	LoadAvg1    float64
	LoadAvg5    float64
	LoadAvg15   float64
	NetInBytes  int64
	NetOutBytes int64
	GPUUsage    *float64
	GPUMemUsage *float64
	GPUTemp     *float64
	GPUName     string    `gorm:"size:128"`
	GPUsJSON    string    `gorm:"type:jsonb;default:'[]'"`
	CollectedAt time.Time `gorm:"not null;index:idx_host_metrics_host_time"`
	CreatedAt   time.Time `gorm:"not null"`
}

func (HostMetricSnapshotPO) TableName() string { return "host_metric_snapshots" }

// ============================
// 通知模块 PO
// ============================

// ContainerPO 容器表
type ContainerPO struct {
	ID            string `gorm:"primaryKey;size:36"`
	HostID        string `gorm:"size:36;not null;index:idx_containers_host;uniqueIndex:uk_container_host_name"`
	ContainerID   string `gorm:"size:64"`
	ContainerName string `gorm:"size:128;not null;uniqueIndex:uk_container_host_name"`
	Image         string `gorm:"size:256"`
	Status        string `gorm:"size:32;not null;default:unknown;index:idx_containers_status"`
	ConfigPaths   string `gorm:"type:jsonb;not null;default:'[]'"`
	Description   string `gorm:"type:text"`
	LastSyncedAt  *time.Time
	CreatedAt     time.Time      `gorm:"not null"`
	UpdatedAt     time.Time      `gorm:"not null"`
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

func (ContainerPO) TableName() string { return "containers" }

// NotificationChannelPO 通知渠道配置表
type NotificationChannelPO struct {
	ID          string         `gorm:"primaryKey;size:36"`
	Name        string         `gorm:"size:128;not null"`
	ChannelType string         `gorm:"size:32;not null"`
	Config      string         `gorm:"type:jsonb;not null;default:'{}'"`
	Enabled     bool           `gorm:"not null;default:true"`
	CreatedAt   time.Time      `gorm:"not null"`
	UpdatedAt   time.Time      `gorm:"not null"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (NotificationChannelPO) TableName() string { return "notification_channels" }

// NotificationRulePO 通知规则表
type NotificationRulePO struct {
	ID        string         `gorm:"primaryKey;size:36"`
	EventType string         `gorm:"size:64;not null;index"`
	ChannelID string         `gorm:"size:36;not null;index"`
	Enabled   bool           `gorm:"not null;default:true"`
	Filter    string         `gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt time.Time      `gorm:"not null"`
	UpdatedAt time.Time      `gorm:"not null"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (NotificationRulePO) TableName() string { return "notification_rules" }

// NotificationLogPO 通知记录表
type NotificationLogPO struct {
	ID           string    `gorm:"primaryKey;size:36"`
	ChannelID    string    `gorm:"size:36;not null;index"`
	EventType    string    `gorm:"size:64;not null;index"`
	Title        string    `gorm:"size:256;not null"`
	Content      string    `gorm:"type:text;not null"`
	Status       string    `gorm:"size:16;not null;default:sent"`
	ErrorMessage string    `gorm:"type:text"`
	CreatedAt    time.Time `gorm:"not null;index"`
}

func (NotificationLogPO) TableName() string { return "notification_logs" }
