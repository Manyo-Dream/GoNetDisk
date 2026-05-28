package service

import (
	"bytes"
	"context"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"GoNetDisk/configs"
	"GoNetDisk/internal/api"
	"GoNetDisk/internal/model"
	"GoNetDisk/internal/repository"
	"GoNetDisk/internal/util"

	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

const lockScript = `
	if redis.call('exists', KEYS[1]) == 0 then
		redis.call('set', KEYS[1], ARGV[1], 'ex', ARGV[2])
		return 1
	end
	return 0
`

const unlockScript = `
	if redis.call('get', KEYS[1]) == ARGV[1] then
		return redis.call('del', KEYS[1])
	end
	return 0
`

type FileService struct {
	redis       *redis.Client
	minioClient *minio.Client
	userRepo    *repository.UserRepo
	fileRepo    *repository.FileRepo
	jwtManger   *util.JWTManager
	config      *configs.Config
}

func NewFileService(redis *redis.Client, minioClient *minio.Client, userRepo *repository.UserRepo, fileRepo *repository.FileRepo, jwtManger *util.JWTManager, config *configs.Config) *FileService {
	return &FileService{
		redis:       redis,
		minioClient: minioClient,
		userRepo:    userRepo,
		fileRepo:    fileRepo,
		jwtManger:   jwtManger,
		config:      config,
	}
}

func (s *FileService) UploadPhyFileAndBindFile(email string, parentID uint64, fileHeader *multipart.FileHeader) (*api.FileUploadResponse, error) {
	validResult, err := s.validFile(fileHeader)
	if err != nil {
		return nil, util.BadRequest(fmt.Sprintf("验证文件信息失败: %s", err))
	}

	userInfo, err := s.userRepo.GetByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, util.NotFound(fmt.Sprintf("用户不存在: %s", email))
		}
		return nil, util.Internal(fmt.Sprintf("查询用户信息失败: %s", err))
	}

	if userInfo.Total_Space > 0 && userInfo.Used_Space+uint64(validResult.FileSize) > userInfo.Total_Space {
		return nil, util.Conflict(fmt.Sprintf("空间不足：已用%d，总计%d", userInfo.Used_Space, userInfo.Total_Space))
	}

	if parentID != 0 {
		_, err := s.fileRepo.GetParentFolderByParentID(userInfo.ID, parentID)
		if err != nil {
			return nil, util.NotFound(fmt.Sprintf("父文件夹不存在: %s", err))
		}
	}

	src, err := fileHeader.Open()
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("打开文件流失败: %s", err))
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("读取文件失败: %s", err))
	}

	physicalID, _, _, err := s.findOrCreatePhysicalFile(data, validResult.FileName, validResult.FileSize)
	if err != nil {
		return nil, err
	}
	if physicalID == 0 {
		return nil, util.Internal("创建物理文件记录异常: physical_id 为 0，请检查 physical_file 表是否配置了 AUTO_INCREMENT")
	}

	return s.createUserFileRecord(userInfo.ID, parentID, physicalID, validResult)
}

// findOrCreatePhysicalFile 计算 hash、查重、上传到 MinIO、创建 PhysicalFile 记录
// 重复文件不上传，直接复用并引用计数+1。返回 physicalID、fileHash、fileExt
func (s *FileService) findOrCreatePhysicalFile(data []byte, fileName string, fileSize int64) (physicalID uint64, fileHash string, fileExt string, err error) {
	fileHash = fmt.Sprintf("%x", md5.Sum(data))
	fileExt = strings.ToLower(filepath.Ext(fileName))

	cacheKey := "hash:" + fileHash
	lockKey := "lock:hash:" + fileHash
	lockValue := fmt.Sprintf("%d", time.Now().UnixNano()) // 唯一标识，防止误删锁

	// 查缓存
	if id, ok := s.tryUseCache(cacheKey); ok {
		return id, fileHash, fileExt, nil
	}

	// 尝试获取锁
	locked, err := s.redis.Eval(
		context.Background(),
		lockScript,
		[]string{lockKey},
		lockValue,
		2).Result()
	if err != nil || locked.(int64) == 0 {
		// 获取锁失败，等待重试
		return s.waitAndRetry(cacheKey, fileHash, fileExt)
	}

	// 确保释放锁
	defer s.redis.Eval(
		context.Background(),
		unlockScript,
		[]string{lockKey},
		lockValue)

	// 双重检查
	if id, ok := s.tryUseCache(cacheKey); ok {
		return id, fileHash, fileExt, nil
	}

	// 查询或创建文件
	return s.createPhysicalFile(data, fileName, fileSize, fileHash, fileExt, cacheKey)
}

