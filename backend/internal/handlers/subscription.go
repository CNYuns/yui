package handlers

import (
	"strconv"

	"y-ui/internal/middleware"
	"y-ui/internal/services"
	"y-ui/pkg/response"

	"github.com/gin-gonic/gin"
)

type SubscriptionHandler struct {
	subService    *services.SubscriptionService
	clientService *services.ClientService
}

func NewSubscriptionHandler() *SubscriptionHandler {
	return &SubscriptionHandler{
		subService:    services.NewSubscriptionService(),
		clientService: services.NewClientService(),
	}
}

// GetClientLinks 获取客户端的订阅链接
func (h *SubscriptionHandler) GetClientLinks(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	// 权限检查：非管理员只能查看自己创建的客户端链接
	client, err := h.clientService.Get(uint(id))
	if err != nil {
		response.NotFound(c, "客户端不存在")
		return
	}
	if !middleware.CanManageResource(c, client.CreatedByID) {
		response.Forbidden(c, "无权访问此资源")
		return
	}

	// 获取服务器地址，优先从查询参数获取
	serverAddr := c.Query("server")
	if serverAddr == "" {
		// 尝试从请求头获取
		serverAddr = c.Request.Host
		// 移除端口
		if idx := len(serverAddr) - 1; idx > 0 {
			for i := idx; i >= 0; i-- {
				if serverAddr[i] == ':' {
					serverAddr = serverAddr[:i]
					break
				}
			}
		}
	}

	links, err := h.subService.GenerateClientLinks(uint(id), serverAddr)
	if err != nil {
		response.Error(c, 7001, err.Error())
		return
	}

	response.Success(c, links)
}

// GetSubscription 获取 Base64 编码的订阅内容
func (h *SubscriptionHandler) GetSubscription(c *gin.Context) {
	uuid := c.Param("uuid")
	if uuid == "" {
		response.BadRequest(c, "无效的UUID")
		return
	}

	// 通过 UUID 获取客户端
	client, err := h.clientService.GetByUUID(uuid)
	if err != nil {
		response.NotFound(c, "用户不存在")
		return
	}

	if !client.Enable {
		response.Forbidden(c, "用户已禁用")
		return
	}

	serverAddr := c.Query("server")
	if serverAddr == "" {
		serverAddr = c.Request.Host
		if idx := len(serverAddr) - 1; idx > 0 {
			for i := idx; i >= 0; i-- {
				if serverAddr[i] == ':' {
					serverAddr = serverAddr[:i]
					break
				}
			}
		}
	}

	content, err := h.subService.GenerateSubscription(client.ID, serverAddr)
	if err != nil {
		response.Error(c, 7002, err.Error())
		return
	}

	// 直接返回 Base64 内容，方便客户端订阅
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=subscription.txt")
	c.String(200, content)
}
