package http

import (
	"errors"
	"net/http"
	appRelease "ops-hub/internal/application/release"
	"ops-hub/internal/domain/release"
	"ops-hub/pkg/apperr"
	"ops-hub/pkg/response"

	"github.com/gin-gonic/gin"
)

// release 领域错误 → AppError 映射
var releaseErrMappings = []apperr.DomainErrorMapping{
	{Target: release.ErrReleaseNotFound, AppErr: apperr.NotFound("发布记录不存在")},
	{Target: release.ErrInvalidReleaseState, AppErr: apperr.BadRequest("发布状态不允许该操作")},
	{Target: release.ErrIdempotencyConflict, AppErr: apperr.Conflict("幂等键冲突，重复发布")},
	{Target: release.ErrNoPreviousVersion, AppErr: apperr.BadRequest("无可回滚的历史版本")},
	{Target: release.ErrReleaseAlreadyActive, AppErr: apperr.Conflict("该服务环境已有进行中的发布")},
}

type ReleaseHandler struct {
	uc *appRelease.UseCase
}

func NewReleaseHandler(uc *appRelease.UseCase) *ReleaseHandler {
	return &ReleaseHandler{uc: uc}
}

func (h *ReleaseHandler) RegisterRoutes(api *gin.RouterGroup) {
	g := api.Group("/releases")
	{
		g.GET("", apperr.Handle(h.ListReleases))
		g.POST("", apperr.Handle(h.CreateRelease))
		g.GET("/:id", apperr.Handle(h.GetRelease))
		g.POST("/:id/execute", apperr.Handle(h.ExecuteRelease))
		g.POST("/:id/rollback", apperr.Handle(h.RollbackRelease))
		g.DELETE("/:id", apperr.Handle(h.DeleteRelease))
		g.GET("/:id/steps", apperr.Handle(h.GetSteps))
	}
}

// ListReleases 发布记录列表
// GET /api/v1/releases?service_id=xxx&status=pending&page=1&page_size=20
func (h *ReleaseHandler) ListReleases(c *gin.Context) error {
	var q appRelease.ReleaseQueryCmd
	if err := c.ShouldBindQuery(&q); err != nil {
		return apperr.BadRequest(err.Error())
	}

	dtos, total, err := h.uc.ListReleases(c.Request.Context(), q)
	if err != nil {
		return apperr.MapDomainError(err, releaseErrMappings)
	}

	response.OKPage(c, dtos, total, q.Page, q.PageSize)
	return nil
}

// GetRelease 发布详情
// GET /api/v1/releases/:id
func (h *ReleaseHandler) GetRelease(c *gin.Context) error {
	id := c.Param("id")

	dto, err := h.uc.GetRelease(c.Request.Context(), id)
	if err != nil {
		return apperr.MapDomainError(err, releaseErrMappings)
	}

	response.OK(c, dto)
	return nil
}

// CreateRelease 创建发布单
// POST /api/v1/releases
func (h *ReleaseHandler) CreateRelease(c *gin.Context) error {
	var cmd appRelease.CreateReleaseCmd
	if err := c.ShouldBindJSON(&cmd); err != nil {
		return apperr.BadRequest(err.Error())
	}

	// TODO: 从 JWT token 提取 operator_id
	// cmd.OperatorID = getOperatorID(c)

	dto, err := h.uc.CreateRelease(c.Request.Context(), cmd)
	if err != nil {
		if errors.Is(err, appRelease.ErrProdTargetBlocked) {
			return &apperr.Error{
				Code:       apperr.Code(appRelease.ErrCodeProdTargetBlocked),
				HTTPStatus: http.StatusBadRequest,
				Message:    err.Error(),
				Cause:      err,
			}
		}
		return apperr.MapDomainError(err, releaseErrMappings)
	}

	response.Created(c, dto)
	return nil
}

// ExecuteRelease 执行发布
// POST /api/v1/releases/:id/execute
func (h *ReleaseHandler) ExecuteRelease(c *gin.Context) error {
	id := c.Param("id")

	dto, err := h.uc.ExecuteRelease(c.Request.Context(), id)
	if err != nil {
		return apperr.MapDomainError(err, releaseErrMappings)
	}

	response.OK(c, dto)
	return nil
}

// RollbackRelease 回滚发布
// POST /api/v1/releases/:id/rollback
func (h *ReleaseHandler) RollbackRelease(c *gin.Context) error {
	id := c.Param("id")

	// TODO: 从 JWT token 提取 operator_id
	operatorID := ""

	dto, err := h.uc.RollbackRelease(c.Request.Context(), id, operatorID)
	if err != nil {
		return apperr.MapDomainError(err, releaseErrMappings)
	}

	response.Created(c, dto)
	return nil
}

// GetSteps 获取发布步骤日志
// GET /api/v1/releases/:id/steps
func (h *ReleaseHandler) GetSteps(c *gin.Context) error {
	id := c.Param("id")
	dtos, err := h.uc.GetReleaseSteps(c.Request.Context(), id)
	if err != nil {
		return apperr.MapDomainError(err, releaseErrMappings)
	}
	response.OK(c, dtos)
	return nil
}

// DeleteRelease 删除发布记录
// DELETE /api/v1/releases/:id
func (h *ReleaseHandler) DeleteRelease(c *gin.Context) error {
	id := c.Param("id")
	if err := h.uc.DeleteRelease(c.Request.Context(), id); err != nil {
		return apperr.MapDomainError(err, releaseErrMappings)
	}
	response.OK(c, nil)
	return nil
}
