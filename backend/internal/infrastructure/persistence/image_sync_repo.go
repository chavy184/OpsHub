package persistence

import (
	"context"
	"ops-hub/internal/domain/image_sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ImageSyncRecordPO struct {
	ID             string `gorm:"primaryKey;size:36"`
	SourceHostID   string `gorm:"size:36;not null;index"`
	SourceHostName string `gorm:"size:100"`
	TargetHostID   string `gorm:"size:36;not null;index"`
	TargetHostName string `gorm:"size:100"`
	Image          string `gorm:"size:500;not null;index"`
	Mode           string `gorm:"size:30;not null"`
	Status         string `gorm:"size:30;not null;index"`
	SourceImageID  string `gorm:"size:200"`
	TargetImageID  string `gorm:"size:200"`
	ImageSize      int64  `gorm:"default:0"`
	Summary        string `gorm:"size:500"`
	Error          string `gorm:"type:text"`
	StartedAt      time.Time
	FinishedAt     *time.Time
	Duration       int64 `gorm:"default:0"`
	CreatedAt      time.Time
}

func (ImageSyncRecordPO) TableName() string { return "image_sync_records" }

func imageSyncRecordEntityToPO(e *image_sync.Record) *ImageSyncRecordPO {
	return &ImageSyncRecordPO{
		ID:             e.ID,
		SourceHostID:   e.SourceHostID,
		SourceHostName: e.SourceHostName,
		TargetHostID:   e.TargetHostID,
		TargetHostName: e.TargetHostName,
		Image:          e.Image,
		Mode:           string(e.Mode),
		Status:         string(e.Status),
		SourceImageID:  e.SourceImageID,
		TargetImageID:  e.TargetImageID,
		ImageSize:      e.ImageSize,
		Summary:        e.Summary,
		Error:          e.Error,
		StartedAt:      e.StartedAt,
		FinishedAt:     e.FinishedAt,
		Duration:       e.Duration,
		CreatedAt:      e.CreatedAt,
	}
}

func imageSyncRecordPOToEntity(po *ImageSyncRecordPO) *image_sync.Record {
	return &image_sync.Record{
		ID:             po.ID,
		SourceHostID:   po.SourceHostID,
		SourceHostName: po.SourceHostName,
		TargetHostID:   po.TargetHostID,
		TargetHostName: po.TargetHostName,
		Image:          po.Image,
		Mode:           image_sync.SyncMode(po.Mode),
		Status:         image_sync.SyncStatus(po.Status),
		SourceImageID:  po.SourceImageID,
		TargetImageID:  po.TargetImageID,
		ImageSize:      po.ImageSize,
		Summary:        po.Summary,
		Error:          po.Error,
		StartedAt:      po.StartedAt,
		FinishedAt:     po.FinishedAt,
		Duration:       po.Duration,
		CreatedAt:      po.CreatedAt,
	}
}

type ImageSyncRecordRepository struct {
	db *gorm.DB
}

func NewImageSyncRecordRepository(db *gorm.DB) *ImageSyncRecordRepository {
	return &ImageSyncRecordRepository{db: db}
}

func (r *ImageSyncRecordRepository) Save(ctx context.Context, entity *image_sync.Record) error {
	if entity.ID == "" {
		entity.ID = uuid.New().String()
	}
	po := imageSyncRecordEntityToPO(entity)
	if err := r.db.WithContext(ctx).Create(po).Error; err != nil {
		return err
	}
	entity.CreatedAt = po.CreatedAt
	return nil
}

func (r *ImageSyncRecordRepository) Update(ctx context.Context, entity *image_sync.Record) error {
	po := imageSyncRecordEntityToPO(entity)
	return r.db.WithContext(ctx).Save(po).Error
}

func (r *ImageSyncRecordRepository) FindByID(ctx context.Context, id string) (*image_sync.Record, error) {
	var po ImageSyncRecordPO
	if err := r.db.WithContext(ctx).First(&po, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return imageSyncRecordPOToEntity(&po), nil
}

func (r *ImageSyncRecordRepository) Find(ctx context.Context, query image_sync.RecordQuery) ([]*image_sync.Record, int64, error) {
	tx := r.db.WithContext(ctx).Model(&ImageSyncRecordPO{})
	if query.SourceHostID != "" {
		tx = tx.Where("source_host_id = ?", query.SourceHostID)
	}
	if query.TargetHostID != "" {
		tx = tx.Where("target_host_id = ?", query.TargetHostID)
	}
	if query.Status != "" {
		tx = tx.Where("status = ?", query.Status)
	}

	var total int64
	tx.Count(&total)

	var pos []ImageSyncRecordPO
	if err := tx.Order("started_at DESC").Offset(query.Offset()).Limit(query.PageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}
	entities := make([]*image_sync.Record, len(pos))
	for i := range pos {
		entities[i] = imageSyncRecordPOToEntity(&pos[i])
	}
	return entities, total, nil
}
