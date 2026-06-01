package repository

import (
	"GoNetDisk/internal/model"
	"time"

	"gorm.io/gorm"
)

type ChunkRepo struct {
	db *gorm.DB
}

func NewChunkRepo(db *gorm.DB) *ChunkRepo {
	return &ChunkRepo{db: db}
}

// WithTx 返回绑定到事务的新实例，方法调用将走同一个事务
func (cr *ChunkRepo) WithTx(tx *gorm.DB) *ChunkRepo {
	return &ChunkRepo{db: tx}
}

func (cr *ChunkRepo) Create(m *model.MultipartUpload) error {
	return cr.db.Model(&model.MultipartUpload{}).Create(m).Error
}

func (cr *ChunkRepo) FindByUploadID(uploadID string) (*model.MultipartUpload, error) {
	var multiUploadInfo *model.MultipartUpload
	err := cr.db.Model(&model.MultipartUpload{}).Where("upload_id = ?", uploadID).First(&multiUploadInfo).Error
	if err != nil {
		return nil, err
	}

	return multiUploadInfo, nil
}

func (cr *ChunkRepo) UpdateStatus(uploadID string, status int) error {
	return cr.db.Model(&model.MultipartUpload{}).
		Where("upload_id = ?", uploadID).
		Update("status", status).Error
}

func (cr *ChunkRepo) FindExpired(d time.Duration) ([]*model.MultipartUpload, error) {
	var expired []*model.MultipartUpload
	err := cr.db.Model(&model.MultipartUpload{}).
		Where("status = ? AND updated_at < ?", model.ChunkStatusUploading, time.Now().Add(-d)).
		Find(&expired).Error
	if err != nil {
		return nil, err
	}
	return expired, nil
}
