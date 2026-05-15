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

	"github.com/manyodream/gonetdisk/configs"
	"github.com/manyodream/gonetdisk/internal/dto"
	"github.com/manyodream/gonetdisk/internal/model"
	"github.com/manyodream/gonetdisk/internal/repository"
	"github.com/manyodream/gonetdisk/internal/util"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
)

type FileService struct {
	minioClient *minio.Client
	userRepo    *repository.UserRepo
	fileRepo    *repository.FileRepo
	jwtManger   *util.JWTManager
	config      *configs.Config
}

func NewFileService(minioClient *minio.Client, userRepo *repository.UserRepo, fileRepo *repository.FileRepo, jwtManger *util.JWTManager, config *configs.Config) *FileService {
	return &FileService{
		minioClient: minioClient,
		userRepo:    userRepo,
		fileRepo:    fileRepo,
		jwtManger:   jwtManger,
		config:      config,
	}
}

func (s *FileService) UploadPhyFileAndBindFile(email string, parentID uint64, fileHeader *multipart.FileHeader) (*dto.FileUploadResponse, error) {
	validResult, err := s.validFile(fileHeader)
	if err != nil {
		return nil, BadRequest(fmt.Sprintf("验证文件信息失败: %s", err))
	}

	userInfo, err := s.userRepo.GetByEmail(email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NotFound(fmt.Sprintf("用户不存在: %s", email))
		}
		return nil, Internal(fmt.Sprintf("查询用户信息失败: %s", err))
	}

	if userInfo.Total_Space > 0 && userInfo.Used_Space+uint64(validResult.FileSize) > userInfo.Total_Space {
		return nil, Conflict(fmt.Sprintf("空间不足：已用%d，总计%d", userInfo.Used_Space, userInfo.Total_Space))
	}

	if parentID != 0 {
		_, err := s.fileRepo.GetParentFolderByParentID(userInfo.ID, parentID)
		if err != nil {
			return nil, NotFound(fmt.Sprintf("父文件夹不存在: %s", err))
		}
	}

	src, err := fileHeader.Open()
	if err != nil {
		return nil, Internal(fmt.Sprintf("打开文件流失败: %s", err))
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		return nil, Internal(fmt.Sprintf("读取文件失败: %s", err))
	}

	physicalID, _, _, err := s.findOrCreatePhysicalFile(data, validResult.FileName, validResult.FileSize)
	if err != nil {
		return nil, err
	}

	return s.createUserFileRecord(userInfo.ID, parentID, physicalID, validResult)
}

// findOrCreatePhysicalFile 计算 hash、查重、上传到 MinIO、创建 PhysicalFile 记录
// 重复文件不上传，直接复用并引用计数+1。返回 physicalID、fileHash、fileExt
func (s *FileService) findOrCreatePhysicalFile(data []byte, fileName string, fileSize int64) (physicalID uint64, fileHash string, fileExt string, err error) {
	fileHash = fmt.Sprintf("%x", md5.Sum(data))
	fileExt = strings.ToLower(filepath.Ext(fileName))

	hashResult, err := s.fileRepo.HashDeduplication(fileHash)

	if err == nil {
		err = s.fileRepo.IncrPhyFileRefCount(hashResult.ID, 1)
		if err != nil {
			return 0, "", "", Internal(fmt.Sprintf("更新物理文件引用数失败: %s", err))
		}
		return hashResult.ID, fileHash, fileExt, nil

	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		objectKey := fmt.Sprintf("objects/%s/%s/%s%s",
			fileHash[:2],
			fileHash[2:4],
			fileHash,
			fileExt,
		)

		_, err = s.minioClient.PutObject(
			context.Background(),
			s.config.Minio.Bucket,
			objectKey,
			bytes.NewReader(data),
			int64(len(data)),
			minio.PutObjectOptions{},
		)
		if err != nil {
			return 0, "", "", Internal(fmt.Sprintf("上传到 MinIO 失败: %s", err))
		}

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
			return 0, "", "", Internal(fmt.Sprintf("创建物理文件记录失败: %s", err))
		}

		return phyFile.ID, fileHash, fileExt, nil

	} else {
		return 0, "", "", Internal(fmt.Sprintf("哈希查重失败: %s", err))
	}
}

func (s *FileService) DownloadUserFile(userID, userFileID uint64) (*dto.FileDownloadMeta, io.ReadCloser, error) {
	userFile, phyFile, err := s.fileRepo.GetFileByDownloadReq(userID, userFileID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, NotFound("文件不存在")
	} else if err != nil {
		return nil, nil, Internal(fmt.Sprintf("查询文件失败: %s", err))
	}

	if phyFile.FilePath == "" {
		return nil, nil, NotFound("文件路径为空")
	}

	obj, err := s.minioClient.GetObject(
		context.Background(),
		s.config.Minio.Bucket,
		phyFile.FilePath,
		minio.GetObjectOptions{},
	)
	if err != nil {
		return nil, nil, Internal(fmt.Sprintf("从 MinIO 读取文件失败: %s", err))
	}

	return &dto.FileDownloadMeta{
		FileName:    userFile.FileName,
		StorageType: phyFile.StorageType,
		FileExt:     userFile.FileExt,
		FileSize:    phyFile.FileSize,
	}, obj, nil
}

