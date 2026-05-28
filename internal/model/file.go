package model

import (
	"time"

	"gorm.io/gorm"
)

type PhysicalFile struct {
	ID          uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	FileHash    string         `gorm:"type:char(32);unique" json:"file_hash"`
	FileName    string         `gorm:"type:varchar(255);not null" json:"file_name"`
	FileExt     string         `gorm:"type:varchar(32);not null" json:"file_ext"`
	FileSize    int64          `json:"file_size"`
	FilePath    string         `gorm:"type:varchar(512)" json:"file_path"`
	StorageType string         `gorm:"type:varchar(32)" json:"storage_type"`
	RefCount    uint64         `json:"ref_count"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at"`
}

type UserFile struct {
	ID           uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID       uint64         `json:"user_id"`
	User         *User          `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
	PhysicalID   *uint64        `json:"physical_id"`
	PhysicalFile *PhysicalFile  `gorm:"foreignKey:PhysicalID;references:ID" json:"physical_file,omitempty"`
	ParentID     uint64         `json:"parent_id"`
	FileName     string         `gorm:"type:varchar(255)" json:"file_name"`
	FileExt      string         `gorm:"type:varchar(32)" json:"file_ext"`
	FileSize     int64          `json:"file_size"`
	PathStack    string         `gorm:"type:varchar(1024)" json:"path_stack"`
	IsDir        bool           `json:"is_dir"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"deleted_at"`
}
