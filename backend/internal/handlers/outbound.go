package handlers

import (
	"strconv"

	"y-ui/internal/services"
	"y-ui/internal/xray"
	"y-ui/pkg/response"

	"github.com/gin-gonic/gin"
)

type OutboundHandler struct {
	outboundService *services.OutboundService
	xrayManager     *xray.Manager
}

func NewOutboundHandler(xrayManager *xray.Manager) *OutboundHandler {
	return &OutboundHandler{
		outboundService: services.NewOutboundService(),
		xrayManager:     xrayManager,
	}
}

// List 获取出站列表
func (h *OutboundHandler) List(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	outbounds, total, err := h.outboundService.List(page, pageSize)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":      outbounds,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Get 获取单个出站
func (h *OutboundHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	outbound, err := h.outboundService.Get(uint(id))
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	response.Success(c, outbound)
}

// Create 创建出站
func (h *OutboundHandler) Create(c *gin.Context) {
	var req services.CreateOutboundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	outbound, err := h.outboundService.Create(&req)
	if err != nil {
		response.Error(c, 5001, err.Error())
		return
	}

	if h.xrayManager != nil {
		h.xrayManager.Reload()
	}

	response.Success(c, outbound)
}

// Update 更新出站
func (h *OutboundHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	var req services.UpdateOutboundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	outbound, err := h.outboundService.Update(uint(id), &req)
	if err != nil {
		response.Error(c, 5002, err.Error())
		return
	}

	if h.xrayManager != nil {
		h.xrayManager.Reload()
	}

	response.Success(c, outbound)
}

// Delete 删除出站
func (h *OutboundHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	if err := h.outboundService.Delete(uint(id)); err != nil {
		response.Error(c, 5003, err.Error())
		return
	}

	if h.xrayManager != nil {
		h.xrayManager.Reload()
	}

	response.SuccessMsg(c, "删除成功")
}
