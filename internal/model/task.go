package model

import (
	"time"

	"gorm.io/gorm"
)

// 任务状态常量
const (
	UploadTaskStatusProcessing = 1
	UploadTaskStatusSuccess    = 2
	UploadTaskStatusFail       = 3
	UploadTaskStatusAllFailed  = 4
)

// 单任务状态常量
const (
	FileStatusWaiting = 1
	FileStatusSuccess = 2
	FileStatusFailed  = 3
	FileStatusSkip    = 4
)

type UploadTask struct {
	ID           uint64         `gorm:"primaryKey;autoIncrement"`
	TaskID       string         `gorm:"column:task_id;type:char(36);uniqueIndex"`
	UserID       uint64         `gorm:"column:user_id"`
	ParentID     uint64         `gorm:"column:parent_id"`
	Status       int            `gorm:"column:status"`
	FileCount    int            `gorm:"column:file_count"`
	SuccessCount int            `gorm:"column:success_count"`
	FailCount    int            `gorm:"column:fail_count"`
	TotalSize    int64          `gorm:"column:total_size"`
	ErrorMsg     string         `gorm:"column:error_msg;type:varchar(512)"`
	CreatedAt    time.Time      `gorm:"column:created_at"`
	UpdatedAt    time.Time      `gorm:"column:updated_at"`
	DeleteAt     gorm.DeletedAt `gorm:"column:delete_at"`
}

type UploadFileRecord struct {
	ID           uint64         `gorm:"primaryKey;autoIncrement"`
	TaskID       string         `gorm:"column:task_id;type:char(36);index"`
	UserID       uint64         `gorm:"column:user_id"`
	FileIndex    int            `gorm:"column:file_index"`
	FileName     string         `gorm:"column:file_name;type:varchar(255)"`
	FileExt      string         `gorm:"column:file_ext;type:varchar(32)"`
	FileSize     int64          `gorm:"column:file_size"`
	FileHash     string         `gorm:"column:file_hash;type:varchar(64)"`
	Status       int            `gorm:"column:status"`
	RelativePath string         `gorm:"column:relative_path;type:varchar(512)"`
	PhysicalID   uint64         `gorm:"column:physical_id"`
	UserFileID   uint64         `gorm:"column:userfile_id"`
	ErrorMsg     string         `gorm:"column:error_msg;type:varchar(512)"`
	CreatedAt    time.Time      `gorm:"column:created_at"`
	UpdatedAt    time.Time      `gorm:"column:updated_at"`
	DeleteAt     gorm.DeletedAt `gorm:"column:delete_at"`
}