// 辅助方法：尝试从缓存获取
func (s *FileService) tryUseCache(cacheKey string) (uint64, bool) {
	val, err := s.redis.Get(context.Background(), cacheKey).Result()
	if err != nil {
		return 0, false
	}

	id, parseErr := strconv.ParseUint(val, 10, 64)
	if parseErr != nil {
		s.redis.Del(context.Background(), cacheKey)
		return 0, false
	}
	if id == 0 {
		s.redis.Del(context.Background(), cacheKey)
		return 0, false
	}

	if incrErr := s.fileRepo.IncrPhyFileRefCount(id, 1); incrErr != nil {
		s.redis.Del(context.Background(), cacheKey)
		return 0, false
	}

	return id, true
}

// 辅助方法：等待并重试
func (s *FileService) waitAndRetry(cacheKey, fileHash, fileExt string) (uint64, string, string, error) {
	for i := range 5 {
		time.Sleep(time.Duration(50*(i+1)) * time.Millisecond)

		if id, ok := s.tryUseCache(cacheKey); ok {
			return id, fileHash, fileExt, nil
		}
	}

	return 0, "", "", util.Internal("文件处理超时，请重试")
}

// 辅助方法：创建物理文件
func (s *FileService) createPhysicalFile(data []byte, fileName string, fileSize int64,
	fileHash, fileExt, cacheKey string) (uint64, string, string, error) {

	hashResult, err := s.fileRepo.HashDeduplication(fileHash)
	if err == nil {
		err = s.fileRepo.IncrPhyFileRefCount(hashResult.ID, 1)
		if err != nil {
			return 0, "", "", util.Internal(fmt.Sprintf("更新物理文件引用数失败: %s", err))
		}
		s.redis.Set(context.Background(), cacheKey, strconv.FormatUint(hashResult.ID, 10), time.Hour)
		return hashResult.ID, fileHash, fileExt, nil
	}

	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, "", "", util.Internal(fmt.Sprintf("哈希查重失败: %s", err))
	}

	objectKey := fmt.Sprintf("objects/%s/%s/%s%s",
		fileHash[:2], fileHash[2:4], fileHash, fileExt)

	// 上传到 MinIO
	_, err = s.minioClient.PutObject(
		context.Background(),
		s.config.Minio.Bucket,
		objectKey,
		bytes.NewReader(data),
		int64(len(data)),
		minio.PutObjectOptions{},
	)
	if err != nil {
		return 0, "", "", util.Internal(fmt.Sprintf("上传到 MinIO 失败: %s", err))
	}

	// 创建数据库记录
	phyFile := &model.PhysicalFile{
		FileHash:    fileHash,
		FileName:    fileName,
		FileExt:     fileExt,
		FileSize:    fileSize,
		FilePath:    objectKey,
		StorageType: "minio",
		RefCount:    1,
	}

	err = s.fileRepo.CreatePhyFile(phyFile)
	if err != nil {
		// 清理 MinIO
		s.minioClient.RemoveObject(context.Background(), s.config.Minio.Bucket,
			objectKey, minio.RemoveObjectOptions{})
		return 0, "", "", util.Internal(fmt.Sprintf("创建物理文件记录失败: %s", err))
	}

	if phyFile.ID == 0 {
		s.minioClient.RemoveObject(context.Background(), s.config.Minio.Bucket,
			objectKey, minio.RemoveObjectOptions{})
		return 0, "", "", util.Internal("创建物理文件记录异常: 数据库表 physical_file 可能缺少 AUTO_INCREMENT，请执行: ALTER TABLE physical_file MODIFY COLUMN id BIGINT UNSIGNED AUTO_INCREMENT")
	}

	s.redis.Set(context.Background(), cacheKey, strconv.FormatUint(phyFile.ID, 10), time.Hour)
	return phyFile.ID, fileHash, fileExt, nil
}

