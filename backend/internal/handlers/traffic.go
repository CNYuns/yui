package handlers

import (
	"strconv"

	"y-ui/internal/middleware"
	"y-ui/internal/services"
	"y-ui/pkg/response"

	"github.com/gin-gonic/gin"
)

type TrafficHandler struct {
	trafficService *services.TrafficService
	clientService  *services.ClientService
}

func NewTrafficHandler() *TrafficHandler {
	return &TrafficHandler{
		trafficService: services.NewTrafficService(),
		clientService:  services.NewClientService(),
	}
}

// GetSummary 获取流量汇总
func (h *TrafficHandler) GetSummary(c *gin.Context) {
	summary, err := h.trafficService.GetSummary()
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, summary)
}

// GetClientTraffic 获取客户端流量
func (h *TrafficHandler) GetClientTraffic(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的客户端ID")
		return
	}

	// 权限检查：非管理员只能查看自己创建的客户端流量
	client, err := h.clientService.Get(uint(id))
	if err != nil {
		response.NotFound(c, "客户端不存在")
		return
	}
	if !middleware.CanManageResource(c, client.CreatedByID) {
		response.Forbidden(c, "无权查看此客户端流量")
		return
	}

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	stats, err := h.trafficService.GetClientTraffic(uint(id), startDate, endDate)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, stats)
}

// GetInboundTraffic 获取入站流量（仅管理员）
func (h *TrafficHandler) GetInboundTraffic(c *gin.Context) {
	// 权限检查：仅管理员可以查看入站流量
	if middleware.GetUserRole(c) != middleware.RoleAdmin {
		response.Forbidden(c, "仅管理员可查看入站流量")
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的入站ID")
		return
	}
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	stats, err := h.trafficService.GetInboundTraffic(uint(id), startDate, endDate)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, stats)
}

// GetDailyStats 获取每日统计
func (h *TrafficHandler) GetDailyStats(c *gin.Context) {
	days, err := strconv.Atoi(c.DefaultQuery("days", "30"))
	if err != nil || days < 1 || days > 365 {
		days = 30
	}

	stats, err := h.trafficService.GetDailyStats(days)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, stats)
}
