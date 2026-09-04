package http

import (
	appCredential "ops-hub/internal/application/credential"
	domainCredential "ops-hub/internal/domain/credential"
	"ops-hub/pkg/apperr"
	"ops-hub/pkg/response"

	"github.com/gin-gonic/gin"
)

var credentialErrMappings = []apperr.DomainErrorMapping{
	{Target: domainCredential.ErrCredentialNotFound, AppErr: apperr.NotFound("凭证不存在")},
	{Target: domainCredential.ErrCredentialNameExists, AppErr: apperr.Conflict("凭证名称已存在")},
}

// CredentialHandler 凭证管理 Handler
type CredentialHandler struct {
	uc *appCredential.UseCase
}

func NewCredentialHandler(uc *appCredential.UseCase) *CredentialHandler {
	return &CredentialHandler{uc: uc}
}

func (h *CredentialHandler) RegisterRoutes(api *gin.RouterGroup) {
	g := api.Group("/credentials")
	g.GET("", apperr.Handle(h.List))
	g.POST("", apperr.Handle(h.Create))
	g.GET("/:id", apperr.Handle(h.Get))
	g.PUT("/:id", apperr.Handle(h.Update))
	g.DELETE("/:id", apperr.Handle(h.Delete))
}

func (h *CredentialHandler) List(c *gin.Context) error {
	var cmd appCredential.CredentialQueryCmd
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

func (h *CredentialHandler) Create(c *gin.Context) error {
	var cmd appCredential.CreateCredentialCmd
	if err := c.ShouldBindJSON(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}
	dto, err := h.uc.Create(c.Request.Context(), cmd)
	if err != nil {
		return apperr.MapDomainError(err, credentialErrMappings)
	}
	response.Created(c, dto)
	return nil
}

func (h *CredentialHandler) Get(c *gin.Context) error {
	dto, err := h.uc.Get(c.Request.Context(), c.Param("id"))
	if err != nil {
		return apperr.MapDomainError(err, credentialErrMappings)
	}
	response.OK(c, dto)
	return nil
}

func (h *CredentialHandler) Update(c *gin.Context) error {
	var cmd appCredential.UpdateCredentialCmd
	if err := c.ShouldBindJSON(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}
	cmd.ID = c.Param("id")
	dto, err := h.uc.Update(c.Request.Context(), cmd)
	if err != nil {
		return apperr.MapDomainError(err, credentialErrMappings)
	}
	response.OK(c, dto)
	return nil
}

func (h *CredentialHandler) Delete(c *gin.Context) error {
	if err := h.uc.Delete(c.Request.Context(), c.Param("id")); err != nil {
		return apperr.MapDomainError(err, credentialErrMappings)
	}
	response.NoContent(c)
	return nil
}
