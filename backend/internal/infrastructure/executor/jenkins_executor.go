package executor

import (
	"context"
	"encoding/json"
	"fmt"
	"ops-hub/internal/domain/release"
	"ops-hub/internal/domain/setting"
	"ops-hub/internal/infrastructure/jenkins"
	"time"
)

// JenkinsExecutor 通过 Jenkins API 执行构建
type JenkinsExecutor struct {
	client      *jenkins.Client
	settingRepo setting.SystemSettingRepository
}

func NewJenkinsExecutor(client *jenkins.Client) *JenkinsExecutor {
	return &JenkinsExecutor{client: client}
}

// SetSettingRepo 允许从设置表动态读取 Jenkins 配置
func (e *JenkinsExecutor) SetSettingRepo(repo setting.SystemSettingRepository) {
	e.settingRepo = repo
}

// getClient 优先从设置表读取最新配置
func (e *JenkinsExecutor) getClient(ctx context.Context) *jenkins.Client {
	if e.settingRepo == nil {
		return e.client
	}
	urlSetting, err1 := e.settingRepo.FindByKey(ctx, "jenkins.url")
	userSetting, err2 := e.settingRepo.FindByKey(ctx, "jenkins.user")
	tokenSetting, err3 := e.settingRepo.FindByKey(ctx, "jenkins.token")
	if err1 == nil && urlSetting.Value != "" {
		user := ""
		token := ""
		if err2 == nil {
			user = userSetting.Value
		}
		if err3 == nil {
			token = tokenSetting.Value
		}
		return jenkins.NewClient(urlSetting.Value, user, token)
	}
	return e.client
}

func (e *JenkinsExecutor) Execute(ctx context.Context, task *release.ExecutionTask) (*release.ExecutionResult, error) {
	start := time.Now()
	client := e.getClient(ctx)

	if task.Scripts == nil || task.Scripts["jenkins_job"] == "" {
		return &release.ExecutionResult{
			Success: false,
			Error:   "未配置 Jenkins Job 路径",
			Elapsed: time.Since(start),
		}, nil
	}

	jobPath := task.Scripts["jenkins_job"]

	// 解析 Jenkins 参数
	params := make(map[string]string)
	if paramsJSON, ok := task.Scripts["jenkins_params"]; ok && paramsJSON != "" {
		_ = json.Unmarshal([]byte(paramsJSON), &params)
	}

	// 1. 触发构建
	queueURL, err := client.TriggerBuild(ctx, jobPath, params)
	if err != nil {
		return &release.ExecutionResult{
			Success: false,
			Error:   fmt.Sprintf("触发 Jenkins 构建失败: %v", err),
			Elapsed: time.Since(start),
		}, nil
	}

	// 2. 等待队列分配 build number（最多等 120 秒）
	var buildNumber int
	if queueURL != "" {
		buildNumber, err = e.waitForBuildNumber(ctx, client, queueURL, 120*time.Second)
		if err != nil {
			return &release.ExecutionResult{
				Success: false,
				Output:  fmt.Sprintf("队列 URL: %s", queueURL),
				Error:   fmt.Sprintf("等待构建号分配失败: %v", err),
				Elapsed: time.Since(start),
			}, nil
		}
	}

	// 3. 轮询构建状态（最多等 30 分钟），期间实时推送日志
	buildInfo, err := e.waitForBuildComplete(ctx, client, jobPath, buildNumber, 30*time.Minute, task.OnOutput)
	if err != nil {
		return &release.ExecutionResult{
			Success: false,
			Output:  fmt.Sprintf("Build #%d", buildNumber),
			Error:   fmt.Sprintf("等待构建完成失败: %v", err),
			Elapsed: time.Since(start),
		}, nil
	}

	// 4. 获取最终完整 console output
	consoleOutput, _ := client.GetConsoleOutput(ctx, jobPath, buildNumber)

	success := buildInfo.Result == "SUCCESS"
	result := &release.ExecutionResult{
		Success:     success,
		Output:      consoleOutput,
		Elapsed:     time.Since(start),
		BuildNumber: buildNumber,
	}
	if !success {
		result.Error = fmt.Sprintf("Jenkins 构建 #%d 结果: %s", buildNumber, buildInfo.Result)
	}

	return result, nil
}

func (e *JenkinsExecutor) HealthCheck(ctx context.Context, endpoint string) (*release.HealthResult, error) {
	// Jenkins 模式不需要 OpsHub 层面的健康检查
	return &release.HealthResult{
		Healthy: true,
		Message: "Jenkins 模式跳过健康检查",
	}, nil
}

// GetBuildNumber 获取当前构建的 build number（用于回填到 ReleaseRecord）
func (e *JenkinsExecutor) GetBuildNumber() int {
	return 0 // 由 Execute 方法内部处理
}

// waitForBuildNumber 等待 queue item 分配 build number
func (e *JenkinsExecutor) waitForBuildNumber(ctx context.Context, client *jenkins.Client, queueURL string, timeout time.Duration) (int, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		item, err := client.GetQueueItem(ctx, queueURL)
		if err != nil {
			// queue item 可能已过期，忽略继续
			time.Sleep(2 * time.Second)
			continue
		}

		if item.Executable != nil && item.Executable.Number > 0 {
			return item.Executable.Number, nil
		}

		time.Sleep(2 * time.Second)
	}

	return 0, fmt.Errorf("等待构建号超时 (%v)", timeout)
}

// waitForBuildComplete 轮询等待构建完成，期间定期推送日志
func (e *JenkinsExecutor) waitForBuildComplete(ctx context.Context, client *jenkins.Client, jobPath string, buildNumber int, timeout time.Duration, onOutput func(string)) (*jenkins.BuildInfo, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		info, err := client.GetBuildInfo(ctx, jobPath, buildNumber)
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}

		// 实时推送日志
		if onOutput != nil {
			if output, outErr := client.GetConsoleOutput(ctx, jobPath, buildNumber); outErr == nil {
				onOutput(output)
			}
		}

		if !info.Building {
			return info, nil
		}

		time.Sleep(5 * time.Second)
	}

	return nil, fmt.Errorf("等待构建完成超时 (%v)", timeout)
}
