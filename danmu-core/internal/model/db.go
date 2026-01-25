package model

import (
	"danmu-core/conf"
	"danmu-core/logger"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init(cfg *conf.Database) error {
	var dsn = fmt.Sprintf("user=%s password=%s host=%s dbname=%s port=%s sslmode=disable search_path=%s",
		cfg.User,
		cfg.Password,
		cfg.Host,
		cfg.DBName,
		cfg.Port,
		cfg.SearchPath)
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.Error().Err(err).Msg("db.Setup failure")
		return err
	}
	/*	sqlDB, err := DB.DB()
		if err != nil {
			log.Fatalf("failed to get Sql DB: %v", err)
		}
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)*/
	return nil
}

// Close closes the database connection
func Close() {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			logger.Error().Err(err).Msg("Error getting SQL DB handle")
			return
		}
		if err := sqlDB.Close(); err != nil {
			logger.Error().Err(err).Msg("Error closing database connection")
		}
	}
}
