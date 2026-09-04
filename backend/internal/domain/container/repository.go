package container

import "context"

// ContainerRepository 容器仓储接口
type ContainerRepository interface {
	Save(ctx context.Context, entity *Container) error
	Update(ctx context.Context, entity *Container) error
	FindByID(ctx context.Context, id string) (*Container, error)
	FindByHostID(ctx context.Context, hostID string) ([]*Container, error)
	FindByHostAndName(ctx context.Context, hostID, name string) (*Container, error)
	Delete(ctx context.Context, id string) error
	DeleteByHostIDNotIn(ctx context.Context, hostID string, names []string) error
}
