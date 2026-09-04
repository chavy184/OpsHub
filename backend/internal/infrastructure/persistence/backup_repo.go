package persistence

import (
	"context"
	"ops-hub/internal/domain/backup"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ============================
// PO (Persistent Objects)
// ============================

type BackupTaskPO struct {
	ID            string `gorm:"primaryKey;size:36"`
	Name          string `gorm:"size:100;not null"`
	BackupType    string `gorm:"size:20;not null"`
	CronExpr      string `gorm:"size:50;not null"`
	Enabled       bool   `gorm:"default:true"`
	DBHost        string `gorm:"size:200;not null"`
	DBPort        int    `gorm:"not null"`
	DBUser        string `gorm:"size:100;not null"`
	DBPassword    string `gorm:"type:text;not null"`
	DBName        string `gorm:"size:100"`
	TargetHostID  string `gorm:"size:36;not null"`
	TargetPath    string `gorm:"size:500;not null"`
	RetentionDays int    `gorm:"default:10"`
	Description   string `gorm:"size:500"`
	LastRunAt     *time.Time
	LastRunStatus string `gorm:"size:20"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`
}

func (BackupTaskPO) TableName() string { return "backup_tasks" }

type BackupRecordPO struct {
	ID         string    `gorm:"primaryKey;size:36"`
	TaskID     string    `gorm:"size:36;not null;index"`
	TaskName   string    `gorm:"size:100"`
	Status     string    `gorm:"size:20;not null"`
	FileName   string    `gorm:"size:500"`
	FileSize   int64     `gorm:"default:0"`
	Duration   int64     `gorm:"default:0"`
	Error      string    `gorm:"type:text"`
	StartedAt  time.Time `gorm:"not null"`
	FinishedAt *time.Time
	CreatedAt  time.Time
}

func (BackupRecordPO) TableName() string { return "backup_records" }

type MigrationTaskPO struct {
	ID             string `gorm:"primaryKey;size:36"`
	Name           string `gorm:"size:100;not null"`
	DBType         string `gorm:"size:20;not null"`
	SourceHost     string `gorm:"size:200;not null"`
	SourcePort     int    `gorm:"not null"`
	SourceUser     string `gorm:"size:100;not null"`
	SourcePassword string `gorm:"type:text;not null"`
	DBNames        string `gorm:"type:text;not null"`
	TargetHost     string `gorm:"size:200;not null"`
	TargetPort     int    `gorm:"not null"`
	TargetUser     string `gorm:"size:100;not null"`
	TargetPassword string `gorm:"type:text;not null"`
	Mode           string `gorm:"size:30;not null"`
	Description    string `gorm:"size:500"`
	LastRunAt      *time.Time
	LastRunStatus  string `gorm:"size:30"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
}

func (MigrationTaskPO) TableName() string { return "migration_tasks" }

type MigrationRecordPO struct {
	ID         string    `gorm:"primaryKey;size:36"`
	TaskID     string    `gorm:"size:36;not null;index"`
	TaskName   string    `gorm:"size:100"`
	DBType     string    `gorm:"size:20;not null"`
	Mode       string    `gorm:"size:30;not null"`
	Status     string    `gorm:"size:30;not null;index"`
	SourceHost string    `gorm:"size:200"`
	TargetHost string    `gorm:"size:200"`
	DBNames    string    `gorm:"type:text"`
	Summary    string    `gorm:"size:500"`
	Error      string    `gorm:"type:text"`
	StartedAt  time.Time `gorm:"not null"`
	FinishedAt *time.Time
	Duration   int64 `gorm:"default:0"`
	CreatedAt  time.Time
}

func (MigrationRecordPO) TableName() string { return "migration_records" }

type MigrationRecordItemPO struct {
	ID         string    `gorm:"primaryKey;size:36"`
	RecordID   string    `gorm:"size:36;not null;index"`
	DBName     string    `gorm:"size:100;not null"`
	Action     string    `gorm:"size:30"`
	Status     string    `gorm:"size:30;not null;index"`
	Message    string    `gorm:"type:text"`
	StartedAt  time.Time `gorm:"not null"`
	FinishedAt *time.Time
	Duration   int64 `gorm:"default:0"`
	CreatedAt  time.Time
}

func (MigrationRecordItemPO) TableName() string { return "migration_record_items" }

type ObjectSyncTaskPO struct {
	ID              string `gorm:"primaryKey;size:36"`
	Name            string `gorm:"size:100;not null"`
	SourceEndpoint  string `gorm:"size:300;not null"`
	SourceRegion    string `gorm:"size:100"`
	SourceBucket    string `gorm:"size:200;not null"`
	SourcePath      string `gorm:"size:1000"`
	SourceAccessKey string `gorm:"type:text;not null"`
	SourceSecretKey string `gorm:"type:text;not null"`
	SourceUseSSL    bool   `gorm:"default:true"`
	TargetEndpoint  string `gorm:"size:300;not null"`
	TargetRegion    string `gorm:"size:100"`
	TargetBucket    string `gorm:"size:200;not null"`
	TargetPath      string `gorm:"size:1000"`
	TargetAccessKey string `gorm:"type:text;not null"`
	TargetSecretKey string `gorm:"type:text;not null"`
	TargetUseSSL    bool   `gorm:"default:true"`
	Mode            string `gorm:"size:30;not null"`
	Description     string `gorm:"size:500"`
	LastRunAt       *time.Time
	LastRunStatus   string `gorm:"size:30"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
	DeletedAt       gorm.DeletedAt `gorm:"index"`
}

func (ObjectSyncTaskPO) TableName() string { return "object_sync_tasks" }

type ObjectSyncRecordPO struct {
	ID           string    `gorm:"primaryKey;size:36"`
	TaskID       string    `gorm:"size:36;not null;index"`
	TaskName     string    `gorm:"size:100"`
	Mode         string    `gorm:"size:30;not null"`
	Status       string    `gorm:"size:30;not null;index"`
	SourceBucket string    `gorm:"size:200"`
	SourcePath   string    `gorm:"size:1000"`
	TargetBucket string    `gorm:"size:200"`
	TargetPath   string    `gorm:"size:1000"`
	ObjectCount  int       `gorm:"default:0"`
	SuccessCount int       `gorm:"default:0"`
	SkippedCount int       `gorm:"default:0"`
	FailedCount  int       `gorm:"default:0"`
	BytesTotal   int64     `gorm:"default:0"`
	Summary      string    `gorm:"size:500"`
	Error        string    `gorm:"type:text"`
	StartedAt    time.Time `gorm:"not null"`
	FinishedAt   *time.Time
	Duration     int64 `gorm:"default:0"`
	CreatedAt    time.Time
}

func (ObjectSyncRecordPO) TableName() string { return "object_sync_records" }

type ObjectSyncRecordItemPO struct {
	ID         string    `gorm:"primaryKey;size:36"`
	RecordID   string    `gorm:"size:36;not null;index"`
	SourceKey  string    `gorm:"size:1000;not null"`
	TargetKey  string    `gorm:"size:1000;not null"`
	Size       int64     `gorm:"default:0"`
	ETag       string    `gorm:"size:200"`
	Action     string    `gorm:"size:30"`
	Status     string    `gorm:"size:30;not null;index"`
	Message    string    `gorm:"type:text"`
	StartedAt  time.Time `gorm:"not null"`
	FinishedAt *time.Time
	Duration   int64 `gorm:"default:0"`
	CreatedAt  time.Time
}

func (ObjectSyncRecordItemPO) TableName() string { return "object_sync_record_items" }

// ============================
// Assemblers
// ============================

func backupTaskEntityToPO(e *backup.BackupTask) *BackupTaskPO {
	return &BackupTaskPO{
		ID:            e.ID,
		Name:          e.Name,
		BackupType:    string(e.BackupType),
		CronExpr:      e.CronExpr,
		Enabled:       e.Enabled,
		DBHost:        e.DBHost,
		DBPort:        e.DBPort,
		DBUser:        e.DBUser,
		DBPassword:    e.DBPassword,
		DBName:        e.DBName,
		TargetHostID:  e.TargetHostID,
		TargetPath:    e.TargetPath,
		RetentionDays: e.RetentionDays,
		Description:   e.Description,
		LastRunAt:     e.LastRunAt,
		LastRunStatus: string(e.LastRunStatus),
		CreatedAt:     e.CreatedAt,
		UpdatedAt:     e.UpdatedAt,
	}
}

func backupTaskPOToEntity(po *BackupTaskPO) *backup.BackupTask {
	return &backup.BackupTask{
		ID:            po.ID,
		Name:          po.Name,
		BackupType:    backup.BackupType(po.BackupType),
		CronExpr:      po.CronExpr,
		Enabled:       po.Enabled,
		DBHost:        po.DBHost,
		DBPort:        po.DBPort,
		DBUser:        po.DBUser,
		DBPassword:    po.DBPassword,
		DBName:        po.DBName,
		TargetHostID:  po.TargetHostID,
		TargetPath:    po.TargetPath,
		RetentionDays: po.RetentionDays,
		Description:   po.Description,
		LastRunAt:     po.LastRunAt,
		LastRunStatus: backup.BackupStatus(po.LastRunStatus),
		CreatedAt:     po.CreatedAt,
		UpdatedAt:     po.UpdatedAt,
	}
}

func backupRecordEntityToPO(e *backup.BackupRecord) *BackupRecordPO {
	return &BackupRecordPO{
		ID:         e.ID,
		TaskID:     e.TaskID,
		TaskName:   e.TaskName,
		Status:     string(e.Status),
		FileName:   e.FileName,
		FileSize:   e.FileSize,
		Duration:   e.Duration,
		Error:      e.Error,
		StartedAt:  e.StartedAt,
		FinishedAt: e.FinishedAt,
		CreatedAt:  e.CreatedAt,
	}
}

func backupRecordPOToEntity(po *BackupRecordPO) *backup.BackupRecord {
	return &backup.BackupRecord{
		ID:         po.ID,
		TaskID:     po.TaskID,
		TaskName:   po.TaskName,
		Status:     backup.BackupStatus(po.Status),
		FileName:   po.FileName,
		FileSize:   po.FileSize,
		Duration:   po.Duration,
		Error:      po.Error,
		StartedAt:  po.StartedAt,
		FinishedAt: po.FinishedAt,
		CreatedAt:  po.CreatedAt,
	}
}

func migrationTaskEntityToPO(e *backup.MigrationTask) *MigrationTaskPO {
	return &MigrationTaskPO{
		ID:             e.ID,
		Name:           e.Name,
		DBType:         string(e.DBType),
		SourceHost:     e.SourceHost,
		SourcePort:     e.SourcePort,
		SourceUser:     e.SourceUser,
		SourcePassword: e.SourcePassword,
		DBNames:        e.DBNames,
		TargetHost:     e.TargetHost,
		TargetPort:     e.TargetPort,
		TargetUser:     e.TargetUser,
		TargetPassword: e.TargetPassword,
		Mode:           string(e.Mode),
		Description:    e.Description,
		LastRunAt:      e.LastRunAt,
		LastRunStatus:  string(e.LastRunStatus),
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}
}

func migrationTaskPOToEntity(po *MigrationTaskPO) *backup.MigrationTask {
	return &backup.MigrationTask{
		ID:             po.ID,
		Name:           po.Name,
		DBType:         backup.BackupType(po.DBType),
		SourceHost:     po.SourceHost,
		SourcePort:     po.SourcePort,
		SourceUser:     po.SourceUser,
		SourcePassword: po.SourcePassword,
		DBNames:        po.DBNames,
		TargetHost:     po.TargetHost,
		TargetPort:     po.TargetPort,
		TargetUser:     po.TargetUser,
		TargetPassword: po.TargetPassword,
		Mode:           backup.MigrationMode(po.Mode),
		Description:    po.Description,
		LastRunAt:      po.LastRunAt,
		LastRunStatus:  backup.MigrationStatus(po.LastRunStatus),
		CreatedAt:      po.CreatedAt,
		UpdatedAt:      po.UpdatedAt,
	}
}

func migrationRecordEntityToPO(e *backup.MigrationRecord) *MigrationRecordPO {
	return &MigrationRecordPO{
		ID:         e.ID,
		TaskID:     e.TaskID,
		TaskName:   e.TaskName,
		DBType:     string(e.DBType),
		Mode:       string(e.Mode),
		Status:     string(e.Status),
		SourceHost: e.SourceHost,
		TargetHost: e.TargetHost,
		DBNames:    e.DBNames,
		Summary:    e.Summary,
		Error:      e.Error,
		StartedAt:  e.StartedAt,
		FinishedAt: e.FinishedAt,
		Duration:   e.Duration,
		CreatedAt:  e.CreatedAt,
	}
}

func migrationRecordPOToEntity(po *MigrationRecordPO) *backup.MigrationRecord {
	return &backup.MigrationRecord{
		ID:         po.ID,
		TaskID:     po.TaskID,
		TaskName:   po.TaskName,
		DBType:     backup.BackupType(po.DBType),
		Mode:       backup.MigrationMode(po.Mode),
		Status:     backup.MigrationStatus(po.Status),
		SourceHost: po.SourceHost,
		TargetHost: po.TargetHost,
		DBNames:    po.DBNames,
		Summary:    po.Summary,
		Error:      po.Error,
		StartedAt:  po.StartedAt,
		FinishedAt: po.FinishedAt,
		Duration:   po.Duration,
		CreatedAt:  po.CreatedAt,
	}
}

func migrationRecordItemEntityToPO(e *backup.MigrationRecordItem) *MigrationRecordItemPO {
	return &MigrationRecordItemPO{
		ID:         e.ID,
		RecordID:   e.RecordID,
		DBName:     e.DBName,
		Action:     string(e.Action),
		Status:     string(e.Status),
		Message:    e.Message,
		StartedAt:  e.StartedAt,
		FinishedAt: e.FinishedAt,
		Duration:   e.Duration,
		CreatedAt:  e.CreatedAt,
	}
}

func migrationRecordItemPOToEntity(po *MigrationRecordItemPO) *backup.MigrationRecordItem {
	return &backup.MigrationRecordItem{
		ID:         po.ID,
		RecordID:   po.RecordID,
		DBName:     po.DBName,
		Action:     backup.MigrationAction(po.Action),
		Status:     backup.MigrationStatus(po.Status),
		Message:    po.Message,
		StartedAt:  po.StartedAt,
		FinishedAt: po.FinishedAt,
		Duration:   po.Duration,
		CreatedAt:  po.CreatedAt,
	}
}

func objectSyncTaskEntityToPO(e *backup.ObjectSyncTask) *ObjectSyncTaskPO {
	return &ObjectSyncTaskPO{
		ID:              e.ID,
		Name:            e.Name,
		SourceEndpoint:  e.SourceEndpoint,
		SourceRegion:    e.SourceRegion,
		SourceBucket:    e.SourceBucket,
		SourcePath:      e.SourcePath,
		SourceAccessKey: e.SourceAccessKey,
		SourceSecretKey: e.SourceSecretKey,
		SourceUseSSL:    e.SourceUseSSL,
		TargetEndpoint:  e.TargetEndpoint,
		TargetRegion:    e.TargetRegion,
		TargetBucket:    e.TargetBucket,
		TargetPath:      e.TargetPath,
		TargetAccessKey: e.TargetAccessKey,
		TargetSecretKey: e.TargetSecretKey,
		TargetUseSSL:    e.TargetUseSSL,
		Mode:            string(e.Mode),
		Description:     e.Description,
		LastRunAt:       e.LastRunAt,
		LastRunStatus:   string(e.LastRunStatus),
		CreatedAt:       e.CreatedAt,
		UpdatedAt:       e.UpdatedAt,
	}
}

func objectSyncTaskPOToEntity(po *ObjectSyncTaskPO) *backup.ObjectSyncTask {
	return &backup.ObjectSyncTask{
		ID:              po.ID,
		Name:            po.Name,
		SourceEndpoint:  po.SourceEndpoint,
		SourceRegion:    po.SourceRegion,
		SourceBucket:    po.SourceBucket,
		SourcePath:      po.SourcePath,
		SourceAccessKey: po.SourceAccessKey,
		SourceSecretKey: po.SourceSecretKey,
		SourceUseSSL:    po.SourceUseSSL,
		TargetEndpoint:  po.TargetEndpoint,
		TargetRegion:    po.TargetRegion,
		TargetBucket:    po.TargetBucket,
		TargetPath:      po.TargetPath,
		TargetAccessKey: po.TargetAccessKey,
		TargetSecretKey: po.TargetSecretKey,
		TargetUseSSL:    po.TargetUseSSL,
		Mode:            backup.ObjectSyncMode(po.Mode),
		Description:     po.Description,
		LastRunAt:       po.LastRunAt,
		LastRunStatus:   backup.ObjectSyncStatus(po.LastRunStatus),
		CreatedAt:       po.CreatedAt,
		UpdatedAt:       po.UpdatedAt,
	}
}

func objectSyncRecordEntityToPO(e *backup.ObjectSyncRecord) *ObjectSyncRecordPO {
	return &ObjectSyncRecordPO{
		ID:           e.ID,
		TaskID:       e.TaskID,
		TaskName:     e.TaskName,
		Mode:         string(e.Mode),
		Status:       string(e.Status),
		SourceBucket: e.SourceBucket,
		SourcePath:   e.SourcePath,
		TargetBucket: e.TargetBucket,
		TargetPath:   e.TargetPath,
		ObjectCount:  e.ObjectCount,
		SuccessCount: e.SuccessCount,
		SkippedCount: e.SkippedCount,
		FailedCount:  e.FailedCount,
		BytesTotal:   e.BytesTotal,
		Summary:      e.Summary,
		Error:        e.Error,
		StartedAt:    e.StartedAt,
		FinishedAt:   e.FinishedAt,
		Duration:     e.Duration,
		CreatedAt:    e.CreatedAt,
	}
}

func objectSyncRecordPOToEntity(po *ObjectSyncRecordPO) *backup.ObjectSyncRecord {
	return &backup.ObjectSyncRecord{
		ID:           po.ID,
		TaskID:       po.TaskID,
		TaskName:     po.TaskName,
		Mode:         backup.ObjectSyncMode(po.Mode),
		Status:       backup.ObjectSyncStatus(po.Status),
		SourceBucket: po.SourceBucket,
		SourcePath:   po.SourcePath,
		TargetBucket: po.TargetBucket,
		TargetPath:   po.TargetPath,
		ObjectCount:  po.ObjectCount,
		SuccessCount: po.SuccessCount,
		SkippedCount: po.SkippedCount,
		FailedCount:  po.FailedCount,
		BytesTotal:   po.BytesTotal,
		Summary:      po.Summary,
		Error:        po.Error,
		StartedAt:    po.StartedAt,
		FinishedAt:   po.FinishedAt,
		Duration:     po.Duration,
		CreatedAt:    po.CreatedAt,
	}
}

func objectSyncRecordItemEntityToPO(e *backup.ObjectSyncRecordItem) *ObjectSyncRecordItemPO {
	return &ObjectSyncRecordItemPO{
		ID:         e.ID,
		RecordID:   e.RecordID,
		SourceKey:  e.SourceKey,
		TargetKey:  e.TargetKey,
		Size:       e.Size,
		ETag:       e.ETag,
		Action:     string(e.Action),
		Status:     string(e.Status),
		Message:    e.Message,
		StartedAt:  e.StartedAt,
		FinishedAt: e.FinishedAt,
		Duration:   e.Duration,
		CreatedAt:  e.CreatedAt,
	}
}

func objectSyncRecordItemPOToEntity(po *ObjectSyncRecordItemPO) *backup.ObjectSyncRecordItem {
	return &backup.ObjectSyncRecordItem{
		ID:         po.ID,
		RecordID:   po.RecordID,
		SourceKey:  po.SourceKey,
		TargetKey:  po.TargetKey,
		Size:       po.Size,
		ETag:       po.ETag,
		Action:     backup.ObjectSyncAction(po.Action),
		Status:     backup.ObjectSyncStatus(po.Status),
		Message:    po.Message,
		StartedAt:  po.StartedAt,
		FinishedAt: po.FinishedAt,
		Duration:   po.Duration,
		CreatedAt:  po.CreatedAt,
	}
}

// ============================
// BackupTaskRepository 实现
// ============================

type GormBackupTaskRepository struct {
	db *gorm.DB
}

func NewBackupTaskRepository(db *gorm.DB) *GormBackupTaskRepository {
	return &GormBackupTaskRepository{db: db}
}

func (r *GormBackupTaskRepository) Save(ctx context.Context, entity *backup.BackupTask) error {
	if entity.ID == "" {
		entity.ID = uuid.New().String()
	}
	po := backupTaskEntityToPO(entity)
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		return err
	}
	entity.CreatedAt = po.CreatedAt
	entity.UpdatedAt = po.UpdatedAt
	return nil
}

