package services

import (
	"encoding/json"
	"errors"

	"xpanel/internal/database"
	"xpanel/internal/models"
)

type OutboundService struct{}

func NewOutboundService() *OutboundService {
	return &OutboundService{}
}

type CreateOutboundRequest struct {
	Tag            string `json:"tag" binding:"required"`
	Protocol       string `json:"protocol" binding:"required"`
	Settings       any    `json:"settings"`
	StreamSettings any    `json:"stream_settings"`
	ProxySettings  any    `json:"proxy_settings"`
	Mux            any    `json:"mux"`
	Remark         string `json:"remark"`
}

type UpdateOutboundRequest struct {
	Tag            string `json:"tag"`
	Protocol       string `json:"protocol"`
	Settings       any    `json:"settings"`
	StreamSettings any    `json:"stream_settings"`
	ProxySettings  any    `json:"proxy_settings"`
	Mux            any    `json:"mux"`
	Enable         *bool  `json:"enable"`
	Remark         string `json:"remark"`
}

// List 获取出站列表
func (s *OutboundService) List(page, pageSize int) ([]models.Outbound, int64, error) {
	var outbounds []models.Outbound
	var total int64

	db := database.DB.Model(&models.Outbound{})
	db.Count(&total)

	offset := (page - 1) * pageSize
	if err := db.Offset(offset).Limit(pageSize).Find(&outbounds).Error; err != nil {
		return nil, 0, err
	}

	return outbounds, total, nil
}

// Get 获取单个出站
func (s *OutboundService) Get(id uint) (*models.Outbound, error) {
	var outbound models.Outbound
	if err := database.DB.First(&outbound, id).Error; err != nil {
		return nil, errors.New("出站配置不存在")
	}
	return &outbound, nil
}

// Create 创建出站
func (s *OutboundService) Create(req *CreateOutboundRequest) (*models.Outbound, error) {
	var count int64
	database.DB.Model(&models.Outbound{}).Where("tag = ?", req.Tag).Count(&count)
	if count > 0 {
		return nil, errors.New("Tag 已存在")
	}

	settingsJSON, _ := json.Marshal(req.Settings)
	streamJSON, _ := json.Marshal(req.StreamSettings)
	proxyJSON, _ := json.Marshal(req.ProxySettings)
	muxJSON, _ := json.Marshal(req.Mux)

	outbound := models.Outbound{
		Tag:            req.Tag,
		Protocol:       req.Protocol,
		Settings:       string(settingsJSON),
		StreamSettings: string(streamJSON),
		ProxySettings:  string(proxyJSON),
		Mux:            string(muxJSON),
		Enable:         true,
		Remark:         req.Remark,
	}

	if err := database.DB.Create(&outbound).Error; err != nil {
		return nil, err
	}

	return &outbound, nil
}

// Update 更新出站
func (s *OutboundService) Update(id uint, req *UpdateOutboundRequest) (*models.Outbound, error) {
	var outbound models.Outbound
	if err := database.DB.First(&outbound, id).Error; err != nil {
		return nil, errors.New("出站配置不存在")
	}

	updates := make(map[string]interface{})

	if req.Tag != "" && req.Tag != outbound.Tag {
		var count int64
		database.DB.Model(&models.Outbound{}).Where("tag = ? AND id != ?", req.Tag, id).Count(&count)
		if count > 0 {
			return nil, errors.New("Tag 已存在")
		}
		updates["tag"] = req.Tag
	}

	if req.Protocol != "" {
		updates["protocol"] = req.Protocol
	}
	if req.Settings != nil {
		settingsJSON, _ := json.Marshal(req.Settings)
		updates["settings"] = string(settingsJSON)
	}
	if req.StreamSettings != nil {
		streamJSON, _ := json.Marshal(req.StreamSettings)
		updates["stream_settings"] = string(streamJSON)
	}
	if req.ProxySettings != nil {
		proxyJSON, _ := json.Marshal(req.ProxySettings)
		updates["proxy_settings"] = string(proxyJSON)
	}
	if req.Mux != nil {
		muxJSON, _ := json.Marshal(req.Mux)
		updates["mux"] = string(muxJSON)
	}
	if req.Enable != nil {
		updates["enable"] = *req.Enable
	}
	if req.Remark != "" {
		updates["remark"] = req.Remark
	}

	if len(updates) > 0 {
		if err := database.DB.Model(&outbound).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	database.DB.First(&outbound, id)
	return &outbound, nil
}

// Delete 删除出站
func (s *OutboundService) Delete(id uint) error {
	var outbound models.Outbound
	if err := database.DB.First(&outbound, id).Error; err != nil {
		return errors.New("出站配置不存在")
	}
	return database.DB.Delete(&outbound).Error
}

// GetAllEnabled 获取所有启用的出站
func (s *OutboundService) GetAllEnabled() ([]models.Outbound, error) {
	var outbounds []models.Outbound
	err := database.DB.Where("enable = ?", true).Find(&outbounds).Error
	return outbounds, err
}

// CreateDefaultOutbounds 创建默认出站
func (s *OutboundService) CreateDefaultOutbounds() error {
	// 检查是否已存在
	var count int64
	database.DB.Model(&models.Outbound{}).Count(&count)
	if count > 0 {
		return nil
	}

	// 创建 direct 出站
	direct := models.Outbound{
		Tag:      "direct",
		Protocol: "freedom",
		Settings: "{}",
		Enable:   true,
		Remark:   "直连",
	}
	database.DB.Create(&direct)

	// 创建 blocked 出站
	blocked := models.Outbound{
		Tag:      "blocked",
		Protocol: "blackhole",
		Settings: "{}",
		Enable:   true,
		Remark:   "阻止",
	}
	database.DB.Create(&blocked)

	return nil
}
