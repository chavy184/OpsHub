package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"ops-hub/internal/domain/host"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// AlertCallback 告警回调函数类型
type AlertCallback func(hostName, hostAddress, alertType, message string, value float64)

// Thresholds 资源告警阈值集
type Thresholds struct {
	CPUWarning   float64
	CPUCritical  float64
	MemWarning   float64
	MemCritical  float64
	DiskWarning  float64
	DiskCritical float64
}

// ThresholdProvider 阈值提供者（运行时读取阈值，允许热更新）
type ThresholdProvider func() Thresholds

// DefaultThresholds 内置默认阈值
var DefaultThresholds = Thresholds{
	CPUWarning:   90.0,
	CPUCritical:  95.0,
	MemWarning:   85.0,
	MemCritical:  95.0,
	DiskWarning:  85.0,
	DiskCritical: 95.0,
}

// SSHProvider 提供连接池能力：GetSSH 返回共享连接，EvictSSH 踢出不可用连接。
type SSHProvider interface {
	GetSSH(ctx context.Context, hostID string) (*ssh.Client, error)
	EvictSSH(hostID string)
}

// Collector 主机指标采集器
type Collector struct {
	hostRepo          host.HostRepository
	metricRepo        host.HostMetricRepository
	sshProv           SSHProvider
	alertCallback     AlertCallback
	thresholdProvider ThresholdProvider
	interval          time.Duration
	stopCh            chan struct{}
}

// NewCollector 创建采集器
func NewCollector(
	hostRepo host.HostRepository,
	metricRepo host.HostMetricRepository,
	sshProv SSHProvider,
) *Collector {
	return &Collector{
		hostRepo:   hostRepo,
		metricRepo: metricRepo,
		sshProv:    sshProv,
		interval:   60 * time.Second,
		stopCh:     make(chan struct{}),
	}
}

// SetAlertCallback 设置告警回调
func (c *Collector) SetAlertCallback(cb AlertCallback) {
	c.alertCallback = cb
}

// SetThresholdProvider 设置阈值提供者（未设置时使用 DefaultThresholds）
func (c *Collector) SetThresholdProvider(p ThresholdProvider) {
	c.thresholdProvider = p
}

// Start 启动定时采集
func (c *Collector) Start() {
	go func() {
		ticker := time.NewTicker(c.interval)
		defer ticker.Stop()
		// Run once immediately
		c.collectAll()
		for {
			select {
			case <-ticker.C:
				c.collectAll()
			case <-c.stopCh:
				return
			}
		}
	}()
	log.Printf("[MetricsCollector] started, interval=%v", c.interval)
}

// Stop 停止采集
func (c *Collector) Stop() {
	close(c.stopCh)
}

func (c *Collector) collectAll() {
	ctx := context.Background()
	hosts, _, err := c.hostRepo.Find(ctx, host.HostQuery{Page: 1, PageSize: 200})
	if err != nil {
		log.Printf("[MetricsCollector] failed to list hosts: %v", err)
		return
	}

	for _, h := range hosts {
		// 跳过未配置凭证的主机（无法 SSH）； offline/unknown 主机仍尝试采集，以便从离线恢复
		if h.CredentialID == "" {
			continue
		}
		go c.collectOne(ctx, h)
	}

	// Clean up old metrics (older than 24h)
	_ = c.metricRepo.DeleteOlderThan(ctx, time.Now().Add(-24*time.Hour))
}

// CollectOne 采集单台主机指标（公开方法，供手动触发调用）
func (c *Collector) CollectOne(ctx context.Context, hostID string) (*host.HostMetricSnapshot, error) {
	h, err := c.hostRepo.FindByID(ctx, hostID)
	if err != nil {
		return nil, fmt.Errorf("主机不存在: %w", err)
	}
	return c.doCollect(ctx, h)
}

func (c *Collector) collectOne(ctx context.Context, h *host.Host) {
	snapshot, err := c.doCollect(ctx, h)
	if err != nil {
		log.Printf("[MetricsCollector] host=%s(%s) collect failed: %v", h.Name, h.HostAddress, err)
		// 采集失败 → 标记主机离线
		c.markHostStatus(ctx, h, false)
		// SSH 连接失败视为主机离线告警
		if c.alertCallback != nil {
			c.alertCallback(h.Name, h.HostAddress, "host_offline", fmt.Sprintf("主机 %s(%s) SSH 连接失败: %v", h.Name, h.HostAddress, err), 0)
		}
		return
	}
	// 采集成功 → 更新心跳，标记在线
	c.markHostStatus(ctx, h, true)
	// 检查阈值告警
	c.checkThresholds(h, snapshot)
}

