package backup

import (
	"errors"
	"time"
)

// BackupType 备份类型
type BackupType string

const (
	BackupTypePostgres BackupType = "postgres"
	BackupTypeMySQL    BackupType = "mysql"
)

// BackupStatus 备份执行状态
type BackupStatus string

const (
	BackupStatusPending BackupStatus = "pending"
	BackupStatusRunning BackupStatus = "running"
	BackupStatusSuccess BackupStatus = "success"
	BackupStatusFailed  BackupStatus = "failed"
)

// MigrationMode 数据库迁移模式
type MigrationMode string

const (
	MigrationModeCreateIfMissing MigrationMode = "create_if_missing"
	MigrationModeOverwrite       MigrationMode = "overwrite"
)

// MigrationStatus 数据库迁移执行状态
type MigrationStatus string

const (
	MigrationStatusRunning        MigrationStatus = "running"
	MigrationStatusSuccess        MigrationStatus = "success"
	MigrationStatusPartialSuccess MigrationStatus = "partial_success"
	MigrationStatusFailed         MigrationStatus = "failed"
	MigrationStatusSkipped        MigrationStatus = "skipped"
)

// MigrationAction 数据库迁移动作
type MigrationAction string

const (
	MigrationActionCreated     MigrationAction = "created"
	MigrationActionOverwritten MigrationAction = "overwritten"
	MigrationActionSkipped     MigrationAction = "skipped"
)

// ObjectSyncMode 对象存储同步模式
type ObjectSyncMode string

const (
	ObjectSyncModeCopyIfMissing ObjectSyncMode = "copy_if_missing"
	ObjectSyncModeOverwrite     ObjectSyncMode = "overwrite"
	ObjectSyncModeChecksumSkip  ObjectSyncMode = "checksum_skip"
)

// ObjectSyncStatus 对象存储同步执行状态
type ObjectSyncStatus string

const (
	ObjectSyncStatusRunning        ObjectSyncStatus = "running"
	ObjectSyncStatusSuccess        ObjectSyncStatus = "success"
	ObjectSyncStatusPartialSuccess ObjectSyncStatus = "partial_success"
	ObjectSyncStatusFailed         ObjectSyncStatus = "failed"
	ObjectSyncStatusSkipped        ObjectSyncStatus = "skipped"
)

// ObjectSyncAction 对象存储同步动作
type ObjectSyncAction string

const (
	ObjectSyncActionCopied      ObjectSyncAction = "copied"
	ObjectSyncActionOverwritten ObjectSyncAction = "overwritten"
	ObjectSyncActionSkipped     ObjectSyncAction = "skipped"
)

// BackupTask 备份任务（聚合根）
type BackupTask struct {
	ID            string
	Name          string
	BackupType    BackupType
	CronExpr      string // cron 表达式，如 "0 2 * * *"
	Enabled       bool
	DBHost        string
	DBPort        int
	DBUser        string
	DBPassword    string // 加密存储
	DBName        string // 空=全库备份
	TargetHostID  string // 目标存储宿主机 ID
	TargetPath    string // 目标存储路径
	RetentionDays int    // 保留天数
	Description   string
	LastRunAt     *time.Time
	LastRunStatus BackupStatus
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// BackupRecord 备份执行记录
type BackupRecord struct {
	ID         string
	TaskID     string
	TaskName   string
	Status     BackupStatus
	FileName   string
	FileSize   int64 // bytes
	Duration   int64 // 秒
	Error      string
	StartedAt  time.Time
	FinishedAt *time.Time
	CreatedAt  time.Time
}

// MigrationTask 数据库迁移任务
type MigrationTask struct {
	ID             string
	Name           string
	DBType         BackupType
	SourceHost     string
	SourcePort     int
	SourceUser     string
	SourcePassword string
	DBNames        string
	TargetHost     string
	TargetPort     int
	TargetUser     string
	TargetPassword string
	Mode           MigrationMode
	Description    string
	LastRunAt      *time.Time
	LastRunStatus  MigrationStatus
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// MigrationRecord 数据库迁移执行记录
type MigrationRecord struct {
	ID         string
	TaskID     string
	TaskName   string
	DBType     BackupType
	Mode       MigrationMode
	Status     MigrationStatus
	SourceHost string
	TargetHost string
	DBNames    string
	Summary    string
	Error      string
	StartedAt  time.Time
	FinishedAt *time.Time
	Duration   int64
	CreatedAt  time.Time
	Items      []*MigrationRecordItem
}

// MigrationRecordItem 数据库迁移执行明细
type MigrationRecordItem struct {
	ID         string
	RecordID   string
	DBName     string
	Action     MigrationAction
	Status     MigrationStatus
	Message    string
	StartedAt  time.Time
	FinishedAt *time.Time
	Duration   int64
	CreatedAt  time.Time
}

// ObjectSyncTask 对象存储同步任务
type ObjectSyncTask struct {
	ID              string
	Name            string
	SourceEndpoint  string
	SourceRegion    string
	SourceBucket    string
	SourcePath      string
	SourceAccessKey string
	SourceSecretKey string
	SourceUseSSL    bool
	TargetEndpoint  string
	TargetRegion    string
	TargetBucket    string
	TargetPath      string
	TargetAccessKey string
	TargetSecretKey string
	TargetUseSSL    bool
	Mode            ObjectSyncMode
	Description     string
	LastRunAt       *time.Time
	LastRunStatus   ObjectSyncStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// ObjectSyncRecord 对象存储同步执行记录
type ObjectSyncRecord struct {
	ID           string
	TaskID       string
	TaskName     string
	Mode         ObjectSyncMode
	Status       ObjectSyncStatus
	SourceBucket string
	SourcePath   string
	TargetBucket string
	TargetPath   string
	ObjectCount  int
	SuccessCount int
	SkippedCount int
	FailedCount  int
	BytesTotal   int64
	Summary      string
	Error        string
	StartedAt    time.Time
	FinishedAt   *time.Time
	Duration     int64
	CreatedAt    time.Time
	Items        []*ObjectSyncRecordItem
}

// ObjectSyncRecordItem 对象存储同步明细
type ObjectSyncRecordItem struct {
	ID         string
	RecordID   string
	SourceKey  string
	TargetKey  string
	Size       int64
	ETag       string
	Action     ObjectSyncAction
	Status     ObjectSyncStatus
	Message    string
	StartedAt  time.Time
	FinishedAt *time.Time
	Duration   int64
	CreatedAt  time.Time
}

// 领域错误
var (
	ErrTaskNotFound             = errors.New("备份任务不存在")
	ErrTaskNameExists           = errors.New("备份任务名称已存在")
	ErrRecordNotFound           = errors.New("备份记录不存在")
	ErrInvalidCronExpr          = errors.New("无效的 cron 表达式")
	ErrTaskDisabled             = errors.New("备份任务已禁用")
	ErrMigrationTaskNotFound    = errors.New("迁移任务不存在")
	ErrMigrationRecordNotFound  = errors.New("迁移记录不存在")
	ErrInvalidMigrationMode     = errors.New("无效的迁移模式")
	ErrInvalidMigrationDBType   = errors.New("无效的迁移数据库类型")
	ErrMigrationDBNamesRequired = errors.New("迁移数据库列表不能为空")
	ErrOverwriteConfirmRequired = errors.New("覆盖迁移需要二次确认")
	ErrObjectSyncTaskNotFound   = errors.New("对象同步任务不存在")
	ErrObjectSyncRecordNotFound = errors.New("对象同步记录不存在")
	ErrInvalidObjectSyncMode    = errors.New("无效的对象同步模式")
)