func (s *FileService) GetUserFileList(userID, parentID uint64, page, pageSize int, sortBy, orderBy string) (*dto.FileListResponse, error) {
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
		return nil, Internal(fmt.Sprintf("获取文件列表失败: %s", err))
	}

	return &dto.FileListResponse{
		List:     userFileList,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *FileService) GetTrashList(userID uint64, page, pageSize int) (*dto.TrashListResponse, error) {
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
		return nil, Internal(fmt.Sprintf("获取文件列表失败: %s", err))
	}

	return &dto.TrashListResponse{
		List:     trashFileList,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *FileService) MoveFileToTrash(userID, userFileID uint64) (*dto.TrashDeleteResponse, error) {
	userFile, err := s.fileRepo.GetUserFileByIDAny(userID, userFileID)
	if err != nil {
		return nil, Internal(fmt.Sprintf("获取用户文件失败: %s", err))
	}
	if userFile.IsDir {
		return nil, Internal("该项是文件夹，请调用文件夹接口")
	}

	if !userFile.DeletedAt.Valid {
		err := s.fileRepo.SoftDeleteUserItem(userID, userFileID)
		if err != nil {
			return nil, Internal(fmt.Sprintf("移入回收站失败: %s", err))
		}
		err = s.userRepo.DecrUserSpace(userID, userFile.FileSize)
		if err != nil {
			return nil, Internal(fmt.Sprintf("更新用户空间失败: %s", err))
		}
		return &dto.TrashDeleteResponse{Message: "文件成功移入回收站"}, nil
	}
	return nil, Conflict("文件已在回收站")
}

func (s *FileService) RestoreFile(userID, userFileID uint64) (*dto.TrashRestoreResponse, error) {
	userFile, err := s.fileRepo.GetUserFileByIDAny(userID, userFileID)
	if err != nil {
		return nil, Internal(err.Error())
	}
	if !userFile.DeletedAt.Valid {
		return nil, Conflict("文件不在回收站，无法还原")
	}

	if util.IsNameExistsInFolder(s.fileRepo, userID, userFile.ParentID, userFile.FileName, userFileID, false) {
		newName, err := util.GenerateUniqueName(s.fileRepo, userID, userFile.ParentID, userFile.FileName, userFile.FileExt, userFileID, false)
		if err != nil {
			return nil, err
		}
		userFile.FileName = newName
		err = s.fileRepo.UpdateUserFile(userFile)
		if err != nil {
			return nil, Internal(fmt.Sprintf("更新用户文件失败: %s", err))
		}
	}

	err = s.fileRepo.RestoreUserFile(userID, userFileID)
	if err != nil {
		return nil, Internal(fmt.Sprintf("还原用户文件失败: %s", err))
	}

	err = s.userRepo.IncrUserSpace(userID, userFile.FileSize)
	if err != nil {
		return nil, Internal(fmt.Sprintf("更新用户空间失败: %s", err))
	}

	return &dto.TrashRestoreResponse{Message: "文件成功还原"}, nil
}

func (s *FileService) RenameFile(userID, userFileID uint64, newFileName string) (*dto.FileRenameResponse, error) {
	userFile, err := s.fileRepo.GetUserFileByIDAny(userID, userFileID)
	if err != nil {
		return nil, NotFound(fmt.Sprintf("文件不存在: %s", err))
	}

	if userFile.DeletedAt.Valid {
		return nil, Conflict("文件在回收站中，无法重命名")
	}

	if err = util.ValidateName(newFileName); err != nil {
		return nil, BadRequest(fmt.Sprintf("文件名不合法: %s", err))
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
			return nil, Internal(fmt.Sprintf("生成唯一名称失败: %s", err))
		}
		newFileName = uniqueName
	}

	userFile.FileName = newFileName
	userFile.FileExt = newExt
	if err = s.fileRepo.UpdateUserFile(userFile); err != nil {
		return nil, Internal(fmt.Sprintf("更新文件名失败: %s", err))
	}

	return &dto.FileRenameResponse{
		UserFileID: userFile.ID,
		FileName:   userFile.FileName,
		FileExt:    userFile.FileExt,
	}, nil
}

