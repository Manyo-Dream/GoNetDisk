package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"GoNetDisk/configs"
	"GoNetDisk/internal/api"
	"GoNetDisk/internal/model"
	"GoNetDisk/internal/repository"
	"GoNetDisk/internal/util"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type ShareService struct {
	shareRepo   *repository.ShareRepo
	fileRepo    *repository.FileRepo
	userRepo    *repository.UserRepo
	redis       *redis.Client
	minioClient *minio.Client
	config      *configs.Config
}

func NewShareService(shareRepo *repository.ShareRepo, fileRepo *repository.FileRepo, userRepo *repository.UserRepo, redis *redis.Client, minioClient *minio.Client, config *configs.Config) *ShareService {
	s := &ShareService{
		shareRepo:   shareRepo,
		fileRepo:    fileRepo,
		userRepo:    userRepo,
		redis:       redis,
		minioClient: minioClient,
		config:      config,
	}

	go s.syncViewCounts()

	return s
}

func (s *ShareService) CreateShare(userID, userFileID uint64, code string, expireDays int) (*api.ShareCreateResponse, error) {
	userFile, err := s.fileRepo.GetUserFileByIDAny(userID, userFileID)
	if err != nil {
		return nil, util.NotFound(fmt.Sprintf("文件或文件夹不存在: %s", err))
	}
	if userFile.DeletedAt.Valid {
		return nil, util.BadRequest("回收站中的文件无法创建分享")
	}

	var expireAt *time.Time
	if expireDays > 0 {
		t := time.Now().Add(time.Duration(expireDays) * 24 * time.Hour)
		expireAt = &t
	}

	shareCode := uuid.New().String()

	share := &model.Share{
		ShareCode:  shareCode,
		UserID:     userID,
		UserFileID: userFileID,
		Code:       code,
		ExpireAt:   expireAt,
	}

	err = s.shareRepo.CreateShare(share)
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("创建分享失败: %s", err))
	}

	return &api.ShareCreateResponse{
		ShareCode: shareCode,
		Code:      code,
		ExpireAt:  expireAt,
		CreatedAt: share.CreatedAt,
	}, nil
}

func (s *ShareService) GetUserShareList(userID uint64, page, pageSize int) (*api.ShareListResponse, error) {
	shares, total, err := s.shareRepo.GetSharesByUserID(userID, page, pageSize)
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("获取分享列表失败: %s", err))
	}

	items := make([]api.ShareItem, 0, len(shares))

	for _, share := range shares {
		redisCount, _ := s.redis.Get(context.Background(), "share:view:"+share.ShareCode).Int64()
		displayCount := share.ViewCount + uint64(redisCount)

		userFile, err := s.fileRepo.GetUserFileByIDAny(userID, share.UserFileID)
		if err != nil {
			userFile = &model.UserFile{FileName: "已删除", IsDir: false}
		}

		items = append(items, api.ShareItem{
			ShareCode: share.ShareCode,
			FileName:  userFile.FileName,
			IsDir:     userFile.IsDir,
			Code:      share.Code,
			ExpireAt:  share.ExpireAt,
			ViewCount: displayCount,
			CreatedAt: share.CreatedAt,
		})
	}

	return &api.ShareListResponse{
		List:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *ShareService) GetShareInfo(shareCode, code string) (*api.ShareInfoResponse, error) {
	share, err := s.shareRepo.GetShareByCode(shareCode)
	if err != nil {
		return nil, util.NotFound(err.Error())
	}

	if share.ExpireAt != nil && time.Now().After(*share.ExpireAt) {
		return nil, util.NotFound("分享已过期")
	}

	if share.Code != "" && share.Code != code {
		return nil, util.Forbidden("提取码错误")
	}

	userFile, err := s.fileRepo.GetUserFileByIDAny(share.UserID, share.UserFileID)
	if err != nil {
		return nil, util.NotFound("分享的文件已被删除")
	}

	return &api.ShareInfoResponse{
		ShareCode: share.ShareCode,
		FileName:  userFile.FileName,
		FileExt:   userFile.FileExt,
		FileSize:  userFile.FileSize,
		IsDir:     userFile.IsDir,
		ExpireAt:  share.ExpireAt,
		HasCode:   share.Code != "",
	}, nil
}

func (s *ShareService) RevokeShare(userID uint64, shareCode string) error {
	err := s.shareRepo.DeleteShare(userID, shareCode)
	if err != nil {
		return util.NotFound(fmt.Sprintf("取消分享失败: %s", err))
	}
	return nil
}

func (s *ShareService) DownloadSharedFile(shareCode, code string, userID uint64) (*api.FileDownloadResponse, io.ReadCloser, error) {
	share, err := s.shareRepo.GetShareByCode(shareCode)
	if err != nil {
		return nil, nil, util.NotFound(err.Error())
	}

	if share.ExpireAt != nil && time.Now().After(*share.ExpireAt) {
		return nil, nil, util.NotFound("分享已过期")
	}

	if share.Code != "" && share.Code != code {
		return nil, nil, util.Forbidden("提取码错误")
	}

	userFile, phyFile, err := s.fileRepo.GetFileByDownloadReq(share.UserID, share.UserFileID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, util.NotFound("文件不存在")
	} else if err != nil {
		return nil, nil, util.Internal(fmt.Sprintf("查询文件失败: %s", err))
	}

	if phyFile.FilePath == "" {
		return nil, nil, util.NotFound("文件路径为空")
	}

	obj, err := s.minioClient.GetObject(
		context.Background(),
		s.config.Minio.Bucket,
		phyFile.FilePath,
		minio.GetObjectOptions{},
	)
	if err != nil {
		return nil, nil, util.Internal(fmt.Sprintf("从 MinIO 读取文件失败: %s", err))
	}

	s.redis.Incr(context.Background(), "share:view:"+shareCode)

	return &api.FileDownloadResponse{
		FileName:    userFile.FileName,
		StorageType: phyFile.StorageType,
		FileExt:     userFile.FileExt,
		FileSize:    phyFile.FileSize,
	}, obj, nil
}

func (s *ShareService) syncViewCounts() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		var cursor uint64

		for {
			keys, nextCursor, err := s.redis.Scan(ctx, cursor, "share:view:*", 100).Result()
			if err != nil {
				break
			}

			for _, key := range keys {
				count, err := s.redis.GetDel(ctx, key).Int64()
				if err == nil && count > 0 {
					shareCode := strings.TrimPrefix(key, "share:view:")
					s.shareRepo.IncrViewCountBy(shareCode, count)
				}
			}

			cursor = nextCursor
			if cursor == 0 {
				break
			}
		}
	}
}
