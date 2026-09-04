package image_sync

import "context"

type RecordRepository interface {
	Save(ctx context.Context, entity *Record) error
	Update(ctx context.Context, entity *Record) error
	FindByID(ctx context.Context, id string) (*Record, error)
	Find(ctx context.Context, query RecordQuery) ([]*Record, int64, error)
}

type RecordQuery struct {
	SourceHostID string
	TargetHostID string
	Status       string
	Page         int
	PageSize     int
}

func (q RecordQuery) Offset() int {
	if q.Page < 1 {
		q.Page = 1
	}
	return (q.Page - 1) * q.PageSize
}
