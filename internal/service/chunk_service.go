package service

import (
	"GoNetDisk/configs"
	"GoNetDisk/internal/api"
	"GoNetDisk/internal/model"
	"GoNetDisk/internal/repository"
	"GoNetDisk/internal/util"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type ChunkService struct {
	redis       *redis.Client
	minio       *minio.Client
	userRepo    *repository.UserRepo
	fileRepo    *repository.FileRepo
	chunkRepo   *repository.ChunkRepo
	fileService *FileService
	config      *configs.Config
}

func NewChunkService(
	redis *redis.Client,
	minio *minio.Client,
	userRepo *repository.UserRepo,
	fileRepo *repository.FileRepo,
	chunkRepo *repository.ChunkRepo,
	fileService *FileService,
	config *configs.Config,
) *ChunkService {
	return &ChunkService{
		redis:       redis,
		minio:       minio,
		userRepo:    userRepo,
		fileRepo:    fileRepo,
		chunkRepo:   chunkRepo,
		fileService: fileService,
		config:      config,
	}
}

func (ch *ChunkService) InitChunkUpload(userID uint64, req *api.ChunkInitRequest) (*api.ChunkInitResponse, error) {
	ctx := context.Background()

	if err := util.ValidateName(req.FileName); err != nil {
		return nil, util.BadRequest("文件名不合法")
	}
	if len(req.FileHash) != 32 {
		return nil, util.BadRequest("文件哈希数值错误")
	}
	if req.FileSize <= 0 || req.ChunkSize <= 0 {
		return nil, util.BadRequest("文件大小或块大小数值错误")
	}

	userInfo, err := ch.userRepo.GetUserByID(userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, util.BadRequest("用户不存在")
	} else if err != nil {
		return nil, util.Internal("查询用户失败")
	}

	if userInfo.Used_Space+uint64(req.FileSize) > userInfo.Total_Space {
		return nil, util.BadRequest("用户空间不足")
	}

	if req.ParentID != 0 {
		_, err := ch.fileRepo.GetParentFolderByParentID(userID, uint64(req.ParentID))
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, util.BadRequest("父文件夹不存在")
		} else if err != nil {
			return nil, util.Internal("查找父文件夹失败")
		}
	}

	fileExt := strings.ToLower(filepath.Ext(req.FileName))
	cacheKey := "hash:" + req.FileHash

	// 1. 先查 Redis 缓存
	physicalID, err := ch.redis.Get(ctx, cacheKey).Result()
	if err != nil && err != redis.Nil {
		return nil, util.Internal("Redis查找物理文件ID失败")
	}
	if err == nil {
		// 缓存命中 → 秒传
		physicalUintID, err := strconv.ParseUint(physicalID, 10, 64)
		if err != nil {
			return nil, util.Internal("物理文件ID转换失败")
		}
		phyFile, err := ch.fileRepo.GetPhyFileByID(physicalUintID)
		if err != nil {
			return nil, util.Internal("查询物理文件失败")
		}
		return ch.instantUpload(userID, req, physicalUintID, phyFile)
	}

	// 2. 缓存未命中，查数据库
	phyFile, err := ch.fileRepo.HashDeduplication(req.FileHash)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, util.Internal("Sql查找物理文件ID失败")
	}
	if err == nil {
		// 数据库命中 → 缓存物理文件 ID + 秒传
		ch.redis.Set(ctx, cacheKey, phyFile.ID, time.Hour)
		return ch.instantUpload(userID, req, phyFile.ID, phyFile)
	}

	// 3. 新文件 → 生成 UploadID，走分片上传流程
	uploadID := uuid.New().String()
	chunkCount := util.CalcChunkCount(req.FileSize, req.ChunkSize)

	// 4. 新建 minio 分片上传
	objectKey := fmt.Sprintf("objects/%s/%s/%s%s",
		req.FileHash[:2], req.FileHash[2:4], req.FileHash, fileExt)

	core := minio.Core{Client: ch.minio}
	minioUploadID, err := core.NewMultipartUpload(ctx, ch.config.Minio.Bucket, objectKey, minio.PutObjectOptions{})
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("MinIO 初始化分片上传失败: %s", err))
	}

	// 5. minio 分片信息写入 Redis
	metaKey := "chunk:" + uploadID + ":meta"
	err = ch.redis.HSet(ctx, metaKey,
		"userID", userID,
		"fileName", req.FileName,
		"fileSize", req.FileSize,
		"fileHash", req.FileHash,
		"chunkSize", req.ChunkSize,
		"chunkCount", chunkCount,
		"parentID", req.ParentID,
		"objectKey", objectKey,
		"minioUploadID", minioUploadID,
		"status", "uploading").Err()
	if err != nil {
		core.AbortMultipartUpload(ctx, ch.config.Minio.Bucket, objectKey, minioUploadID)
		return nil, util.Internal("Redis 写入分片元信息失败")
	}
	if err = ch.redis.Expire(ctx, metaKey, 24*time.Hour).Err(); err != nil {
		core.AbortMultipartUpload(ctx, ch.config.Minio.Bucket, objectKey, minioUploadID)
		ch.redis.Del(ctx, metaKey)
		return nil, util.Internal("Redis 设置过期时间失败")
	}

	// 6. minio 分片信息写入 Mysql
	err = ch.chunkRepo.Create(&model.MultipartUpload{
		UploadID:   uploadID,
		UserID:     userID,
		ParentID:   uint64(req.ParentID),
		FileName:   req.FileName,
		FileExt:    fileExt,
		FileSize:   req.FileSize,
		FileHash:   req.FileHash,
		ObjectKey:  objectKey,
		ChunkSize:  req.ChunkSize,
		ChunkCount: int(chunkCount),
		Status:     model.ChunkStatusUploading,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	})
	if err != nil {
		core.AbortMultipartUpload(ctx, ch.config.Minio.Bucket, objectKey, minioUploadID)
		ch.redis.Del(ctx, metaKey)
		return nil, util.Internal("创建分片表记录失败")
	}

	return &api.ChunkInitResponse{
		UploadID:      uploadID,
		ChunkSize:     req.ChunkSize,
		ChunkCount:    chunkCount,
		InstantUpload: false,
	}, nil
}

