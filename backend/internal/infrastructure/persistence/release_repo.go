package persistence

import (
	"context"
	"ops-hub/internal/domain/release"

	"gorm.io/gorm"
)

// ============================
// Assemblers: Release
// ============================

func releaseEntityToPO(e *release.ReleaseRecord) *ReleaseRecordPO {
	return &ReleaseRecordPO{
		ID:              e.ID,
		ServiceID:       e.ServiceID,
		EnvID:           e.EnvID,
		TenantID:        e.TenantID,
		TargetVersionID: e.TargetVersionID,
		PrevVersionID:   e.PrevVersionID,
		ReleaseType:     string(e.ReleaseType),
		Strategy:        e.Strategy,
		Status:          string(e.Status),
		ErrorMessage:    e.ErrorMessage,
		OperatorID:      e.OperatorID,
		IdempotencyKey:  e.IdempotencyKey,
		JenkinsParams:   e.JenkinsParams,
		JenkinsBuildNo:  e.JenkinsBuildNo,
		StartedAt:       e.StartedAt,
		EndedAt:         e.EndedAt,
		CreatedAt:       e.CreatedAt,
		UpdatedAt:       e.UpdatedAt,
	}
}

func releasePOToEntity(po *ReleaseRecordPO) *release.ReleaseRecord {
	return &release.ReleaseRecord{
		ID:              po.ID,
		ServiceID:       po.ServiceID,
		EnvID:           po.EnvID,
		TenantID:        po.TenantID,
		TargetVersionID: po.TargetVersionID,
		PrevVersionID:   po.PrevVersionID,
		ReleaseType:     release.ReleaseType(po.ReleaseType),
		Strategy:        po.Strategy,
		Status:          release.ReleaseStatus(po.Status),
		ErrorMessage:    po.ErrorMessage,
		OperatorID:      po.OperatorID,
		IdempotencyKey:  po.IdempotencyKey,
		JenkinsParams:   po.JenkinsParams,
		JenkinsBuildNo:  po.JenkinsBuildNo,
		StartedAt:       po.StartedAt,
		EndedAt:         po.EndedAt,
		CreatedAt:       po.CreatedAt,
		UpdatedAt:       po.UpdatedAt,
	}
}

// ============================
// GormReleaseRecordRepository
// ============================

type GormReleaseRecordRepository struct {
	db *gorm.DB
}

func NewGormReleaseRecordRepository(db *gorm.DB) release.ReleaseRecordRepository {
	return &GormReleaseRecordRepository{db: db}
}

func (r *GormReleaseRecordRepository) Save(ctx context.Context, e *release.ReleaseRecord) error {
	po := releaseEntityToPO(e)
	return r.db.WithContext(ctx).Create(po).Error
}

func (r *GormReleaseRecordRepository) FindByID(ctx context.Context, id string) (*release.ReleaseRecord, error) {
	var po ReleaseRecordPO
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&po).Error; err != nil {
		return nil, err
	}
	return releasePOToEntity(&po), nil
}

func (r *GormReleaseRecordRepository) Find(ctx context.Context, query release.ReleaseQuery) ([]*release.ReleaseRecord, int64, error) {
	q := r.db.WithContext(ctx).Model(&ReleaseRecordPO{})
	if query.ServiceID != "" {
		q = q.Where("service_id = ?", query.ServiceID)
	}
	if query.EnvID != "" {
		q = q.Where("env_id = ?", query.EnvID)
	}
	if query.TenantID != "" {
		q = q.Where("tenant_id = ?", query.TenantID)
	}
	if query.Status != "" {
		q = q.Where("status = ?", query.Status)
	}
	if query.ReleaseType != "" {
		q = q.Where("release_type = ?", query.ReleaseType)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var pos []*ReleaseRecordPO
	if err := q.Offset(query.Offset()).Limit(query.PageSize).Order("created_at DESC").Find(&pos).Error; err != nil {
		return nil, 0, err
	}

	entities := make([]*release.ReleaseRecord, len(pos))
	for i, po := range pos {
		entities[i] = releasePOToEntity(po)
	}
	return entities, total, nil
}

func (r *GormReleaseRecordRepository) FindLastSuccess(ctx context.Context, serviceID, envID string) (*release.ReleaseRecord, error) {
	var po ReleaseRecordPO
	err := r.db.WithContext(ctx).
		Where("service_id = ? AND env_id = ? AND status = ? AND release_type = ?",
			serviceID, envID, string(release.ReleaseStatusSuccess), string(release.ReleaseTypeDeploy)).
		Order("ended_at DESC").
		First(&po).Error
	if err != nil {
		return nil, err
	}
	return releasePOToEntity(&po), nil
}

func (r *GormReleaseRecordRepository) FindActiveByServiceEnv(ctx context.Context, serviceID, envID string) (*release.ReleaseRecord, error) {
	var po ReleaseRecordPO
	err := r.db.WithContext(ctx).
		Where("service_id = ? AND env_id = ? AND status IN ?",
			serviceID, envID, []string{string(release.ReleaseStatusPending), string(release.ReleaseStatusRunning)}).
		First(&po).Error
	if err != nil {
		return nil, err
	}
	return releasePOToEntity(&po), nil
}

func (r *GormReleaseRecordRepository) Update(ctx context.Context, e *release.ReleaseRecord) error {
	po := releaseEntityToPO(e)
	return r.db.WithContext(ctx).Save(po).Error
}
func (r *GormReleaseRecordRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&ReleaseRecordPO{}, "id = ?", id).Error
}
