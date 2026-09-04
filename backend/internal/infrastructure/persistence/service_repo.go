package persistence

import (
	"context"
	"ops-hub/internal/domain/service"

	"gorm.io/gorm"
)

// ============================
// Assemblers: Service
// ============================

func serviceEntityToPO(e *service.Service) *ServicePO {
	return &ServicePO{
		ID:          e.ID,
		ServiceKey:  e.ServiceKey,
		ServiceName: e.ServiceName,
		OwnerUserID: e.OwnerUserID,
		RepoURL:     e.RepoURL,
		CreatedAt:   e.CreatedAt,
		UpdatedAt:   e.UpdatedAt,
	}
}

func servicePOToEntity(po *ServicePO) *service.Service {
	return &service.Service{
		ID:          po.ID,
		ServiceKey:  po.ServiceKey,
		ServiceName: po.ServiceName,
		OwnerUserID: po.OwnerUserID,
		RepoURL:     po.RepoURL,
		CreatedAt:   po.CreatedAt,
		UpdatedAt:   po.UpdatedAt,
	}
}

func envEntityToPO(e *service.ServiceEnv) *ServiceEnvPO {
	return &ServiceEnvPO{
		ID:                      e.ID,
		ServiceID:               e.ServiceID,
		EnvCode:                 e.EnvCode,
		ClusterName:             e.ClusterName,
		Namespace:               e.Namespace,
		AccessEndpoint:          e.AccessEndpoint,
		HealthcheckURL:          e.HealthcheckURL,
		LogSourceType:           e.LogSourceType,
		LogSourceConfig:         e.LogSourceConfig,
		HostID:                  e.HostID,
		DeployPath:              e.DeployPath,
		EnvVars:                 e.EnvVars,
		HealthCheckInterval:     e.HealthCheckInterval,
		HealthCheckTimeout:      e.HealthCheckTimeout,
		HealthCheckSuccessCodes: e.HealthCheckSuccessCodes,
		HealthCheckEnabled:      e.HealthCheckEnabled,
		HealthStatus:            e.HealthStatus,
		HealthLastCheckedAt:     e.HealthLastCheckedAt,
		HealthLastMessage:       e.HealthLastMessage,
		JenkinsJobs:             e.JenkinsJobs,
		CreatedAt:               e.CreatedAt,
		UpdatedAt:               e.UpdatedAt,
	}
}

func envPOToEntity(po *ServiceEnvPO) *service.ServiceEnv {
	return &service.ServiceEnv{
		ID:                      po.ID,
		ServiceID:               po.ServiceID,
		EnvCode:                 po.EnvCode,
		ClusterName:             po.ClusterName,
		Namespace:               po.Namespace,
		AccessEndpoint:          po.AccessEndpoint,
		HealthcheckURL:          po.HealthcheckURL,
		LogSourceType:           po.LogSourceType,
		LogSourceConfig:         po.LogSourceConfig,
		HostID:                  po.HostID,
		DeployPath:              po.DeployPath,
		EnvVars:                 po.EnvVars,
		HealthCheckInterval:     po.HealthCheckInterval,
		HealthCheckTimeout:      po.HealthCheckTimeout,
		HealthCheckSuccessCodes: po.HealthCheckSuccessCodes,
		HealthCheckEnabled:      po.HealthCheckEnabled,
		HealthStatus:            po.HealthStatus,
		HealthLastCheckedAt:     po.HealthLastCheckedAt,
		HealthLastMessage:       po.HealthLastMessage,
		JenkinsJobs:             po.JenkinsJobs,
		CreatedAt:               po.CreatedAt,
		UpdatedAt:               po.UpdatedAt,
	}
}

// ============================
// GormServiceRepository
// ============================

type GormServiceRepository struct {
	db *gorm.DB
}

func NewGormServiceRepository(db *gorm.DB) service.ServiceRepository {
	return &GormServiceRepository{db: db}
}

func (r *GormServiceRepository) Save(ctx context.Context, e *service.Service) error {
	po := serviceEntityToPO(e)
	return r.db.WithContext(ctx).Save(po).Error
}

func (r *GormServiceRepository) FindByID(ctx context.Context, id string) (*service.Service, error) {
	var po ServicePO
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&po).Error; err != nil {
		return nil, err
	}
	return servicePOToEntity(&po), nil
}

func (r *GormServiceRepository) FindByKey(ctx context.Context, key string) (*service.Service, error) {
	var po ServicePO
	if err := r.db.WithContext(ctx).Where("service_key = ?", key).First(&po).Error; err != nil {
		return nil, err
	}
	return servicePOToEntity(&po), nil
}

func (r *GormServiceRepository) Find(ctx context.Context, query service.ServiceQuery) ([]*service.Service, int64, error) {
	q := r.db.WithContext(ctx).Model(&ServicePO{})
	if query.Keyword != "" {
		like := "%" + query.Keyword + "%"
		q = q.Where("service_name LIKE ? OR service_key LIKE ?", like, like)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var pos []*ServicePO
	if err := q.Offset(query.Offset()).Limit(query.PageSize).Order("created_at DESC").Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	entities := make([]*service.Service, len(pos))
	for i, po := range pos {
		entities[i] = servicePOToEntity(po)
	}
	return entities, total, nil
}

func (r *GormServiceRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&ServicePO{}).Error
}

// ============================
// GormServiceEnvRepository
// ============================

type GormServiceEnvRepository struct {
	db *gorm.DB
}

func NewGormServiceEnvRepository(db *gorm.DB) service.ServiceEnvRepository {
	return &GormServiceEnvRepository{db: db}
}

func (r *GormServiceEnvRepository) Save(ctx context.Context, e *service.ServiceEnv) error {
	po := envEntityToPO(e)
	return r.db.WithContext(ctx).Save(po).Error
}

func (r *GormServiceEnvRepository) FindByID(ctx context.Context, id string) (*service.ServiceEnv, error) {
	var po ServiceEnvPO
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&po).Error; err != nil {
		return nil, err
	}
	return envPOToEntity(&po), nil
}

func (r *GormServiceEnvRepository) FindByServiceID(ctx context.Context, serviceID string) ([]*service.ServiceEnv, error) {
	var pos []*ServiceEnvPO
	if err := r.db.WithContext(ctx).Where("service_id = ?", serviceID).Find(&pos).Error; err != nil {
		return nil, err
	}
	entities := make([]*service.ServiceEnv, len(pos))
	for i, po := range pos {
		entities[i] = envPOToEntity(po)
	}
	return entities, nil
}

func (r *GormServiceEnvRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&ServiceEnvPO{}).Error
}

func (r *GormServiceEnvRepository) FindAllHealthCheckEnabled(ctx context.Context) ([]*service.ServiceEnv, error) {
	var pos []ServiceEnvPO
	if err := r.db.WithContext(ctx).Where("health_check_enabled = ? AND deleted_at IS NULL", true).Find(&pos).Error; err != nil {
		return nil, err
	}
	entities := make([]*service.ServiceEnv, len(pos))
	for i, po := range pos {
		poCopy := po
		entities[i] = envPOToEntity(&poCopy)
	}
	return entities, nil
}

func (r *GormServiceEnvRepository) UpdateHealthStatus(ctx context.Context, env *service.ServiceEnv) error {
	return r.db.WithContext(ctx).Model(&ServiceEnvPO{}).Where("id = ?", env.ID).Updates(map[string]interface{}{
		"health_status":          env.HealthStatus,
		"health_last_checked_at": env.HealthLastCheckedAt,
		"health_last_message":    env.HealthLastMessage,
	}).Error
}
