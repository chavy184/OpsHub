package notificationuc

import (
	"context"
	"fmt"
	"log"
	"ops-hub/internal/domain/notification"
	"ops-hub/internal/infrastructure/sender"
	"sync"
	"time"
)

// ============================
// DTO
// ============================

type CreateChannelCmd struct {
	Name        string `json:"name" binding:"required"`
	ChannelType string `json:"channel_type" binding:"required"`
	Config      string `json:"config" binding:"required"`
	Enabled     *bool  `json:"enabled"`
}

type UpdateChannelCmd struct {
	ID          string `json:"-"`
	Name        string `json:"name"`
	ChannelType string `json:"channel_type"`
	Config      string `json:"config"`
	Enabled     *bool  `json:"enabled"`
}

type CreateRuleCmd struct {
	EventType string `json:"event_type" binding:"required"`
	ChannelID string `json:"channel_id" binding:"required"`
	Enabled   *bool  `json:"enabled"`
	Filter    string `json:"filter"`
}

type UpdateRuleCmd struct {
	ID        string `json:"-"`
	EventType string `json:"event_type"`
	ChannelID string `json:"channel_id"`
	Enabled   *bool  `json:"enabled"`
	Filter    string `json:"filter"`
}

type LogQueryCmd struct {
	ChannelID string `form:"channel_id"`
	EventType string `form:"event_type"`
	Status    string `form:"status"`
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
}

type ChannelDTO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	ChannelType string `json:"channel_type"`
	Config      string `json:"config"`
	Enabled     bool   `json:"enabled"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type RuleDTO struct {
	ID        string `json:"id"`
	EventType string `json:"event_type"`
	ChannelID string `json:"channel_id"`
	Enabled   bool   `json:"enabled"`
	Filter    string `json:"filter"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type LogDTO struct {
	ID           string `json:"id"`
	ChannelID    string `json:"channel_id"`
	EventType    string `json:"event_type"`
	Title        string `json:"title"`
	Content      string `json:"content"`
	Status       string `json:"status"`
	ErrorMessage string `json:"error_message"`
	CreatedAt    string `json:"created_at"`
}

func channelToDTO(e *notification.NotificationChannel) *ChannelDTO {
	return &ChannelDTO{
		ID:          e.ID,
		Name:        e.Name,
		ChannelType: string(e.ChannelType),
		Config:      e.Config,
		Enabled:     e.Enabled,
		CreatedAt:   e.CreatedAt.Format(time.DateTime),
		UpdatedAt:   e.UpdatedAt.Format(time.DateTime),
	}
}

func ruleToDTO(e *notification.NotificationRule) *RuleDTO {
	return &RuleDTO{
		ID:        e.ID,
		EventType: string(e.EventType),
		ChannelID: e.ChannelID,
		Enabled:   e.Enabled,
		Filter:    e.Filter,
		CreatedAt: e.CreatedAt.Format(time.DateTime),
		UpdatedAt: e.UpdatedAt.Format(time.DateTime),
	}
}

func logToDTO(e *notification.NotificationLog) *LogDTO {
	return &LogDTO{
		ID:           e.ID,
		ChannelID:    e.ChannelID,
		EventType:    string(e.EventType),
		Title:        e.Title,
		Content:      e.Content,
		Status:       e.Status,
		ErrorMessage: e.ErrorMessage,
		CreatedAt:    e.CreatedAt.Format(time.DateTime),
	}
}

// ============================
// UseCase
// ============================

type UseCase struct {
	channelRepo notification.ChannelRepository
	ruleRepo    notification.RuleRepository
	logRepo     notification.LogRepository
	cooldownMu  sync.Mutex
	cooldownMap map[string]time.Time // key: eventType+fingerprint → last sent time
}

