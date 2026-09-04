package persistence

import (
	"context"
	"ops-hub/internal/domain/notification"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ============================
// Assemblers: NotificationChannel
// ============================

func channelEntityToPO(e *notification.NotificationChannel) *NotificationChannelPO {
	return &NotificationChannelPO{
		ID:          e.ID,
		Name:        e.Name,
		ChannelType: string(e.ChannelType),
		Config:      e.Config,
		Enabled:     e.Enabled,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

func channelPOToEntity(po *NotificationChannelPO) *notification.NotificationChannel {
	return &notification.NotificationChannel{
		ID:          po.ID,
		Name:        po.Name,
		ChannelType: notification.ChannelType(po.ChannelType),
		Config:      po.Config,
		Enabled:     po.Enabled,
		CreatedAt:   po.CreatedAt,
		UpdatedAt:   po.UpdatedAt,
	}
}

// NotificationChannelRepository GORM 实现
type NotificationChannelRepository struct {
	db *gorm.DB
}

func NewNotificationChannelRepository(db *gorm.DB) *NotificationChannelRepository {
	return &NotificationChannelRepository{db: db}
}

func (r *NotificationChannelRepository) Save(ctx context.Context, e *notification.NotificationChannel) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	e.CreatedAt = time.Now()
	e.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Create(channelEntityToPO(e)).Error
}

func (r *NotificationChannelRepository) Update(ctx context.Context, e *notification.NotificationChannel) error {
	e.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(channelEntityToPO(e)).Error
}

func (r *NotificationChannelRepository) FindByID(ctx context.Context, id string) (*notification.NotificationChannel, error) {
	var po NotificationChannelPO
	if err := r.db.WithContext(ctx).First(&po, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, notification.ErrChannelNotFound
		}
		return nil, err
	}
	return channelPOToEntity(&po), nil
}

func (r *NotificationChannelRepository) FindAll(ctx context.Context) ([]*notification.NotificationChannel, error) {
	var pos []NotificationChannelPO
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&pos).Error; err != nil {
		return nil, err
	}
	result := make([]*notification.NotificationChannel, len(pos))
	for i := range pos {
		result[i] = channelPOToEntity(&pos[i])
	}
	return result, nil
}

func (r *NotificationChannelRepository) FindEnabled(ctx context.Context) ([]*notification.NotificationChannel, error) {
	var pos []NotificationChannelPO
	if err := r.db.WithContext(ctx).Where("enabled = ?", true).Find(&pos).Error; err != nil {
		return nil, err
	}
	result := make([]*notification.NotificationChannel, len(pos))
	for i := range pos {
		result[i] = channelPOToEntity(&pos[i])
	}
	return result, nil
}

func (r *NotificationChannelRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&NotificationChannelPO{}, "id = ?", id).Error
}

// ============================
// Assemblers: NotificationRule
// ============================

func ruleEntityToPO(e *notification.NotificationRule) *NotificationRulePO {
	return &NotificationRulePO{
		ID:        e.ID,
		EventType: string(e.EventType),
		ChannelID: e.ChannelID,
		Enabled:   e.Enabled,
		Filter:    e.Filter,
		CreatedAt: e.CreatedAt,
		UpdatedAt: e.UpdatedAt,
	}
}

func rulePOToEntity(po *NotificationRulePO) *notification.NotificationRule {
	return &notification.NotificationRule{
		ID:        po.ID,
		EventType: notification.EventType(po.EventType),
		ChannelID: po.ChannelID,
		Enabled:   po.Enabled,
		Filter:    po.Filter,
		CreatedAt: po.CreatedAt,
		UpdatedAt: po.UpdatedAt,
	}
}

// NotificationRuleRepository GORM 实现
type NotificationRuleRepository struct {
	db *gorm.DB
}

func NewNotificationRuleRepository(db *gorm.DB) *NotificationRuleRepository {
	return &NotificationRuleRepository{db: db}
}

