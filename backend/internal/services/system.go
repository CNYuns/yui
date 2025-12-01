package services

import (
	"os/exec"
	"runtime"
	"strings"
	"time"

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
		CPU: CPUStatus{
			Cores: runtime.NumCPU(),
			Usage: []float64{0},
		},
	}

	// 主机信息
	hostInfo, err := host.Info()
	if err == nil {
		status.Hostname = hostInfo.Hostname
		status.Platform = hostInfo.Platform
		status.Uptime = hostInfo.Uptime
	}

	// CPU 信息 - 使用 500ms 间隔获取更准确的数据
	cpuPercent, err := cpu.Percent(500*time.Millisecond, true)
	if err == nil && len(cpuPercent) > 0 {
		status.CPU.Usage = cpuPercent
		status.CPU.Cores = len(cpuPercent)
	} else {
		// 如果获取失败，尝试获取总体使用率
		cpuTotal, err := cpu.Percent(500*time.Millisecond, false)
		if err == nil && len(cpuTotal) > 0 {
			status.CPU.Usage = cpuTotal
		}
	}

	// 内存信息
	memInfo, err := mem.VirtualMemory()
	if err == nil && memInfo != nil {
		status.Memory = MemStatus{
			Total:       memInfo.Total,
			Used:        memInfo.Used,
			Free:        memInfo.Free,
			UsedPercent: memInfo.UsedPercent,
		}
	}

	// 磁盘信息 - 尝试多个路径
	diskPaths := []string{"/", "/home", "C:\\"}
	for _, path := range diskPaths {
		diskInfo, err := disk.Usage(path)
		if err == nil && diskInfo != nil {
			status.Disk = DiskStatus{
				Total:       diskInfo.Total,
				Used:        diskInfo.Used,
				Free:        diskInfo.Free,
				UsedPercent: diskInfo.UsedPercent,
			}
			break
		}
	}

	// Xray 状态检测
	status.XrayRunning = s.isXrayRunning()
	status.XrayVersion = s.getXrayVersion()

	return status, nil
}

// isXrayRunning 检测 Xray 是否运行
func (s *SystemService) isXrayRunning() bool {
	// 方法1：使用 gopsutil 检测进程
	processes, err := process.Processes()
	if err == nil {
		for _, p := range processes {
			name, err := p.Name()
			if err == nil && strings.ToLower(name) == "xray" {
				// 检查是否是 run 命令（排除 version 检查等）
				cmdline, _ := p.Cmdline()
				if strings.Contains(cmdline, "run") {
					return true
				}
			}
		}
	}

	// 方法2：使用 pgrep 检查进程（Linux 备用）
	cmd := exec.Command("pgrep", "-x", "xray")
	if err := cmd.Run(); err == nil {
		return true
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