// instantUpload 秒传
func (ch *ChunkService) instantUpload(
	userID uint64,
	req *api.ChunkInitRequest,
	physicalID uint64,
	phyFile *model.PhysicalFile,
) (*api.ChunkInitResponse, error) {
	if err := ch.fileRepo.IncrPhyFileRefCount(physicalID, 1); err != nil {
		return nil, util.Internal("更新文件引用数失败")
	}

	metaData, err := ch.fileService.createUserFileRecord(
		userID,
		uint64(req.ParentID),
		physicalID,
		phyFile,
	)
	if err != nil {
		return nil, util.Internal("创建用户文件记录失败")
	}

	return &api.ChunkInitResponse{
		UploadID:      "",
		ChunkSize:     0,
		ChunkCount:    0,
		InstantUpload: true,
		UserFileID:    metaData.UserFileID,
	}, nil
}

func (ch *ChunkService) UploadChunk(userID uint64, req *api.ChunkUploadRequest, chunkFile *multipart.FileHeader) (*api.ChunkUploadResponse, error) {
	ctx := context.Background()

	// 校验参数
	if len(req.UploadID) == 0 || req.ChunkIndex <= 0 {
		return nil, util.BadRequest("upload_id 或 chunk_index 无效")
	}

	metaKey := "chunk:" + req.UploadID + ":meta"
	meta, err := ch.redis.HGetAll(ctx, metaKey).Result()
	if err != nil {
		return nil, util.Internal("读取上传元信息失败")
	}
	if len(meta) == 0 {
		return nil, util.NotFound("上传任务不存在或已过期")
	}

	metaUserID, err := strconv.ParseUint(meta["userID"], 10, 64)
	if err != nil || metaUserID != userID {
		return nil, util.Forbidden("无权操作此上传任务")
	}

	if meta["status"] != "uploading" {
		return nil, util.BadRequest("上传任务已结束")
	}

	chunkCount, err := strconv.Atoi(meta["chunkCount"])
	if err != nil {
		return nil, util.Internal("解析分片数量失败")
	}
	if req.ChunkIndex < 1 || req.ChunkIndex > chunkCount {
		return nil, util.BadRequest("分片索引超出范围")
	}

	// 上传分片
	partsKey := "chunk:" + req.UploadID + ":parts"
	ok, err := ch.redis.SIsMember(ctx, partsKey, req.ChunkIndex).Result()
	if err != nil {
		return nil, util.Internal("检查分片状态失败")
	}
	if ok {
		return &api.ChunkUploadResponse{
			UploadID:   req.UploadID,
			ChunkIndex: req.ChunkIndex,
		}, nil
	}

	file, err := chunkFile.Open()
	if err != nil {
		return nil, util.Internal("读取分片文件失败")
	}
	defer file.Close()

	core := minio.Core{Client: ch.minio}
	objPart, err := core.PutObjectPart(
		ctx,
		ch.config.Minio.Bucket,
		meta["objectKey"],
		meta["minioUploadID"],
		req.ChunkIndex,
		file,
		chunkFile.Size,
		minio.PutObjectPartOptions{},
	)
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("MinIO 上传分片失败: %s", err))
	}

	etagsKey := "chunk:" + req.UploadID + ":etags"

	pipe := ch.redis.Pipeline()
	pipe.SAdd(ctx, partsKey, req.ChunkIndex)
	pipe.HSet(ctx, etagsKey, strconv.Itoa(req.ChunkIndex), objPart.ETag)
	pipe.Expire(ctx, partsKey, 24*time.Hour)
	pipe.Expire(ctx, etagsKey, 24*time.Hour)
	pipe.Expire(ctx, metaKey, 24*time.Hour)
	_, err = pipe.Exec(ctx)
	if err != nil {
		return nil, util.Internal("记录分片状态失败")
	}

	return &api.ChunkUploadResponse{
		UploadID:   req.UploadID,
		ChunkIndex: req.ChunkIndex,
	}, nil
}

