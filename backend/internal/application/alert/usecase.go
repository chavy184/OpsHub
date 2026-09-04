package alert

import (
	"context"
	"fmt"
	"ops-hub/internal/domain/alert"
	"time"
)

type UseCase struct {
	alertRepo alert.AlertEventRepository
}

func NewUseCase(alertRepo alert.AlertEventRepository) *UseCase {
	return &UseCase{alertRepo: alertRepo}
}

// ============================
// DTO
// ============================

type AlertDTO struct {
	ID               string `json:"id"`
	ServiceID        string `json:"service_id"`
	EnvID            string `json:"env_id"`
	AlertSource      string `json:"alert_source"`
	AlertFingerprint string `json:"alert_fingerprint"`
	Severity         string `json:"severity"`
	Title            string `json:"title"`
	Content          string `json:"content"`
	Status           string `json:"status"`
	FirstSeenAt      string `json:"first_seen_at"`
	LastSeenAt       string `json:"last_seen_at"`
	AssigneeUserID   string `json:"assignee_user_id"`
	CreatedAt        string `json:"created_at"`
}

func toDTO(e *alert.AlertEvent) *AlertDTO {
	return &AlertDTO{
		ID:               e.ID,
		ServiceID:        e.ServiceID,
		EnvID:            e.EnvID,
		AlertSource:      e.AlertSource,
		AlertFingerprint: e.AlertFingerprint,
		Severity:         string(e.Severity),
		Title:            e.Title,
		Content:          e.Content,
		Status:           string(e.Status),
		FirstSeenAt:      e.FirstSeenAt.Format(time.DateTime),
		LastSeenAt:       e.LastSeenAt.Format(time.DateTime),
		AssigneeUserID:   e.AssigneeUserID,
		CreatedAt:        e.CreatedAt.Format(time.DateTime),
	}
}

// ============================
// Commands
// ============================

type CreateAlertCmd struct {
	ServiceID   string `json:"service_id"`
	EnvID       string `json:"env_id"`
	AlertSource string `json:"alert_source"`
	Severity    string `json:"severity"`
	Title       string `json:"title"`
	Content     string `json:"content"`
}

func (uc *UseCase) CreateAlert(ctx context.Context, cmd CreateAlertCmd) (*AlertDTO, error) {
	if cmd.Title == "" {
		return nil, fmt.Errorf("告警标题不能为空")
	}
	if cmd.Severity == "" {
		cmd.Severity = "P3"
	}
	if cmd.AlertSource == "" {
		cmd.AlertSource = "manual"
	}

	entity := &alert.AlertEvent{
		ServiceID:   cmd.ServiceID,
		EnvID:       cmd.EnvID,
		AlertSource: cmd.AlertSource,
		Severity:    alert.Severity(cmd.Severity),
		Title:       cmd.Title,
		Content:     cmd.Content,
		Status:      alert.AlertStatusOpen,
	}
	if err := uc.alertRepo.Save(ctx, entity); err != nil {
		return nil, fmt.Errorf("创建告警失败: %w", err)
	}
	return toDTO(entity), nil
}

func (uc *UseCase) AckAlert(ctx context.Context, id string, userID string) error {
	entity, err := uc.alertRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if err := entity.Ack(userID); err != nil {
		return err
	}
	return uc.alertRepo.Update(ctx, entity)
}

func (uc *UseCase) CloseAlert(ctx context.Context, id string) error {
	entity, err := uc.alertRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if err := entity.Close(); err != nil {
		return err
	}
	return uc.alertRepo.Update(ctx, entity)
}

// ============================
// Queries
// ============================

type AlertListResult struct {
	Items []*AlertDTO `json:"items"`
	Total int64       `json:"total"`
}

func (uc *UseCase) ListAlerts(ctx context.Context, query alert.AlertQuery) (*AlertListResult, error) {
	items, total, err := uc.alertRepo.Find(ctx, query)
	if err != nil {
		return nil, err
	}
	dtos := make([]*AlertDTO, len(items))
	for i, item := range items {
		dtos[i] = toDTO(item)
	}
	return &AlertListResult{Items: dtos, Total: total}, nil
}

func (uc *UseCase) GetAlert(ctx context.Context, id string) (*AlertDTO, error) {
	entity, err := uc.alertRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toDTO(entity), nil
}

// GetStats 获取告警统计
type AlertStats struct {
	TotalOpen   int64 `json:"total_open"`
	TotalAcked  int64 `json:"total_acked"`
	TotalClosed int64 `json:"total_closed"`
	P1Open      int64 `json:"p1_open"`
	P2Open      int64 `json:"p2_open"`
}

func (uc *UseCase) GetStats(ctx context.Context) (*AlertStats, error) {
	openItems, openTotal, err := uc.alertRepo.Find(ctx, alert.AlertQuery{Status: "open", PageSize: 1})
	_ = openItems
	if err != nil {
		return nil, err
	}
	ackedItems, ackedTotal, err := uc.alertRepo.Find(ctx, alert.AlertQuery{Status: "acked", PageSize: 1})
	_ = ackedItems
	if err != nil {
		return nil, err
	}
	closedItems, closedTotal, err := uc.alertRepo.Find(ctx, alert.AlertQuery{Status: "closed", PageSize: 1})
	_ = closedItems
	if err != nil {
		return nil, err
	}
	p1Items, p1Total, err := uc.alertRepo.Find(ctx, alert.AlertQuery{Status: "open", Severity: "P1", PageSize: 1})
	_ = p1Items
	if err != nil {
		return nil, err
	}
	p2Items, p2Total, err := uc.alertRepo.Find(ctx, alert.AlertQuery{Status: "open", Severity: "P2", PageSize: 1})
	_ = p2Items
	if err != nil {
		return nil, err
	}

	return &AlertStats{
		TotalOpen:   openTotal,
		TotalAcked:  ackedTotal,
		TotalClosed: closedTotal,
		P1Open:      p1Total,
		P2Open:      p2Total,
	}, nil
}
