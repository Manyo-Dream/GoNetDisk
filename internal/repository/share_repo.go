package repository

import (
	"errors"
	"fmt"

	"GoNetDisk/internal/model"

	"gorm.io/gorm"
)

type ShareRepo struct {
	db *gorm.DB
}

func NewShareRepo(db *gorm.DB) *ShareRepo {
	return &ShareRepo{db: db}
}

func (r *ShareRepo) CreateShare(share *model.Share) error {
	return r.db.Create(share).Error
}

func (r *ShareRepo) GetShareByCode(shareCode string) (*model.Share, error) {
	var share model.Share
	err := r.db.Where("share_code = ?", shareCode).First(&share).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("分享不存在或已失效")
		}
		return nil, fmt.Errorf("查询分享失败: %w", err)
	}
	return &share, nil
}

func (r *ShareRepo) GetSharesByUserID(userID uint64, page, pageSize int) ([]model.Share, int64, error) {
	var total int64
	var shares []model.Share

	query := r.db.Model(&model.Share{}).Where("user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计分享总数失败: %w", err)
	}
	if total == 0 {
		return []model.Share{}, 0, nil
	}

	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&shares).Error
	if err != nil {
		return nil, 0, fmt.Errorf("分页查询分享列表失败: %w", err)
	}

	return shares, total, nil
}

func (r *ShareRepo) DeleteShare(userID uint64, shareCode string) error {
	result := r.db.Where("user_id = ? AND share_code = ?", userID, shareCode).Delete(&model.Share{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("分享不存在")
	}
	return nil
}

func (r *ShareRepo) IncrViewCountBy(shareCode string, delta int64) error {
	return r.db.Model(&model.Share{}).
		Where("share_code = ?", shareCode).
		UpdateColumn("view_count", gorm.Expr("COALESCE(view_count, 0) + ?", delta)).
		Error
}
