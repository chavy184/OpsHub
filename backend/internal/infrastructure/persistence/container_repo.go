package persistence

import (
	"context"
	"encoding/json"
	"ops-hub/internal/domain/container"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ============================
// Assemblers: Container
// ============================

func containerEntityToPO(e *container.Container) *ContainerPO {
	paths, _ := json.Marshal(e.ConfigPaths)
	return &ContainerPO{
		ID:            e.ID,
		HostID:        e.HostID,
		ContainerID:   e.ContainerID,
		ContainerName: e.ContainerName,
		Image:         e.Image,
		Status:        e.Status,
		ConfigPaths:   string(paths),
		Description:   e.Description,
		LastSyncedAt:  e.LastSyncedAt,
		CreatedAt:     e.CreatedAt,
		UpdatedAt:     e.UpdatedAt,
	}
}

func containerPOToEntity(po *ContainerPO) *container.Container {
	var paths []string
	_ = json.Unmarshal([]byte(po.ConfigPaths), &paths)
	if paths == nil {
		paths = []string{}
	}
	return &container.Container{
		ID:            po.ID,
		HostID:        po.HostID,
		ContainerID:   po.ContainerID,
		ContainerName: po.ContainerName,
		Image:         po.Image,
		Status:        po.Status,
		ConfigPaths:   paths,
		Description:   po.Description,
		LastSyncedAt:  po.LastSyncedAt,
		CreatedAt:     po.CreatedAt,
		UpdatedAt:     po.UpdatedAt,
	}
}

// ContainerRepository GORM 实现
type ContainerRepository struct {
	db *gorm.DB
}

func NewContainerRepository(db *gorm.DB) *ContainerRepository {
	return &ContainerRepository{db: db}
}

func (r *ContainerRepository) Save(ctx context.Context, e *container.Container) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	now := time.Now()
	e.CreatedAt = now
	e.UpdatedAt = now
	po := containerEntityToPO(e)
	return r.db.WithContext(ctx).Create(po).Error
}

func (r *ContainerRepository) Update(ctx context.Context, e *container.Container) error {
	e.UpdatedAt = time.Now()
	po := containerEntityToPO(e)
	return r.db.WithContext(ctx).Model(&ContainerPO{}).Where("id = ?", po.ID).Updates(map[string]interface{}{
		"container_id":   po.ContainerID,
		"container_name": po.ContainerName,
		"image":          po.Image,
		"status":         po.Status,
		"config_paths":   po.ConfigPaths,
		"description":    po.Description,
		"last_synced_at": po.LastSyncedAt,
		"updated_at":     po.UpdatedAt,
	}).Error
}

func (r *ContainerRepository) FindByID(ctx context.Context, id string) (*container.Container, error) {
	var po ContainerPO
	if err := r.db.WithContext(ctx).First(&po, "id = ?", id).Error; err != nil {
		return nil, container.ErrContainerNotFound
	}
	return containerPOToEntity(&po), nil
}

func (r *ContainerRepository) FindByHostID(ctx context.Context, hostID string) ([]*container.Container, error) {
	var pos []ContainerPO
	if err := r.db.WithContext(ctx).Where("host_id = ?", hostID).Order("container_name ASC").Find(&pos).Error; err != nil {
		return nil, err
	}
	entities := make([]*container.Container, len(pos))
	for i := range pos {
		entities[i] = containerPOToEntity(&pos[i])
	}
	return entities, nil
}

func (r *ContainerRepository) FindByHostAndName(ctx context.Context, hostID, name string) (*container.Container, error) {
	var po ContainerPO
	if err := r.db.WithContext(ctx).Where("host_id = ? AND container_name = ?", hostID, name).First(&po).Error; err != nil {
		return nil, container.ErrContainerNotFound
	}
	return containerPOToEntity(&po), nil
}

func (r *ContainerRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&ContainerPO{}, "id = ?", id).Error
}

func (r *ContainerRepository) DeleteByHostIDNotIn(ctx context.Context, hostID string, names []string) error {
	return r.db.WithContext(ctx).
		Where("host_id = ? AND container_name NOT IN ?", hostID, names).
		Update("status", "removed").Error
}
