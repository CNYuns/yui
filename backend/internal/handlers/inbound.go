package handlers

import (
	"strconv"
	"strings"

	"y-ui/internal/services"
	"y-ui/internal/xray"
	"y-ui/pkg/response"

	"github.com/gin-gonic/gin"
)

type InboundHandler struct {
	inboundService *services.InboundService
	xrayManager    *xray.Manager
}

func NewInboundHandler(xrayManager *xray.Manager) *InboundHandler {
	return &InboundHandler{
		inboundService: services.NewInboundService(),
		xrayManager:    xrayManager,
	}
}

// List 获取入站列表
func (h *InboundHandler) List(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	inbounds, total, err := h.inboundService.List(page, pageSize)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":      inbounds,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Get 获取单个入站
func (h *InboundHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	inbound, err := h.inboundService.Get(uint(id))
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	response.Success(c, inbound)
}

// Create 创建入站
func (h *InboundHandler) Create(c *gin.Context) {
	// 先用 map 接收，以便标准化字段
	var rawReq map[string]interface{}
	if err := c.ShouldBindJSON(&rawReq); err != nil {
		response.BadRequest(c, "无效的 JSON 数据")
		return
	}

	// 标准化 protocol 字段
	if protocol, ok := rawReq["protocol"].(string); ok {
		rawReq["protocol"] = strings.ToLower(strings.TrimSpace(protocol))
	}

	// 验证协议
	protocol, _ := rawReq["protocol"].(string)
	validProtocols := map[string]bool{
		"vmess": true, "vless": true, "trojan": true, "shadowsocks": true,
		"socks": true, "http": true, "dokodemo-door": true, "wireguard": true,
	}
	if !validProtocols[protocol] {
		response.BadRequest(c, "无效的协议类型，支持: vmess, vless, trojan, shadowsocks, socks, http, dokodemo-door, wireguard")
		return
	}

	// 验证必填字段
	tag, _ := rawReq["tag"].(string)
	if strings.TrimSpace(tag) == "" {
		response.BadRequest(c, "标签不能为空")
		return
	}

	port, _ := rawReq["port"].(float64)
	if port < 1 || port > 65535 {
		response.BadRequest(c, "端口必须在 1-65535 之间")
		return
	}

	// 构建请求
	listen, _ := rawReq["listen"].(string)
	if listen == "" {
		listen = "0.0.0.0"
	}
	remark, _ := rawReq["remark"].(string)
	enable := true
	if e, ok := rawReq["enable"].(bool); ok {
		enable = e
	}

	req := &services.CreateInboundRequest{
		Tag:            strings.TrimSpace(tag),
		Protocol:       protocol,
		Port:           int(port),
		Listen:         listen,
		Settings:       rawReq["settings"],
		StreamSettings: rawReq["stream_settings"],
		Sniffing:       rawReq["sniffing"],
		Enable:         &enable,
		Remark:         remark,
	}

	inbound, err := h.inboundService.Create(req)
	if err != nil {
		response.Error(c, 4001, err.Error())
		return
	}

	// 重载 Xray 配置
	if h.xrayManager != nil {
		h.xrayManager.Reload()
	}

	response.Success(c, inbound)
}

// Update 更新入站
func (h *InboundHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	// 使用 map 接收以便标准化
	var rawReq map[string]interface{}
	if err := c.ShouldBindJSON(&rawReq); err != nil {
		response.BadRequest(c, "无效的 JSON 数据")
		return
	}

	// 标准化 protocol 字段
	if protocol, ok := rawReq["protocol"].(string); ok {
		protocol = strings.ToLower(strings.TrimSpace(protocol))
		if protocol != "" {
			validProtocols := map[string]bool{
				"vmess": true, "vless": true, "trojan": true, "shadowsocks": true,
				"socks": true, "http": true, "dokodemo-door": true, "wireguard": true,
			}
			if !validProtocols[protocol] {
				response.BadRequest(c, "无效的协议类型")
				return
			}
		}
		rawReq["protocol"] = protocol
	}

	// 构建更新请求
	req := &services.UpdateInboundRequest{}
	if tag, ok := rawReq["tag"].(string); ok {
		req.Tag = strings.TrimSpace(tag)
	}
	if protocol, ok := rawReq["protocol"].(string); ok {
		req.Protocol = protocol
	}
	if port, ok := rawReq["port"].(float64); ok {
		req.Port = int(port)
	}
	if listen, ok := rawReq["listen"].(string); ok {
		req.Listen = listen
	}
	if remark, ok := rawReq["remark"].(string); ok {
		req.Remark = remark
	}
	if enable, ok := rawReq["enable"].(bool); ok {
		req.Enable = &enable
	}
	req.Settings = rawReq["settings"]
	req.StreamSettings = rawReq["stream_settings"]
	req.Sniffing = rawReq["sniffing"]

	inbound, err := h.inboundService.Update(uint(id), req)
	if err != nil {
		response.Error(c, 4002, err.Error())
		return
	}

	// 重载 Xray 配置
	if h.xrayManager != nil {
		h.xrayManager.Reload()
	}

	response.Success(c, inbound)
}

// Delete 删除入站
func (h *InboundHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	if err := h.inboundService.Delete(uint(id)); err != nil {
		response.Error(c, 4003, err.Error())
		return
	}

	// 重载 Xray 配置
	if h.xrayManager != nil {
		h.xrayManager.Reload()
	}

	response.SuccessMsg(c, "删除成功")
}

// AddClient 添加客户端到入站
func (h *InboundHandler) AddClient(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的入站ID")
		return
	}

	var req struct {
		ClientID uint `json:"client_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if err := h.inboundService.AddClient(uint(id), req.ClientID); err != nil {
		response.Error(c, 4004, err.Error())
		return
	}

	// 重载 Xray 配置
	if h.xrayManager != nil {
		h.xrayManager.Reload()
	}

	response.SuccessMsg(c, "添加成功")
}

// RemoveClient 从入站移除客户端
func (h *InboundHandler) RemoveClient(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的入站ID")
		return
	}
	clientID, err := strconv.ParseUint(c.Param("client_id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的客户端ID")
		return
	}

	if err := h.inboundService.RemoveClient(uint(id), uint(clientID)); err != nil {
		response.Error(c, 4005, err.Error())
		return
	}

	// 重载 Xray 配置
	if h.xrayManager != nil {
		h.xrayManager.Reload()
	}

	response.SuccessMsg(c, "移除成功")
}

// GetClients 获取入站的客户端列表
func (h *InboundHandler) GetClients(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的入站ID")
		return
	}

	clients, err := h.inboundService.GetClients(uint(id))
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, clients)
}
