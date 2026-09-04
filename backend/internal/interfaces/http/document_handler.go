package http

import (
	"path/filepath"

	appDocument "ops-hub/internal/application/document"
	"ops-hub/pkg/apperr"
	"ops-hub/pkg/response"

	"github.com/gin-gonic/gin"
)

// document 领域错误 → AppError 映射
var documentErrMappings = []apperr.DomainErrorMapping{
	{Target: appDocument.ErrInvalidDocType, AppErr: apperr.BadRequest("无效的文档类型")},
	{Target: appDocument.ErrInvalidPath, AppErr: apperr.BadRequest("路径非法")},
	{Target: appDocument.ErrFileExists, AppErr: apperr.Conflict("同名文件已存在")},
	{Target: appDocument.ErrFileNotFound, AppErr: apperr.NotFound("文件不存在")},
	{Target: appDocument.ErrFileTooLarge, AppErr: &apperr.Error{Code: 4001, HTTPStatus: 413, Message: "文件过大"}},
	{Target: appDocument.ErrFileTypeDenied, AppErr: apperr.BadRequest("文件类型不允许")},
	{Target: appDocument.ErrNotTextFile, AppErr: &apperr.Error{Code: 4002, HTTPStatus: 415, Message: "非文本文件，无法预览"}},
	{Target: appDocument.ErrTypeExists, AppErr: apperr.Conflict("文档类型已存在")},
}

type DocumentHandler struct {
	uc *appDocument.UseCase
}

func NewDocumentHandler(uc *appDocument.UseCase) *DocumentHandler {
	return &DocumentHandler{uc: uc}
}

func (h *DocumentHandler) RegisterRoutes(api *gin.RouterGroup) {
	g := api.Group("/documents")
	{
		g.GET("/types", apperr.Handle(h.listTypes))
		g.POST("/types", apperr.Handle(h.createType))
		g.DELETE("/types/:name", apperr.Handle(h.deleteType))
		g.GET("/tree", apperr.Handle(h.getTree))
		g.POST("/upload", apperr.Handle(h.upload))
		g.POST("/mkdir", apperr.Handle(h.mkdir))
		g.DELETE("", apperr.Handle(h.delete))
		g.GET("/content", apperr.Handle(h.getContent))
		g.GET("/download", apperr.Handle(h.download))
	}
}

func (h *DocumentHandler) listTypes(c *gin.Context) error {
	types, err := h.uc.ListTypes(c.Request.Context())
	if err != nil {
		return apperr.MapDomainError(err, documentErrMappings)
	}
	if types == nil {
		types = []appDocument.DocCategory{}
	}
	response.OK(c, types)
	return nil
}

type createTypeReq struct {
	Name string `json:"name" binding:"required"`
}

func (h *DocumentHandler) createType(c *gin.Context) error {
	var req createTypeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return apperr.BadRequest(err.Error())
	}
	cat, err := h.uc.CreateType(c.Request.Context(), req.Name)
	if err != nil {
		return apperr.MapDomainError(err, documentErrMappings)
	}
	response.OK(c, cat)
	return nil
}

func (h *DocumentHandler) deleteType(c *gin.Context) error {
	name := c.Param("name")
	if name == "" {
		return apperr.BadRequest("类型名称不能为空")
	}
	if err := h.uc.DeleteType(c.Request.Context(), name); err != nil {
		return apperr.MapDomainError(err, documentErrMappings)
	}
	response.OK(c, nil)
	return nil
}

func (h *DocumentHandler) getTree(c *gin.Context) error {
	docType := appDocument.DocType(c.Query("type"))
	nodes, err := h.uc.GetTree(c.Request.Context(), docType)
	if err != nil {
		return apperr.MapDomainError(err, documentErrMappings)
	}
	if nodes == nil {
		nodes = []*appDocument.FileNode{}
	}
	response.OK(c, nodes)
	return nil
}

func (h *DocumentHandler) upload(c *gin.Context) error {
	docType := appDocument.DocType(c.PostForm("type"))
	dir := c.PostForm("path")
	overwrite := c.PostForm("overwrite") == "true"

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		return apperr.BadRequest("请选择要上传的文件")
	}
	defer file.Close()

	node, err := h.uc.Upload(c.Request.Context(), docType, dir, header.Filename, file, overwrite)
	if err != nil {
		return apperr.MapDomainError(err, documentErrMappings)
	}
	response.OK(c, node)
	return nil
}

type mkdirReq struct {
	Type string `json:"type" binding:"required"`
	Path string `json:"path" binding:"required"`
}

func (h *DocumentHandler) mkdir(c *gin.Context) error {
	var req mkdirReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return apperr.BadRequest(err.Error())
	}

	node, err := h.uc.Mkdir(c.Request.Context(), appDocument.DocType(req.Type), req.Path)
	if err != nil {
		return apperr.MapDomainError(err, documentErrMappings)
	}
	response.OK(c, node)
	return nil
}

func (h *DocumentHandler) delete(c *gin.Context) error {
	docType := appDocument.DocType(c.Query("type"))
	path := c.Query("path")
	if path == "" {
		return apperr.BadRequest("path 参数不能为空")
	}

	if err := h.uc.Delete(c.Request.Context(), docType, path); err != nil {
		return apperr.MapDomainError(err, documentErrMappings)
	}
	response.OK(c, nil)
	return nil
}

func (h *DocumentHandler) getContent(c *gin.Context) error {
	docType := appDocument.DocType(c.Query("type"))
	path := c.Query("path")
	if path == "" {
		return apperr.BadRequest("path 参数不能为空")
	}

	content, err := h.uc.GetContent(c.Request.Context(), docType, path)
	if err != nil {
		return apperr.MapDomainError(err, documentErrMappings)
	}
	response.OK(c, content)
	return nil
}

func (h *DocumentHandler) download(c *gin.Context) error {
	docType := appDocument.DocType(c.Query("type"))
	path := c.Query("path")
	if path == "" {
		return apperr.BadRequest("path 参数不能为空")
	}

	fullPath, err := h.uc.GetFilePath(c.Request.Context(), docType, path)
	if err != nil {
		return apperr.MapDomainError(err, documentErrMappings)
	}

	filename := filepath.Base(fullPath)
	c.FileAttachment(fullPath, filename)
	return nil
}
