package image_sync

import (
	"errors"
	"time"
)

type SyncMode string

const (
	SyncModeSkipIfExists SyncMode = "skip_if_exists"
	SyncModeOverwrite    SyncMode = "overwrite"
)

type SyncStatus string

const (
	SyncStatusRunning SyncStatus = "running"
	SyncStatusSuccess SyncStatus = "success"
	SyncStatusFailed  SyncStatus = "failed"
	SyncStatusSkipped SyncStatus = "skipped"
)

type Record struct {
	ID             string
	SourceHostID   string
	SourceHostName string
	TargetHostID   string
	TargetHostName string
	Image          string
	Mode           SyncMode
	Status         SyncStatus
	SourceImageID  string
	TargetImageID  string
	ImageSize      int64
	Summary        string
	Error          string
	StartedAt      time.Time
	FinishedAt     *time.Time
	Duration       int64
	CreatedAt      time.Time
}

var (
	ErrRecordNotFound    = errors.New("镜像同步记录不存在")
	ErrInvalidMode       = errors.New("无效的镜像同步模式")
	ErrInvalidImageName  = errors.New("无效的镜像名称")
	ErrSameHost          = errors.New("源主机和目标主机不能相同")
	ErrSourceImageAbsent = errors.New("源主机镜像不存在")
)
