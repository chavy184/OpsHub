package backup

import "context"

// BackupTaskRepository 备份任务仓储接口
type BackupTaskRepository interface {
	Save(ctx context.Context, entity *BackupTask) error
	Update(ctx context.Context, entity *BackupTask) error
	FindByID(ctx context.Context, id string) (*BackupTask, error)
	Find(ctx context.Context, query TaskQuery) ([]*BackupTask, int64, error)
	FindAllEnabled(ctx context.Context) ([]*BackupTask, error)
	Delete(ctx context.Context, id string) error
}

// BackupRecordRepository 备份记录仓储接口
type BackupRecordRepository interface {
	Save(ctx context.Context, entity *BackupRecord) error
	Update(ctx context.Context, entity *BackupRecord) error
	FindByID(ctx context.Context, id string) (*BackupRecord, error)
	Find(ctx context.Context, query RecordQuery) ([]*BackupRecord, int64, error)
	DeleteByTaskID(ctx context.Context, taskID string) error
}

// MigrationTaskRepository 数据库迁移任务仓储接口
type MigrationTaskRepository interface {
	Save(ctx context.Context, entity *MigrationTask) error
	Update(ctx context.Context, entity *MigrationTask) error
	FindByID(ctx context.Context, id string) (*MigrationTask, error)
	Find(ctx context.Context, query MigrationTaskQuery) ([]*MigrationTask, int64, error)
	Delete(ctx context.Context, id string) error
}

// MigrationRecordRepository 数据库迁移记录仓储接口
type MigrationRecordRepository interface {
	Save(ctx context.Context, entity *MigrationRecord) error
	Update(ctx context.Context, entity *MigrationRecord) error
	FindByID(ctx context.Context, id string) (*MigrationRecord, error)
	Find(ctx context.Context, query MigrationRecordQuery) ([]*MigrationRecord, int64, error)
	DeleteByTaskID(ctx context.Context, taskID string) error
}

// MigrationRecordItemRepository 数据库迁移明细仓储接口
type MigrationRecordItemRepository interface {
	Save(ctx context.Context, entity *MigrationRecordItem) error
	FindByRecordID(ctx context.Context, recordID string) ([]*MigrationRecordItem, error)
}

// ObjectSyncTaskRepository 对象存储同步任务仓储接口
type ObjectSyncTaskRepository interface {
	Save(ctx context.Context, entity *ObjectSyncTask) error
	Update(ctx context.Context, entity *ObjectSyncTask) error
	FindByID(ctx context.Context, id string) (*ObjectSyncTask, error)
	Find(ctx context.Context, query ObjectSyncTaskQuery) ([]*ObjectSyncTask, int64, error)
	Delete(ctx context.Context, id string) error
}

// ObjectSyncRecordRepository 对象存储同步记录仓储接口
type ObjectSyncRecordRepository interface {
	Save(ctx context.Context, entity *ObjectSyncRecord) error
	Update(ctx context.Context, entity *ObjectSyncRecord) error
	FindByID(ctx context.Context, id string) (*ObjectSyncRecord, error)
	Find(ctx context.Context, query ObjectSyncRecordQuery) ([]*ObjectSyncRecord, int64, error)
	DeleteByTaskID(ctx context.Context, taskID string) error
}

// ObjectSyncRecordItemRepository 对象存储同步明细仓储接口
type ObjectSyncRecordItemRepository interface {
	Save(ctx context.Context, entity *ObjectSyncRecordItem) error
	FindByRecordID(ctx context.Context, recordID string) ([]*ObjectSyncRecordItem, error)
}

// TaskQuery 任务查询对象
type TaskQuery struct {
	Keyword  string
	Page     int
	PageSize int
}

func (q TaskQuery) Offset() int {
	if q.Page < 1 {
		q.Page = 1
	}
	return (q.Page - 1) * q.PageSize
}

// RecordQuery 记录查询对象
type RecordQuery struct {
	TaskID   string
	Status   string
	Page     int
	PageSize int
}

func (q RecordQuery) Offset() int {
	if q.Page < 1 {
		q.Page = 1
	}
	return (q.Page - 1) * q.PageSize
}

// MigrationTaskQuery 迁移任务查询对象
type MigrationTaskQuery struct {
	Keyword  string
	Page     int
	PageSize int
}

func (q MigrationTaskQuery) Offset() int {
	if q.Page < 1 {
		q.Page = 1
	}
	return (q.Page - 1) * q.PageSize
}

// MigrationRecordQuery 迁移记录查询对象
type MigrationRecordQuery struct {
	TaskID   string
	Status   string
	Page     int
	PageSize int
}

func (q MigrationRecordQuery) Offset() int {
	if q.Page < 1 {
		q.Page = 1
	}
	return (q.Page - 1) * q.PageSize
}

// ObjectSyncTaskQuery 对象同步任务查询对象
type ObjectSyncTaskQuery struct {
	Keyword  string
	Page     int
	PageSize int
}

func (q ObjectSyncTaskQuery) Offset() int {
	if q.Page < 1 {
		q.Page = 1
	}
	return (q.Page - 1) * q.PageSize
}

// ObjectSyncRecordQuery 对象同步记录查询对象
type ObjectSyncRecordQuery struct {
	TaskID   string
	Status   string
	Page     int
	PageSize int
}

func (q ObjectSyncRecordQuery) Offset() int {
	if q.Page < 1 {
		q.Page = 1
	}
	return (q.Page - 1) * q.PageSize
}
