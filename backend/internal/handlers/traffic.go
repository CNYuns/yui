package handlers

import (
	"strconv"

	"y-ui/internal/services"
	"y-ui/pkg/response"

	"github.com/gin-gonic/gin"
)

type TrafficHandler struct {
	trafficService *services.TrafficService
}

func NewTrafficHandler() *TrafficHandler {
	return &TrafficHandler{
		trafficService: services.NewTrafficService(),
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
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	stats, err := h.trafficService.GetClientTraffic(uint(id), startDate, endDate)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, stats)
}

// GetInboundTraffic 获取入站流量
func (h *TrafficHandler) GetInboundTraffic(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
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
	days, _ := strconv.Atoi(c.DefaultQuery("days", "30"))

	stats, err := h.trafficService.GetDailyStats(days)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, stats)
}