func (r *GormBackupTaskRepository) Update(ctx context.Context, entity *backup.BackupTask) error {
	po := backupTaskEntityToPO(entity)
	return r.db.WithContext(ctx).Save(po).Error
}

func (r *GormBackupTaskRepository) FindByID(ctx context.Context, id string) (*backup.BackupTask, error) {
	var po BackupTaskPO
	if err := r.db.WithContext(ctx).First(&po, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return backupTaskPOToEntity(&po), nil
}

func (r *GormBackupTaskRepository) Find(ctx context.Context, query backup.TaskQuery) ([]*backup.BackupTask, int64, error) {
	tx := r.db.WithContext(ctx).Model(&BackupTaskPO{})

	if query.Keyword != "" {
		kw := "%" + query.Keyword + "%"
		tx = tx.Where("name ILIKE ? OR description ILIKE ?", kw, kw)
	}

	var total int64
	tx.Count(&total)

	var pos []BackupTaskPO
	if err := tx.Order("created_at DESC").Offset(query.Offset()).Limit(query.PageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	entities := make([]*backup.BackupTask, len(pos))
	for i := range pos {
		entities[i] = backupTaskPOToEntity(&pos[i])
	}
	return entities, total, nil
}

func (r *GormBackupTaskRepository) FindAllEnabled(ctx context.Context) ([]*backup.BackupTask, error) {
	var pos []BackupTaskPO
	if err := r.db.WithContext(ctx).Where("enabled = ?", true).Find(&pos).Error; err != nil {
		return nil, err
	}
	entities := make([]*backup.BackupTask, len(pos))
	for i := range pos {
		entities[i] = backupTaskPOToEntity(&pos[i])
	}
	return entities, nil
}

func (r *GormBackupTaskRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&BackupTaskPO{}, "id = ?", id).Error
}

// ============================
// BackupRecordRepository 实现
// ============================

type GormBackupRecordRepository struct {
	db *gorm.DB
}

func NewBackupRecordRepository(db *gorm.DB) *GormBackupRecordRepository {
	return &GormBackupRecordRepository{db: db}
}

func (r *GormBackupRecordRepository) Save(ctx context.Context, entity *backup.BackupRecord) error {
	if entity.ID == "" {
		entity.ID = uuid.New().String()
	}
	po := backupRecordEntityToPO(entity)
	return r.db.WithContext(ctx).Create(po).Error
}

func (r *GormBackupRecordRepository) Update(ctx context.Context, entity *backup.BackupRecord) error {
	po := backupRecordEntityToPO(entity)
	return r.db.WithContext(ctx).Save(po).Error
}

func (r *GormBackupRecordRepository) FindByID(ctx context.Context, id string) (*backup.BackupRecord, error) {
	var po BackupRecordPO
	if err := r.db.WithContext(ctx).First(&po, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return backupRecordPOToEntity(&po), nil
}

func (r *GormBackupRecordRepository) Find(ctx context.Context, query backup.RecordQuery) ([]*backup.BackupRecord, int64, error) {
	tx := r.db.WithContext(ctx).Model(&BackupRecordPO{})

	if query.TaskID != "" {
		tx = tx.Where("task_id = ?", query.TaskID)
	}
	if query.Status != "" {
		tx = tx.Where("status = ?", query.Status)
	}

	var total int64
	tx.Count(&total)

	var pos []BackupRecordPO
	if err := tx.Order("started_at DESC").Offset(query.Offset()).Limit(query.PageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	entities := make([]*backup.BackupRecord, len(pos))
	for i := range pos {
		entities[i] = backupRecordPOToEntity(&pos[i])
	}
	return entities, total, nil
}

func (r *GormBackupRecordRepository) DeleteByTaskID(ctx context.Context, taskID string) error {
	return r.db.WithContext(ctx).Where("task_id = ?", taskID).Delete(&BackupRecordPO{}).Error
}

// ============================
// MigrationTaskRepository 实现
// ============================

type GormMigrationTaskRepository struct {
	db *gorm.DB
}

func NewMigrationTaskRepository(db *gorm.DB) *GormMigrationTaskRepository {
	return &GormMigrationTaskRepository{db: db}
}

func (r *GormMigrationTaskRepository) Save(ctx context.Context, entity *backup.MigrationTask) error {
	if entity.ID == "" {
		entity.ID = uuid.New().String()
	}
	po := migrationTaskEntityToPO(entity)
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		return err
	}
	entity.CreatedAt = po.CreatedAt
	entity.UpdatedAt = po.UpdatedAt
	return nil
}

func (r *GormMigrationTaskRepository) Update(ctx context.Context, entity *backup.MigrationTask) error {
	po := migrationTaskEntityToPO(entity)
	return r.db.WithContext(ctx).Save(po).Error
}

func (r *GormMigrationTaskRepository) FindByID(ctx context.Context, id string) (*backup.MigrationTask, error) {
	var po MigrationTaskPO
	if err := r.db.WithContext(ctx).First(&po, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return migrationTaskPOToEntity(&po), nil
}

func (r *GormMigrationTaskRepository) Find(ctx context.Context, query backup.MigrationTaskQuery) ([]*backup.MigrationTask, int64, error) {
	tx := r.db.WithContext(ctx).Model(&MigrationTaskPO{})

	if query.Keyword != "" {
		kw := "%" + query.Keyword + "%"
		tx = tx.Where("name ILIKE ? OR description ILIKE ? OR source_host ILIKE ? OR target_host ILIKE ?", kw, kw, kw, kw)
	}

	var total int64
	tx.Count(&total)

	var pos []MigrationTaskPO
	if err := tx.Order("created_at DESC").Offset(query.Offset()).Limit(query.PageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	entities := make([]*backup.MigrationTask, len(pos))
	for i := range pos {
		entities[i] = migrationTaskPOToEntity(&pos[i])
	}
	return entities, total, nil
}

func (r *GormMigrationTaskRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&MigrationTaskPO{}, "id = ?", id).Error
}

// ============================
// MigrationRecordRepository 实现
// ============================

type GormMigrationRecordRepository struct {
	db *gorm.DB
}

func NewMigrationRecordRepository(db *gorm.DB) *GormMigrationRecordRepository {
	return &GormMigrationRecordRepository{db: db}
}

func (r *GormMigrationRecordRepository) Save(ctx context.Context, entity *backup.MigrationRecord) error {
	if entity.ID == "" {
		entity.ID = uuid.New().String()
	}
	po := migrationRecordEntityToPO(entity)
	return r.db.WithContext(ctx).Create(po).Error
}

func (r *GormMigrationRecordRepository) Update(ctx context.Context, entity *backup.MigrationRecord) error {
	po := migrationRecordEntityToPO(entity)
	return r.db.WithContext(ctx).Save(po).Error
}

func (r *GormMigrationRecordRepository) FindByID(ctx context.Context, id string) (*backup.MigrationRecord, error) {
	var po MigrationRecordPO
	if err := r.db.WithContext(ctx).First(&po, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return migrationRecordPOToEntity(&po), nil
}

func (r *GormMigrationRecordRepository) Find(ctx context.Context, query backup.MigrationRecordQuery) ([]*backup.MigrationRecord, int64, error) {
	tx := r.db.WithContext(ctx).Model(&MigrationRecordPO{})

	if query.TaskID != "" {
		tx = tx.Where("task_id = ?", query.TaskID)
	}
	if query.Status != "" {
		tx = tx.Where("status = ?", query.Status)
	}

	var total int64
	tx.Count(&total)

	var pos []MigrationRecordPO
	if err := tx.Order("started_at DESC").Offset(query.Offset()).Limit(query.PageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	entities := make([]*backup.MigrationRecord, len(pos))
	for i := range pos {
		entities[i] = migrationRecordPOToEntity(&pos[i])
	}
	return entities, total, nil
}

func (r *GormMigrationRecordRepository) DeleteByTaskID(ctx context.Context, taskID string) error {
	return r.db.WithContext(ctx).Where("task_id = ?", taskID).Delete(&MigrationRecordPO{}).Error
}

// ============================
// MigrationRecordItemRepository 实现
// ============================

type GormMigrationRecordItemRepository struct {
	db *gorm.DB
}

func NewMigrationRecordItemRepository(db *gorm.DB) *GormMigrationRecordItemRepository {
	return &GormMigrationRecordItemRepository{db: db}
}

func (r *GormMigrationRecordItemRepository) Save(ctx context.Context, entity *backup.MigrationRecordItem) error {
	if entity.ID == "" {
		entity.ID = uuid.New().String()
	}
	po := migrationRecordItemEntityToPO(entity)
	return r.db.WithContext(ctx).Create(po).Error
}

func (r *GormMigrationRecordItemRepository) FindByRecordID(ctx context.Context, recordID string) ([]*backup.MigrationRecordItem, error) {
	var pos []MigrationRecordItemPO
	if err := r.db.WithContext(ctx).Where("record_id = ?", recordID).Order("started_at ASC").Find(&pos).Error; err != nil {
		return nil, err
	}
	entities := make([]*backup.MigrationRecordItem, len(pos))
	for i := range pos {
		entities[i] = migrationRecordItemPOToEntity(&pos[i])
	}
	return entities, nil
}

// ============================
// ObjectSyncTaskRepository 实现
// ============================

type GormObjectSyncTaskRepository struct {
	db *gorm.DB
}

func NewObjectSyncTaskRepository(db *gorm.DB) *GormObjectSyncTaskRepository {
	return &GormObjectSyncTaskRepository{db: db}
}

func (r *GormObjectSyncTaskRepository) Save(ctx context.Context, entity *backup.ObjectSyncTask) error {
	if entity.ID == "" {
		entity.ID = uuid.New().String()
	}
	po := objectSyncTaskEntityToPO(entity)
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		return err
	}
	entity.CreatedAt = po.CreatedAt
	entity.UpdatedAt = po.UpdatedAt
	return nil
}

func (r *GormObjectSyncTaskRepository) Update(ctx context.Context, entity *backup.ObjectSyncTask) error {
	po := objectSyncTaskEntityToPO(entity)
	return r.db.WithContext(ctx).Save(po).Error
}

func (r *GormObjectSyncTaskRepository) FindByID(ctx context.Context, id string) (*backup.ObjectSyncTask, error) {
	var po ObjectSyncTaskPO
	if err := r.db.WithContext(ctx).First(&po, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return objectSyncTaskPOToEntity(&po), nil
}

func (r *GormObjectSyncTaskRepository) Find(ctx context.Context, query backup.ObjectSyncTaskQuery) ([]*backup.ObjectSyncTask, int64, error) {
	tx := r.db.WithContext(ctx).Model(&ObjectSyncTaskPO{})
	if query.Keyword != "" {
		kw := "%" + query.Keyword + "%"
		tx = tx.Where("name ILIKE ? OR description ILIKE ? OR source_endpoint ILIKE ? OR target_endpoint ILIKE ?", kw, kw, kw, kw)
	}

	var total int64
	tx.Count(&total)

	var pos []ObjectSyncTaskPO
	if err := tx.Order("created_at DESC").Offset(query.Offset()).Limit(query.PageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}
	entities := make([]*backup.ObjectSyncTask, len(pos))
	for i := range pos {
		entities[i] = objectSyncTaskPOToEntity(&pos[i])
	}
	return entities, total, nil
}

func (r *GormObjectSyncTaskRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&ObjectSyncTaskPO{}, "id = ?", id).Error
}

// ============================
// ObjectSyncRecordRepository 实现
// ============================

type GormObjectSyncRecordRepository struct {
	db *gorm.DB
}

func NewObjectSyncRecordRepository(db *gorm.DB) *GormObjectSyncRecordRepository {
	return &GormObjectSyncRecordRepository{db: db}
}

func (r *GormObjectSyncRecordRepository) Save(ctx context.Context, entity *backup.ObjectSyncRecord) error {
	if entity.ID == "" {
		entity.ID = uuid.New().String()
	}
	po := objectSyncRecordEntityToPO(entity)
	return r.db.WithContext(ctx).Create(po).Error
}

func (r *GormObjectSyncRecordRepository) Update(ctx context.Context, entity *backup.ObjectSyncRecord) error {
	po := objectSyncRecordEntityToPO(entity)
	return r.db.WithContext(ctx).Save(po).Error
}

func (r *GormObjectSyncRecordRepository) FindByID(ctx context.Context, id string) (*backup.ObjectSyncRecord, error) {
	var po ObjectSyncRecordPO
	if err := r.db.WithContext(ctx).First(&po, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return objectSyncRecordPOToEntity(&po), nil
}

func (r *GormObjectSyncRecordRepository) Find(ctx context.Context, query backup.ObjectSyncRecordQuery) ([]*backup.ObjectSyncRecord, int64, error) {
	tx := r.db.WithContext(ctx).Model(&ObjectSyncRecordPO{})
	if query.TaskID != "" {
		tx = tx.Where("task_id = ?", query.TaskID)
	}
	if query.Status != "" {
		tx = tx.Where("status = ?", query.Status)
	}

	var total int64
	tx.Count(&total)

	var pos []ObjectSyncRecordPO
	if err := tx.Order("started_at DESC").Offset(query.Offset()).Limit(query.PageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}
	entities := make([]*backup.ObjectSyncRecord, len(pos))
	for i := range pos {
		entities[i] = objectSyncRecordPOToEntity(&pos[i])
	}
	return entities, total, nil
}

func (r *GormObjectSyncRecordRepository) DeleteByTaskID(ctx context.Context, taskID string) error {
	return r.db.WithContext(ctx).Where("task_id = ?", taskID).Delete(&ObjectSyncRecordPO{}).Error
}

// ============================
// ObjectSyncRecordItemRepository 实现
// ============================

type GormObjectSyncRecordItemRepository struct {
	db *gorm.DB
}

func NewObjectSyncRecordItemRepository(db *gorm.DB) *GormObjectSyncRecordItemRepository {
	return &GormObjectSyncRecordItemRepository{db: db}
}

func (r *GormObjectSyncRecordItemRepository) Save(ctx context.Context, entity *backup.ObjectSyncRecordItem) error {
	if entity.ID == "" {
		entity.ID = uuid.New().String()
	}
	po := objectSyncRecordItemEntityToPO(entity)
	return r.db.WithContext(ctx).Create(po).Error
}

func (r *GormObjectSyncRecordItemRepository) FindByRecordID(ctx context.Context, recordID string) ([]*backup.ObjectSyncRecordItem, error) {
	var pos []ObjectSyncRecordItemPO
	if err := r.db.WithContext(ctx).Where("record_id = ?", recordID).Order("started_at ASC").Find(&pos).Error; err != nil {
		return nil, err
	}
	entities := make([]*backup.ObjectSyncRecordItem, len(pos))
	for i := range pos {
		entities[i] = objectSyncRecordItemPOToEntity(&pos[i])
	}
	return entities, nil
}