// markHostStatus 更新主机上下线状态与心跳。仅在状态变化或心跳超过 30s 未更新时写库，避免频繁 UPDATE。
func (c *Collector) markHostStatus(ctx context.Context, h *host.Host, online bool) {
	now := time.Now()
	targetStatus := host.AgentStatusOffline
	if online {
		targetStatus = host.AgentStatusOnline
	}

	needWrite := h.AgentStatus != targetStatus
	if online && !needWrite {
		if h.LastHeartbeat == nil || now.Sub(*h.LastHeartbeat) >= 30*time.Second {
			needWrite = true
		}
	}
	if !needWrite {
		return
	}
	h.AgentStatus = targetStatus
	if online {
		h.LastHeartbeat = &now
	}
	if err := c.hostRepo.Update(ctx, h); err != nil {
		log.Printf("[MetricsCollector] update host status host=%s err=%v", h.Name, err)
	}
}

func (c *Collector) doCollect(ctx context.Context, h *host.Host) (*host.HostMetricSnapshot, error) {
	client, err := c.sshProv.GetSSH(ctx, h.ID)
	if err != nil {
		return nil, fmt.Errorf("SSH connect failed: %w", err)
	}
	// 池化连接不在这里 Close

	snapshot := &host.HostMetricSnapshot{
		HostID:      h.ID,
		CollectedAt: time.Now(),
	}

	// 单脚本一次性采集所有指标，减少 SSH session 开销
	combinedOut, runErr := runCmd(client, collectScript)
	if runErr != nil {
		// 连接可能已断开：踢出让下轮重拨
		c.sshProv.EvictSSH(h.ID)
		return nil, fmt.Errorf("collect cmd failed: %w", runErr)
	}
	sections := splitSections(combinedOut)

	// CPU：第一行=核数，第二行=top 输出
	cpuCombined := strings.TrimSpace(sections["CPU_CORES"]) + "\n" + strings.TrimSpace(sections["CPU_TOP"])
	parseCPU(cpuCombined, snapshot)
	parseMem(strings.TrimSpace(sections["MEM"]), snapshot)
	parseDisks(strings.TrimSpace(sections["DISK"]), snapshot)
	parseLoad(strings.TrimSpace(sections["LOAD"]), snapshot)
	parseGPU(strings.TrimSpace(sections["GPU"]), snapshot)

	if err := c.metricRepo.Save(ctx, snapshot); err != nil {
		return nil, fmt.Errorf("save metric failed: %w", err)
	}

	return snapshot, nil
}

// collectScript 在一个 SSH session 中依次打印各节区并以隔离符划分。
const collectScript = `
echo "===CPU_CORES==="
grep -c ^processor /proc/cpuinfo 2>/dev/null
echo "===CPU_TOP==="
top -b -n1 2>/dev/null | grep '%Cpu' | head -1
echo "===MEM==="
LC_ALL=C free -m 2>/dev/null | grep '^Mem:'
echo "===DISK==="
LC_ALL=C df -BG -P 2>/dev/null | awk 'NR==1{next} $1 ~ "^/dev/" && $2+0 > 5 && $6 !~ "^/boot"{print}'
echo "===LOAD==="
cat /proc/loadavg 2>/dev/null
echo "===GPU==="
nvidia-smi --query-gpu=utilization.gpu,memory.used,memory.total,temperature.gpu,name --format=csv,noheader,nounits 2>/dev/null
echo "===END==="
`

// splitSections 按 ===NAME=== 标记拆分输出。
func splitSections(out string) map[string]string {
	sections := make(map[string]string)
	lines := strings.Split(out, "\n")
	curName := ""
	var curBuf strings.Builder
	flush := func() {
		if curName != "" {
			sections[curName] = curBuf.String()
		}
		curBuf.Reset()
	}
	for _, line := range lines {
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "===") && strings.HasSuffix(trim, "===") && len(trim) > 6 {
			flush()
			name := strings.TrimSuffix(strings.TrimPrefix(trim, "==="), "===")
			if name == "END" {
				curName = ""
				continue
			}
			curName = name
			continue
		}
		if curName != "" {
			curBuf.WriteString(line)
			curBuf.WriteByte('\n')
		}
	}
	flush()
	return sections
}

