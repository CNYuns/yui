package services

import (
	"os/exec"
	"runtime"
	"strings"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
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

	// Xray 状态检测
	status.XrayRunning = s.isXrayRunning()
	status.XrayVersion = s.getXrayVersion()

	return status, nil
}

// isXrayRunning 检测 Xray 是否运行
func (s *SystemService) isXrayRunning() bool {
	processes, err := process.Processes()
	if err != nil {
		return false
	}
	for _, p := range processes {
		name, err := p.Name()
		if err == nil && strings.Contains(strings.ToLower(name), "xray") {
			return true
		}
	}
	return false
}

// getXrayVersion 获取 Xray 版本
func (s *SystemService) getXrayVersion() string {
	// 尝试常见的 Xray 路径
	paths := []string{
		"/usr/local/xray/xray",
		"/usr/local/bin/xray",
		"/usr/bin/xray",
		"xray",
	}
	for _, path := range paths {
		out, err := exec.Command(path, "version").Output()
		if err == nil {
			lines := strings.Split(string(out), "\n")
			if len(lines) > 0 {
				// 提取版本号，格式: "Xray 1.8.x ..."
				parts := strings.Fields(lines[0])
				if len(parts) >= 2 {
					return parts[1]
				}
				return lines[0]
			}
		}
	}
	return "未安装"
}
