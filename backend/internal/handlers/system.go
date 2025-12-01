package handlers

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"y-ui/internal/services"
	"y-ui/internal/xray"
	"y-ui/pkg/response"

	"github.com/gin-gonic/gin"
)

type SystemHandler struct {
	systemService *services.SystemService
	xrayManager   *xray.Manager
}

func NewSystemHandler(xrayManager *xray.Manager) *SystemHandler {
	return &SystemHandler{
		systemService: services.NewSystemService(),
		xrayManager:   xrayManager,
	}
}

// GetStatus 获取系统状态
func (h *SystemHandler) GetStatus(c *gin.Context) {
	status, err := h.systemService.GetStatus()
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	// 添加 Xray 状态
	if h.xrayManager != nil {
		status.XrayRunning = h.xrayManager.IsRunning()
		status.XrayVersion = h.xrayManager.GetVersion()
	}

	response.Success(c, status)
}

// ReloadXray 重载 Xray 配置
func (h *SystemHandler) ReloadXray(c *gin.Context) {
	if h.xrayManager == nil {
		response.Error(c, 6001, "Xray 管理器未初始化")
		return
	}

	if err := h.xrayManager.Reload(); err != nil {
		response.Error(c, 6002, "重载失败: "+err.Error())
		return
	}

	response.SuccessMsg(c, "重载成功")
}

// RestartXray 重启 Xray
func (h *SystemHandler) RestartXray(c *gin.Context) {
	if h.xrayManager == nil {
		response.Error(c, 6001, "Xray 管理器未初始化")
		return
	}

	if err := h.xrayManager.Restart(); err != nil {
		response.Error(c, 6003, "重启失败: "+err.Error())
		return
	}

	response.SuccessMsg(c, "重启成功")
}

// GetXrayConfig 获取当前 Xray 配置
func (h *SystemHandler) GetXrayConfig(c *gin.Context) {
	if h.xrayManager == nil {
		response.Error(c, 6001, "Xray 管理器未初始化")
		return
	}

	config, err := h.xrayManager.GetCurrentConfig()
	if err != nil {
		response.Error(c, 6004, "获取配置失败: "+err.Error())
		return
	}

	response.Success(c, config)
}

// CheckPort 检查端口是否被占用
func (h *SystemHandler) CheckPort(c *gin.Context) {
	portStr := c.Query("port")
	if portStr == "" {
		response.BadRequest(c, "端口参数缺失")
		return
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		response.BadRequest(c, "无效的端口号")
		return
	}

	// 检查 TCP 端口
	tcpInUse := isPortInUse("tcp", port)
	// 检查 UDP 端口
	udpInUse := isPortInUse("udp", port)

	response.Success(c, gin.H{
		"port":       port,
		"tcp_in_use": tcpInUse,
		"udp_in_use": udpInUse,
		"available":  !tcpInUse && !udpInUse,
	})
}

// isPortInUse 检查端口是否被占用
func isPortInUse(protocol string, port int) bool {
	address := fmt.Sprintf(":%d", port)

	if protocol == "tcp" {
		listener, err := net.Listen("tcp", address)
		if err != nil {
			return true
		}
		listener.Close()
		return false
	}

	// UDP
	conn, err := net.ListenPacket("udp", address)
	if err != nil {
		return true
	}
	conn.Close()
	return false
}

// CheckUpdate 检查 GitHub 版本更新
func (h *SystemHandler) CheckUpdate(c *gin.Context) {
	// 当前版本
	currentVersion := "1.3.5"

	// 从 GitHub API 获取最新版本
	latestVersion, releaseURL, err := getLatestVersion()
	if err != nil {
		response.Error(c, 6005, "检查更新失败: "+err.Error())
		return
	}

	// 比较版本
	hasUpdate := compareVersions(latestVersion, currentVersion)

	response.Success(c, gin.H{
		"current_version": currentVersion,
		"latest_version":  latestVersion,
		"has_update":      hasUpdate,
		"release_url":     releaseURL,
	})
}

// getLatestVersion 从 GitHub 获取最新版本
func getLatestVersion() (string, string, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get("https://api.github.com/repos/CNYuns/yui/releases/latest")
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("GitHub API 返回状态码: %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", err
	}

	// 去掉 v 前缀
	version := release.TagName
	if len(version) > 0 && version[0] == 'v' {
		version = version[1:]
	}

	return version, release.HTMLURL, nil
}

// compareVersions 比较版本号，返回 true 表示有更新
func compareVersions(latest, current string) bool {
	// 简单的版本比较
	return latest > current
}