func (r *NotificationRuleRepository) Save(ctx context.Context, e *notification.NotificationRule) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	e.CreatedAt = time.Now()
	e.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Create(ruleEntityToPO(e)).Error
}

func (r *NotificationRuleRepository) Update(ctx context.Context, e *notification.NotificationRule) error {
	e.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Save(ruleEntityToPO(e)).Error
}

func (r *NotificationRuleRepository) FindByID(ctx context.Context, id string) (*notification.NotificationRule, error) {
	var po NotificationRulePO
	if err := r.db.WithContext(ctx).First(&po, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, notification.ErrRuleNotFound
		}
		return nil, err
	}
	return rulePOToEntity(&po), nil
}

func (r *NotificationRuleRepository) FindAll(ctx context.Context) ([]*notification.NotificationRule, error) {
	var pos []NotificationRulePO
	if err := r.db.WithContext(ctx).Order("created_at DESC").Find(&pos).Error; err != nil {
		return nil, err
	}
	result := make([]*notification.NotificationRule, len(pos))
	for i := range pos {
		result[i] = rulePOToEntity(&pos[i])
	}
	return result, nil
}

func (r *NotificationRuleRepository) FindByEventType(ctx context.Context, eventType notification.EventType) ([]*notification.NotificationRule, error) {
	var pos []NotificationRulePO
	if err := r.db.WithContext(ctx).Where("event_type = ? AND enabled = ?", string(eventType), true).Find(&pos).Error; err != nil {
		return nil, err
	}
	result := make([]*notification.NotificationRule, len(pos))
	for i := range pos {
		result[i] = rulePOToEntity(&pos[i])
	}
	return result, nil
}

func (r *NotificationRuleRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&NotificationRulePO{}, "id = ?", id).Error
}

// ============================
// Assemblers: NotificationLog
// ============================

func logEntityToPO(e *notification.NotificationLog) *NotificationLogPO {
	return &NotificationLogPO{
		ID:           e.ID,
		ChannelID:    e.ChannelID,
		EventType:    string(e.EventType),
		Title:        e.Title,
		Content:      e.Content,
		Status:       e.Status,
		ErrorMessage: e.ErrorMessage,
		CreatedAt:    e.CreatedAt,
	}
}

func logPOToEntity(po *NotificationLogPO) *notification.NotificationLog {
	return &notification.NotificationLog{
		ID:           po.ID,
		ChannelID:    po.ChannelID,
		EventType:    notification.EventType(po.EventType),
		Title:        po.Title,
		Content:      po.Content,
		Status:       po.Status,
		ErrorMessage: po.ErrorMessage,
		CreatedAt:    po.CreatedAt,
	}
}

// NotificationLogRepository GORM 实现
type NotificationLogRepository struct {
	db *gorm.DB
}

func NewNotificationLogRepository(db *gorm.DB) *NotificationLogRepository {
	return &NotificationLogRepository{db: db}
}

func (r *NotificationLogRepository) Save(ctx context.Context, e *notification.NotificationLog) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	e.CreatedAt = time.Now()
	return r.db.WithContext(ctx).Create(logEntityToPO(e)).Error
}

func (r *NotificationLogRepository) Find(ctx context.Context, q notification.LogQuery) ([]*notification.NotificationLog, int64, error) {
	db := r.db.WithContext(ctx).Model(&NotificationLogPO{})
	if q.ChannelID != "" {
		db = db.Where("channel_id = ?", q.ChannelID)
	}
	if q.EventType != "" {
		db = db.Where("event_type = ?", q.EventType)
	}
	if q.Status != "" {
		db = db.Where("status = ?", q.Status)
	}

	var total int64
	db.Count(&total)

	pageSize := q.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	page := q.Page
	if page < 1 {
		page = 1
	}

	var pos []NotificationLogPO
	if err := db.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}
	result := make([]*notification.NotificationLog, len(pos))
	for i := range pos {
		result[i] = logPOToEntity(&pos[i])
	}
	return result, total, nil
}
