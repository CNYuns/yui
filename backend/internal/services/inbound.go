package services

import (
	"encoding/json"
	"errors"
	"fmt"

	"y-ui/internal/database"
	"y-ui/internal/models"
)

type InboundService struct{}

func NewInboundService() *InboundService {
	return &InboundService{}
}

type CreateInboundRequest struct {
	Tag            string `json:"tag" binding:"required"`
	Protocol       string `json:"protocol" binding:"required,oneof=vmess vless trojan shadowsocks wireguard"`
	Port           int    `json:"port" binding:"required,min=1,max=65535"`
	Listen         string `json:"listen"`
	Settings       any    `json:"settings"`
	StreamSettings any    `json:"stream_settings"`
	Sniffing       any    `json:"sniffing"`
	Remark         string `json:"remark"`
}

type UpdateInboundRequest struct {
	Tag            string `json:"tag"`
	Protocol       string `json:"protocol" binding:"omitempty,oneof=vmess vless trojan shadowsocks wireguard"`
	Port           int    `json:"port" binding:"omitempty,min=1,max=65535"`
	Listen         string `json:"listen"`
	Settings       any    `json:"settings"`
	StreamSettings any    `json:"stream_settings"`
	Sniffing       any    `json:"sniffing"`
	Enable         *bool  `json:"enable"`
	Remark         string `json:"remark"`
}

// List 获取入站列表
func (s *InboundService) List(page, pageSize int) ([]models.Inbound, int64, error) {
	var inbounds []models.Inbound
	var total int64

	db := database.DB.Model(&models.Inbound{})
	db.Count(&total)

	offset := (page - 1) * pageSize
	if err := db.Offset(offset).Limit(pageSize).Find(&inbounds).Error; err != nil {
		return nil, 0, err
	}

	return inbounds, total, nil
}

// Get 获取单个入站
func (s *InboundService) Get(id uint) (*models.Inbound, error) {
	var inbound models.Inbound
	if err := database.DB.First(&inbound, id).Error; err != nil {
		return nil, errors.New("入站配置不存在")
	}
	return &inbound, nil
}

// GetByTag 通过 Tag 获取入站
func (s *InboundService) GetByTag(tag string) (*models.Inbound, error) {
	var inbound models.Inbound
	if err := database.DB.Where("tag = ?", tag).First(&inbound).Error; err != nil {
		return nil, errors.New("入站配置不存在")
	}
	return &inbound, nil
}

// Create 创建入站
func (s *InboundService) Create(req *CreateInboundRequest) (*models.Inbound, error) {
	// 检查 Tag 唯一性
	var count int64
	database.DB.Model(&models.Inbound{}).Where("tag = ?", req.Tag).Count(&count)
	if count > 0 {
		return nil, errors.New("Tag 已存在")
	}

	// 检查端口占用
	database.DB.Model(&models.Inbound{}).Where("port = ?", req.Port).Count(&count)
	if count > 0 {
		return nil, errors.New("端口已被使用")
	}

	listen := req.Listen
	if listen == "" {
		listen = "0.0.0.0"
	}

	settingsJSON, err := json.Marshal(req.Settings)
	if err != nil {
		return nil, fmt.Errorf("序列化 Settings 失败: %v", err)
	}
	streamJSON, err := json.Marshal(req.StreamSettings)
	if err != nil {
		return nil, fmt.Errorf("序列化 StreamSettings 失败: %v", err)
	}
	sniffingJSON, err := json.Marshal(req.Sniffing)
	if err != nil {
		return nil, fmt.Errorf("序列化 Sniffing 失败: %v", err)
	}

	inbound := models.Inbound{
		Tag:            req.Tag,
		Protocol:       req.Protocol,
		Port:           req.Port,
		Listen:         listen,
		Settings:       string(settingsJSON),
		StreamSettings: string(streamJSON),
		Sniffing:       string(sniffingJSON),
		Enable:         true,
		Remark:         req.Remark,
	}

	if err := database.DB.Create(&inbound).Error; err != nil {
		return nil, err
	}

	return &inbound, nil
}

