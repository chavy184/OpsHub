package container

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"ops-hub/internal/application/host"
	"ops-hub/internal/domain/container"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// ============================
// DTO
// ============================

type ContainerDTO struct {
	ID            string   `json:"id"`
	HostID        string   `json:"host_id"`
	ContainerID   string   `json:"container_id"`
	ContainerName string   `json:"container_name"`
	Image         string   `json:"image"`
	Status        string   `json:"status"`
	ConfigPaths   []string `json:"config_paths"`
	Description   string   `json:"description"`
	LastSyncedAt  *string  `json:"last_synced_at"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
}

type UpdateContainerCmd struct {
	ID          string   `json:"-"`
	ConfigPaths []string `json:"config_paths"`
	Description string   `json:"description"`
}

type ConfigFileDTO struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type WriteConfigCmd struct {
	Path    string `json:"path" binding:"required"`
	Content string `json:"content" binding:"required"`
	Restart bool   `json:"restart"`
}

type InspectDTO struct {
	Raw string `json:"raw"`
}

func toDTO(e *container.Container) *ContainerDTO {
	dto := &ContainerDTO{
		ID:            e.ID,
		HostID:        e.HostID,
		ContainerID:   e.ContainerID,
		ContainerName: e.ContainerName,
		Image:         e.Image,
		Status:        e.Status,
		ConfigPaths:   e.ConfigPaths,
		Description:   e.Description,
		CreatedAt:     e.CreatedAt.Format(time.DateTime),
		UpdatedAt:     e.UpdatedAt.Format(time.DateTime),
	}
	if e.LastSyncedAt != nil {
		s := e.LastSyncedAt.Format(time.DateTime)
		dto.LastSyncedAt = &s
	}
	if dto.ConfigPaths == nil {
		dto.ConfigPaths = []string{}
	}
	return dto
}

// ============================
// UseCase
// ============================

type UseCase struct {
	repo   container.ContainerRepository
	hostUC *host.UseCase
}

func NewUseCase(repo container.ContainerRepository, hostUC *host.UseCase) *UseCase {
	return &UseCase{repo: repo, hostUC: hostUC}
}

// SyncContainers 同步宿主机上的所有容器
func (uc *UseCase) SyncContainers(ctx context.Context, hostID string) ([]*ContainerDTO, error) {
	output, err := uc.execDocker(ctx, hostID, `docker ps -a --format '{{.ID}}\t{{.Names}}\t{{.Image}}\t{{.Status}}'`)
	if err != nil {
		return nil, fmt.Errorf("获取容器列表失败: %w", err)
	}

	now := time.Now()
	var syncedNames []string

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) < 4 {
			continue
		}

		containerID := parts[0]
		containerName := parts[1]
		image := parts[2]
		status := parseStatus(parts[3])
		syncedNames = append(syncedNames, containerName)

		existing, findErr := uc.repo.FindByHostAndName(ctx, hostID, containerName)
		if findErr != nil {
			// 不存在则新建
			entity := &container.Container{
				HostID:        hostID,
				ContainerID:   containerID,
				ContainerName: containerName,
				Image:         image,
				Status:        status,
				ConfigPaths:   []string{},
				LastSyncedAt:  &now,
			}
			if saveErr := uc.repo.Save(ctx, entity); saveErr != nil {
				return nil, saveErr
			}
		} else {
			// 已存在则更新状态
			existing.ContainerID = containerID
			existing.Image = image
			existing.Status = status
			existing.LastSyncedAt = &now
			if updateErr := uc.repo.Update(ctx, existing); updateErr != nil {
				return nil, updateErr
			}
		}
	}

	// 标记不再存在的容器为 removed
	if len(syncedNames) > 0 {
		_ = uc.repo.DeleteByHostIDNotIn(ctx, hostID, syncedNames)
	}

	return uc.List(ctx, hostID)
}

// List 获取容器列表
func (uc *UseCase) List(ctx context.Context, hostID string) ([]*ContainerDTO, error) {
	entities, err := uc.repo.FindByHostID(ctx, hostID)
	if err != nil {
		return nil, err
	}
	dtos := make([]*ContainerDTO, len(entities))
	for i, e := range entities {
		dtos[i] = toDTO(e)
	}
	return dtos, nil
}

// Update 更新容器配置路径或描述
func (uc *UseCase) Update(ctx context.Context, cmd UpdateContainerCmd) (*ContainerDTO, error) {
	e, err := uc.repo.FindByID(ctx, cmd.ID)
	if err != nil {
		return nil, container.ErrContainerNotFound
	}
	if cmd.ConfigPaths != nil {
		e.ConfigPaths = cmd.ConfigPaths
	}
	if cmd.Description != "" {
		e.Description = cmd.Description
	}
	if err := uc.repo.Update(ctx, e); err != nil {
		return nil, err
	}
	return toDTO(e), nil
}

// Start 启动容器
func (uc *UseCase) Start(ctx context.Context, hostID, id string) error {
	e, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return container.ErrContainerNotFound
	}
	if !validContainerName.MatchString(e.ContainerName) {
		return errors.New("容器名称不合法")
	}
	_, err = uc.execDocker(ctx, hostID, fmt.Sprintf("docker start %s", shellQuote(e.ContainerName)))
	if err == nil {
		e.Status = "running"
		_ = uc.repo.Update(ctx, e)
	}
	return err
}

// Stop 停止容器
func (uc *UseCase) Stop(ctx context.Context, hostID, id string) error {
	e, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return container.ErrContainerNotFound
	}
	if !validContainerName.MatchString(e.ContainerName) {
		return errors.New("容器名称不合法")
	}
	_, err = uc.execDocker(ctx, hostID, fmt.Sprintf("docker stop %s", shellQuote(e.ContainerName)))
	if err == nil {
		e.Status = "exited"
		_ = uc.repo.Update(ctx, e)
	}
	return err
}

// Restart 重启容器
func (uc *UseCase) Restart(ctx context.Context, hostID, id string) error {
	e, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return container.ErrContainerNotFound
	}
	if !validContainerName.MatchString(e.ContainerName) {
		return errors.New("容器名称不合法")
	}
	_, err = uc.execDockerWithTimeout(ctx, hostID, fmt.Sprintf("docker restart %s", shellQuote(e.ContainerName)), 60*time.Second)
	if err == nil {
		e.Status = "running"
		_ = uc.repo.Update(ctx, e)
	}
	return err
}

// Inspect 容器详情
func (uc *UseCase) Inspect(ctx context.Context, hostID, id string) (*InspectDTO, error) {
	e, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, container.ErrContainerNotFound
	}
	output, err := uc.execDocker(ctx, hostID, fmt.Sprintf("docker inspect %s", shellQuote(e.ContainerName)))
	if err != nil {
		return nil, err
	}
	return &InspectDTO{Raw: output}, nil
}

// ReadConfig 读取容器内配置文件
func (uc *UseCase) ReadConfig(ctx context.Context, hostID, id, filePath string) (*ConfigFileDTO, error) {
	if err := validateFilePath(filePath); err != nil {
		return nil, err
	}
	e, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, container.ErrContainerNotFound
	}
	cmd := fmt.Sprintf("docker exec %s cat %s", shellQuote(e.ContainerName), shellQuote(filePath))
	content, err := uc.execDockerWithTimeout(ctx, hostID, cmd, 15*time.Second)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}
	return &ConfigFileDTO{Path: filePath, Content: content}, nil
}

// WriteConfig 写入配置文件，可选重启容器
func (uc *UseCase) WriteConfig(ctx context.Context, hostID, id string, cmd WriteConfigCmd) error {
	if err := validateFilePath(cmd.Path); err != nil {
		return err
	}
	e, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return container.ErrContainerNotFound
	}
	if !validContainerName.MatchString(e.ContainerName) {
		return errors.New("容器名称不合法")
	}

	// 限制文件大小 1MB
	if len(cmd.Content) > 1024*1024 {
		return errors.New("配置文件超过 1MB 大小限制")
	}

	containerName := e.ContainerName

	// 1. 备份原配置
	backupCmd := fmt.Sprintf("docker exec %s cp %s %s.bak",
		shellQuote(containerName), shellQuote(cmd.Path), shellQuote(cmd.Path))
	if _, backupErr := uc.execDocker(ctx, hostID, backupCmd); backupErr != nil {
		return fmt.Errorf("备份配置文件失败: %w", backupErr)
	}

	// 2. 写入新配置（通过 stdin 管道）
	if writeErr := uc.writeFileViaStdin(ctx, hostID, containerName, cmd.Path, cmd.Content); writeErr != nil {
		return fmt.Errorf("写入配置文件失败: %w", writeErr)
	}

	// 3. 如果需要重启容器
	if cmd.Restart {
		restartCmd := fmt.Sprintf("docker restart %s", shellQuote(containerName))
		if _, restartErr := uc.execDockerWithTimeout(ctx, hostID, restartCmd, 60*time.Second); restartErr != nil {
			// 重启失败则回滚
			rollbackCmd := fmt.Sprintf("docker exec %s cp %s.bak %s",
				shellQuote(containerName), shellQuote(cmd.Path), shellQuote(cmd.Path))
			_, _ = uc.execDocker(ctx, hostID, rollbackCmd)
			return fmt.Errorf("重启容器失败，已回滚配置: %w", restartErr)
		}
		e.Status = "running"
		_ = uc.repo.Update(ctx, e)
	}

	return nil
}

// ============================
// 内部方法
// ============================

func (uc *UseCase) execDocker(ctx context.Context, hostID, command string) (string, error) {
	return uc.execDockerWithTimeout(ctx, hostID, command, 30*time.Second)
}

func (uc *UseCase) execDockerWithTimeout(ctx context.Context, hostID, command string, timeout time.Duration) (string, error) {
	client, err := uc.hostUC.GetSSH(ctx, hostID)
	if err != nil {
		return "", fmt.Errorf("SSH 连接失败: %w", err)
	}
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("创建 SSH Session 失败: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	done := make(chan error, 1)
	go func() { done <- session.Run(command) }()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case err := <-done:
		if err != nil {
			return "", fmt.Errorf("%s: %s", err, stderr.String())
		}
		return stdout.String(), nil
	case <-timer.C:
		_ = session.Signal(ssh.SIGKILL)
		return "", fmt.Errorf("命令执行超时")
	case <-ctx.Done():
		_ = session.Signal(ssh.SIGKILL)
		return "", ctx.Err()
	}
}

func (uc *UseCase) writeFileViaStdin(ctx context.Context, hostID, containerName, filePath, content string) error {
	client, err := uc.hostUC.GetSSH(ctx, hostID)
	if err != nil {
		return err
	}
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()

	session.Stdin = strings.NewReader(content)
	cmd := fmt.Sprintf("docker exec -i %s tee %s > /dev/null",
		shellQuote(containerName), shellQuote(filePath))
	return session.Run(cmd)
}

// ============================
// 安全辅助函数
// ============================

var validContainerName = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

func validateFilePath(path string) error {
	if strings.Contains(path, "..") {
		return container.ErrInvalidFilePath
	}
	if !strings.HasPrefix(path, "/") {
		return container.ErrInvalidFilePath
	}
	return nil
}

func parseStatus(rawStatus string) string {
	lower := strings.ToLower(rawStatus)
	switch {
	case strings.HasPrefix(lower, "up"):
		return "running"
	case strings.Contains(lower, "paused"):
		return "paused"
	case strings.Contains(lower, "restarting"):
		return "restarting"
	default:
		return "exited"
	}
}
