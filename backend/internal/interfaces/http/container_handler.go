package http

import (
	appContainer "ops-hub/internal/application/container"
	domainContainer "ops-hub/internal/domain/container"
	"ops-hub/pkg/apperr"
	"ops-hub/pkg/response"

	"github.com/gin-gonic/gin"
)

// container 领域错误 → AppError 映射
var containerErrMappings = []apperr.DomainErrorMapping{
	{Target: domainContainer.ErrContainerNotFound, AppErr: apperr.NotFound("容器不存在")},
	{Target: domainContainer.ErrInvalidFilePath, AppErr: apperr.BadRequest("非法文件路径")},
}

// ContainerHandler 容器管理 Handler
type ContainerHandler struct {
	uc *appContainer.UseCase
}

func NewContainerHandler(uc *appContainer.UseCase) *ContainerHandler {
	return &ContainerHandler{uc: uc}
}

func (h *ContainerHandler) RegisterRoutes(api *gin.RouterGroup) {
	g := api.Group("/hosts/:id/containers")
	g.POST("/sync", apperr.Handle(h.Sync))
	g.GET("", apperr.Handle(h.List))
	g.PUT("/:containerId", apperr.Handle(h.Update))
	g.POST("/:containerId/start", apperr.Handle(h.Start))
	g.POST("/:containerId/stop", apperr.Handle(h.Stop))
	g.POST("/:containerId/restart", apperr.Handle(h.Restart))
	g.GET("/:containerId/inspect", apperr.Handle(h.Inspect))
	g.GET("/:containerId/config", apperr.Handle(h.ReadConfig))
	g.PUT("/:containerId/config", apperr.Handle(h.WriteConfig))
}

// Sync 同步容器列表
// POST /api/v1/hosts/:id/containers/sync
func (h *ContainerHandler) Sync(c *gin.Context) error {
	hostID := c.Param("id")
	dtos, err := h.uc.SyncContainers(c.Request.Context(), hostID)
	if err != nil {
		return apperr.MapDomainError(err, containerErrMappings)
	}
	response.OK(c, dtos)
	return nil
}

// List 获取容器列表
// GET /api/v1/hosts/:id/containers
func (h *ContainerHandler) List(c *gin.Context) error {
	hostID := c.Param("id")
	dtos, err := h.uc.List(c.Request.Context(), hostID)
	if err != nil {
		return apperr.MapDomainError(err, containerErrMappings)
	}
	response.OK(c, dtos)
	return nil
}

// Update 更新容器信息（配置路径、描述）
// PUT /api/v1/hosts/:id/containers/:containerId
func (h *ContainerHandler) Update(c *gin.Context) error {
	var cmd appContainer.UpdateContainerCmd
	if err := c.ShouldBindJSON(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}
	cmd.ID = c.Param("containerId")
	dto, err := h.uc.Update(c.Request.Context(), cmd)
	if err != nil {
		return apperr.MapDomainError(err, containerErrMappings)
	}
	response.OK(c, dto)
	return nil
}

// Start 启动容器
// POST /api/v1/hosts/:id/containers/:containerId/start
func (h *ContainerHandler) Start(c *gin.Context) error {
	hostID := c.Param("id")
	id := c.Param("containerId")
	if err := h.uc.Start(c.Request.Context(), hostID, id); err != nil {
		return apperr.MapDomainError(err, containerErrMappings)
	}
	response.OK(c, nil)
	return nil
}

// Stop 停止容器
// POST /api/v1/hosts/:id/containers/:containerId/stop
func (h *ContainerHandler) Stop(c *gin.Context) error {
	hostID := c.Param("id")
	id := c.Param("containerId")
	if err := h.uc.Stop(c.Request.Context(), hostID, id); err != nil {
		return apperr.MapDomainError(err, containerErrMappings)
	}
	response.OK(c, nil)
	return nil
}

// Restart 重启容器
// POST /api/v1/hosts/:id/containers/:containerId/restart
func (h *ContainerHandler) Restart(c *gin.Context) error {
	hostID := c.Param("id")
	id := c.Param("containerId")
	if err := h.uc.Restart(c.Request.Context(), hostID, id); err != nil {
		return apperr.MapDomainError(err, containerErrMappings)
	}
	response.OK(c, nil)
	return nil
}

// Inspect 容器详情
// GET /api/v1/hosts/:id/containers/:containerId/inspect
func (h *ContainerHandler) Inspect(c *gin.Context) error {
	hostID := c.Param("id")
	id := c.Param("containerId")
	dto, err := h.uc.Inspect(c.Request.Context(), hostID, id)
	if err != nil {
		return apperr.MapDomainError(err, containerErrMappings)
	}
	response.OK(c, dto)
	return nil
}

// ReadConfig 读取配置文件
// GET /api/v1/hosts/:id/containers/:containerId/config?path=/etc/nginx/nginx.conf
func (h *ContainerHandler) ReadConfig(c *gin.Context) error {
	hostID := c.Param("id")
	id := c.Param("containerId")
	filePath := c.Query("path")
	if filePath == "" {
		return apperr.BadRequest("path 参数必填")
	}
	dto, err := h.uc.ReadConfig(c.Request.Context(), hostID, id, filePath)
	if err != nil {
		return apperr.MapDomainError(err, containerErrMappings)
	}
	response.OK(c, dto)
	return nil
}

// WriteConfig 写入配置文件并重启
// PUT /api/v1/hosts/:id/containers/:containerId/config
func (h *ContainerHandler) WriteConfig(c *gin.Context) error {
	hostID := c.Param("id")
	id := c.Param("containerId")
	var cmd appContainer.WriteConfigCmd
	if err := c.ShouldBindJSON(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}
	if err := h.uc.WriteConfig(c.Request.Context(), hostID, id, cmd); err != nil {
		return apperr.MapDomainError(err, containerErrMappings)
	}
	response.OK(c, nil)
	return nil
}
