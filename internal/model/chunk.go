package model

import "time"

const (
	ChunkStatusUploading = 1
	ChunkStatusCompleted = 2
	ChunkStatusFailed    = 3
	ChunkStatusExpired   = 4
)

type MultipartUpload struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement"`
	UploadID   string    `gorm:"column:upload_id;type:char(36);uniqueIndex"`
	UserID     uint64    `gorm:"column:user_id;index"`
	ParentID   uint64    `gorm:"column:parent_id"`
	FileName   string    `gorm:"column:file_name;type:varchar(255)"`
	FileExt    string    `gorm:"column:file_ext;type:varchar(32)"`
	FileSize   int64     `gorm:"column:file_size"`
	FileHash   string    `gorm:"column:file_hash;type:varchar(32)"`
	ObjectKey  string    `gorm:"column:object_key;type:varchar(255)"`
	ChunkSize  int64     `gorm:"column:chunk_size"`
	ChunkCount int       `gorm:"column:chunk_count"`
	Status     int       `gorm:"column:status"`
	CreatedAt  time.Time `gorm:"column:created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at"`
}