func runCmd(client *ssh.Client, cmd string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()
	out, err := session.Output(cmd)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func parseCPU(output string, s *host.HostMetricSnapshot) {
	lines := strings.Split(output, "\n")
	if len(lines) >= 1 {
		cores, _ := strconv.Atoi(strings.TrimSpace(lines[0]))
		s.CPUCores = cores
	}
	if len(lines) >= 2 {
		// Parse: %Cpu(s):  5.3 us,  1.2 sy,  0.0 ni, 93.0 id, ...
		line := lines[1]
		if idx := strings.Index(line, "id"); idx > 0 {
			parts := strings.Split(line[:idx], ",")
			if len(parts) > 0 {
				last := strings.TrimSpace(parts[len(parts)-1])
				idle, err := strconv.ParseFloat(last, 64)
				if err == nil {
					s.CPUUsage = 100.0 - idle
				}
			}
		}
	}
}

func parseMem(output string, s *host.HostMetricSnapshot) {
	// Mem:           7855        3214        2547         187        2093        4270
	fields := strings.Fields(output)
	if len(fields) >= 3 {
		total, _ := strconv.ParseInt(fields[1], 10, 64)
		used, _ := strconv.ParseInt(fields[2], 10, 64)
		s.MemTotalMB = total
		s.MemUsedMB = used
		if total > 0 {
			s.MemUsage = float64(used) / float64(total) * 100.0
		}
	}
}

func parseDisks(output string, s *host.HostMetricSnapshot) {
	var disks []host.DiskInfo
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		mountPoint := fields[5]
		totalGB := parseGB(fields[1])
		usedGB := parseGB(fields[2])
		var usage float64
		if totalGB > 0 {
			usage = float64(usedGB) / float64(totalGB) * 100.0
		}
		disks = append(disks, host.DiskInfo{
			MountPoint: mountPoint,
			TotalGB:    totalGB,
			UsedGB:     usedGB,
			Usage:      usage,
		})
	}
	// 主分区（根 /）保持向后兼容
	for _, d := range disks {
		if d.MountPoint == "/" {
			s.DiskTotalGB = d.TotalGB
			s.DiskUsedGB = d.UsedGB
			s.DiskUsage = d.Usage
			break
		}
	}
	if len(disks) == 0 {
		return
	}
	if s.DiskTotalGB == 0 {
		// 没有根分区时用第一个
		s.DiskTotalGB = disks[0].TotalGB
		s.DiskUsedGB = disks[0].UsedGB
		s.DiskUsage = disks[0].Usage
	}
	if b, err := json.Marshal(disks); err == nil {
		s.DisksJSON = string(b)
	}
}

func parseGB(s string) int64 {
	s = strings.TrimSuffix(s, "G")
	v, _ := strconv.ParseInt(s, 10, 64)
	return v
}

func parseLoad(output string, s *host.HostMetricSnapshot) {
	// 0.15 0.10 0.05 1/123 12345
	fields := strings.Fields(output)
	if len(fields) >= 3 {
		s.LoadAvg1, _ = strconv.ParseFloat(fields[0], 64)
		s.LoadAvg5, _ = strconv.ParseFloat(fields[1], 64)
		s.LoadAvg15, _ = strconv.ParseFloat(fields[2], 64)
	}
}

