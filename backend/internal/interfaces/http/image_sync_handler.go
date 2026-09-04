package http

import (
	appImageSync "ops-hub/internal/application/image_sync"
	domainImageSync "ops-hub/internal/domain/image_sync"
	"ops-hub/pkg/apperr"
	"ops-hub/pkg/response"

	"github.com/gin-gonic/gin"
)

var imageSyncErrMappings = []apperr.DomainErrorMapping{
	{Target: domainImageSync.ErrRecordNotFound, AppErr: apperr.NotFound("镜像同步记录不存在")},
	{Target: domainImageSync.ErrInvalidMode, AppErr: apperr.BadRequest("无效的镜像同步模式")},
	{Target: domainImageSync.ErrInvalidImageName, AppErr: apperr.BadRequest("无效的镜像名称")},
	{Target: domainImageSync.ErrSameHost, AppErr: apperr.BadRequest("源主机和目标主机不能相同")},
	{Target: domainImageSync.ErrSourceImageAbsent, AppErr: apperr.BadRequest("源主机镜像不存在")},
}

type ImageSyncHandler struct {
	uc *appImageSync.UseCase
}

func NewImageSyncHandler(uc *appImageSync.UseCase) *ImageSyncHandler {
	return &ImageSyncHandler{uc: uc}
}

func (h *ImageSyncHandler) RegisterRoutes(api *gin.RouterGroup) {
	api.GET("/hosts/:id/images", apperr.Handle(h.ListHostImages))

	g := api.Group("/hosts/image-sync")
	g.POST("/execute", apperr.Handle(h.Execute))
	g.GET("/records", apperr.Handle(h.ListRecords))
	g.GET("/records/:id", apperr.Handle(h.GetRecord))
}

func (h *ImageSyncHandler) ListHostImages(c *gin.Context) error {
	dtos, err := h.uc.ListHostImages(c.Request.Context(), c.Param("id"))
	if err != nil {
		return apperr.MapDomainError(err, imageSyncErrMappings)
	}
	response.OK(c, dtos)
	return nil
}

func (h *ImageSyncHandler) Execute(c *gin.Context) error {
	var cmd appImageSync.ExecuteCmd
	if err := c.ShouldBindJSON(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}
	dto, err := h.uc.Execute(c.Request.Context(), cmd)
	if err != nil {
		return apperr.MapDomainError(err, imageSyncErrMappings)
	}
	response.OK(c, dto)
	return nil
}

func (h *ImageSyncHandler) ListRecords(c *gin.Context) error {
	var cmd appImageSync.RecordQueryCmd
	if err := c.ShouldBindQuery(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}
	dtos, total, err := h.uc.ListRecords(c.Request.Context(), cmd)
	if err != nil {
		return apperr.Internal(err)
	}
	response.OKPage(c, dtos, total, cmd.Page, cmd.PageSize)
	return nil
}

func (h *ImageSyncHandler) GetRecord(c *gin.Context) error {
	dto, err := h.uc.GetRecord(c.Request.Context(), c.Param("id"))
	if err != nil {
		return apperr.MapDomainError(err, imageSyncErrMappings)
	}
	response.OK(c, dto)
	return nil
}
