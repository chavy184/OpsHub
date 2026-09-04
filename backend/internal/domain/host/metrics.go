package host

import "time"

// DiskInfo 单个磁盘分区信息
type DiskInfo struct {
	MountPoint string  `json:"mount_point"`
	TotalGB    int64   `json:"total_gb"`
	UsedGB     int64   `json:"used_gb"`
	Usage      float64 `json:"usage"`
}

// GPUInfo 单块 GPU 信息
type GPUInfo struct {
	Index    int     `json:"index"`
	Name     string  `json:"name"`
	Usage    float64 `json:"usage"`
	MemUsage float64 `json:"mem_usage"`
	Temp     float64 `json:"temp"`
}

// HostMetricSnapshot 主机指标快照
type HostMetricSnapshot struct {
	ID          string
	HostID      string
	CPUUsage    float64
	CPUCores    int
	MemTotalMB  int64
	MemUsedMB   int64
	MemUsage    float64
	DiskTotalGB int64   // 保留：主分区总量（向后兼容）
	DiskUsedGB  int64   // 保留：主分区已用
	DiskUsage   float64 // 保留：主分区使用率
	DisksJSON   string  // 多磁盘 JSON 序列化
	LoadAvg1    float64
	LoadAvg5    float64
	LoadAvg15   float64
	NetInBytes  int64
	NetOutBytes int64
	GPUUsage    *float64
	GPUMemUsage *float64
	GPUTemp     *float64
	GPUName     string
	GPUsJSON    string // 多 GPU JSON 数组（新增）
	CollectedAt time.Time
	CreatedAt   time.Time
}

// HostMetricQuery 指标查询参数
type HostMetricQuery struct {
	HostID    string
	StartTime *time.Time
	EndTime   *time.Time
	Limit     int
}
