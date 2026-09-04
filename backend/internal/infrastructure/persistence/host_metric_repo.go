package persistence

import (
	"context"
	"ops-hub/internal/domain/host"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type HostMetricRepo struct {
	db *gorm.DB
}

func NewHostMetricRepo(db *gorm.DB) *HostMetricRepo {
	return &HostMetricRepo{db: db}
}

func (r *HostMetricRepo) Save(ctx context.Context, s *host.HostMetricSnapshot) error {
	if s.ID == "" {
		s.ID = uuid.New().String()
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = time.Now()
	}
	po := metricToPO(s)
	return r.db.WithContext(ctx).Create(&po).Error
}

func (r *HostMetricRepo) FindLatest(ctx context.Context, hostID string) (*host.HostMetricSnapshot, error) {
	var po HostMetricSnapshotPO
	err := r.db.WithContext(ctx).
		Where("host_id = ?", hostID).
		Order("collected_at DESC").
		First(&po).Error
	if err != nil {
		return nil, err
	}
	return metricToEntity(&po), nil
}

func (r *HostMetricRepo) FindHistory(ctx context.Context, query host.HostMetricQuery) ([]*host.HostMetricSnapshot, error) {
	db := r.db.WithContext(ctx).Where("host_id = ?", query.HostID)
	if query.StartTime != nil {
		db = db.Where("collected_at >= ?", *query.StartTime)
	}
	if query.EndTime != nil {
		db = db.Where("collected_at <= ?", *query.EndTime)
	}
	limit := query.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	var pos []HostMetricSnapshotPO
	err := db.Order("collected_at DESC").Limit(limit).Find(&pos).Error
	if err != nil {
		return nil, err
	}
	result := make([]*host.HostMetricSnapshot, len(pos))
	for i := range pos {
		result[i] = metricToEntity(&pos[i])
	}
	return result, nil
}

func (r *HostMetricRepo) FindAllLatest(ctx context.Context) ([]*host.HostMetricSnapshot, error) {
	// Use a subquery to find the latest metric for each host
	var pos []HostMetricSnapshotPO
	subQuery := r.db.WithContext(ctx).
		Model(&HostMetricSnapshotPO{}).
		Select("DISTINCT ON (host_id) *").
		Order("host_id, collected_at DESC")
	err := subQuery.Find(&pos).Error
	if err != nil {
		// Fallback for databases that don't support DISTINCT ON
		err = r.db.WithContext(ctx).
			Raw(`SELECT m.* FROM host_metric_snapshots m 
				 INNER JOIN (SELECT host_id, MAX(collected_at) as max_time FROM host_metric_snapshots GROUP BY host_id) latest 
				 ON m.host_id = latest.host_id AND m.collected_at = latest.max_time`).
			Scan(&pos).Error
		if err != nil {
			return nil, err
		}
	}
	result := make([]*host.HostMetricSnapshot, len(pos))
	for i := range pos {
		result[i] = metricToEntity(&pos[i])
	}
	return result, nil
}

func (r *HostMetricRepo) DeleteOlderThan(ctx context.Context, before time.Time) error {
	return r.db.WithContext(ctx).
		Where("collected_at < ?", before).
		Delete(&HostMetricSnapshotPO{}).Error
}

// Assemblers

func metricToPO(e *host.HostMetricSnapshot) *HostMetricSnapshotPO {
	return &HostMetricSnapshotPO{
		ID:          e.ID,
		HostID:      e.HostID,
		CPUUsage:    e.CPUUsage,
		CPUCores:    e.CPUCores,
		MemTotalMB:  e.MemTotalMB,
		MemUsedMB:   e.MemUsedMB,
		MemUsage:    e.MemUsage,
		DiskTotalGB: e.DiskTotalGB,
		DiskUsedGB:  e.DiskUsedGB,
		DiskUsage:   e.DiskUsage,
		DisksJSON:   e.DisksJSON,
		LoadAvg1:    e.LoadAvg1,
		LoadAvg5:    e.LoadAvg5,
		LoadAvg15:   e.LoadAvg15,
		NetInBytes:  e.NetInBytes,
		NetOutBytes: e.NetOutBytes,
		GPUUsage:    e.GPUUsage,
		GPUMemUsage: e.GPUMemUsage,
		GPUTemp:     e.GPUTemp,
		GPUName:     e.GPUName,
		GPUsJSON:    e.GPUsJSON,
		CollectedAt: e.CollectedAt,
		CreatedAt:   e.CreatedAt,
	}
}

func metricToEntity(po *HostMetricSnapshotPO) *host.HostMetricSnapshot {
	return &host.HostMetricSnapshot{
		ID:          po.ID,
		HostID:      po.HostID,
		CPUUsage:    po.CPUUsage,
		CPUCores:    po.CPUCores,
		MemTotalMB:  po.MemTotalMB,
		MemUsedMB:   po.MemUsedMB,
		MemUsage:    po.MemUsage,
		DiskTotalGB: po.DiskTotalGB,
		DiskUsedGB:  po.DiskUsedGB,
		DiskUsage:   po.DiskUsage,
		DisksJSON:   po.DisksJSON,
		LoadAvg1:    po.LoadAvg1,
		LoadAvg5:    po.LoadAvg5,
		LoadAvg15:   po.LoadAvg15,
		NetInBytes:  po.NetInBytes,
		NetOutBytes: po.NetOutBytes,
		GPUUsage:    po.GPUUsage,
		GPUMemUsage: po.GPUMemUsage,
		GPUTemp:     po.GPUTemp,
		GPUName:     po.GPUName,
		GPUsJSON:    po.GPUsJSON,
		CollectedAt: po.CollectedAt,
		CreatedAt:   po.CreatedAt,
	}
}