func NewUseCase(
	channelRepo notification.ChannelRepository,
	ruleRepo notification.RuleRepository,
	logRepo notification.LogRepository,
) *UseCase {
	return &UseCase{
		channelRepo: channelRepo,
		ruleRepo:    ruleRepo,
		logRepo:     logRepo,
		cooldownMap: make(map[string]time.Time),
	}
}

// ─── Channel CRUD ────────────────────────────────────────

func (uc *UseCase) CreateChannel(ctx context.Context, cmd CreateChannelCmd) (*ChannelDTO, error) {
	enabled := true
	if cmd.Enabled != nil {
		enabled = *cmd.Enabled
	}
	e := &notification.NotificationChannel{
		Name:        cmd.Name,
		ChannelType: notification.ChannelType(cmd.ChannelType),
		Config:      cmd.Config,
		Enabled:     enabled,
	}
	if err := uc.channelRepo.Save(ctx, e); err != nil {
		return nil, fmt.Errorf("创建通知渠道失败: %w", err)
	}
	return channelToDTO(e), nil
}

func (uc *UseCase) UpdateChannel(ctx context.Context, cmd UpdateChannelCmd) (*ChannelDTO, error) {
	e, err := uc.channelRepo.FindByID(ctx, cmd.ID)
	if err != nil {
		return nil, notification.ErrChannelNotFound
	}
	if cmd.Name != "" {
		e.Name = cmd.Name
	}
	if cmd.ChannelType != "" {
		e.ChannelType = notification.ChannelType(cmd.ChannelType)
	}
	if cmd.Config != "" {
		e.Config = cmd.Config
	}
	if cmd.Enabled != nil {
		e.Enabled = *cmd.Enabled
	}
	if err := uc.channelRepo.Update(ctx, e); err != nil {
		return nil, fmt.Errorf("更新通知渠道失败: %w", err)
	}
	return channelToDTO(e), nil
}

func (uc *UseCase) GetChannel(ctx context.Context, id string) (*ChannelDTO, error) {
	e, err := uc.channelRepo.FindByID(ctx, id)
	if err != nil {
		return nil, notification.ErrChannelNotFound
	}
	return channelToDTO(e), nil
}

func (uc *UseCase) ListChannels(ctx context.Context) ([]*ChannelDTO, error) {
	list, err := uc.channelRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	dtos := make([]*ChannelDTO, len(list))
	for i, e := range list {
		dtos[i] = channelToDTO(e)
	}
	return dtos, nil
}

func (uc *UseCase) DeleteChannel(ctx context.Context, id string) error {
	if _, err := uc.channelRepo.FindByID(ctx, id); err != nil {
		return notification.ErrChannelNotFound
	}
	return uc.channelRepo.Delete(ctx, id)
}

func (uc *UseCase) TestChannel(ctx context.Context, id string) error {
	ch, err := uc.channelRepo.FindByID(ctx, id)
	if err != nil {
		return notification.ErrChannelNotFound
	}
	s := sender.SenderFactory(ch.ChannelType)
	if s == nil {
		return fmt.Errorf("不支持的渠道类型: %s", ch.ChannelType)
	}
	return s.Send(ctx, ch, "[OpsHub 测试] 通知渠道测试", "这是一条测试消息，如果您收到此消息说明通知渠道配置正确。\n\n时间: "+time.Now().Format(time.DateTime))
}

// ─── Rule CRUD ───────────────────────────────────────────

func (uc *UseCase) CreateRule(ctx context.Context, cmd CreateRuleCmd) (*RuleDTO, error) {
	enabled := true
	if cmd.Enabled != nil {
		enabled = *cmd.Enabled
	}
	filter := cmd.Filter
	if filter == "" {
		filter = "{}"
	}
	e := &notification.NotificationRule{
		EventType: notification.EventType(cmd.EventType),
		ChannelID: cmd.ChannelID,
		Enabled:   enabled,
		Filter:    filter,
	}
	if err := uc.ruleRepo.Save(ctx, e); err != nil {
		return nil, fmt.Errorf("创建通知规则失败: %w", err)
	}
	return ruleToDTO(e), nil
}