func (s *FileService) DownloadUserFile(userID, userFileID uint64) (*api.FileDownloadResponse, io.ReadCloser, error) {
	userFile, phyFile, err := s.fileRepo.GetFileByDownloadReq(userID, userFileID)
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

	return &api.FileDownloadResponse{
		FileName:    userFile.FileName,
		StorageType: phyFile.StorageType,
		FileExt:     userFile.FileExt,
		FileSize:    phyFile.FileSize,
	}, obj, nil
}

func (s *FileService) GetUserFileList(userID, parentID uint64, page, pageSize int, sortBy, orderBy string) (*api.FileListResponse, error) {
	switch {
	case page < 1:
		page = 1
	}
	switch {
	case pageSize < 1:
		pageSize = 5
	case pageSize > 100:
		pageSize = 100
	}
	switch sortBy {
	case "":
		sortBy = "updated_at"
	case "file_name", "file_size", "created_at", "updated_at":
	default:
		sortBy = "updated_at"
	}
	switch orderBy {
	case "":
		orderBy = "desc"
	case "asc", "desc":
	default:
		orderBy = "desc"
	}

	userFileList, total, err := s.fileRepo.GetUserFileList(userID, parentID, page, pageSize, sortBy, orderBy)
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("获取文件列表失败: %s", err))
	}

	return &api.FileListResponse{
		List:     userFileList,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *FileService) GetTrashList(userID uint64, page, pageSize int) (*api.TrashListResponse, error) {
	switch {
	case page < 1:
		page = 1
	}
	switch {
	case pageSize < 1:
		pageSize = 5
	case pageSize > 100:
		pageSize = 100
	}

	trashFileList, total, err := s.fileRepo.GetTrashFileList(userID, page, pageSize)
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("获取文件列表失败: %s", err))
	}

	return &api.TrashListResponse{
		List:     trashFileList,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *FileService) MoveFileToTrash(userID, userFileID uint64) (*api.TrashDeleteResponse, error) {
	userFile, err := s.fileRepo.GetUserFileByIDAny(userID, userFileID)
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("获取用户文件失败: %s", err))
	}
	if userFile.IsDir {
		return nil, util.Internal("该项是文件夹，请调用文件夹接口")
	}

	if !userFile.DeletedAt.Valid {
		err := s.fileRepo.SoftDeleteUserItem(userID, userFileID)
		if err != nil {
			return nil, util.Internal(fmt.Sprintf("移入回收站失败: %s", err))
		}
		err = s.userRepo.DecrUserSpace(userID, userFile.FileSize)
		if err != nil {
			return nil, util.Internal(fmt.Sprintf("更新用户空间失败: %s", err))
		}
		return &api.TrashDeleteResponse{Message: "文件成功移入回收站"}, nil
	}
	return nil, util.Conflict("文件已在回收站")
}

func (s *FileService) RestoreFile(userID, userFileID uint64) (*api.TrashRestoreResponse, error) {
	userFile, err := s.fileRepo.GetUserFileByIDAny(userID, userFileID)
	if err != nil {
		return nil, util.Internal(err.Error())
	}
	if !userFile.DeletedAt.Valid {
		return nil, util.Conflict("文件不在回收站，无法还原")
	}

	if util.IsNameExistsInFolder(s.fileRepo, userID, userFile.ParentID, userFile.FileName, userFileID, false) {
		newName, err := util.GenerateUniqueName(s.fileRepo, userID, userFile.ParentID, userFile.FileName, userFile.FileExt, userFileID, false)
		if err != nil {
			return nil, err
		}
		userFile.FileName = newName
		err = s.fileRepo.UpdateUserFile(userFile)
		if err != nil {
			return nil, util.Internal(fmt.Sprintf("更新用户文件失败: %s", err))
		}
	}

	err = s.fileRepo.RestoreUserFile(userID, userFileID)
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("还原用户文件失败: %s", err))
	}

	err = s.userRepo.IncrUserSpace(userID, userFile.FileSize)
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("更新用户空间失败: %s", err))
	}

	return &api.TrashRestoreResponse{Message: "文件成功还原"}, nil
}

