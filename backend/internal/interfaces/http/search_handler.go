package http

import (
	appHost "ops-hub/internal/application/host"
	appService "ops-hub/internal/application/service"
	"ops-hub/pkg/apperr"
	"ops-hub/pkg/response"

	"github.com/gin-gonic/gin"
)

// SearchHandler 全局聚合搜索（命令面板）
//
// 设计原则：
//   - 只读、纯新增模块，不修改现有 service/host 模块代码。
//   - 复用现有 UseCase 的 List 方法，按 Keyword 过滤。
//   - 仅返回服务、主机两类，每类最多 8 条。
type SearchHandler struct {
	serviceUC *appService.UseCase
	hostUC    *appHost.UseCase
}

func NewSearchHandler(serviceUC *appService.UseCase, hostUC *appHost.UseCase) *SearchHandler {
	return &SearchHandler{serviceUC: serviceUC, hostUC: hostUC}
}

func (h *SearchHandler) RegisterRoutes(api *gin.RouterGroup) {
	api.GET("/search", apperr.Handle(h.Search))
}

// SearchHit 命令面板候选项
type SearchHit struct {
	Type     string `json:"type"` // service / host
	ID       string `json:"id"`
	Title    string `json:"title"`    // 主显示
	Subtitle string `json:"subtitle"` // 副显示
	URL      string `json:"url"`      // 前端跳转路径
}

type SearchResult struct {
	Query string      `json:"query"`
	Items []SearchHit `json:"items"`
}

// Search 全局搜索
// GET /api/v1/search?q=xxx
func (h *SearchHandler) Search(c *gin.Context) error {
	q := c.Query("q")
	if len(q) == 0 {
		response.OK(c, SearchResult{Query: "", Items: []SearchHit{}})
		return nil
	}

	const perType = 8
	hits := make([]SearchHit, 0, perType*2)

	// 服务
	if svcs, _, err := h.serviceUC.ListServices(c.Request.Context(), appService.ServiceQueryCmd{
		Keyword: q, Page: 1, PageSize: perType,
	}); err == nil {
		for _, s := range svcs {
			hits = append(hits, SearchHit{
				Type:     "service",
				ID:       s.ID,
				Title:    s.ServiceName,
				Subtitle: s.ServiceKey,
				URL:      "/services/" + s.ID,
			})
		}
	}

	// 主机
	if hosts, _, err := h.hostUC.List(c.Request.Context(), appHost.HostQueryCmd{
		Keyword: q, Page: 1, PageSize: perType,
	}); err == nil {
		for _, hh := range hosts {
			hits = append(hits, SearchHit{
				Type:     "host",
				ID:       hh.ID,
				Title:    hh.Name,
				Subtitle: hh.HostAddress,
				URL:      "/hosts",
			})
		}
	}

	response.OK(c, SearchResult{Query: q, Items: hits})
	return nil
}
