package database

import (
	"fmt"
	"time"

	"GoNetDisk/configs"
	"GoNetDisk/internal/model"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

func InitDB(dsn string, dataConfig *configs.DatabaseConfig) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true,
		},
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("获取底层数据库示例失败：%w", err)
	}

	sqlDB.SetMaxOpenConns(dataConfig.MaxOpenConns)
	sqlDB.SetMaxIdleConns(dataConfig.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(dataConfig.ConnMaxLifetimeMin * time.Minute)
	sqlDB.SetConnMaxIdleTime(dataConfig.ConnMaxIdleTimeMin * time.Minute)

	if err := db.AutoMigrate(
		&model.User{},
		&model.PhysicalFile{},
		&model.UserFile{},
		&model.UploadTask{},
		&model.UploadFileRecord{},
		&model.MultipartUpload{},
		&model.Share{},
	); err != nil {
		return nil, fmt.Errorf("自动迁移操作失败: %w", err)
	}

	return db, nil
}