func (s *FileService) RenameFile(userID, userFileID uint64, newFileName string) (*api.FileRenameResponse, error) {
	userFile, err := s.fileRepo.GetUserFileByIDAny(userID, userFileID)
	if err != nil {
		return nil, util.NotFound(fmt.Sprintf("文件不存在: %s", err))
	}

	if userFile.DeletedAt.Valid {
		return nil, util.Conflict("文件在回收站中，无法重命名")
	}

	if err = util.ValidateName(newFileName); err != nil {
		return nil, util.BadRequest(fmt.Sprintf("文件名不合法: %s", err))
	}

	newExt := userFile.FileExt
	if !userFile.IsDir {
		ext := filepath.Ext(newFileName)
		if ext == "" {
			newFileName = newFileName + userFile.FileExt
		} else {
			newExt = strings.ToLower(ext)
		}
	}

	if util.IsNameExistsInFolder(s.fileRepo, userID, userFile.ParentID, newFileName, userFileID, false) {
		uniqueName, err := util.GenerateUniqueName(s.fileRepo, userID, userFile.ParentID, newFileName, newExt, userFileID, false)
		if err != nil {
			return nil, util.Internal(fmt.Sprintf("生成唯一名称失败: %s", err))
		}
		newFileName = uniqueName
	}

	userFile.FileName = newFileName
	userFile.FileExt = newExt
	if err = s.fileRepo.UpdateUserFile(userFile); err != nil {
		return nil, util.Internal(fmt.Sprintf("更新文件名失败: %s", err))
	}

	return &api.FileRenameResponse{
		UserFileID: userFile.ID,
		FileName:   userFile.FileName,
		FileExt:    userFile.FileExt,
	}, nil
}

func (s *FileService) MoveFile(userID, userFileID, targetParentID uint64) (*api.FileMoveResponse, error) {
	userFile, err := s.fileRepo.GetUserFileByID(userID, userFileID)
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("获取用户文件失败: %s", err))
	}

	if userFile.IsDir {
		return nil, util.Conflict("该项是文件夹，不能使用文件移动接口")
	}

	if userFile.DeletedAt.Valid {
		return nil, util.Conflict("改文件在回收站中，请先回复再进行移动")
	}

	if targetParentID == userFile.ParentID {
		return &api.FileMoveResponse{
			UserFileID:   userFile.ID,
			FileName:     userFile.FileName,
			NewParentID:  userFile.ParentID,
			NewPathStack: userFile.PathStack,
		}, nil
	}

	var targetPathStack string
	if targetParentID == 0 {
		targetPathStack = "/0"
	} else {
		targetFolder, err := s.fileRepo.GetUserFolderByID(userID, targetParentID)
		if err != nil {
			return nil, util.NotFound(fmt.Sprintf("目标目录不存在: %s", err))
		}
		targetPathStack = targetFolder.PathStack
	}

	resolvedName := userFile.FileName
	if util.IsNameExistsInFolder(s.fileRepo, userID, targetParentID, userFile.FileName, userFileID, false) {
		uniqueName, err := util.GenerateUniqueName(s.fileRepo, userID, targetParentID, userFile.FileName, userFile.FileExt, userFileID, false)
		if err != nil {
			return nil, util.Internal(fmt.Sprintf("生成唯一名称失败: %s", err))
		}
		resolvedName = uniqueName
	}

	newPathStack := targetPathStack + "/" + strconv.FormatUint(userFileID, 10)

	userFile.ParentID = targetParentID
	userFile.FileName = resolvedName
	userFile.PathStack = newPathStack
	if err := s.fileRepo.UpdateUserFile(userFile); err != nil {
		return nil, util.Internal(fmt.Sprintf("更新文件信息失败: %s", err))
	}

	return &api.FileMoveResponse{
		UserFileID:   userFile.ID,
		FileName:     resolvedName,
		NewParentID:  targetParentID,
		NewPathStack: newPathStack,
	}, nil
}

