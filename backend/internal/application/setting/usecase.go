package setting

import (
	"context"
	"fmt"
	"ops-hub/internal/domain/setting"
	"time"
)

// ============================
// DTO
// ============================

type SettingDTO struct {
	ID          string `json:"id"`
	SettingKey  string `json:"setting_key"`
	Value       string `json:"value"`
	ValueType   string `json:"value_type"`
	Category    string `json:"category"`
	Description string `json:"description"`
	UpdatedAt   string `json:"updated_at"`
}

type UpdateSettingCmd struct {
	Key       string `json:"setting_key" binding:"required"`
	Value     string `json:"value" binding:"required"`
	UpdatedBy string `json:"updated_by"`
}

type BatchUpdateCmd struct {
	Items []UpdateSettingCmd `json:"items" binding:"required,min=1"`
}

func toDTO(e *setting.SystemSetting) *SettingDTO {
	return &SettingDTO{
		ID:          e.ID,
		SettingKey:  e.SettingKey,
		Value:       e.Value,
		ValueType:   e.ValueType,
		Category:    e.Category,
		Description: e.Description,
		UpdatedAt:   e.UpdatedAt.Format(time.DateTime),
	}
}

// ============================
// UseCase
// ============================

type UseCase struct {
	repo setting.SystemSettingRepository
}

func NewUseCase(repo setting.SystemSettingRepository) *UseCase {
	return &UseCase{repo: repo}
}

// GetAll 获取所有设置
func (uc *UseCase) GetAll(ctx context.Context) ([]*SettingDTO, error) {
	entities, err := uc.repo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	dtos := make([]*SettingDTO, len(entities))
	for i, e := range entities {
		dtos[i] = toDTO(e)
	}
	return dtos, nil
}

// GetByCategory 按分类获取设置
func (uc *UseCase) GetByCategory(ctx context.Context, category string) ([]*SettingDTO, error) {
	entities, err := uc.repo.FindByCategory(ctx, category)
	if err != nil {
		return nil, err
	}
	dtos := make([]*SettingDTO, len(entities))
	for i, e := range entities {
		dtos[i] = toDTO(e)
	}
	return dtos, nil
}

// Update 更新单个设置
func (uc *UseCase) Update(ctx context.Context, cmd UpdateSettingCmd) (*SettingDTO, error) {
	existing, err := uc.repo.FindByKey(ctx, cmd.Key)
	if err != nil {
		// 设置项不存在时创建
		existing = &setting.SystemSetting{
			SettingKey: cmd.Key,
			ValueType:  "string",
			Category:   "general",
		}
	}

	existing.Value = cmd.Value
	existing.UpdatedBy = cmd.UpdatedBy
	existing.UpdatedAt = time.Now()

	if err := uc.repo.Upsert(ctx, existing); err != nil {
		return nil, fmt.Errorf("更新设置失败: %w", err)
	}
	return toDTO(existing), nil
}

// BatchUpdate 批量更新设置
func (uc *UseCase) BatchUpdate(ctx context.Context, cmd BatchUpdateCmd) error {
	for _, s := range cmd.Items {
		if _, err := uc.Update(ctx, s); err != nil {
			return err
		}
	}
	return nil
}

// GetByKey 获取单个设置值
func (uc *UseCase) GetByKey(ctx context.Context, key string) (string, error) {
	e, err := uc.repo.FindByKey(ctx, key)
	if err != nil {
		return "", err
	}
	return e.Value, nil
}

// SeedDefaults 种子数据: 如果 key 不存在则创建
func (uc *UseCase) SeedDefaults(ctx context.Context, defaults []setting.SystemSetting) error {
	for _, d := range defaults {
		_, err := uc.repo.FindByKey(ctx, d.SettingKey)
		if err == nil {
			continue // 已存在，跳过
		}
		if err := uc.repo.Upsert(ctx, &d); err != nil {
			return fmt.Errorf("seed %s failed: %w", d.SettingKey, err)
		}
	}
	return nil
}