// Update 更新入站
func (s *InboundService) Update(id uint, req *UpdateInboundRequest) (*models.Inbound, error) {
	var inbound models.Inbound
	if err := database.DB.First(&inbound, id).Error; err != nil {
		return nil, errors.New("入站配置不存在")
	}

	updates := make(map[string]interface{})

	if req.Tag != "" && req.Tag != inbound.Tag {
		var count int64
		database.DB.Model(&models.Inbound{}).Where("tag = ? AND id != ?", req.Tag, id).Count(&count)
		if count > 0 {
			return nil, errors.New("Tag 已存在")
		}
		updates["tag"] = req.Tag
	}

	if req.Port > 0 && req.Port != inbound.Port {
		var count int64
		database.DB.Model(&models.Inbound{}).Where("port = ? AND id != ?", req.Port, id).Count(&count)
		if count > 0 {
			return nil, errors.New("端口已被使用")
		}
		updates["port"] = req.Port
	}

	if req.Protocol != "" {
		updates["protocol"] = req.Protocol
	}
	if req.Listen != "" {
		updates["listen"] = req.Listen
	}
	if req.Settings != nil {
		settingsJSON, err := json.Marshal(req.Settings)
		if err != nil {
			return nil, fmt.Errorf("序列化 Settings 失败: %v", err)
		}
		updates["settings"] = string(settingsJSON)
	}
	if req.StreamSettings != nil {
		streamJSON, err := json.Marshal(req.StreamSettings)
		if err != nil {
			return nil, fmt.Errorf("序列化 StreamSettings 失败: %v", err)
		}
		updates["stream_settings"] = string(streamJSON)
	}
	if req.Sniffing != nil {
		sniffingJSON, err := json.Marshal(req.Sniffing)
		if err != nil {
			return nil, fmt.Errorf("序列化 Sniffing 失败: %v", err)
		}
		updates["sniffing"] = string(sniffingJSON)
	}
	if req.Enable != nil {
		updates["enable"] = *req.Enable
	}
	if req.Remark != "" {
		updates["remark"] = req.Remark
	}

	if len(updates) > 0 {
		if err := database.DB.Model(&inbound).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	database.DB.First(&inbound, id)
	return &inbound, nil
}

// Delete 删除入站
func (s *InboundService) Delete(id uint) error {
	var inbound models.Inbound
	if err := database.DB.First(&inbound, id).Error; err != nil {
		return errors.New("入站配置不存在")
	}

	// 删除关联记录
	database.DB.Where("inbound_id = ?", id).Delete(&models.InboundClient{})

	return database.DB.Delete(&inbound).Error
}

// GetAllEnabled 获取所有启用的入站
func (s *InboundService) GetAllEnabled() ([]models.Inbound, error) {
	var inbounds []models.Inbound
	err := database.DB.Where("enable = ?", true).Find(&inbounds).Error
	return inbounds, err
}

// AddClient 添加客户端到入站
func (s *InboundService) AddClient(inboundID, clientID uint) error {
	var count int64
	database.DB.Model(&models.InboundClient{}).
		Where("inbound_id = ? AND client_id = ?", inboundID, clientID).
		Count(&count)
	if count > 0 {
		return errors.New("客户端已添加到此入站")
	}

	ic := models.InboundClient{
		InboundID: inboundID,
		ClientID:  clientID,
	}
	return database.DB.Create(&ic).Error
}

// RemoveClient 从入站移除客户端
func (s *InboundService) RemoveClient(inboundID, clientID uint) error {
	return database.DB.Where("inbound_id = ? AND client_id = ?", inboundID, clientID).
		Delete(&models.InboundClient{}).Error
}

// GetClients 获取入站的客户端列表
func (s *InboundService) GetClients(inboundID uint) ([]models.Client, error) {
	var clients []models.Client
	err := database.DB.Joins("JOIN inbound_clients ON inbound_clients.client_id = clients.id").
		Where("inbound_clients.inbound_id = ?", inboundID).
		Find(&clients).Error
	return clients, err
}
