package http

import (
	appSetting "ops-hub/internal/application/setting"
	"ops-hub/internal/domain/setting"
	"ops-hub/pkg/apperr"
	"ops-hub/pkg/response"

	"github.com/gin-gonic/gin"
)

// setting 领域错误 → AppError 映射
var settingErrMappings = []apperr.DomainErrorMapping{
	{Target: setting.ErrSettingNotFound, AppErr: apperr.NotFound("设置项不存在")},
}

// SettingHandler 系统设置 Handler
type SettingHandler struct {
	uc *appSetting.UseCase
}

func NewSettingHandler(uc *appSetting.UseCase) *SettingHandler {
	return &SettingHandler{uc: uc}
}

func (h *SettingHandler) RegisterRoutes(api *gin.RouterGroup) {
	g := api.Group("/settings")
	g.GET("", apperr.Handle(h.GetSettings))
	g.PUT("", apperr.Handle(h.UpdateSetting))
	g.PATCH("/batch", apperr.Handle(h.BatchUpdate))
}

// GetSettings godoc
// GET /api/v1/settings?category=xxx
func (h *SettingHandler) GetSettings(c *gin.Context) error {
	category := c.Query("category")
	var (
		dtos []*appSetting.SettingDTO
		err  error
	)
	if category != "" {
		dtos, err = h.uc.GetByCategory(c.Request.Context(), category)
	} else {
		dtos, err = h.uc.GetAll(c.Request.Context())
	}
	if err != nil {
		return apperr.MapDomainError(err, settingErrMappings)
	}
	response.OK(c, dtos)
	return nil
}

// UpdateSetting godoc
// PUT /api/v1/settings
func (h *SettingHandler) UpdateSetting(c *gin.Context) error {
	var cmd appSetting.UpdateSettingCmd
	if err := c.ShouldBindJSON(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}
	dto, err := h.uc.Update(c.Request.Context(), cmd)
	if err != nil {
		return apperr.MapDomainError(err, settingErrMappings)
	}
	response.OK(c, dto)
	return nil
}

// BatchUpdate godoc
// PATCH /api/v1/settings/batch
func (h *SettingHandler) BatchUpdate(c *gin.Context) error {
	var cmd appSetting.BatchUpdateCmd
	if err := c.ShouldBindJSON(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}
	if err := h.uc.BatchUpdate(c.Request.Context(), cmd); err != nil {
		return apperr.MapDomainError(err, settingErrMappings)
	}
	response.OK(c, gin.H{"message": "批量更新成功"})
	return nil
}
