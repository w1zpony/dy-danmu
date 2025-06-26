package model

import (
	"danmu-core/logger"
	"danmu-core/setting"
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func init() {
	var dsn = fmt.Sprintf("user=%s password=%s host=%s dbname=%s port=%s sslmode=disable search_path=%s",
		setting.DatabaseSetting.User,
		setting.DatabaseSetting.Password,
		setting.DatabaseSetting.Host,
		setting.DatabaseSetting.DBName,
		setting.DatabaseSetting.Port,
		setting.DatabaseSetting.SearchPath)
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		logger.Fatal().Err(err).Msg("db.Setup failure")
		return
	}
	/*	sqlDB, err := DB.DB()
		if err != nil {
			log.Fatalf("failed to get Sql DB: %v", err)
		}
		sqlDB.SetMaxIdleConns(settings.DatabaseSetting.MaxIdleConns)
		sqlDB.SetMaxOpenConns(settings.DatabaseSetting.MaxOpenConns)*/
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
