package services

import (
	"fmt"

	"y-ui/internal/database"
	"y-ui/internal/models"
)

type AuditService struct{}

func NewAuditService() *AuditService {
	return &AuditService{}
}

// List 获取审计日志列表
func (s *AuditService) List(page, pageSize int, userID uint, action, resource string) ([]models.AuditLog, int64, error) {
	var logs []models.AuditLog
	var total int64

	db := database.DB.Model(&models.AuditLog{})

	if userID > 0 {
		db = db.Where("user_id = ?", userID)
	}
	if action != "" {
		db = db.Where("action = ?", action)
	}
	if resource != "" {
		db = db.Where("resource = ?", resource)
	}

	db.Count(&total)

	offset := (page - 1) * pageSize
	if err := db.Preload("User").Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// Create 创建审计日志
func (s *AuditService) Create(userID uint, action, resource string, resourceID uint, detail, ip, userAgent, status string) error {
	log := models.AuditLog{
		UserID:     userID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Detail:     detail,
		IP:         ip,
		UserAgent:  userAgent,
		Status:     status,
	}
	return database.DB.Create(&log).Error
}

// LogAction 记录操作
func (s *AuditService) LogAction(userID uint, action, resource string, resourceID uint, detail string, c interface{}) {
	// 从 gin.Context 提取信息的简化版本
	s.Create(userID, action, resource, resourceID, detail, "", "", "success")
}

// CleanOldLogs 清理旧日志
func (s *AuditService) CleanOldLogs(days int) error {
	return database.DB.Exec("DELETE FROM audit_logs WHERE created_at < datetime('now', ?)",
		fmt.Sprintf("-%d days", days)).Error
}
