package persistence

import (
	"context"
	"ops-hub/internal/domain/setting"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ============================
// Assemblers: SystemSetting
// ============================

func settingEntityToPO(e *setting.SystemSetting) *SystemSettingPO {
	return &SystemSettingPO{
		ID:          e.ID,
		SettingKey:  e.SettingKey,
		Value:       e.Value,
		ValueType:   e.ValueType,
		Category:    e.Category,
		Description: e.Description,
		UpdatedBy:   e.UpdatedBy,
		UpdatedAt:   e.UpdatedAt,
	}
}

func settingPOToEntity(po *SystemSettingPO) *setting.SystemSetting {
	return &setting.SystemSetting{
		ID:          po.ID,
		SettingKey:  po.SettingKey,
		Value:       po.Value,
		ValueType:   po.ValueType,
		Category:    po.Category,
		Description: po.Description,
		UpdatedBy:   po.UpdatedBy,
		UpdatedAt:   po.UpdatedAt,
	}
}

// SettingRepository GORM 实现
type SettingRepository struct {
	db *gorm.DB
}

func NewSettingRepository(db *gorm.DB) *SettingRepository {
	return &SettingRepository{db: db}
}

func (r *SettingRepository) FindAll(ctx context.Context) ([]*setting.SystemSetting, error) {
	var pos []SystemSettingPO
	if err := r.db.WithContext(ctx).Order("category, setting_key").Find(&pos).Error; err != nil {
		return nil, err
	}
	entities := make([]*setting.SystemSetting, len(pos))
	for i, po := range pos {
		poCopy := po
		entities[i] = settingPOToEntity(&poCopy)
	}
	return entities, nil
}

func (r *SettingRepository) FindByKey(ctx context.Context, key string) (*setting.SystemSetting, error) {
	var po SystemSettingPO
	if err := r.db.WithContext(ctx).First(&po, "setting_key = ?", key).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, setting.ErrSettingNotFound
		}
		return nil, err
	}
	return settingPOToEntity(&po), nil
}

func (r *SettingRepository) FindByCategory(ctx context.Context, category string) ([]*setting.SystemSetting, error) {
	var pos []SystemSettingPO
	if err := r.db.WithContext(ctx).Where("category = ?", category).Order("setting_key").Find(&pos).Error; err != nil {
		return nil, err
	}
	entities := make([]*setting.SystemSetting, len(pos))
	for i, po := range pos {
		poCopy := po
		entities[i] = settingPOToEntity(&poCopy)
	}
	return entities, nil
}

func (r *SettingRepository) Upsert(ctx context.Context, e *setting.SystemSetting) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	e.UpdatedAt = time.Now()
	po := settingEntityToPO(e)
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "setting_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_by", "updated_at"}),
		}).Create(po).Error
}

func (r *SettingRepository) BatchUpsert(ctx context.Context, entities []*setting.SystemSetting) error {
	for _, e := range entities {
		if err := r.Upsert(ctx, e); err != nil {
			return err
		}
	}
	return nil
}
