package http

import (
	appHost "ops-hub/internal/application/host"
	domainHost "ops-hub/internal/domain/host"
	"ops-hub/internal/infrastructure/metrics"
	"ops-hub/pkg/apperr"
	"ops-hub/pkg/response"
	"time"

	"github.com/gin-gonic/gin"
)

var hostErrMappings = []apperr.DomainErrorMapping{
	{Target: domainHost.ErrHostNotFound, AppErr: apperr.NotFound("机器不存在")},
	{Target: domainHost.ErrHostNameExists, AppErr: apperr.Conflict("机器名称已存在")},
	{Target: domainHost.ErrHostUsernameRequired, AppErr: apperr.BadRequest("选择凭证时 SSH 用户名不能为空")},
}

// HostHandler 机器管理 Handler
type HostHandler struct {
	uc        *appHost.UseCase
	collector *metrics.Collector
}

func NewHostHandler(uc *appHost.UseCase, collector *metrics.Collector) *HostHandler {
	return &HostHandler{uc: uc, collector: collector}
}

func (h *HostHandler) RegisterRoutes(api *gin.RouterGroup) {
	g := api.Group("/hosts")
	g.GET("", apperr.Handle(h.List))
	g.POST("", apperr.Handle(h.Create))
	g.GET("/:id", apperr.Handle(h.Get))
	g.PUT("/:id", apperr.Handle(h.Update))
	g.DELETE("/:id", apperr.Handle(h.Delete))
	g.POST("/:id/test-connection", apperr.Handle(h.TestConnection))

	// 资源监控
	g.GET("/:id/metrics/latest", apperr.Handle(h.GetMetricsLatest))
	g.GET("/:id/metrics/history", apperr.Handle(h.GetMetricsHistory))
	g.GET("/metrics/overview", apperr.Handle(h.GetMetricsOverview))
	g.POST("/:id/metrics/collect", apperr.Handle(h.CollectMetrics))
}

func (h *HostHandler) List(c *gin.Context) error {
	var cmd appHost.HostQueryCmd
	if err := c.ShouldBindQuery(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}
	dtos, total, err := h.uc.List(c.Request.Context(), cmd)
	if err != nil {
		return apperr.Internal(err)
	}
	response.OKPage(c, dtos, total, cmd.Page, cmd.PageSize)
	return nil
}

func (h *HostHandler) Create(c *gin.Context) error {
	var cmd appHost.CreateHostCmd
	if err := c.ShouldBindJSON(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}
	dto, err := h.uc.Create(c.Request.Context(), cmd)
	if err != nil {
		return apperr.MapDomainError(err, hostErrMappings)
	}
	response.Created(c, dto)
	return nil
}

func (h *HostHandler) Get(c *gin.Context) error {
	dto, err := h.uc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		return apperr.MapDomainError(err, hostErrMappings)
	}
	response.OK(c, dto)
	return nil
}

func (h *HostHandler) Update(c *gin.Context) error {
	var cmd appHost.UpdateHostCmd
	if err := c.ShouldBindJSON(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}
	cmd.ID = c.Param("id")
	dto, err := h.uc.Update(c.Request.Context(), cmd)
	if err != nil {
		return apperr.MapDomainError(err, hostErrMappings)
	}
	response.OK(c, dto)
	return nil
}

func (h *HostHandler) Delete(c *gin.Context) error {
	if err := h.uc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		return apperr.MapDomainError(err, hostErrMappings)
	}
	response.NoContent(c)
	return nil
}

func (h *HostHandler) TestConnection(c *gin.Context) error {
	result, err := h.uc.TestConnection(c.Request.Context(), c.Param("id"))
	if err != nil {
		return apperr.MapDomainError(err, hostErrMappings)
	}
	response.OK(c, result)
	return nil
}

func (h *HostHandler) GetMetricsLatest(c *gin.Context) error {
	dto, err := h.uc.GetLatestMetrics(c.Request.Context(), c.Param("id"))
	if err != nil {
		return apperr.MapDomainError(err, hostErrMappings)
	}
	response.OK(c, dto)
	return nil
}

func (h *HostHandler) GetMetricsHistory(c *gin.Context) error {
	id := c.Param("id")
	var q struct {
		StartTime string `form:"start_time"`
		EndTime   string `form:"end_time"`
		Limit     int    `form:"limit"`
	}
	_ = c.ShouldBindQuery(&q)

	var startTime, endTime *time.Time
	if q.StartTime != "" {
		t, _ := time.Parse(time.RFC3339, q.StartTime)
		startTime = &t
	}
	if q.EndTime != "" {
		t, _ := time.Parse(time.RFC3339, q.EndTime)
		endTime = &t
	}

	dtos, err := h.uc.GetMetricsHistory(c.Request.Context(), id, startTime, endTime, q.Limit)
	if err != nil {
		return apperr.Internal(err)
	}
	response.OK(c, dtos)
	return nil
}

func (h *HostHandler) GetMetricsOverview(c *gin.Context) error {
	dtos, err := h.uc.GetAllLatestMetrics(c.Request.Context())
	if err != nil {
		return apperr.Internal(err)
	}
	response.OK(c, dtos)
	return nil
}

func (h *HostHandler) CollectMetrics(c *gin.Context) error {
	if h.collector == nil {
		return apperr.Internal(nil)
	}
	snapshot, err := h.collector.CollectOne(c.Request.Context(), c.Param("id"))
	if err != nil {
		return apperr.MapDomainError(err, hostErrMappings)
	}
	response.OK(c, h.uc.MetricSnapshotToDTO(snapshot))
	return nil
}
