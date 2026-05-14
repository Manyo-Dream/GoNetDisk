package model

import (
	"time"

	"gorm.io/gorm"
)

type Share struct {
	ID         uint64         `gorm:"primaryKey"`
	ShareCode  string         `gorm:"type:char(36);uniqueIndex;not null"`
	UserID     uint64         `gorm:"not null;index"`
	UserFileID uint64         `gorm:"not null"`
	Code       string         `gorm:"type:varchar(64);default:''"`
	ExpireAt   *time.Time
	ViewCount  uint64         `gorm:"default:0"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}
