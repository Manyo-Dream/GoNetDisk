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
	TaskID       string         `gorm:"column:task_id"`
	UserID       uint64         `gorm:"column:user_id"`
	ParentID     uint64         `gorm:"column:parent_id"`
	Status       int            `gorm:"column:status"`
	FileCount    int            `gorm:"column:file_count"`
	SuccessCount int            `gorm:"column:success_count"`
	FailCount    int            `gorm:"column:fail_count"`
	TotalSize    int64          `gorm:"column:total_size"`
	ErrorMsg     string         `gorm:"column:error_msg"`
	CreatedAt    time.Time      `gorm:"column:created_at"`
	UpdatedAt    time.Time      `gorm:"column:updated_at"`
	DeleteAt     gorm.DeletedAt `gorm:"column:delete_at"`
}

type UploadFileRecord struct {
	ID           uint64         `gorm:"primaryKey;autoIncrement"`
	TaskID       string         `gorm:"column:task_id"`
	UserID       uint64         `gorm:"column:user_id"`
	FileIndex    int            `gorm:"column:file_index"`
	FileName     string         `gorm:"column:file_name"`
	FileExt      string         `gorm:"column:file_ext"`
	FileSize     int64          `gorm:"column:file_size"`
	FileHash     string         `gorm:"column:file_hash"`
	Status       int            `gorm:"column:status"`
	RelativePath string         `gorm:"column:relative_path"`
	PhysicalID   uint64         `gorm:"column:physical_id"`
	UserFileID   uint64         `gorm:"column:userfile_id"`
	ErrorMsg     string         `gorm:"column:error_msg"`
	CreatedAt    time.Time      `gorm:"column:created_at"`
	UpdatedAt    time.Time      `gorm:"column:updated_at"`
	DeleteAt     gorm.DeletedAt `gorm:"column:delete_at"`
}
