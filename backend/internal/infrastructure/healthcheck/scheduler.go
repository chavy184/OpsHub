package healthcheck

import (
	"context"
	"fmt"
	"net/http"
	"ops-hub/internal/domain/service"
	"strings"
	"sync"
	"time"
)

// Scheduler 健康检查调度器
// 启动后台 goroutine，定时扫描并检查所有启用了健康检查的环境
type Scheduler struct {
	envRepo     service.ServiceEnvRepository
	minInterval time.Duration
	stopCh      chan struct{}
	wg          sync.WaitGroup
}

// NewScheduler 创建调度器
func NewScheduler(envRepo service.ServiceEnvRepository) *Scheduler {
	return &Scheduler{
		envRepo:     envRepo,
		minInterval: 15 * time.Second,
		stopCh:      make(chan struct{}),
	}
}

// Start 启动健康检查调度
func (s *Scheduler) Start() {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(s.minInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.runChecks()
			case <-s.stopCh:
				return
			}
		}
	}()
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	close(s.stopCh)
	s.wg.Wait()
}

// runChecks 执行一轮健康检查
func (s *Scheduler) runChecks() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	envs, err := s.envRepo.FindAllHealthCheckEnabled(ctx)
	if err != nil {
		return
	}

	for _, env := range envs {
		// 未到检查间隔则跳过
		interval := env.HealthCheckInterval
		if interval <= 0 {
			interval = 60
		}
		if env.HealthLastCheckedAt != nil &&
			time.Since(*env.HealthLastCheckedAt) < time.Duration(interval)*time.Second {
			continue
		}
		// 无健康检查 URL 则跳过
		if env.HealthcheckURL == "" {
			continue
		}
		e := env
		go s.checkOne(e)
	}
}

// checkOne 检查单个环境
func (s *Scheduler) checkOne(env *service.ServiceEnv) {
	timeout := env.HealthCheckTimeout
	if timeout <= 0 {
		timeout = 10
	}
	successCodes := env.HealthCheckSuccessCodes
	if successCodes == "" {
		successCodes = "200"
	}

	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	now := time.Now()

	resp, err := client.Get(env.HealthcheckURL)
	if err != nil {
		env.HealthStatus = "unreachable"
		env.HealthLastMessage = err.Error()
	} else {
		resp.Body.Close()
		if isSuccessCode(resp.StatusCode, successCodes) {
			env.HealthStatus = "healthy"
			env.HealthLastMessage = fmt.Sprintf("HTTP %d", resp.StatusCode)
		} else {
			env.HealthStatus = "unhealthy"
			env.HealthLastMessage = fmt.Sprintf("HTTP %d (expected: %s)", resp.StatusCode, successCodes)
		}
	}

	env.HealthLastCheckedAt = &now

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.envRepo.UpdateHealthStatus(ctx, env)
}

// isSuccessCode 判断状态码是否符合预期
// successCodes 格式: "200" 或 "200,201,204" 或 "2xx"
func isSuccessCode(code int, successCodes string) bool {
	parts := strings.Split(successCodes, ",")
	codeStr := fmt.Sprintf("%d", code)
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == codeStr {
			return true
		}
		// 支持 2xx 格式
		if len(part) == 3 && part[1] == 'x' && part[2] == 'x' {
			if codeStr[0] == part[0] {
				return true
			}
		}
	}
	return false
}
