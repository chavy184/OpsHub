package http

import (
	"log"
	appService "ops-hub/internal/application/service"
	"ops-hub/internal/domain/service"
	"ops-hub/pkg/apperr"
	"ops-hub/pkg/response"

	"github.com/gin-gonic/gin"
)

// service 领域错误 → AppError 映射
var serviceErrMappings = []apperr.DomainErrorMapping{
	{Target: service.ErrServiceNotFound, AppErr: apperr.NotFound("服务不存在")},
	{Target: service.ErrServiceKeyDuplicated, AppErr: apperr.Conflict("服务标识已存在")},
	{Target: service.ErrServiceEnvNotFound, AppErr: apperr.NotFound("服务环境不存在")},
}

type ServiceHandler struct {
	uc *appService.UseCase
}

func NewServiceHandler(uc *appService.UseCase) *ServiceHandler {
	return &ServiceHandler{uc: uc}
}

func (h *ServiceHandler) RegisterRoutes(api *gin.RouterGroup) {
	g := api.Group("/services")
	{
		g.GET("", apperr.Handle(h.ListServices))
		g.POST("", apperr.Handle(h.CreateService))
		g.GET("/:id", apperr.Handle(h.GetService))
		g.PUT("/:id", apperr.Handle(h.UpdateService))
		g.DELETE("/:id", apperr.Handle(h.DeleteService))

		// 环境
		g.GET("/:id/envs", apperr.Handle(h.ListEnvs))
		g.POST("/:id/envs", apperr.Handle(h.CreateEnv))
		g.PUT("/:id/envs/:envId", apperr.Handle(h.UpdateEnv))
		g.DELETE("/:id/envs/:envId", apperr.Handle(h.DeleteEnv))
	}
}

// ListServices 服务列表
// GET /api/v1/services?status=active&tech_stack=go&keyword=xxx&page=1&page_size=20
func (h *ServiceHandler) ListServices(c *gin.Context) error {
	var q appService.ServiceQueryCmd
	if err := c.ShouldBindQuery(&q); err != nil {
		return apperr.BadRequest(err.Error())
	}

	dtos, total, err := h.uc.ListServices(c.Request.Context(), q)
	if err != nil {
		return apperr.MapDomainError(err, serviceErrMappings)
	}

	response.OKPage(c, dtos, total, q.Page, q.PageSize)
	return nil
}

// GetService 服务详情
// GET /api/v1/services/:id
func (h *ServiceHandler) GetService(c *gin.Context) error {
	id := c.Param("id")

	dto, err := h.uc.GetService(c.Request.Context(), id)
	if err != nil {
		return apperr.MapDomainError(err, serviceErrMappings)
	}

	response.OK(c, dto)
	return nil
}

// CreateService 创建服务
// POST /api/v1/services
func (h *ServiceHandler) CreateService(c *gin.Context) error {
	var cmd appService.CreateServiceCmd
	if err := c.ShouldBindJSON(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}

	dto, err := h.uc.CreateService(c.Request.Context(), cmd)
	if err != nil {
		return apperr.MapDomainError(err, serviceErrMappings)
	}

	response.Created(c, dto)
	return nil
}

// UpdateService 更新服务
// PUT /api/v1/services/:id
func (h *ServiceHandler) UpdateService(c *gin.Context) error {
	var cmd appService.UpdateServiceCmd
	if err := c.ShouldBindJSON(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}
	cmd.ServiceID = c.Param("id")

	dto, err := h.uc.UpdateService(c.Request.Context(), cmd)
	if err != nil {
		return apperr.MapDomainError(err, serviceErrMappings)
	}

	response.OK(c, dto)
	return nil
}

// ListEnvs 服务环境列表
// GET /api/v1/services/:id/envs
func (h *ServiceHandler) ListEnvs(c *gin.Context) error {
	serviceID := c.Param("id")

	dtos, err := h.uc.ListEnvs(c.Request.Context(), serviceID)
	if err != nil {
		return apperr.MapDomainError(err, serviceErrMappings)
	}

	response.OK(c, dtos)
	return nil
}

// CreateEnv 创建服务环境
// POST /api/v1/services/:id/envs
func (h *ServiceHandler) CreateEnv(c *gin.Context) error {
	var cmd appService.CreateEnvCmd
	if err := c.ShouldBindJSON(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}
	cmd.ServiceID = c.Param("id")

	dto, err := h.uc.CreateEnv(c.Request.Context(), cmd)
	if err != nil {
		return apperr.MapDomainError(err, serviceErrMappings)
	}

	response.Created(c, dto)
	return nil
}

// UpdateEnv 更新服务环境
// PUT /api/v1/services/:id/envs/:envId
func (h *ServiceHandler) UpdateEnv(c *gin.Context) error {
	var cmd appService.UpdateEnvCmd
	if err := c.ShouldBindJSON(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}
	cmd.EnvID = c.Param("envId")

	dto, err := h.uc.UpdateEnv(c.Request.Context(), cmd)
	if err != nil {
		log.Printf("UpdateEnv failed: envId=%s err=%v", cmd.EnvID, err)
		return apperr.MapDomainError(err, serviceErrMappings)
	}

	response.OK(c, dto)
	return nil
}

// DeleteEnv 删除服务环境
// DELETE /api/v1/services/:id/envs/:envId
func (h *ServiceHandler) DeleteEnv(c *gin.Context) error {
	envID := c.Param("envId")

	if err := h.uc.DeleteEnv(c.Request.Context(), envID); err != nil {
		return apperr.MapDomainError(err, serviceErrMappings)
	}

	response.OK(c, nil)
	return nil
}

// DeleteService 删除服务
// DELETE /api/v1/services/:id
func (h *ServiceHandler) DeleteService(c *gin.Context) error {
	id := c.Param("id")

	if err := h.uc.DeleteService(c.Request.Context(), id); err != nil {
		return apperr.MapDomainError(err, serviceErrMappings)
	}

	response.OK(c, nil)
	return nil
}
