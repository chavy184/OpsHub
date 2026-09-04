package notification

import (
	"errors"
	"time"
)

// ChannelType 通知渠道类型
type ChannelType string

const (
	ChannelTypeEmail    ChannelType = "email"
	ChannelTypeWecomBot ChannelType = "wecom_bot"
)

// EventType 通知事件类型
type EventType string

const (
	EventHealthCheckFail EventType = "health_check_fail"
	EventHealthCheckOK   EventType = "health_check_ok"
	EventDeploySuccess   EventType = "deploy_success"
	EventDeployFail      EventType = "deploy_fail"
	EventResourceAlert   EventType = "resource_alert"
	EventHostOffline     EventType = "host_offline"
)

// NotificationChannel 通知渠道配置
type NotificationChannel struct {
	ID          string
	Name        string
	ChannelType ChannelType
	Config      string // JSON
	Enabled     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NotificationRule 通知规则
type NotificationRule struct {
	ID        string
	EventType EventType
	ChannelID string
	Enabled   bool
	Filter    string // JSON
	CreatedAt time.Time
	UpdatedAt time.Time
}

// NotificationLog 通知记录
type NotificationLog struct {
	ID           string
	ChannelID    string
	EventType    EventType
	Title        string
	Content      string
	Status       string // sent / failed
	ErrorMessage string
	CreatedAt    time.Time
}

var (
	ErrChannelNotFound = errors.New("通知渠道不存在")
	ErrRuleNotFound    = errors.New("通知规则不存在")
)
