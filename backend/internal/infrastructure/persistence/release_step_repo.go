package persistence

import (
	"context"
	"ops-hub/internal/domain/release"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ============================
// Assemblers: ReleaseStepLog
// ============================

func stepLogEntityToPO(e *release.ReleaseStepLog) *ReleaseStepLogPO {
	return &ReleaseStepLogPO{
		ID:          e.ID,
		ReleaseID:   e.ReleaseID,
		StepOrder:   e.StepOrder,
		StepName:    e.StepName,
		StepStatus:  string(e.StepStatus),
		StartedAt:   e.StartedAt,
		EndedAt:     e.EndedAt,
		DurationMs:  e.DurationMs,
		Output:      e.Output,
		ErrorOutput: e.ErrorOutput,
		CreatedAt:   e.CreatedAt,
	}
}

func stepLogPOToEntity(po *ReleaseStepLogPO) *release.ReleaseStepLog {
	return &release.ReleaseStepLog{
		ID:          po.ID,
		ReleaseID:   po.ReleaseID,
		StepOrder:   po.StepOrder,
		StepName:    po.StepName,
		StepStatus:  release.StepStatus(po.StepStatus),
		StartedAt:   po.StartedAt,
		EndedAt:     po.EndedAt,
		DurationMs:  po.DurationMs,
		Output:      po.Output,
		ErrorOutput: po.ErrorOutput,
		CreatedAt:   po.CreatedAt,
	}
}

// ReleaseStepLogRepository GORM 实现
type ReleaseStepLogRepository struct {
	db *gorm.DB
}

func NewReleaseStepLogRepository(db *gorm.DB) *ReleaseStepLogRepository {
	return &ReleaseStepLogRepository{db: db}
}

func (r *ReleaseStepLogRepository) Save(ctx context.Context, e *release.ReleaseStepLog) error {
	if e.ID == "" {
		e.ID = uuid.New().String()
	}
	e.CreatedAt = time.Now()
	po := stepLogEntityToPO(e)
	return r.db.WithContext(ctx).Create(po).Error
}

func (r *ReleaseStepLogRepository) Update(ctx context.Context, e *release.ReleaseStepLog) error {
	po := stepLogEntityToPO(e)
	return r.db.WithContext(ctx).Save(po).Error
}

func (r *ReleaseStepLogRepository) FindByReleaseID(ctx context.Context, releaseID string) ([]*release.ReleaseStepLog, error) {
	var pos []ReleaseStepLogPO
	if err := r.db.WithContext(ctx).
		Where("release_id = ?", releaseID).
		Order("step_order ASC").
		Find(&pos).Error; err != nil {
		return nil, err
	}

	entities := make([]*release.ReleaseStepLog, len(pos))
	for i, po := range pos {
		poCopy := po
		entities[i] = stepLogPOToEntity(&poCopy)
	}
	return entities, nil
}
