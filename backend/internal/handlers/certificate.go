package handlers

import (
	"strconv"

	"y-ui/internal/config"
	"y-ui/internal/services"
	"y-ui/pkg/response"

	"github.com/gin-gonic/gin"
)

type CertificateHandler struct {
	certService *services.CertificateService
}

func NewCertificateHandler() *CertificateHandler {
	return &CertificateHandler{
		certService: services.NewCertificateService(),
	}
}

// List 获取证书列表
func (h *CertificateHandler) List(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	certs, total, err := h.certService.List(page, pageSize)
	if err != nil {
		response.ServerError(c, err.Error())
		return
	}

	response.Success(c, gin.H{
		"list":      certs,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// Get 获取单个证书
func (h *CertificateHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	cert, err := h.certService.Get(uint(id))
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	response.Success(c, cert)
}

// Request 申请证书
func (h *CertificateHandler) Request(c *gin.Context) {
	var req services.RequestCertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误: "+err.Error())
		return
	}

	cfg := config.GlobalConfig
	certDir := cfg.TLS.CertPath
	if certDir == "" {
		certDir = "/etc/y-ui/certs"
	}

	cert, err := h.certService.Request(&req, certDir)
	if err != nil {
		response.Error(c, 7001, "证书申请失败: "+err.Error())
		return
	}

	response.Success(c, cert)
}

// Renew 续签证书
func (h *CertificateHandler) Renew(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	if err := h.certService.Renew(uint(id)); err != nil {
		response.Error(c, 7002, "续签失败: "+err.Error())
		return
	}

	response.SuccessMsg(c, "续签成功")
}

// Delete 删除证书
func (h *CertificateHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	if err := h.certService.Delete(uint(id)); err != nil {
		response.Error(c, 7003, err.Error())
		return
	}

	response.SuccessMsg(c, "删除成功")
}

// UpdateAutoRenew 更新自动续签
func (h *CertificateHandler) UpdateAutoRenew(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的ID")
		return
	}

	var req struct {
		AutoRenew bool `json:"auto_renew"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "参数错误")
		return
	}

	if err := h.certService.UpdateAutoRenew(uint(id), req.AutoRenew); err != nil {
		response.Error(c, 7004, err.Error())
		return
	}

	response.SuccessMsg(c, "更新成功")
}
