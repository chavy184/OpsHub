package persistence

import (
	"context"
	"ops-hub/internal/domain/alert"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GormAlertEventRepository struct {
	db *gorm.DB
}

func NewAlertEventRepository(db *gorm.DB) *GormAlertEventRepository {
	return &GormAlertEventRepository{db: db}
}

func (r *GormAlertEventRepository) Save(ctx context.Context, entity *alert.AlertEvent) error {
	if entity.ID == "" {
		entity.ID = uuid.New().String()
	}
	now := time.Now()
	if entity.CreatedAt.IsZero() {
		entity.CreatedAt = now
	}
	entity.UpdatedAt = now
	if entity.FirstSeenAt.IsZero() {
		entity.FirstSeenAt = now
	}
	if entity.LastSeenAt.IsZero() {
		entity.LastSeenAt = now
	}
	po := alertEventToPO(entity)
	return r.db.WithContext(ctx).Create(&po).Error
}

func (r *GormAlertEventRepository) FindByID(ctx context.Context, id string) (*alert.AlertEvent, error) {
	var po AlertEventPO
	if err := r.db.WithContext(ctx).First(&po, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, alert.ErrAlertNotFound
		}
		return nil, err
	}
	return alertEventToDomain(&po), nil
}

func (r *GormAlertEventRepository) Find(ctx context.Context, query alert.AlertQuery) ([]*alert.AlertEvent, int64, error) {
	db := r.db.WithContext(ctx).Model(&AlertEventPO{})
	if query.ServiceID != "" {
		db = db.Where("service_id = ?", query.ServiceID)
	}
	if query.EnvID != "" {
		db = db.Where("env_id = ?", query.EnvID)
	}
	if query.Severity != "" {
		db = db.Where("severity = ?", query.Severity)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.Keyword != "" {
		db = db.Where("title ILIKE ?", "%"+query.Keyword+"%")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if query.PageSize <= 0 {
		query.PageSize = 20
	}

	var pos []AlertEventPO
	if err := db.Order("last_seen_at DESC").
		Offset(query.Offset()).
		Limit(query.PageSize).
		Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	result := make([]*alert.AlertEvent, len(pos))
	for i, po := range pos {
		result[i] = alertEventToDomain(&po)
	}
	return result, total, nil
}

func (r *GormAlertEventRepository) FindByFingerprint(ctx context.Context, fingerprint string) (*alert.AlertEvent, error) {
	var po AlertEventPO
	if err := r.db.WithContext(ctx).First(&po, "alert_fingerprint = ?", fingerprint).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return alertEventToDomain(&po), nil
}

func (r *GormAlertEventRepository) Update(ctx context.Context, entity *alert.AlertEvent) error {
	entity.UpdatedAt = time.Now()
	po := alertEventToPO(entity)
	return r.db.WithContext(ctx).Save(&po).Error
}

// assemblers
func alertEventToPO(e *alert.AlertEvent) *AlertEventPO {
	return &AlertEventPO{
		ID:               e.ID,
		ServiceID:        e.ServiceID,
		EnvID:            e.EnvID,
		TenantID:         e.TenantID,
		AlertSource:      e.AlertSource,
		AlertFingerprint: e.AlertFingerprint,
		Severity:         string(e.Severity),
		Title:            e.Title,
		Content:          e.Content,
		Status:           string(e.Status),
		FirstSeenAt:      e.FirstSeenAt,
		LastSeenAt:       e.LastSeenAt,
		AssigneeUserID:   e.AssigneeUserID,
		CreatedAt:        e.CreatedAt,
		UpdatedAt:        e.UpdatedAt,
	}
}

func alertEventToDomain(po *AlertEventPO) *alert.AlertEvent {
	return &alert.AlertEvent{
		ID:               po.ID,
		ServiceID:        po.ServiceID,
		EnvID:            po.EnvID,
		TenantID:         po.TenantID,
		AlertSource:      po.AlertSource,
		AlertFingerprint: po.AlertFingerprint,
		Severity:         alert.Severity(po.Severity),
		Title:            po.Title,
		Content:          po.Content,
		Status:           alert.AlertStatus(po.Status),
		FirstSeenAt:      po.FirstSeenAt,
		LastSeenAt:       po.LastSeenAt,
		AssigneeUserID:   po.AssigneeUserID,
		CreatedAt:        po.CreatedAt,
		UpdatedAt:        po.UpdatedAt,
	}
}
