package database

import (
	"fmt"

	"github.com/rozoomcool/sihkaromicro/projects/internal/config"
	"github.com/rozoomcool/sihkaromicro/projects/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func New(cfg config.DBConf) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DBDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // В проде: logger.Warn
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Настройка пула соединений
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)       // 10
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)       // 100
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime) // 1 час

	// Проверка подключения
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}

// AutoMigrate - для разработки (в проде лучше использовать golang-migrate)
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.Project{},
	)
}
