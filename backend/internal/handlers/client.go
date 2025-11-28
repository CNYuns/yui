package handlers

import (
	"strconv"

	"xpanel/internal/middleware"
	"xpanel/internal/services"
	"xpanel/pkg/response"

	"github.com/gin-gonic/gin"
)

type ClientHandler struct {
	clientService *services.ClientService
}

func NewClientHandler() *ClientHandler {
	return &ClientHandler{
		clientService: services.NewClientService(),
	}
}

// List 获取客户端列表
func (h *ClientHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	// 非管理员只能看到自己创建的
	var createdByID uint
	if middleware.GetUserRole(c) != middleware.RoleAdmin {
		createdByID = middleware.GetUserID(c)
	}

	clients, total, err := h.clientService.List(page, pageSize, createdByID)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":      clients,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Get 获取单个客户端
func (h *ClientHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	client, err := h.clientService.Get(uint(id))
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	response.Success(c, client)
}

// Create 创建客户端
func (h *ClientHandler) Create(c *gin.Context) {
	var req services.CreateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	userID := middleware.GetUserID(c)
	client, err := h.clientService.Create(&req, userID)
	if err != nil {
		response.Error(c, 3001, err.Error())
		return
	}

	response.Success(c, client)
}

// Update 更新客户端
func (h *ClientHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	var req services.UpdateClientRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	client, err := h.clientService.Update(uint(id), &req)
	if err != nil {
		response.Error(c, 3002, err.Error())
		return
	}

	response.Success(c, client)
}

// Delete 删除客户端
func (h *ClientHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	if err := h.clientService.Delete(uint(id)); err != nil {
		response.Error(c, 3003, err.Error())
		return
	}

	response.SuccessMsg(c, "删除成功")
}

// ResetTraffic 重置流量
func (h *ClientHandler) ResetTraffic(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	if err := h.clientService.ResetTraffic(uint(id)); err != nil {
		response.Error(c, 3004, err.Error())
		return
	}

	response.SuccessMsg(c, "流量重置成功")
}
