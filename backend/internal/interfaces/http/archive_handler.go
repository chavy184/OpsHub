package http

import (
	appArchive "ops-hub/internal/application/archive"
	domainArchive "ops-hub/internal/domain/archive"
	"ops-hub/pkg/apperr"
	"ops-hub/pkg/response"

	"github.com/gin-gonic/gin"
)

var archiveErrMappings = []apperr.DomainErrorMapping{
	{Target: domainArchive.ErrUnsupportedFile, AppErr: apperr.BadRequest("暂不支持该文件类型")},
	{Target: domainArchive.ErrInvalidArchive, AppErr: apperr.BadRequest("压缩包格式无效")},
	{Target: domainArchive.ErrFileTooLarge, AppErr: &apperr.Error{Code: 4003, HTTPStatus: 413, Message: "文件过大"}},
}

type ArchiveHandler struct {
	uc *appArchive.UseCase
}

func NewArchiveHandler(uc *appArchive.UseCase) *ArchiveHandler {
	return &ArchiveHandler{uc: uc}
}

func (h *ArchiveHandler) RegisterRoutes(api *gin.RouterGroup) {
	g := api.Group("/archives")
	{
		g.POST("/analyze", apperr.Handle(h.analyze))
	}
}

func (h *ArchiveHandler) analyze(c *gin.Context) error {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		return apperr.BadRequest("请选择要解析的文件")
	}
	defer file.Close()

	result, err := h.uc.Analyze(c.Request.Context(), header.Filename, file, header.Size)
	if err != nil {
		return apperr.MapDomainError(err, archiveErrMappings)
	}
	response.OK(c, result)
	return nil
}
