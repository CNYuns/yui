package services

import (
	"runtime"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
)

type SystemService struct{}

func NewSystemService() *SystemService {
	return &SystemService{}
}

type SystemStatus struct {
	Hostname    string      `json:"hostname"`
	Platform    string      `json:"platform"`
	OS          string      `json:"os"`
	Arch        string      `json:"arch"`
	Uptime      uint64      `json:"uptime"`
	CPU         CPUStatus   `json:"cpu"`
	Memory      MemStatus   `json:"memory"`
	Disk        DiskStatus  `json:"disk"`
	XrayRunning bool        `json:"xray_running"`
	XrayVersion string      `json:"xray_version"`
}

type CPUStatus struct {
	Cores   int       `json:"cores"`
	Usage   []float64 `json:"usage"`
}

type MemStatus struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
}

type DiskStatus struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
}

// GetStatus 获取系统状态
func (s *SystemService) GetStatus() (*SystemStatus, error) {
	status := &SystemStatus{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	// 主机信息
	hostInfo, err := host.Info()
	if err == nil {
		status.Hostname = hostInfo.Hostname
		status.Platform = hostInfo.Platform
		status.Uptime = hostInfo.Uptime
	}

	// CPU 信息
	cpuPercent, err := cpu.Percent(0, true)
	if err == nil {
		status.CPU.Usage = cpuPercent
		status.CPU.Cores = len(cpuPercent)
	}

	// 内存信息
	memInfo, err := mem.VirtualMemory()
	if err == nil {
		status.Memory = MemStatus{
			Total:       memInfo.Total,
			Used:        memInfo.Used,
			Free:        memInfo.Free,
			UsedPercent: memInfo.UsedPercent,
		}
	}

	// 磁盘信息
	diskInfo, err := disk.Usage("/")
	if err == nil {
		status.Disk = DiskStatus{
			Total:       diskInfo.Total,
			Used:        diskInfo.Used,
			Free:        diskInfo.Free,
			UsedPercent: diskInfo.UsedPercent,
		}
	}

	return status, nil
}