func (uc *UseCase) UpdateRule(ctx context.Context, cmd UpdateRuleCmd) (*RuleDTO, error) {
	e, err := uc.ruleRepo.FindByID(ctx, cmd.ID)
	if err != nil {
		return nil, notification.ErrRuleNotFound
	}
	if cmd.EventType != "" {
		e.EventType = notification.EventType(cmd.EventType)
	}
	if cmd.ChannelID != "" {
		e.ChannelID = cmd.ChannelID
	}
	if cmd.Enabled != nil {
		e.Enabled = *cmd.Enabled
	}
	if cmd.Filter != "" {
		e.Filter = cmd.Filter
	}
	if err := uc.ruleRepo.Update(ctx, e); err != nil {
		return nil, fmt.Errorf("更新通知规则失败: %w", err)
	}
	return ruleToDTO(e), nil
}

func (uc *UseCase) ListRules(ctx context.Context) ([]*RuleDTO, error) {
	list, err := uc.ruleRepo.FindAll(ctx)
	if err != nil {
		return nil, err
	}
	dtos := make([]*RuleDTO, len(list))
	for i, e := range list {
		dtos[i] = ruleToDTO(e)
	}
	return dtos, nil
}

func (uc *UseCase) DeleteRule(ctx context.Context, id string) error {
	if _, err := uc.ruleRepo.FindByID(ctx, id); err != nil {
		return notification.ErrRuleNotFound
	}
	return uc.ruleRepo.Delete(ctx, id)
}

// ─── Log Query ───────────────────────────────────────────

func (uc *UseCase) ListLogs(ctx context.Context, cmd LogQueryCmd) ([]*LogDTO, int64, error) {
	logs, total, err := uc.logRepo.Find(ctx, notification.LogQuery{
		ChannelID: cmd.ChannelID,
		EventType: cmd.EventType,
		Status:    cmd.Status,
		Page:      cmd.Page,
		PageSize:  cmd.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}
	dtos := make([]*LogDTO, len(logs))
	for i, e := range logs {
		dtos[i] = logToDTO(e)
	}
	return dtos, total, nil
}

// ─── Dispatch (被其他模块调用) ────────────────────────────

const DefaultCooldownSeconds = 300

// Dispatch 根据事件类型查找匹配规则，发送通知
func (uc *UseCase) Dispatch(ctx context.Context, eventType notification.EventType, fingerprint, title, content string) {
	// 冷却检查
	cooldownKey := string(eventType) + ":" + fingerprint
	uc.cooldownMu.Lock()
	if lastSent, ok := uc.cooldownMap[cooldownKey]; ok {
		if time.Since(lastSent) < time.Duration(DefaultCooldownSeconds)*time.Second {
			uc.cooldownMu.Unlock()
			return
		}
	}
	uc.cooldownMap[cooldownKey] = time.Now()
	uc.cooldownMu.Unlock()

	rules, err := uc.ruleRepo.FindByEventType(ctx, eventType)
	if err != nil || len(rules) == 0 {
		return
	}

	for _, rule := range rules {
		ch, err := uc.channelRepo.FindByID(ctx, rule.ChannelID)
		if err != nil || !ch.Enabled {
			continue
		}

		s := sender.SenderFactory(ch.ChannelType)
		if s == nil {
			continue
		}

		logEntry := &notification.NotificationLog{
			ChannelID: ch.ID,
			EventType: eventType,
			Title:     title,
			Content:   content,
			Status:    "sent",
		}

		if sendErr := s.Send(ctx, ch, title, content); sendErr != nil {
			logEntry.Status = "failed"
			logEntry.ErrorMessage = sendErr.Error()
			log.Printf("[Notification] send failed: channel=%s err=%v", ch.Name, sendErr)
		}

		_ = uc.logRepo.Save(ctx, logEntry)
	}
}
