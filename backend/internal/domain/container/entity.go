package container

import (
	"errors"
	"time"
)

// Container 容器聚合根
type Container struct {
	ID            string
	HostID        string
	ContainerID   string // Docker 短 ID
	ContainerName string
	Image         string
	Status        string // running / exited / paused / removed
	ConfigPaths   []string
	Description   string
	LastSyncedAt  *time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// 领域错误
var (
	ErrContainerNotFound = errors.New("容器不存在")
	ErrInvalidFilePath   = errors.New("非法文件路径")
)
