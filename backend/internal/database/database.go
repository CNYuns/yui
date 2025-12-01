package database

import (
	"y-ui/internal/config"
	"y-ui/internal/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Init(cfg *config.DatabaseConfig) error {
	var err error

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	switch cfg.Driver {
	case "sqlite":
		DB, err = gorm.Open(sqlite.Open(cfg.DSN), gormConfig)
	default:
		DB, err = gorm.Open(sqlite.Open(cfg.DSN), gormConfig)
	}

	if err != nil {
		return err
	}

	// 自动迁移
	if err := autoMigrate(); err != nil {
		return err
	}

	return nil
}

func autoMigrate() error {
	return DB.AutoMigrate(
		&models.User{},
		&models.Client{},
		&models.Inbound{},
		&models.Outbound{},
		&models.Certificate{},
		&models.TrafficStats{},
		&models.AuditLog{},
		&models.InboundClient{},
	)
}

func Close() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
