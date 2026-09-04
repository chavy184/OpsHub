package http

import (
	appAlert "ops-hub/internal/application/alert"
	"ops-hub/internal/domain/alert"
	"ops-hub/pkg/apperr"
	"ops-hub/pkg/response"
	"strconv"

	"github.com/gin-gonic/gin"
)

// alert 领域错误 → AppError 映射
var alertErrMappings = []apperr.DomainErrorMapping{
	{Target: alert.ErrAlertNotFound, AppErr: apperr.NotFound("告警事件不存在")},
	{Target: alert.ErrInvalidAlertState, AppErr: apperr.BadRequest("告警状态不允许该操作")},
}

type AlertHandler struct {
	uc *appAlert.UseCase
}

func NewAlertHandler(uc *appAlert.UseCase) *AlertHandler {
	return &AlertHandler{uc: uc}
}

func (h *AlertHandler) RegisterRoutes(rg *gin.RouterGroup) {
	g := rg.Group("/alerts")
	g.GET("", apperr.Handle(h.ListAlerts))
	g.GET("/stats", apperr.Handle(h.GetStats))
	g.GET("/:id", apperr.Handle(h.GetAlert))
	g.POST("", apperr.Handle(h.CreateAlert))
	g.POST("/:id/ack", apperr.Handle(h.AckAlert))
	g.POST("/:id/close", apperr.Handle(h.CloseAlert))
}

func (h *AlertHandler) ListAlerts(c *gin.Context) error {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	query := alert.AlertQuery{
		ServiceID: c.Query("service_id"),
		EnvID:     c.Query("env_id"),
		Severity:  c.Query("severity"),
		Status:    c.Query("status"),
		Keyword:   c.Query("keyword"),
		Page:      page,
		PageSize:  pageSize,
	}

	result, err := h.uc.ListAlerts(c.Request.Context(), query)
	if err != nil {
		return apperr.MapDomainError(err, alertErrMappings)
	}
	response.OKPage(c, result.Items, result.Total, page, pageSize)
	return nil
}

func (h *AlertHandler) GetStats(c *gin.Context) error {
	stats, err := h.uc.GetStats(c.Request.Context())
	if err != nil {
		return apperr.Internal(err)
	}
	response.OK(c, stats)
	return nil
}

func (h *AlertHandler) GetAlert(c *gin.Context) error {
	dto, err := h.uc.GetAlert(c.Request.Context(), c.Param("id"))
	if err != nil {
		return apperr.MapDomainError(err, alertErrMappings)
	}
	response.OK(c, dto)
	return nil
}

func (h *AlertHandler) CreateAlert(c *gin.Context) error {
	var cmd appAlert.CreateAlertCmd
	if err := c.ShouldBindJSON(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}
	dto, err := h.uc.CreateAlert(c.Request.Context(), cmd)
	if err != nil {
		return apperr.MapDomainError(err, alertErrMappings)
	}
	response.Created(c, dto)
	return nil
}

func (h *AlertHandler) AckAlert(c *gin.Context) error {
	var body struct {
		UserID string `json:"user_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		return apperr.BadRequest(err.Error())
	}
	if body.UserID == "" {
		body.UserID = "system"
	}
	if err := h.uc.AckAlert(c.Request.Context(), c.Param("id"), body.UserID); err != nil {
		return apperr.MapDomainError(err, alertErrMappings)
	}
	response.OK(c, nil)
	return nil
}

func (h *AlertHandler) CloseAlert(c *gin.Context) error {
	if err := h.uc.CloseAlert(c.Request.Context(), c.Param("id")); err != nil {
		return apperr.MapDomainError(err, alertErrMappings)
	}
	response.OK(c, nil)
	return nil
}
