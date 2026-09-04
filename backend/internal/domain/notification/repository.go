package notification

import "context"

// ChannelRepository 通知渠道仓储接口
type ChannelRepository interface {
	Save(ctx context.Context, ch *NotificationChannel) error
	Update(ctx context.Context, ch *NotificationChannel) error
	FindByID(ctx context.Context, id string) (*NotificationChannel, error)
	FindAll(ctx context.Context) ([]*NotificationChannel, error)
	FindEnabled(ctx context.Context) ([]*NotificationChannel, error)
	Delete(ctx context.Context, id string) error
}

// RuleRepository 通知规则仓储接口
type RuleRepository interface {
	Save(ctx context.Context, rule *NotificationRule) error
	Update(ctx context.Context, rule *NotificationRule) error
	FindByID(ctx context.Context, id string) (*NotificationRule, error)
	FindAll(ctx context.Context) ([]*NotificationRule, error)
	FindByEventType(ctx context.Context, eventType EventType) ([]*NotificationRule, error)
	Delete(ctx context.Context, id string) error
}

// LogRepository 通知记录仓储接口
type LogRepository interface {
	Save(ctx context.Context, log *NotificationLog) error
	Find(ctx context.Context, query LogQuery) ([]*NotificationLog, int64, error)
}

// LogQuery 通知记录查询
type LogQuery struct {
	ChannelID string
	EventType string
	Status    string
	Page      int
	PageSize  int
}

// Sender 通知发送接口
type Sender interface {
	Send(ctx context.Context, channel *NotificationChannel, title, content string) error
}
