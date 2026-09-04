package http

import (
	appNotification "ops-hub/internal/application/notification"
	"ops-hub/internal/domain/notification"
	"ops-hub/pkg/apperr"
	"ops-hub/pkg/response"

	"github.com/gin-gonic/gin"
)

// notification 领域错误 → AppError 映射
var notificationErrMappings = []apperr.DomainErrorMapping{
	{Target: notification.ErrChannelNotFound, AppErr: apperr.NotFound("通知渠道不存在")},
	{Target: notification.ErrRuleNotFound, AppErr: apperr.NotFound("通知规则不存在")},
}

type NotificationHandler struct {
	uc *appNotification.UseCase
}

func NewNotificationHandler(uc *appNotification.UseCase) *NotificationHandler {
	return &NotificationHandler{uc: uc}
}

func (h *NotificationHandler) RegisterRoutes(api *gin.RouterGroup) {
	g := api.Group("/notification")

	g.GET("/channels", apperr.Handle(h.ListChannels))
	g.POST("/channels", apperr.Handle(h.CreateChannel))
	g.PUT("/channels/:id", apperr.Handle(h.UpdateChannel))
	g.DELETE("/channels/:id", apperr.Handle(h.DeleteChannel))
	g.POST("/channels/:id/test", apperr.Handle(h.TestChannel))

	g.GET("/rules", apperr.Handle(h.ListRules))
	g.POST("/rules", apperr.Handle(h.CreateRule))
	g.PUT("/rules/:id", apperr.Handle(h.UpdateRule))
	g.DELETE("/rules/:id", apperr.Handle(h.DeleteRule))

	g.GET("/logs", apperr.Handle(h.ListLogs))
}

// ─── Channel ─────────────────────────────────────────────

func (h *NotificationHandler) ListChannels(c *gin.Context) error {
	dtos, err := h.uc.ListChannels(c.Request.Context())
	if err != nil {
		return apperr.Internal(err)
	}
	response.OK(c, dtos)
	return nil
}

func (h *NotificationHandler) CreateChannel(c *gin.Context) error {
	var cmd appNotification.CreateChannelCmd
	if err := c.ShouldBindJSON(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}
	dto, err := h.uc.CreateChannel(c.Request.Context(), cmd)
	if err != nil {
		return apperr.MapDomainError(err, notificationErrMappings)
	}
	response.Created(c, dto)
	return nil
}

func (h *NotificationHandler) UpdateChannel(c *gin.Context) error {
	var cmd appNotification.UpdateChannelCmd
	if err := c.ShouldBindJSON(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}
	cmd.ID = c.Param("id")
	dto, err := h.uc.UpdateChannel(c.Request.Context(), cmd)
	if err != nil {
		return apperr.MapDomainError(err, notificationErrMappings)
	}
	response.OK(c, dto)
	return nil
}

func (h *NotificationHandler) DeleteChannel(c *gin.Context) error {
	if err := h.uc.DeleteChannel(c.Request.Context(), c.Param("id")); err != nil {
		return apperr.MapDomainError(err, notificationErrMappings)
	}
	response.NoContent(c)
	return nil
}

func (h *NotificationHandler) TestChannel(c *gin.Context) error {
	if err := h.uc.TestChannel(c.Request.Context(), c.Param("id")); err != nil {
		return apperr.MapDomainError(err, notificationErrMappings)
	}
	response.OK(c, gin.H{"message": "测试通知已发送"})
	return nil
}

// ─── Rule ────────────────────────────────────────────────

func (h *NotificationHandler) ListRules(c *gin.Context) error {
	dtos, err := h.uc.ListRules(c.Request.Context())
	if err != nil {
		return apperr.Internal(err)
	}
	response.OK(c, dtos)
	return nil
}

func (h *NotificationHandler) CreateRule(c *gin.Context) error {
	var cmd appNotification.CreateRuleCmd
	if err := c.ShouldBindJSON(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}
	dto, err := h.uc.CreateRule(c.Request.Context(), cmd)
	if err != nil {
		return apperr.MapDomainError(err, notificationErrMappings)
	}
	response.Created(c, dto)
	return nil
}

func (h *NotificationHandler) UpdateRule(c *gin.Context) error {
	var cmd appNotification.UpdateRuleCmd
	if err := c.ShouldBindJSON(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}
	cmd.ID = c.Param("id")
	dto, err := h.uc.UpdateRule(c.Request.Context(), cmd)
	if err != nil {
		return apperr.MapDomainError(err, notificationErrMappings)
	}
	response.OK(c, dto)
	return nil
}

func (h *NotificationHandler) DeleteRule(c *gin.Context) error {
	if err := h.uc.DeleteRule(c.Request.Context(), c.Param("id")); err != nil {
		return apperr.MapDomainError(err, notificationErrMappings)
	}
	response.NoContent(c)
	return nil
}

// ─── Log ─────────────────────────────────────────────────

func (h *NotificationHandler) ListLogs(c *gin.Context) error {
	var cmd appNotification.LogQueryCmd
	if err := c.ShouldBindQuery(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}
	dtos, total, err := h.uc.ListLogs(c.Request.Context(), cmd)
	if err != nil {
		return apperr.Internal(err)
	}
	response.OKPage(c, dtos, total, cmd.Page, cmd.PageSize)
	return nil
}
