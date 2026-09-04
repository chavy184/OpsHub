package host

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"ops-hub/internal/domain/credential"
	"ops-hub/internal/domain/host"
	"ops-hub/internal/infrastructure/crypto"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// ============================
// DTO
// ============================

type CreateHostCmd struct {
	Name         string `json:"name" binding:"required"`
	HostAddress  string `json:"host_address" binding:"required"`
	SSHPort      int    `json:"ssh_port"`
	Username     string `json:"username"`
	CredentialID string `json:"credential_id"`
	IsProd       bool   `json:"is_prod"`
	Labels       string `json:"labels"`
	Description  string `json:"description"`
}

type UpdateHostCmd struct {
	ID           string `json:"-"`
	Name         string `json:"name"`
	HostAddress  string `json:"host_address"`
	SSHPort      int    `json:"ssh_port"`
	Username     string `json:"username"`
	CredentialID string `json:"credential_id"`
	IsProd       *bool  `json:"is_prod"`
	Labels       string `json:"labels"`
	Description  string `json:"description"`
}

type HostQueryCmd struct {
	Keyword  string `form:"keyword"`
	Page     int    `form:"page"`
	PageSize int    `form:"page_size"`
}

type HostDTO struct {
	ID            string  `json:"id"`
	Name          string  `json:"name"`
	HostAddress   string  `json:"host_address"`
	SSHPort       int     `json:"ssh_port"`
	Username      string  `json:"username"`
	CredentialID  string  `json:"credential_id"`
	IsProd        bool    `json:"is_prod"`
	Labels        string  `json:"labels"`
	OsInfo        string  `json:"os_info"`
	AgentStatus   string  `json:"agent_status"`
	LastHeartbeat *string `json:"last_heartbeat"`
	Description   string  `json:"description"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type TestConnectionResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	OsInfo  string `json:"os_info"`
}

func toDTO(e *host.Host) *HostDTO {
	dto := &HostDTO{
		ID:           e.ID,
		Name:         e.Name,
		HostAddress:  e.HostAddress,
		SSHPort:      e.SSHPort,
		Username:     e.Username,
		CredentialID: e.CredentialID,
		IsProd:       e.IsProd,
		Labels:       e.Labels,
		OsInfo:       e.OsInfo,
		AgentStatus:  string(e.AgentStatus),
		Description:  e.Description,
		CreatedAt:    e.CreatedAt.Format(time.DateTime),
		UpdatedAt:    e.UpdatedAt.Format(time.DateTime),
	}
	if e.LastHeartbeat != nil {
		s := e.LastHeartbeat.Format(time.DateTime)
		dto.LastHeartbeat = &s
	}
	return dto
}

// ============================
// UseCase
// ============================

type UseCase struct {
	repo       host.HostRepository
	credRepo   credential.CredentialRepository
	encryptor  crypto.Encryptor
	metricRepo host.HostMetricRepository

	// SSH 连接池：hostID -> shared *ssh.Client。
	// 多个 goroutine 可并发从同一 client 上 NewSession，sessions 相互独立。
	sshPoolMu sync.Mutex
	sshPool   map[string]*ssh.Client
}

func NewUseCase(repo host.HostRepository, credRepo credential.CredentialRepository, encryptor crypto.Encryptor, metricRepo ...host.HostMetricRepository) *UseCase {
	uc := &UseCase{repo: repo, credRepo: credRepo, encryptor: encryptor, sshPool: make(map[string]*ssh.Client)}
	if len(metricRepo) > 0 {
		uc.metricRepo = metricRepo[0]
	}
	return uc
}

// Create 创建机器
func (uc *UseCase) Create(ctx context.Context, cmd CreateHostCmd) (*HostDTO, error) {
	sshPort := cmd.SSHPort
	if sshPort <= 0 {
		sshPort = 22
	}
	labels := cmd.Labels
	if labels == "" {
		labels = "{}"
	}
	username := strings.TrimSpace(cmd.Username)
	if cmd.CredentialID != "" && username == "" {
		return nil, host.ErrHostUsernameRequired
	}

	e := &host.Host{
		Name:         cmd.Name,
		HostAddress:  cmd.HostAddress,
		SSHPort:      sshPort,
		Username:     username,
		CredentialID: cmd.CredentialID,
		IsProd:       cmd.IsProd,
		Labels:       labels,
		AgentStatus:  host.AgentStatusUnknown,
		Description:  cmd.Description,
	}

	if err := uc.repo.Save(ctx, e); err != nil {
		return nil, fmt.Errorf("保存机器失败: %w", err)
	}
	return toDTO(e), nil
}

// Update 更新机器
func (uc *UseCase) Update(ctx context.Context, cmd UpdateHostCmd) (*HostDTO, error) {
	e, err := uc.repo.FindByID(ctx, cmd.ID)
	if err != nil {
		return nil, host.ErrHostNotFound
	}

	if cmd.Name != "" {
		e.Name = cmd.Name
	}
	if cmd.HostAddress != "" {
		e.HostAddress = cmd.HostAddress
	}
	if cmd.SSHPort > 0 {
		e.SSHPort = cmd.SSHPort
	}
	if strings.TrimSpace(cmd.Username) != "" {
		e.Username = strings.TrimSpace(cmd.Username)
	}
	if cmd.CredentialID != "" {
		e.CredentialID = cmd.CredentialID
	}
	if cmd.IsProd != nil {
		e.IsProd = *cmd.IsProd
	}
	if e.CredentialID != "" && strings.TrimSpace(e.Username) == "" {
		return nil, host.ErrHostUsernameRequired
	}
	if cmd.Labels != "" {
		e.Labels = cmd.Labels
	}
	if cmd.Description != "" {
		e.Description = cmd.Description
	}

	if err := uc.repo.Update(ctx, e); err != nil {
		return nil, fmt.Errorf("更新机器失败: %w", err)
	}
	return toDTO(e), nil
}

// Get 获取机器
func (uc *UseCase) Get(ctx context.Context, id string) (*HostDTO, error) {
	e, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, host.ErrHostNotFound
	}
	return toDTO(e), nil
}

// List 查询机器列表
func (uc *UseCase) List(ctx context.Context, cmd HostQueryCmd) ([]*HostDTO, int64, error) {
	if cmd.Page < 1 {
		cmd.Page = 1
	}
	if cmd.PageSize < 1 || cmd.PageSize > 100 {
		cmd.PageSize = 20
	}

	entities, total, err := uc.repo.Find(ctx, host.HostQuery{
		Keyword:  cmd.Keyword,
		Page:     cmd.Page,
		PageSize: cmd.PageSize,
	})
	if err != nil {
		return nil, 0, err
	}

	dtos := make([]*HostDTO, len(entities))
	for i, e := range entities {
		dtos[i] = toDTO(e)
	}
	return dtos, total, nil
}

// Delete 删除机器
func (uc *UseCase) Delete(ctx context.Context, id string) error {
	if _, err := uc.repo.FindByID(ctx, id); err != nil {
		return host.ErrHostNotFound
	}
	return uc.repo.Delete(ctx, id)
}

// ConnectSSH 使用主机绑定凭证建立 SSH 连接，供日志搜索等功能复用
func (uc *UseCase) ConnectSSH(ctx context.Context, hostID string) (*ssh.Client, error) {
	return uc.dialSSH(ctx, hostID)
}

// GetSSH 从连接池获取（或新建）一个共享 SSH 连接。
// 调用方不要 Close 该 client；发现连接不可用时调用 EvictSSH。
func (uc *UseCase) GetSSH(ctx context.Context, hostID string) (*ssh.Client, error) {
	uc.sshPoolMu.Lock()
	cli, ok := uc.sshPool[hostID]
	uc.sshPoolMu.Unlock()

	if ok && pingSSH(cli) {
		return cli, nil
	}
	// 废弃不可用连接
	if ok {
		uc.evictLocked(hostID)
	}

	newCli, err := uc.dialSSH(ctx, hostID)
	if err != nil {
		return nil, err
	}
	uc.sshPoolMu.Lock()
	// 双重检查：如果同时另一调用者已填入，复用他们的
	if existing, exists := uc.sshPool[hostID]; exists {
		uc.sshPoolMu.Unlock()
		_ = newCli.Close()
		return existing, nil
	}
	uc.sshPool[hostID] = newCli
	uc.sshPoolMu.Unlock()
	log.Printf("[SSHPool] dial host=%s", hostID)
	return newCli, nil
}

// EvictSSH 从连接池踢出并关闭指定主机的 SSH 连接。
func (uc *UseCase) EvictSSH(hostID string) {
	uc.evictLocked(hostID)
}

func (uc *UseCase) evictLocked(hostID string) {
	uc.sshPoolMu.Lock()
	cli, ok := uc.sshPool[hostID]
	if ok {
		delete(uc.sshPool, hostID)
	}
	uc.sshPoolMu.Unlock()
	if ok && cli != nil {
		_ = cli.Close()
		log.Printf("[SSHPool] evict host=%s", hostID)
	}
}

// pingSSH 发一个 noop 全局请求探活；连接断开时会返回 error。
func pingSSH(cli *ssh.Client) bool {
	if cli == nil {
		return false
	}
	done := make(chan error, 1)
	go func() {
		_, _, err := cli.SendRequest("keepalive@opshub", true, nil)
		done <- err
	}()
	select {
	case err := <-done:
		return err == nil
	case <-time.After(3 * time.Second):
		return false
	}
}

// dialSSH 创建一个全新的 SSH 连接。不进入连接池。
func (uc *UseCase) dialSSH(ctx context.Context, hostID string) (*ssh.Client, error) {
	h, err := uc.repo.FindByID(ctx, hostID)
	if err != nil {
		return nil, fmt.Errorf("主机不存在: %w", err)
	}
	if h.CredentialID == "" {
		return nil, fmt.Errorf("主机未配置 SSH 凭证")
	}
	if strings.TrimSpace(h.Username) == "" {
		return nil, fmt.Errorf("主机未配置 SSH 用户名")
	}
	cred, err := uc.credRepo.FindByID(ctx, h.CredentialID)
	if err != nil {
		return nil, fmt.Errorf("凭证不存在: %w", err)
	}
	secret, err := uc.encryptor.Decrypt(cred.SecretData)
	if err != nil {
		return nil, fmt.Errorf("凭证解密失败: %w", err)
	}
	var authMethods []ssh.AuthMethod
	switch cred.CredType {
	case "ssh_key":
		signer, parseErr := ssh.ParsePrivateKey([]byte(secret))
		if parseErr != nil {
			// 尝试使用 passphrase 解析加密的私钥
			if cred.Passphrase != "" {
				passphrase, ppErr := uc.encryptor.Decrypt(cred.Passphrase)
				if ppErr != nil {
					return nil, fmt.Errorf("密码短语解密失败: %w", ppErr)
				}
				signer, parseErr = ssh.ParsePrivateKeyWithPassphrase([]byte(secret), []byte(passphrase))
				if parseErr != nil {
					return nil, fmt.Errorf("SSH 密钥解析失败（含密码短语）: %w", parseErr)
				}
			} else {
				return nil, fmt.Errorf("SSH 密钥解析失败: %w（如果私钥有密码保护，请在凭证中配置密码短语）", parseErr)
			}
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	default:
		authMethods = append(authMethods, ssh.Password(secret))
	}
	sshPort := h.SSHPort
	if sshPort <= 0 {
		sshPort = 22
	}
	cfg := &ssh.ClientConfig{
		User:            h.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         15 * time.Second,
	}
	addr := fmt.Sprintf("%s:%d", h.HostAddress, sshPort)
	return ssh.Dial("tcp", addr, cfg)
}

// TestConnection 测试 SSH 连通性
func (uc *UseCase) TestConnection(ctx context.Context, id string) (*TestConnectionResult, error) {
	h, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, host.ErrHostNotFound
	}

	if h.CredentialID == "" {
		return &TestConnectionResult{
			Success: false,
			Message: "未配置凭证，无法测试 SSH 连接",
		}, nil
	}
	if strings.TrimSpace(h.Username) == "" {
		return &TestConnectionResult{
			Success: false,
			Message: "未配置 SSH 用户名，无法测试 SSH 连接",
		}, nil
	}

	cred, err := uc.credRepo.FindByID(ctx, h.CredentialID)
	if err != nil {
		return &TestConnectionResult{
			Success: false,
			Message: "凭证不存在",
		}, nil
	}

	secret, err := uc.encryptor.Decrypt(cred.SecretData)
	if err != nil {
		return &TestConnectionResult{
			Success: false,
			Message: "凭证解密失败",
		}, nil
	}
	// 构建 SSH 认证
	var authMethods []ssh.AuthMethod
	switch cred.CredType {
	case "ssh_key":
		signer, parseErr := ssh.ParsePrivateKey([]byte(secret))
		if parseErr != nil {
			// 尝试使用 passphrase 解析加密的私钥
			if cred.Passphrase != "" {
				passphrase, ppErr := uc.encryptor.Decrypt(cred.Passphrase)
				if ppErr != nil {
					return &TestConnectionResult{
						Success: false,
						Message: fmt.Sprintf("密码短语解密失败: %v", ppErr),
					}, nil
				}
				signer, parseErr = ssh.ParsePrivateKeyWithPassphrase([]byte(secret), []byte(passphrase))
				if parseErr != nil {
					return &TestConnectionResult{
						Success: false,
						Message: fmt.Sprintf("SSH 密钥解析失败（含密码短语）: %v", parseErr),
					}, nil
				}
			} else {
				return &TestConnectionResult{
					Success: false,
					Message: fmt.Sprintf("SSH 密钥解析失败: %v（如果私钥有密码保护，请在凭证中配置密码短语）", parseErr),
				}, nil
			}
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	case "ssh_password":
		authMethods = append(authMethods, ssh.Password(secret))
	}

	config := &ssh.ClientConfig{
		User:            h.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", h.HostAddress, h.SSHPort)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return &TestConnectionResult{
			Success: false,
			Message: fmt.Sprintf("SSH 连接失败: %v", err),
		}, nil
	}
	defer client.Close()

	// 获取系统信息
	session, err := client.NewSession()
	if err != nil {
		return &TestConnectionResult{
			Success: true,
			Message: "SSH 连接成功",
			OsInfo:  "",
		}, nil
	}
	defer session.Close()

	out, _ := session.Output("uname -a 2>/dev/null || ver")
	osInfo := strings.TrimSpace(string(out))

	// 更新机器状态
	h.OsInfo = osInfo
	h.AgentStatus = host.AgentStatusOnline
	now := time.Now()
	h.LastHeartbeat = &now
	_ = uc.repo.Update(ctx, h)

	return &TestConnectionResult{
		Success: true,
		Message: "SSH 连接成功",
		OsInfo:  osInfo,
	}, nil
}

// ============================
// Metrics DTO & Methods
// ============================

type HostMetricDTO struct {
	ID          string         `json:"id"`
	HostID      string         `json:"host_id"`
	CPUUsage    float64        `json:"cpu_usage"`
	CPUCores    int            `json:"cpu_cores"`
	MemTotalMB  int64          `json:"mem_total_mb"`
	MemUsedMB   int64          `json:"mem_used_mb"`
	MemUsage    float64        `json:"mem_usage"`
	DiskTotalGB int64          `json:"disk_total_gb"`
	DiskUsedGB  int64          `json:"disk_used_gb"`
	DiskUsage   float64        `json:"disk_usage"`
	DisksJSON   string         `json:"disks_json"`
	LoadAvg1    float64        `json:"load_avg_1"`
	LoadAvg5    float64        `json:"load_avg_5"`
	LoadAvg15   float64        `json:"load_avg_15"`
	NetInBytes  int64          `json:"net_in_bytes"`
	NetOutBytes int64          `json:"net_out_bytes"`
	GPUUsage    *float64       `json:"gpu_usage"`
	GPUMemUsage *float64       `json:"gpu_mem_usage"`
	GPUTemp     *float64       `json:"gpu_temp"`
	GPUName     string         `json:"gpu_name"`
	GPUs        []host.GPUInfo `json:"gpus"`
	CollectedAt string         `json:"collected_at"`
}

func (uc *UseCase) MetricSnapshotToDTO(s *host.HostMetricSnapshot) *HostMetricDTO {
	return &HostMetricDTO{
		ID:          s.ID,
		HostID:      s.HostID,
		CPUUsage:    s.CPUUsage,
		CPUCores:    s.CPUCores,
		MemTotalMB:  s.MemTotalMB,
		MemUsedMB:   s.MemUsedMB,
		MemUsage:    s.MemUsage,
		DiskTotalGB: s.DiskTotalGB,
		DiskUsedGB:  s.DiskUsedGB,
		DiskUsage:   s.DiskUsage,
		DisksJSON:   s.DisksJSON,
		LoadAvg1:    s.LoadAvg1,
		LoadAvg5:    s.LoadAvg5,
		LoadAvg15:   s.LoadAvg15,
		NetInBytes:  s.NetInBytes,
		NetOutBytes: s.NetOutBytes,
		GPUUsage:    s.GPUUsage,
		GPUMemUsage: s.GPUMemUsage,
		GPUTemp:     s.GPUTemp,
		GPUName:     s.GPUName,
		GPUs:        parseGPUsJSON(s.GPUsJSON),
		CollectedAt: s.CollectedAt.Format(time.DateTime),
	}
}

func parseGPUsJSON(raw string) []host.GPUInfo {
	if raw == "" || raw == "[]" {
		return []host.GPUInfo{}
	}
	var list []host.GPUInfo
	if err := json.Unmarshal([]byte(raw), &list); err != nil {
		return []host.GPUInfo{}
	}
	return list
}

func (uc *UseCase) GetLatestMetrics(ctx context.Context, hostID string) (*HostMetricDTO, error) {
	if uc.metricRepo == nil {
		return nil, fmt.Errorf("指标仓储未初始化")
	}
	s, err := uc.metricRepo.FindLatest(ctx, hostID)
	if err != nil {
		return nil, fmt.Errorf("获取最新指标失败: %w", err)
	}
	return uc.MetricSnapshotToDTO(s), nil
}

func (uc *UseCase) GetMetricsHistory(ctx context.Context, hostID string, startTime, endTime *time.Time, limit int) ([]*HostMetricDTO, error) {
	if uc.metricRepo == nil {
		return nil, fmt.Errorf("指标仓储未初始化")
	}
	snapshots, err := uc.metricRepo.FindHistory(ctx, host.HostMetricQuery{
		HostID:    hostID,
		StartTime: startTime,
		EndTime:   endTime,
		Limit:     limit,
	})
	if err != nil {
		return nil, err
	}
	dtos := make([]*HostMetricDTO, len(snapshots))
	for i, s := range snapshots {
		dtos[i] = uc.MetricSnapshotToDTO(s)
	}
	return dtos, nil
}

func (uc *UseCase) GetAllLatestMetrics(ctx context.Context) ([]*HostMetricDTO, error) {
	if uc.metricRepo == nil {
		return nil, fmt.Errorf("指标仓储未初始化")
	}
	snapshots, err := uc.metricRepo.FindAllLatest(ctx)
	if err != nil {
		return nil, err
	}
	dtos := make([]*HostMetricDTO, len(snapshots))
	for i, s := range snapshots {
		dtos[i] = uc.MetricSnapshotToDTO(s)
	}
	return dtos, nil
}