func parseGPU(output string, s *host.HostMetricSnapshot) {
	if output == "" {
		return
	}
	// nvidia-smi 多卡输出每行一张卡（按顺序）：
	//   utilization.gpu, memory.used, memory.total, temperature.gpu, name
	//   45, 1024, 16384, 65, Tesla V100-SXM2-16GB
	gpus := make([]host.GPUInfo, 0, 4)
	for i, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, ",", 5)
		if len(fields) < 4 {
			continue
		}
		usage, _ := strconv.ParseFloat(strings.TrimSpace(fields[0]), 64)
		memUsed, _ := strconv.ParseFloat(strings.TrimSpace(fields[1]), 64)
		memTotal, _ := strconv.ParseFloat(strings.TrimSpace(fields[2]), 64)
		temp, _ := strconv.ParseFloat(strings.TrimSpace(fields[3]), 64)
		name := ""
		if len(fields) >= 5 {
			name = strings.TrimSpace(fields[4])
		}
		memUsage := 0.0
		if memTotal > 0 {
			memUsage = memUsed / memTotal * 100
		}
		gpus = append(gpus, host.GPUInfo{
			Index: i, Name: name, Usage: usage, MemUsage: memUsage, Temp: temp,
		})
	}
	if len(gpus) == 0 {
		return
	}
	// 兼容字段：首张卡数据写入旧字段
	first := gpus[0]
	u, m, t := first.Usage, first.MemUsage, first.Temp
	s.GPUUsage = &u
	s.GPUMemUsage = &m
	s.GPUTemp = &t
	s.GPUName = first.Name
	// 全量数据写入 GPUsJSON
	if data, err := json.Marshal(gpus); err == nil {
		s.GPUsJSON = string(data)
	}
}

// ─── 阈值检查 ────────────────────────────────────────────

func (c *Collector) checkThresholds(h *host.Host, s *host.HostMetricSnapshot) {
	if c.alertCallback == nil {
		return
	}

	t := DefaultThresholds
	if c.thresholdProvider != nil {
		t = c.thresholdProvider()
	}

	if s.CPUUsage >= t.CPUCritical {
		c.alertCallback(h.Name, h.HostAddress, "cpu_critical",
			fmt.Sprintf("主机 %s CPU 使用率 %.1f%% 超过严重阈值 %.0f%%", h.Name, s.CPUUsage, t.CPUCritical), s.CPUUsage)
	} else if s.CPUUsage >= t.CPUWarning {
		c.alertCallback(h.Name, h.HostAddress, "cpu_warning",
			fmt.Sprintf("主机 %s CPU 使用率 %.1f%% 超过警告阈值 %.0f%%", h.Name, s.CPUUsage, t.CPUWarning), s.CPUUsage)
	}

	if s.MemUsage >= t.MemCritical {
		c.alertCallback(h.Name, h.HostAddress, "mem_critical",
			fmt.Sprintf("主机 %s 内存使用率 %.1f%% 超过严重阈值 %.0f%%", h.Name, s.MemUsage, t.MemCritical), s.MemUsage)
	} else if s.MemUsage >= t.MemWarning {
		c.alertCallback(h.Name, h.HostAddress, "mem_warning",
			fmt.Sprintf("主机 %s 内存使用率 %.1f%% 超过警告阈值 %.0f%%", h.Name, s.MemUsage, t.MemWarning), s.MemUsage)
	}

	if s.DiskUsage >= t.DiskCritical {
		c.alertCallback(h.Name, h.HostAddress, "disk_critical",
			fmt.Sprintf("主机 %s 磁盘(/)使用率 %.1f%% 超过严重阈值 %.0f%%", h.Name, s.DiskUsage, t.DiskCritical), s.DiskUsage)
	} else if s.DiskUsage >= t.DiskWarning {
		c.alertCallback(h.Name, h.HostAddress, "disk_warning",
			fmt.Sprintf("主机 %s 磁盘(/)使用率 %.1f%% 超过警告阈值 %.0f%%", h.Name, s.DiskUsage, t.DiskWarning), s.DiskUsage)
	}

	// 检查所有磁盘分区
	if s.DisksJSON != "" {
		var disks []host.DiskInfo
		if err := json.Unmarshal([]byte(s.DisksJSON), &disks); err == nil {
			for _, d := range disks {
				if d.MountPoint == "/" {
					continue // 根分区已在上面检查
				}
				if d.Usage >= t.DiskCritical {
					c.alertCallback(h.Name, h.HostAddress, "disk_critical_"+d.MountPoint,
						fmt.Sprintf("主机 %s 磁盘(%s)使用率 %.1f%% 超过严重阈值 %.0f%%", h.Name, d.MountPoint, d.Usage, t.DiskCritical), d.Usage)
				} else if d.Usage >= t.DiskWarning {
					c.alertCallback(h.Name, h.HostAddress, "disk_warning_"+d.MountPoint,
						fmt.Sprintf("主机 %s 磁盘(%s)使用率 %.1f%% 超过警告阈值 %.0f%%", h.Name, d.MountPoint, d.Usage, t.DiskWarning), d.Usage)
				}
			}
		}
	}
}
