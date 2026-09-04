package http

import (
	"fmt"

	appSetting "ops-hub/internal/application/setting"
	"ops-hub/internal/infrastructure/jenkins"
	"ops-hub/pkg/apperr"
	"ops-hub/pkg/response"

	"github.com/gin-gonic/gin"
)

// JenkinsHandler Jenkins API 代理处理器
type JenkinsHandler struct {
	client    *jenkins.Client
	settingUC *appSetting.UseCase
}

func NewJenkinsHandler(client *jenkins.Client, settingUC *appSetting.UseCase) *JenkinsHandler {
	return &JenkinsHandler{client: client, settingUC: settingUC}
}

// getClient 从设置表加载最新的 Jenkins 配置，若有变更则更新 client
func (h *JenkinsHandler) getClient(c *gin.Context) *jenkins.Client {
	if h.settingUC == nil {
		return h.client
	}
	ctx := c.Request.Context()
	url, _ := h.settingUC.GetByKey(ctx, "jenkins.url")
	user, _ := h.settingUC.GetByKey(ctx, "jenkins.user")
	token, _ := h.settingUC.GetByKey(ctx, "jenkins.token")
	if url != "" {
		return jenkins.NewClient(url, user, token)
	}
	return h.client
}

func (h *JenkinsHandler) RegisterRoutes(api *gin.RouterGroup) {
	g := api.Group("/jenkins")
	{
		g.GET("/job-info", apperr.Handle(h.GetJobInfo))
		g.GET("/builds", apperr.Handle(h.GetRecentBuilds))
		g.GET("/build-info", apperr.Handle(h.GetBuildInfo))
		g.GET("/console", apperr.Handle(h.GetConsoleOutput))
	}
}

// GetJobInfo 获取 Jenkins Job 信息
// GET /api/v1/jenkins/job-info?job=folder/job-name
func (h *JenkinsHandler) GetJobInfo(c *gin.Context) error {
	jobPath := c.Query("job")
	if jobPath == "" {
		return apperr.BadRequest("参数 job 不能为空")
	}

	info, err := h.getClient(c).GetJobInfo(c.Request.Context(), jobPath)
	if err != nil {
		return apperr.Internal(err)
	}

	// 转换为前端友好的格式
	type ParamDTO struct {
		Name         string   `json:"name"`
		Type         string   `json:"type"`
		Description  string   `json:"description"`
		DefaultValue string   `json:"default_value"`
		Choices      []string `json:"choices,omitempty"`
	}

	type BuildDTO struct {
		Number int    `json:"number"`
		URL    string `json:"url"`
	}

	type JobInfoDTO struct {
		Description         string     `json:"description"`
		Buildable           bool       `json:"buildable"`
		FullName            string     `json:"full_name"`
		URL                 string     `json:"url"`
		LastBuild           *BuildDTO  `json:"last_build"`
		LastSuccessfulBuild *BuildDTO  `json:"last_successful_build"`
		Parameters          []ParamDTO `json:"parameters"`
	}

	dto := JobInfoDTO{
		Description: info.Description,
		Buildable:   info.Buildable,
		FullName:    info.FullName,
		URL:         info.URL,
	}

	if info.LastBuild != nil {
		dto.LastBuild = &BuildDTO{Number: info.LastBuild.Number, URL: info.LastBuild.URL}
	}
	if info.LastSuccessfulBuild != nil {
		dto.LastSuccessfulBuild = &BuildDTO{Number: info.LastSuccessfulBuild.Number, URL: info.LastSuccessfulBuild.URL}
	}

	for _, p := range info.ParameterDefinitions {
		pd := ParamDTO{
			Name:        p.Name,
			Type:        p.Type,
			Description: p.Description,
			Choices:     p.Choices,
		}
		if p.DefaultParameterValue != nil {
			pd.DefaultValue = fmt.Sprintf("%v", p.DefaultParameterValue.Value)
		}
		dto.Parameters = append(dto.Parameters, pd)
	}

	response.OK(c, dto)
	return nil
}

// GetRecentBuilds 获取最近构建历史
// GET /api/v1/jenkins/builds?job=folder/job-name&count=10
func (h *JenkinsHandler) GetRecentBuilds(c *gin.Context) error {
	jobPath := c.Query("job")
	if jobPath == "" {
		return apperr.BadRequest("参数 job 不能为空")
	}

	count := 10
	if c.Query("count") != "" {
		if _, err := fmt.Sscanf(c.Query("count"), "%d", &count); err != nil || count < 1 {
			count = 10
		}
		if count > 50 {
			count = 50
		}
	}

	builds, err := h.getClient(c).GetRecentBuilds(c.Request.Context(), jobPath, count)
	if err != nil {
		return apperr.Internal(err)
	}

	type BuildDTO struct {
		Number    int    `json:"number"`
		Result    string `json:"result"`
		Building  bool   `json:"building"`
		Timestamp int64  `json:"timestamp"`
		Duration  int64  `json:"duration"`
		URL       string `json:"url"`
	}

	dtos := make([]BuildDTO, len(builds))
	for i, b := range builds {
		dtos[i] = BuildDTO{
			Number:    b.Number,
			Result:    b.Result,
			Building:  b.Building,
			Timestamp: b.Timestamp,
			Duration:  b.Duration,
			URL:       b.URL,
		}
	}

	response.OK(c, dtos)
	return nil
}

// GetBuildInfo 获取单个构建详情
// GET /api/v1/jenkins/build-info?job=folder/job-name&number=123
func (h *JenkinsHandler) GetBuildInfo(c *gin.Context) error {
	jobPath := c.Query("job")
	if jobPath == "" {
		return apperr.BadRequest("参数 job 不能为空")
	}

	var buildNumber int
	if _, err := fmt.Sscanf(c.Query("number"), "%d", &buildNumber); err != nil || buildNumber < 1 {
		return apperr.BadRequest("参数 number 无效")
	}

	info, err := h.getClient(c).GetBuildInfo(c.Request.Context(), jobPath, buildNumber)
	if err != nil {
		return apperr.Internal(err)
	}

	response.OK(c, info)
	return nil
}

// GetConsoleOutput 获取构建日志
// GET /api/v1/jenkins/console?job=folder/job-name&number=123
func (h *JenkinsHandler) GetConsoleOutput(c *gin.Context) error {
	jobPath := c.Query("job")
	if jobPath == "" {
		return apperr.BadRequest("参数 job 不能为空")
	}

	var buildNumber int
	if _, err := fmt.Sscanf(c.Query("number"), "%d", &buildNumber); err != nil || buildNumber < 1 {
		return apperr.BadRequest("参数 number 无效")
	}

	output, err := h.getClient(c).GetConsoleOutput(c.Request.Context(), jobPath, buildNumber)
	if err != nil {
		return apperr.Internal(err)
	}

	response.OK(c, gin.H{"output": output})
	return nil
}
