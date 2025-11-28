package services

import (
	"time"

	"xpanel/internal/database"
	"xpanel/internal/models"
)

type TrafficService struct{}

func NewTrafficService() *TrafficService {
	return &TrafficService{}
}

type TrafficSummary struct {
	TotalUpload   int64 `json:"total_upload"`
	TotalDownload int64 `json:"total_download"`
	TotalClients  int64 `json:"total_clients"`
	ActiveClients int64 `json:"active_clients"`
}

// GetClientTraffic 获取客户端流量统计
func (s *TrafficService) GetClientTraffic(clientID uint, startDate, endDate string) ([]models.TrafficStats, error) {
	var stats []models.TrafficStats
	db := database.DB.Where("client_id = ?", clientID)

	if startDate != "" {
		db = db.Where("date >= ?", startDate)
	}
	if endDate != "" {
		db = db.Where("date <= ?", endDate)
	}

	err := db.Order("date DESC").Find(&stats).Error
	return stats, err
}

// GetInboundTraffic 获取入站流量统计
func (s *TrafficService) GetInboundTraffic(inboundID uint, startDate, endDate string) ([]models.TrafficStats, error) {
	var stats []models.TrafficStats
	db := database.DB.Where("inbound_id = ?", inboundID)

	if startDate != "" {
		db = db.Where("date >= ?", startDate)
	}
	if endDate != "" {
		db = db.Where("date <= ?", endDate)
	}

	err := db.Order("date DESC").Find(&stats).Error
	return stats, err
}

// GetSummary 获取流量汇总
func (s *TrafficService) GetSummary() (*TrafficSummary, error) {
	var summary TrafficSummary

	// 总上传下载
	database.DB.Model(&models.TrafficStats{}).
		Select("COALESCE(SUM(upload), 0) as total_upload, COALESCE(SUM(download), 0) as total_download").
		Scan(&summary)

	// 总客户端数
	database.DB.Model(&models.Client{}).Count(&summary.TotalClients)

	// 活跃客户端数（有流量的）
	database.DB.Model(&models.Client{}).Where("enable = ? AND used_gb > 0", true).Count(&summary.ActiveClients)

	return &summary, nil
}

// RecordTraffic 记录流量
func (s *TrafficService) RecordTraffic(clientID, inboundID uint, upload, download int64) error {
	date := time.Now().Format("2006-01-02")

	var stats models.TrafficStats
	result := database.DB.Where("client_id = ? AND inbound_id = ? AND date = ?", clientID, inboundID, date).First(&stats)

	if result.RowsAffected == 0 {
		// 创建新记录
		stats = models.TrafficStats{
			ClientID:  clientID,
			InboundID: inboundID,
			Upload:    upload,
			Download:  download,
			Date:      date,
		}
		return database.DB.Create(&stats).Error
	}

	// 更新现有记录
	return database.DB.Model(&stats).Updates(map[string]interface{}{
		"upload":   stats.Upload + upload,
		"download": stats.Download + download,
	}).Error
}

// GetDailyStats 获取每日统计
func (s *TrafficService) GetDailyStats(days int) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	rows, err := database.DB.Model(&models.TrafficStats{}).
		Select("date, SUM(upload) as upload, SUM(download) as download").
		Where("date >= ?", startDate).
		Group("date").
		Order("date ASC").
		Rows()

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var date string
		var upload, download int64
		rows.Scan(&date, &upload, &download)
		results = append(results, map[string]interface{}{
			"date":     date,
			"upload":   upload,
			"download": download,
		})
	}

	return results, nil
}

// CleanOldStats 清理旧统计数据
func (s *TrafficService) CleanOldStats(days int) error {
	cutoffDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
	return database.DB.Where("date < ?", cutoffDate).Delete(&models.TrafficStats{}).Error
}