func (s *FileService) MoveFile(userID, userFileID, targetParentID uint64) (*dto.FileMoveResponse, error) {
	userFile, err := s.fileRepo.GetUserFileByID(userID, userFileID)
	if err != nil {
		return nil, Internal(fmt.Sprintf("获取用户文件失败: %s", err))
	}

	if userFile.IsDir {
		return nil, Conflict("该项是文件夹，不能使用文件移动接口")
	}

	if userFile.DeletedAt.Valid {
		return nil, Conflict("改文件在回收站中，请先回复再进行移动")
	}

	if targetParentID == userFile.ParentID {
		return &dto.FileMoveResponse{
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
			return nil, NotFound(fmt.Sprintf("目标目录不存在: %s", err))
		}
		targetPathStack = targetFolder.PathStack
	}

	resolvedName := userFile.FileName
	if util.IsNameExistsInFolder(s.fileRepo, userID, targetParentID, userFile.FileName, userFileID, false) {
		uniqueName, err := util.GenerateUniqueName(s.fileRepo, userID, targetParentID, userFile.FileName, userFile.FileExt, userFileID, false)
		if err != nil {
			return nil, Internal(fmt.Sprintf("生成唯一名称失败: %s", err))
		}
		resolvedName = uniqueName
	}

	newPathStack := targetPathStack + "/" + strconv.FormatUint(userFileID, 10)

	userFile.ParentID = targetParentID
	userFile.FileName = resolvedName
	userFile.PathStack = newPathStack
	if err := s.fileRepo.UpdateUserFile(userFile); err != nil {
		return nil, Internal(fmt.Sprintf("更新文件信息失败: %s", err))
	}

	return &dto.FileMoveResponse{
		UserFileID:   userFile.ID,
		FileName:     resolvedName,
		NewParentID:  targetParentID,
		NewPathStack: newPathStack,
	}, nil
}

func (s *FileService) RemoveFile(userID, userFileID uint64) (*dto.TrashDeleteResponse, error) {
	userFile, err := s.fileRepo.GetUserFileByIDAny(userID, userFileID)
	if err != nil {
		return nil, NotFound(fmt.Sprintf("文件不存在: %s", err))
	}
	if userFile.IsDir {
		return nil, BadRequest("该项是文件夹，请调用文件夹接口")
	}

	physicalID := userFile.PhysicalID

	err = s.fileRepo.HardDeleteUserFile(userID, userFileID)
	if err != nil {
		return nil, Internal(fmt.Sprintf("删除文件记录失败: %s", err))
	}

	if physicalID != nil {
		phyFile, err := s.fileRepo.GetPhyFileByID(*physicalID)
		if err != nil {
			return nil, Internal(fmt.Sprintf("获取物理文件失败: %s", err))
		}

		if phyFile.RefCount <= 1 {
			err = s.minioClient.RemoveObject(
				context.Background(),
				s.config.Minio.Bucket,
				phyFile.FilePath,
				minio.RemoveObjectOptions{},
			)
			if err != nil {
				return nil, Internal(fmt.Sprintf("从 MinIO 删除文件失败: %s", err))
			}
			err = s.fileRepo.DeletePhysicalFile(phyFile.ID)
			if err != nil {
				return nil, Internal(fmt.Sprintf("删除物理文件记录失败: %s", err))
			}
		} else {
			err = s.fileRepo.DecrPhyFileRefCount(phyFile.ID, 1)
			if err != nil {
				return nil, Internal(fmt.Sprintf("更新物理文件引用数失败: %s", err))
			}
		}
	}

	if !userFile.DeletedAt.Valid {
		err = s.userRepo.DecrUserSpace(userID, userFile.FileSize)
		if err != nil {
			return nil, Internal(fmt.Sprintf("更新用户空间失败: %s", err))
		}
	}

	return &dto.TrashDeleteResponse{Message: "文件已彻底删除"}, nil
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

func (s *FileService) createUserFileRecord(userID, parentID, physicalID uint64, validResult *model.PhysicalFile) (*dto.FileUploadResponse, error) {
	respFileName := validResult.FileName
	existing, err := s.fileRepo.GetUserFileByFileName(userID, parentID, validResult.FileName)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, Internal(fmt.Sprintf("查询用户文件记录失败: %s", err))
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
		return nil, Internal(fmt.Sprintf("创建用户文件记录失败: %s", err))
	}

	pathStack, err := util.BuildPathStack(s.fileRepo, userID, parentID, userFile.ID)
	if err != nil {
		return nil, Internal(fmt.Sprintf("构建路径栈失败: %s", err))
	}

	err = s.fileRepo.UpdateUserFilePath(userFile.ID, pathStack)
	if err != nil {
		return nil, Internal(fmt.Sprintf("更新用户文件表失败: %s", err))
	}

	err = s.userRepo.IncrUserSpace(userID, validResult.FileSize)
	if err != nil {
		return nil, Internal(fmt.Sprintf("更新用户已使用空间失败: %s", err))
	}

	return &dto.FileUploadResponse{
		UserFileID: userFile.ID,
		FileName:   respFileName,
		FileExt:    fileExt,
		FIleSize:   validResult.FileSize,
		ParentID:   parentID,
	}, nil
}