func (ch *ChunkService) CompleteChunkUpload(userID uint64, req *api.ChunkCompleteRequest) (*api.ChunkCompleteResponse, error) {
	ctx := context.Background()

	if len(req.UploadID) == 0 {
		return nil, util.BadRequest("upload_id 为空")
	}

	metaKey := "chunk:" + req.UploadID + ":meta"
	meta, err := ch.redis.HGetAll(ctx, metaKey).Result()
	if err != nil {
		return nil, util.Internal("读取上传元信息失败")
	}
	if len(meta) == 0 {
		return nil, util.NotFound("上传任务不存在或已过期")
	}

	metaUserID, err := strconv.ParseUint(meta["userID"], 10, 64)
	if err != nil || metaUserID != userID {
		return nil, util.Forbidden("无权操作此上传任务")
	}

	if meta["status"] != "uploading" {
		return nil, util.BadRequest("上传任务已结束")
	}

	fileHash := meta["fileHash"]
	fileSize, _ := strconv.ParseInt(meta["fileSize"], 10, 64)
	parentID, _ := strconv.ParseUint(meta["parentID"], 10, 64)
	objectKey := meta["objectKey"]
	minioUploadID := meta["minioUploadID"]
	chunkCount, _ := strconv.Atoi(meta["chunkCount"])

	// —— 秒传检查：hash 是否已在 physical_files 表中 ——
	existingPhy, err := ch.fileRepo.HashDeduplication(fileHash)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, util.Internal("查询物理文件失败")
	}

	partsKey := "chunk:" + req.UploadID + ":parts"
	etagsKey := "chunk:" + req.UploadID + ":etags"

	if existingPhy != nil {
		// 文件已存在 → 废弃本次 MinIO 分片，走秒传
		core := minio.Core{Client: ch.minio}
		core.AbortMultipartUpload(ctx, ch.config.Minio.Bucket, objectKey, minioUploadID)

		if err := ch.fileRepo.IncrPhyFileRefCount(existingPhy.ID, 1); err != nil {
			return nil, util.Internal("更新文件引用数失败")
		}

		result, err := ch.fileService.createUserFileRecord(userID, parentID, existingPhy.ID, existingPhy)
		if err != nil {
			return nil, util.Internal("创建用户文件记录失败")
		}

		ch.chunkRepo.UpdateStatus(req.UploadID, model.ChunkStatusCompleted)
		ch.redis.Del(ctx, metaKey, partsKey, etagsKey)

		return &api.ChunkCompleteResponse{
			UserFileID: result.UserFileID,
			FileName:   result.FileName,
			FileExt:    result.FileExt,
			FileSize:   result.FIleSize,
			ParentID:   result.ParentID,
		}, nil
	}

	// —— 校验所有分片是否已上传 ——
	uploadedParts, err := ch.redis.SMembers(ctx, partsKey).Result()
	if err != nil {
		return nil, util.Internal("读取已上传分片失败")
	}
	if len(uploadedParts) != chunkCount {
		return nil, util.BadRequest(fmt.Sprintf("分片未全部上传 (%d/%d)", len(uploadedParts), chunkCount))
	}

	// —— 构建 CompletePart 列表 ——
	etagMap, err := ch.redis.HGetAll(ctx, etagsKey).Result()
	if err != nil {
		return nil, util.Internal("读取分片ETag失败")
	}

	completeParts := make([]minio.CompletePart, 0, len(uploadedParts))
	for _, partStr := range uploadedParts {
		partNum, _ := strconv.Atoi(partStr)
		completeParts = append(completeParts, minio.CompletePart{
			PartNumber: partNum,
			ETag:       etagMap[partStr],
		})
	}

	// —— MinIO 合并分片 ——
	core := minio.Core{Client: ch.minio}
	_, err = core.CompleteMultipartUpload(
		ctx,
		ch.config.Minio.Bucket,
		objectKey,
		minioUploadID,
		completeParts,
		minio.PutObjectOptions{},
	)
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("MinIO 合并分片失败: %s", err))
	}

	fileExt := strings.ToLower(filepath.Ext(meta["fileName"]))
	phyFile := &model.PhysicalFile{
		FileHash:    fileHash,
		FileName:    meta["fileName"],
		FileExt:     fileExt,
		FileSize:    fileSize,
		FilePath:    objectKey,
		StorageType: "minio",
		RefCount:    1,
	}

	if err := ch.fileRepo.CreatePhyFile(phyFile); err != nil {
		return nil, util.Internal("创建物理文件记录失败")
	}

	// —— Redis 缓存 hash → physicalID（秒传用） ——
	cacheKey := "hash:" + fileHash
	ch.redis.Set(ctx, cacheKey, phyFile.ID, time.Hour)

	// —— 创建用户文件记录 ——
	result, err := ch.fileService.createUserFileRecord(userID, parentID, phyFile.ID, phyFile)
	if err != nil {
		return nil, util.Internal("创建用户文件记录失败")
	}

	// —— 更新 MySQL 状态 ——
	ch.chunkRepo.UpdateStatus(req.UploadID, model.ChunkStatusCompleted)

	// —— 清理 Redis 临时数据 ——
	ch.redis.Del(ctx, metaKey, partsKey, etagsKey)

	return &api.ChunkCompleteResponse{
		UserFileID: result.UserFileID,
		FileName:   result.FileName,
		FileExt:    result.FileExt,
		FileSize:   result.FIleSize,
		ParentID:   result.ParentID,
	}, nil
}