func (s *FileService) RemoveFile(userID, userFileID uint64) (*api.TrashDeleteResponse, error) {
	userFile, err := s.fileRepo.GetUserFileByIDAny(userID, userFileID)
	if err != nil {
		return nil, util.NotFound(fmt.Sprintf("文件不存在: %s", err))
	}
	if userFile.IsDir {
		return nil, util.BadRequest("该项是文件夹，请调用文件夹接口")
	}

	physicalID := userFile.PhysicalID

	err = s.fileRepo.HardDeleteUserFile(userID, userFileID)
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("删除文件记录失败: %s", err))
	}

	if physicalID != nil {
		phyFile, err := s.fileRepo.GetPhyFileByID(*physicalID)
		if err != nil {
			return nil, util.Internal(fmt.Sprintf("获取物理文件失败: %s", err))
		}

		if phyFile.RefCount <= 1 {
			err = s.minioClient.RemoveObject(
				context.Background(),
				s.config.Minio.Bucket,
				phyFile.FilePath,
				minio.RemoveObjectOptions{},
			)
			if err != nil {
				return nil, util.Internal(fmt.Sprintf("从 MinIO 删除文件失败: %s", err))
			}

			_, err := s.redis.Del(context.Background(), "hash:"+phyFile.FileHash).Result()
			if err != nil {
				return nil, util.Internal(fmt.Sprintf("从 Redis 删除缓存失败: %s", err))
			}

			err = s.fileRepo.DeletePhysicalFile(phyFile.ID)
			if err != nil {
				return nil, util.Internal(fmt.Sprintf("删除物理文件记录失败: %s", err))
			}
		} else {
			err = s.fileRepo.DecrPhyFileRefCount(phyFile.ID, 1)
			if err != nil {
				return nil, util.Internal(fmt.Sprintf("更新物理文件引用数失败: %s", err))
			}
		}
	}

	if !userFile.DeletedAt.Valid {
		err = s.userRepo.DecrUserSpace(userID, userFile.FileSize)
		if err != nil {
			return nil, util.Internal(fmt.Sprintf("更新用户空间失败: %s", err))
		}
	}

	return &api.TrashDeleteResponse{Message: "文件已彻底删除"}, nil
}

func (s *FileService) validFile(fileHeader *multipart.FileHeader) (*model.PhysicalFile, error) {
	fileName := fileHeader.Filename
	fileSize := fileHeader.Size

	ext := filepath.Ext(fileName)
	name := strings.TrimSuffix(fileName, ext)

	err := util.ValidateName(name)
	if err != nil {
		return nil, err
	}

	maxFileSize := s.config.Upload.MaxFileSizeMB * 1024 * 1024
	if fileSize > maxFileSize {
		return nil, fmt.Errorf("上传文件过大(超过%dMB)", s.config.Upload.MaxFileSizeMB)
	} else if fileSize <= 0 {
		return nil, errors.New("上传文件为空")
	}

	return &model.PhysicalFile{
		FileName: fileName,
		FileSize: fileSize,
	}, nil
}

func (s *FileService) createUserFileRecord(
	userID,
	parentID,
	physicalID uint64,
	validResult *model.PhysicalFile,
) (*api.FileUploadResponse, error) {
	respFileName := validResult.FileName
	existing, err := s.fileRepo.GetUserFileByFileName(userID, parentID, validResult.FileName)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, util.Internal(fmt.Sprintf("查询用户文件记录失败: %s", err))
	}
	if existing != nil {
		ext := filepath.Ext(validResult.FileName)
		name := strings.TrimSuffix(validResult.FileName, ext)
		respFileName = fmt.Sprintf("%s_%d%s", name, time.Now().UnixNano(), ext)
	}

	fileExt := strings.ToLower(filepath.Ext(respFileName))

	userFile := &model.UserFile{
		UserID:     userID,
		PhysicalID: &physicalID,
		ParentID:   parentID,
		FileName:   respFileName,
		FileExt:    fileExt,
		FileSize:   validResult.FileSize,
		IsDir:      false,
	}

	err = s.fileRepo.CreateUserFile(userFile)
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("创建用户文件记录失败: %s", err))
	}

	pathStack, err := util.BuildPathStack(s.fileRepo, userID, parentID, userFile.ID)
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("构建路径栈失败: %s", err))
	}

	err = s.fileRepo.UpdateUserFilePath(userFile.ID, pathStack)
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("更新用户文件表失败: %s", err))
	}

	err = s.userRepo.IncrUserSpace(userID, validResult.FileSize)
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("更新用户已使用空间失败: %s", err))
	}

	return &api.FileUploadResponse{
		UserFileID: userFile.ID,
		FileName:   respFileName,
		FileExt:    fileExt,
		FIleSize:   validResult.FileSize,
		ParentID:   parentID,
	}, nil
}
