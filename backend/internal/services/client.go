package services

import (
	"errors"
	"time"

	"xpanel/internal/database"
	"xpanel/internal/models"
	"xpanel/pkg/utils"
)

type ClientService struct{}

func NewClientService() *ClientService {
	return &ClientService{}
}

type CreateClientRequest struct {
	Email    string     `json:"email"`
	Remark   string     `json:"remark"`
	TotalGB  int64      `json:"total_gb"`
	ExpireAt *time.Time `json:"expire_at"`
}

type UpdateClientRequest struct {
	Email    string     `json:"email"`
	Remark   string     `json:"remark"`
	Enable   *bool      `json:"enable"`
	TotalGB  *int64     `json:"total_gb"`
	ExpireAt *time.Time `json:"expire_at"`
}

// List 获取客户端列表
func (s *ClientService) List(page, pageSize int, createdByID uint) ([]models.Client, int64, error) {
	var clients []models.Client
	var total int64

	db := database.DB.Model(&models.Client{})
	if createdByID > 0 {
		db = db.Where("created_by_id = ?", createdByID)
	}
	db.Count(&total)

	offset := (page - 1) * pageSize
	if err := db.Preload("CreatedBy").Offset(offset).Limit(pageSize).Find(&clients).Error; err != nil {
		return nil, 0, err
	}

	return clients, total, nil
}

// Get 获取单个客户端
func (s *ClientService) Get(id uint) (*models.Client, error) {
	var client models.Client
	if err := database.DB.Preload("CreatedBy").First(&client, id).Error; err != nil {
		return nil, errors.New("客户端不存在")
	}
	return &client, nil
}

// GetByUUID 通过 UUID 获取客户端
func (s *ClientService) GetByUUID(uuid string) (*models.Client, error) {
	var client models.Client
	if err := database.DB.Where("uuid = ?", uuid).First(&client).Error; err != nil {
		return nil, errors.New("客户端不存在")
	}
	return &client, nil
}

// Create 创建客户端
func (s *ClientService) Create(req *CreateClientRequest, createdByID uint) (*models.Client, error) {
	client := models.Client{
		UUID:        utils.GenerateUUID(),
		Email:       req.Email,
		Remark:      req.Remark,
		Enable:      true,
		TotalGB:     req.TotalGB * 1024 * 1024 * 1024, // GB 转 bytes
		UsedGB:      0,
		ExpireAt:    req.ExpireAt,
		CreatedByID: createdByID,
	}

	if err := database.DB.Create(&client).Error; err != nil {
		return nil, err
	}

	return &client, nil
}

// Update 更新客户端
func (s *ClientService) Update(id uint, req *UpdateClientRequest) (*models.Client, error) {
	var client models.Client
	if err := database.DB.First(&client, id).Error; err != nil {
		return nil, errors.New("客户端不存在")
	}

	updates := make(map[string]interface{})

	if req.Email != "" {
		updates["email"] = req.Email
	}
	if req.Remark != "" {
		updates["remark"] = req.Remark
	}
	if req.Enable != nil {
		updates["enable"] = *req.Enable
	}
	if req.TotalGB != nil {
		updates["total_gb"] = *req.TotalGB * 1024 * 1024 * 1024
	}
	if req.ExpireAt != nil {
		updates["expire_at"] = req.ExpireAt
	}

	if len(updates) > 0 {
		if err := database.DB.Model(&client).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	database.DB.Preload("CreatedBy").First(&client, id)
	return &client, nil
}

// Delete 删除客户端
func (s *ClientService) Delete(id uint) error {
	var client models.Client
	if err := database.DB.First(&client, id).Error; err != nil {
		return errors.New("客户端不存在")
	}

	// 删除关联记录
	database.DB.Where("client_id = ?", id).Delete(&models.InboundClient{})
	database.DB.Where("client_id = ?", id).Delete(&models.TrafficStats{})

	return database.DB.Delete(&client).Error
}

// ResetTraffic 重置流量
func (s *ClientService) ResetTraffic(id uint) error {
	return database.DB.Model(&models.Client{}).Where("id = ?", id).Update("used_gb", 0).Error
}

// UpdateTraffic 更新流量使用量
func (s *ClientService) UpdateTraffic(id uint, bytes int64) error {
	return database.DB.Model(&models.Client{}).Where("id = ?", id).
		UpdateColumn("used_gb", database.DB.Raw("used_gb + ?", bytes)).Error
}

// GetExpiredClients 获取过期的客户端
func (s *ClientService) GetExpiredClients() ([]models.Client, error) {
	var clients []models.Client
	now := time.Now()
	err := database.DB.Where("enable = ? AND expire_at < ?", true, now).Find(&clients).Error
	return clients, err
}

// GetOverQuotaClients 获取超量的客户端
func (s *ClientService) GetOverQuotaClients() ([]models.Client, error) {
	var clients []models.Client
	err := database.DB.Where("enable = ? AND total_gb > 0 AND used_gb >= total_gb", true).Find(&clients).Error
	return clients, err
}

// DisableClient 禁用客户端
func (s *ClientService) DisableClient(id uint) error {
	return database.DB.Model(&models.Client{}).Where("id = ?", id).Update("enable", false).Error
}