func (ch *ChunkService) GetChunkStatus(userID uint64, req *api.ChunkStatusRequest) (*api.ChunkStatusResponse, error) {
	ctx := context.Background()

	if len(req.UploadID) == 0 {
		return nil, util.BadRequest("上传任务 ID 错误")
	}

	metaKey := "chunk:" + req.UploadID + ":meta"
	meta, err := ch.redis.HGetAll(ctx, metaKey).Result()
	if err != nil {
		return nil, util.Internal("读取上传状态失败")
	}

	if len(meta) > 0 {
		return ch.buildStatusFromRedis(ctx, userID, req.UploadID, meta)
	}

	record, err := ch.chunkRepo.FindByUploadID(req.UploadID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, util.NotFound("上传任务不存在")
	}
	if err != nil {
		return nil, util.Internal("查询上传任务失败")
	}
	if record.UserID != userID {
		return nil, util.Forbidden("无权查看此上传任务")
	}

	return &api.ChunkStatusResponse{
		UploadID:       record.UploadID,
		FileName:       record.FileName,
		ChunkSize:      record.ChunkSize,
		ChunkCount:     record.ChunkCount,
		Status:         record.Status,
		UploadedChunks: nil,
	}, nil
}

func (ch *ChunkService) buildStatusFromRedis(ctx context.Context, userID uint64, uploadID string, meta map[string]string) (*api.ChunkStatusResponse, error) {
	metaUserID, err := strconv.ParseUint(meta["userID"], 10, 64)
	if err != nil || metaUserID != userID {
		return nil, util.Forbidden("无权查看此上传任务")
	}

	chunkCount, _ := strconv.Atoi(meta["chunkCount"])
	chunkSize, _ := strconv.ParseInt(meta["chunkSize"], 10, 64)

	status := model.ChunkStatusUploading
	if meta["status"] == "completed" {
		status = model.ChunkStatusCompleted
	}

	partsKey := "chunk:" + uploadID + ":parts"
	partStrs, err := ch.redis.SMembers(ctx, partsKey).Result()
	if err != nil {
		return nil, util.Internal("读取上传进度失败")
	}

	uploadedChunks := make([]int, 0, len(partStrs))
	for _, s := range partStrs {
		n, _ := strconv.Atoi(s)
		uploadedChunks = append(uploadedChunks, n)
	}

	return &api.ChunkStatusResponse{
		UploadID:       uploadID,
		FileName:       meta["fileName"],
		ChunkSize:      chunkSize,
		ChunkCount:     chunkCount,
		Status:         status,
		UploadedChunks: uploadedChunks,
	}, nil
}
