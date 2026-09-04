package image_sync

import (
	"bytes"
	"context"
	"fmt"
	"log"
	appHost "ops-hub/internal/application/host"
	domain "ops-hub/internal/domain/image_sync"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type ExecuteCmd struct {
	SourceHostID string `json:"source_host_id" binding:"required"`
	TargetHostID string `json:"target_host_id" binding:"required"`
	Image        string `json:"image" binding:"required"`
	Mode         string `json:"mode" binding:"required"`
}

type RecordQueryCmd struct {
	SourceHostID string `form:"source_host_id"`
	TargetHostID string `form:"target_host_id"`
	Status       string `form:"status"`
	Page         int    `form:"page"`
	PageSize     int    `form:"page_size"`
}

type ExecuteResultDTO struct {
	Message  string `json:"message"`
	RecordID string `json:"record_id"`
}

type RecordDTO struct {
	ID             string  `json:"id"`
	SourceHostID   string  `json:"source_host_id"`
	SourceHostName string  `json:"source_host_name"`
	TargetHostID   string  `json:"target_host_id"`
	TargetHostName string  `json:"target_host_name"`
	Image          string  `json:"image"`
	Mode           string  `json:"mode"`
	Status         string  `json:"status"`
	SourceImageID  string  `json:"source_image_id"`
	TargetImageID  string  `json:"target_image_id"`
	ImageSize      int64   `json:"image_size"`
	Summary        string  `json:"summary"`
	Error          string  `json:"error"`
	StartedAt      string  `json:"started_at"`
	FinishedAt     *string `json:"finished_at"`
	Duration       int64   `json:"duration"`
	CreatedAt      string  `json:"created_at"`
}

type HostImageDTO struct {
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	ImageID    string `json:"image_id"`
	CreatedAt  string `json:"created_at"`
	Size       string `json:"size"`
	Name       string `json:"name"`
}

func toDTO(e *domain.Record) *RecordDTO {
	dto := &RecordDTO{
		ID:             e.ID,
		SourceHostID:   e.SourceHostID,
		SourceHostName: e.SourceHostName,
		TargetHostID:   e.TargetHostID,
		TargetHostName: e.TargetHostName,
		Image:          e.Image,
		Mode:           string(e.Mode),
		Status:         string(e.Status),
		SourceImageID:  e.SourceImageID,
		TargetImageID:  e.TargetImageID,
		ImageSize:      e.ImageSize,
		Summary:        e.Summary,
		Error:          e.Error,
		StartedAt:      e.StartedAt.Format(time.DateTime),
		Duration:       e.Duration,
		CreatedAt:      e.CreatedAt.Format(time.DateTime),
	}
	if e.FinishedAt != nil {
		s := e.FinishedAt.Format(time.DateTime)
		dto.FinishedAt = &s
	}
	return dto
}

type UseCase struct {
	recordRepo domain.RecordRepository
	hostUC     *appHost.UseCase
}

func NewUseCase(recordRepo domain.RecordRepository, hostUC *appHost.UseCase) *UseCase {
	return &UseCase{recordRepo: recordRepo, hostUC: hostUC}
}

func (uc *UseCase) Execute(ctx context.Context, cmd ExecuteCmd) (*ExecuteResultDTO, error) {
	mode := domain.SyncMode(cmd.Mode)
	if mode != domain.SyncModeSkipIfExists && mode != domain.SyncModeOverwrite {
		return nil, domain.ErrInvalidMode
	}
	image := strings.TrimSpace(cmd.Image)
	if !validImageName.MatchString(image) {
		return nil, domain.ErrInvalidImageName
	}
	if cmd.SourceHostID == cmd.TargetHostID {
		return nil, domain.ErrSameHost
	}

	sourceHost, err := uc.hostUC.Get(ctx, cmd.SourceHostID)
	if err != nil {
		return nil, err
	}
	targetHost, err := uc.hostUC.Get(ctx, cmd.TargetHostID)
	if err != nil {
		return nil, err
	}

	record := &domain.Record{
		SourceHostID:   cmd.SourceHostID,
		SourceHostName: sourceHost.Name,
		TargetHostID:   cmd.TargetHostID,
		TargetHostName: targetHost.Name,
		Image:          image,
		Mode:           mode,
		Status:         domain.SyncStatusRunning,
		StartedAt:      time.Now(),
		Summary:        "镜像同步已触发",
	}
	if err := uc.recordRepo.Save(ctx, record); err != nil {
		return nil, fmt.Errorf("保存镜像同步记录失败: %w", err)
	}

	go uc.executeRecord(record)

	return &ExecuteResultDTO{Message: "镜像同步任务已触发", RecordID: record.ID}, nil
}

func (uc *UseCase) ListRecords(ctx context.Context, cmd RecordQueryCmd) ([]*RecordDTO, int64, error) {
	if cmd.Page < 1 {
		cmd.Page = 1
	}
	if cmd.PageSize < 1 || cmd.PageSize > 100 {
		cmd.PageSize = 20
	}
	entities, total, err := uc.recordRepo.Find(ctx, domain.RecordQuery{
		SourceHostID: cmd.SourceHostID,
		TargetHostID: cmd.TargetHostID,
		Status:       cmd.Status,
		Page:         cmd.Page,
		PageSize:     cmd.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}
	dtos := make([]*RecordDTO, len(entities))
	for i, e := range entities {
		dtos[i] = toDTO(e)
	}
	return dtos, total, nil
}

func (uc *UseCase) GetRecord(ctx context.Context, id string) (*RecordDTO, error) {
	e, err := uc.recordRepo.FindByID(ctx, id)
	if err != nil {
		return nil, domain.ErrRecordNotFound
	}
	return toDTO(e), nil
}

func (uc *UseCase) ListHostImages(ctx context.Context, hostID string) ([]*HostImageDTO, error) {
	if err := uc.checkDocker(ctx, hostID, "主机"); err != nil {
		return nil, err
	}
	output, err := uc.exec(ctx, hostID, `docker images --format '{{.Repository}}\t{{.Tag}}\t{{.ID}}\t{{.CreatedAt}}\t{{.Size}}'`, 60*time.Second)
	if err != nil {
		return nil, fmt.Errorf("获取主机镜像列表失败: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	images := make([]*HostImageDTO, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 5)
		if len(parts) < 5 {
			continue
		}
		repository := strings.TrimSpace(parts[0])
		tag := strings.TrimSpace(parts[1])
		name := repository
		if tag != "" && tag != "<none>" {
			name = repository + ":" + tag
		}
		images = append(images, &HostImageDTO{
			Repository: repository,
			Tag:        tag,
			ImageID:    strings.TrimSpace(parts[2]),
			CreatedAt:  strings.TrimSpace(parts[3]),
			Size:       strings.TrimSpace(parts[4]),
			Name:       name,
		})
	}
	return images, nil
}

func (uc *UseCase) executeRecord(record *domain.Record) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if err := uc.checkDocker(ctx, record.SourceHostID, "源主机"); err != nil {
		uc.finishRecord(ctx, record, domain.SyncStatusFailed, err.Error())
		return
	}
	if err := uc.checkDocker(ctx, record.TargetHostID, "目标主机"); err != nil {
		uc.finishRecord(ctx, record, domain.SyncStatusFailed, err.Error())
		return
	}

	sourceInfo, exists, err := uc.inspectImage(ctx, record.SourceHostID, record.Image)
	if err != nil {
		uc.finishRecord(ctx, record, domain.SyncStatusFailed, "检查源镜像失败: "+err.Error())
		return
	}
	if !exists {
		uc.finishRecord(ctx, record, domain.SyncStatusFailed, domain.ErrSourceImageAbsent.Error())
		return
	}
	record.SourceImageID = sourceInfo.ID
	record.ImageSize = sourceInfo.Size

	targetInfo, targetExists, err := uc.inspectImage(ctx, record.TargetHostID, record.Image)
	if err != nil {
		uc.finishRecord(ctx, record, domain.SyncStatusFailed, "检查目标镜像失败: "+err.Error())
		return
	}
	if targetExists {
		record.TargetImageID = targetInfo.ID
		if record.Mode == domain.SyncModeSkipIfExists {
			uc.finishRecord(ctx, record, domain.SyncStatusSkipped, "目标主机已存在同名镜像，已跳过")
			return
		}
	}

	if err := uc.streamImage(ctx, record.SourceHostID, record.TargetHostID, record.Image); err != nil {
		uc.finishRecord(ctx, record, domain.SyncStatusFailed, "镜像同步失败: "+err.Error())
		return
	}

	loadedInfo, loadedExists, err := uc.inspectImage(ctx, record.TargetHostID, record.Image)
	if err == nil && loadedExists {
		record.TargetImageID = loadedInfo.ID
	}
	uc.finishRecord(ctx, record, domain.SyncStatusSuccess, "")
}

func (uc *UseCase) checkDocker(ctx context.Context, hostID, label string) error {
	output, err := uc.exec(ctx, hostID, `docker version --format '{{.Server.Version}}'`, 30*time.Second)
	if err != nil {
		return fmt.Errorf("%s Docker 不可用: %w", label, err)
	}
	if strings.TrimSpace(output) == "" {
		return fmt.Errorf("%s Docker 版本为空", label)
	}
	return nil
}

type imageInfo struct {
	ID   string
	Size int64
}

func (uc *UseCase) inspectImage(ctx context.Context, hostID, image string) (imageInfo, bool, error) {
	cmd := fmt.Sprintf("docker image inspect %s --format '{{.Id}}\\t{{.Size}}'", shellQuote(image))
	output, err := uc.exec(ctx, hostID, cmd, 30*time.Second)
	if err != nil {
		if strings.Contains(err.Error(), "No such image") || strings.Contains(err.Error(), "No such object") {
			return imageInfo{}, false, nil
		}
		return imageInfo{}, false, err
	}
	parts := strings.Split(strings.TrimSpace(output), "\t")
	if len(parts) < 2 {
		return imageInfo{ID: strings.TrimSpace(output)}, true, nil
	}
	size, _ := strconv.ParseInt(strings.TrimSpace(parts[1]), 10, 64)
	return imageInfo{ID: strings.TrimSpace(parts[0]), Size: size}, true, nil
}

func (uc *UseCase) streamImage(ctx context.Context, sourceHostID, targetHostID, image string) error {
	sourceClient, err := uc.hostUC.GetSSH(ctx, sourceHostID)
	if err != nil {
		return fmt.Errorf("连接源主机失败: %w", err)
	}
	targetClient, err := uc.hostUC.GetSSH(ctx, targetHostID)
	if err != nil {
		return fmt.Errorf("连接目标主机失败: %w", err)
	}

	sourceSession, err := sourceClient.NewSession()
	if err != nil {
		return fmt.Errorf("创建源主机 SSH Session 失败: %w", err)
	}
	defer sourceSession.Close()
	targetSession, err := targetClient.NewSession()
	if err != nil {
		return fmt.Errorf("创建目标主机 SSH Session 失败: %w", err)
	}
	defer targetSession.Close()

	sourceStdout, err := sourceSession.StdoutPipe()
	if err != nil {
		return fmt.Errorf("获取源镜像输出失败: %w", err)
	}
	var sourceStderr, targetStderr bytes.Buffer
	sourceSession.Stderr = &sourceStderr
	targetSession.Stderr = &targetStderr
	targetSession.Stdin = sourceStdout

	if err := targetSession.Start("docker load"); err != nil {
		return fmt.Errorf("启动目标 docker load 失败: %w", err)
	}
	if err := sourceSession.Start(fmt.Sprintf("docker save %s", shellQuote(image))); err != nil {
		return fmt.Errorf("启动源 docker save 失败: %w", err)
	}

	sourceDone := make(chan error, 1)
	targetDone := make(chan error, 1)
	go func() { sourceDone <- sourceSession.Wait() }()
	go func() { targetDone <- targetSession.Wait() }()

	var sourceErr, targetErr error
	for i := 0; i < 2; i++ {
		select {
		case sourceErr = <-sourceDone:
			if sourceErr != nil {
				_ = targetSession.Signal(ssh.SIGKILL)
			}
		case targetErr = <-targetDone:
			if targetErr != nil {
				_ = sourceSession.Signal(ssh.SIGKILL)
			}
		case <-ctx.Done():
			_ = sourceSession.Signal(ssh.SIGKILL)
			_ = targetSession.Signal(ssh.SIGKILL)
			return ctx.Err()
		}
	}
	if sourceErr != nil {
		return fmt.Errorf("源 docker save 失败: %w: %s", sourceErr, sourceStderr.String())
	}
	if targetErr != nil {
		return fmt.Errorf("目标 docker load 失败: %w: %s", targetErr, targetStderr.String())
	}
	return nil
}

func (uc *UseCase) exec(ctx context.Context, hostID, command string, timeout time.Duration) (string, error) {
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

func (uc *UseCase) finishRecord(ctx context.Context, record *domain.Record, status domain.SyncStatus, errMessage string) {
	now := time.Now()
	record.FinishedAt = &now
	record.Duration = int64(now.Sub(record.StartedAt).Seconds())
	record.Status = status
	record.Error = errMessage
	switch status {
	case domain.SyncStatusSuccess:
		record.Summary = fmt.Sprintf("镜像 %s 已同步到目标主机", record.Image)
	case domain.SyncStatusSkipped:
		record.Summary = "目标主机已存在镜像，跳过同步"
	default:
		record.Summary = errMessage
	}
	if err := uc.recordRepo.Update(ctx, record); err != nil {
		log.Printf("[ImageSync] update record failed: %v", err)
	}
}

var validImageName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/@:-]{0,499}$`)

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}
