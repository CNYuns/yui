package handlers

import (
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
