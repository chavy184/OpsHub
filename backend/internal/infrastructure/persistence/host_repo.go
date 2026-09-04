package persistence

import (
	"context"
	"ops-hub/internal/domain/host"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ============================
// Assemblers: Host
// ============================

func hostEntityToPO(e *host.Host) *HostPO {
	return &HostPO{
		ID:            e.ID,
		Name:          e.Name,
		HostAddress:   e.HostAddress,
		SSHPort:       e.SSHPort,
		Username:      e.Username,
		CredentialID:  e.CredentialID,
		IsProd:        e.IsProd,
		Labels:        e.Labels,
		OsInfo:        e.OsInfo,
		AgentStatus:   string(e.AgentStatus),
		LastHeartbeat: e.LastHeartbeat,
		Description:   e.Description,
		CreatedAt:     e.CreatedAt,
		UpdatedAt:     e.UpdatedAt,
	}
}

func hostPOToEntity(po *HostPO) *host.Host {
	return &host.Host{
		ID:            po.ID,
		Name:          po.Name,
		HostAddress:   po.HostAddress,
		SSHPort:       po.SSHPort,
		Username:      po.Username,
		CredentialID:  po.CredentialID,
		IsProd:        po.IsProd,
		Labels:        po.Labels,
		OsInfo:        po.OsInfo,
		AgentStatus:   host.AgentStatus(po.AgentStatus),
		LastHeartbeat: po.LastHeartbeat,
		Description:   po.Description,
		CreatedAt:     po.CreatedAt,
		UpdatedAt:     po.UpdatedAt,
	}
}

// HostRepository GORM 实现
type HostRepository struct {
	db *gorm.DB
}

func NewHostRepository(db *gorm.DB) *HostRepository {
	return &HostRepository{db: db}
}

func (r *HostRepository) Save(ctx context.Context, e *host.Host) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	e.CreatedAt = time.Now()
	e.UpdatedAt = time.Now()
	if e.SSHPort == 0 {
		e.SSHPort = 22
	}
	if e.AgentStatus == "" {
		e.AgentStatus = host.AgentStatusUnknown
	}
	po := hostEntityToPO(e)
	return r.db.WithContext(ctx).Create(po).Error
}

func (r *HostRepository) Update(ctx context.Context, e *host.Host) error {
	e.UpdatedAt = time.Now()
	po := hostEntityToPO(e)
	return r.db.WithContext(ctx).Save(po).Error
}

func (r *HostRepository) FindByID(ctx context.Context, id string) (*host.Host, error) {
	var po HostPO
	if err := r.db.WithContext(ctx).First(&po, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, host.ErrHostNotFound
		}
		return nil, err
	}
	return hostPOToEntity(&po), nil
}

func (r *HostRepository) FindByIsProd(ctx context.Context, isProd bool) ([]*host.Host, error) {
	var pos []HostPO
	if err := r.db.WithContext(ctx).Where("is_prod = ?", isProd).Order("created_at DESC").Find(&pos).Error; err != nil {
		return nil, err
	}

	entities := make([]*host.Host, len(pos))
	for i, po := range pos {
		poCopy := po
		entities[i] = hostPOToEntity(&poCopy)
	}
	return entities, nil
}

func (r *HostRepository) Find(ctx context.Context, q host.HostQuery) ([]*host.Host, int64, error) {
	db := r.db.WithContext(ctx).Model(&HostPO{})
	if q.Keyword != "" {
		db = db.Where("name ILIKE ? OR host_address ILIKE ?", "%"+q.Keyword+"%", "%"+q.Keyword+"%")
	}

	var total int64
	db.Count(&total)

	pageSize := q.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}

	var pos []HostPO
	if err := db.Order("created_at DESC").Offset(q.Offset()).Limit(pageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	entities := make([]*host.Host, len(pos))
	for i, po := range pos {
		poCopy := po
		entities[i] = hostPOToEntity(&poCopy)
	}
	return entities, total, nil
}

func (r *HostRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&HostPO{}, "id = ?", id).Error
}
