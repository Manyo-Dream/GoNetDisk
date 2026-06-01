package service

import (
	"context"
	"errors"
	"fmt"
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

type FolderService struct {
	redis       *redis.Client
	userRepo    *repository.UserRepo
	fileRepo    *repository.FileRepo
	jwtManger   *util.JWTManager
	minioClient *minio.Client
	config      *configs.Config
	txManager   *repository.TxManager
}

func NewFolderService(
	redis *redis.Client,
	userRepo *repository.UserRepo,
	fileRepo *repository.FileRepo,
	jwtManger *util.JWTManager,
	minioClient *minio.Client,
	config *configs.Config,
	txManager *repository.TxManager,
) *FolderService {
	return &FolderService{
		redis:       redis,
		userRepo:    userRepo,
		fileRepo:    fileRepo,
		jwtManger:   jwtManger,
		minioClient: minioClient,
		config:      config,
		txManager:   txManager,
	}
}

type cleanupItem struct {
	filePath string
	fileHash string
}

func (fds *FolderService) CreateFolder(email, folderName string, parentID uint64) (*api.FolderResponse, error) {
	// 参数校验
	if err := util.ValidateName(folderName); err != nil {
		return nil, util.BadRequest(fmt.Sprintf("校验FolderName失败: %s", err.Error()))
	}

	var userFolder *model.UserFile

	err := fds.txManager.Transaction(func(tx *gorm.DB) error {
		txFileRepo := fds.fileRepo.WithTx(tx)
		txUserRepo := fds.userRepo.WithTx(tx)

		userInfo, err := txUserRepo.GetByEmail(email)
		if err != nil {
			return err
		}

		if parentID != 0 {
			_, err := txFileRepo.GetParentFolderByParentID(userInfo.ID, parentID)
			if err != nil {
				return err
			}
		}

		_, err = txFileRepo.GetUserFileByFolderName(userInfo.ID, parentID, folderName)
		if err == nil {
			folderName = fmt.Sprintf("%s_%d", folderName, time.Now().UnixNano())
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		userFolder = &model.UserFile{
			UserID:     userInfo.ID,
			PhysicalID: nil,
			ParentID:   parentID,
			FileName:   folderName,
			IsDir:      true,
		}

		err = txFileRepo.CreateUserFile(userFolder)
		if err != nil {
			return err
		}

		pathStack, err := util.BuildPathStack(txFileRepo, userInfo.ID, parentID, userFolder.ID)
		if err != nil {
			return err
		}

		err = txFileRepo.UpdateUserFilePath(userFolder.ID, pathStack)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("创建文件夹失败: %s", err))
	}

	// 返回响应
	return &api.FolderResponse{
		FolderName: userFolder.FileName,
		ParentID:   userFolder.ParentID,
		FolderID:   userFolder.ID,
	}, nil
}

func (fds *FolderService) MoveFolderToTrash(userID, userFileID uint64) (*api.TrashDeleteResponse, error) {
	userFile, err := fds.fileRepo.GetUserFileByIDAny(userID, userFileID)
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("获取用户文件夹失败: %s", err))
	}
	if !userFile.IsDir {
		return nil, util.Internal("该项是文件，请调用文件接口")
	}
	if !userFile.DeletedAt.Time.IsZero() {
		return nil, util.Conflict("文件夹已在回收站")
	}

	err = fds.txManager.Transaction(func(tx *gorm.DB) error {
		txFileRepo := fds.fileRepo.WithTx(tx)
		txUserRepo := fds.userRepo.WithTx(tx)

		err = softDeleteFolderRecursive(txFileRepo, txUserRepo, userID, userFileID)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("文件夹移入回收站失败: %s", err))
	}

	return &api.TrashDeleteResponse{Message: "文件夹成功移入回收站"}, nil
}

func (fds *FolderService) RemoveFolder(userID, folderID uint64) (*api.TrashDeleteResponse, error) {
	userFolder, err := fds.fileRepo.GetUserFileByIDAny(userID, folderID)
	if err != nil {
		return nil, util.NotFound(fmt.Sprintf("文件夹不存在: %s", err))
	}
	if !userFolder.IsDir {
		return nil, util.BadRequest("该项是文件，请调用文件接口")
	}

	var outCleanUp []cleanupItem
	err = fds.txManager.Transaction(func(tx *gorm.DB) error {
		txFileRepo := fds.fileRepo.WithTx(tx)
		txUserRepo := fds.userRepo.WithTx(tx)

		outCleanUp, err = hardDeleteFolderRecursive(txFileRepo, txUserRepo, userID, folderID)
		if err != nil {
			return fmt.Errorf("文件夹硬删除失败: %s", err)
		}

		return nil
	})
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("文件夹删除失败: %s", err))
	}

	for _, item := range outCleanUp {
		err = fds.minioClient.RemoveObject(
			context.Background(),
			fds.config.Minio.Bucket,
			item.filePath,
			minio.RemoveObjectOptions{},
		)
		if err != nil {
			return nil, util.Internal(fmt.Sprintf("从 MinIO 删除文件失败: %s", err))
		}

		_, err := fds.redis.Del(context.Background(), "hash:"+item.fileHash).Result()
		if err != nil {
			return nil, util.Internal(fmt.Sprintf("从 Redis 删除缓存失败: %s", err))
		}
	}

	return &api.TrashDeleteResponse{Message: "文件夹已彻底删除"}, nil
}

func hardDeleteFolderRecursive(
	txFileRepo *repository.FileRepo,
	txUserRepo *repository.UserRepo,
	userID, folderID uint64,
) ([]cleanupItem, error) {
	children, err := txFileRepo.GetTrashChildrenFiles(userID, folderID)
	if err != nil {
		return nil, fmt.Errorf("获取回收站子文件失败: %s", err)
	}

	var cleanup []cleanupItem
	for _, child := range children {
		if child.IsDir {
			sub, err := hardDeleteFolderRecursive(txFileRepo, txUserRepo, userID, child.ID)
			if err != nil {
				return nil, fmt.Errorf("递归删除子文件失败: %s", err)
			}
			cleanup = append(cleanup, sub...)
		} else {
			err = txFileRepo.HardDeleteUserFile(userID, child.ID)
			if err != nil {
				return nil, fmt.Errorf("删除文件记录失败: %s", err)
			}

			if child.PhysicalID != nil {
				phyFile, err := txFileRepo.GetPhyFileByID(*child.PhysicalID)
				if err != nil {
					return nil, fmt.Errorf("获取物理文件失败: %s", err)
				}

				if phyFile.RefCount <= 1 {
					cleanup = append(cleanup, cleanupItem{
						filePath: phyFile.FilePath,
						fileHash: phyFile.FileHash,
					})

					err = txFileRepo.DeletePhysicalFile(phyFile.ID)
					if err != nil {
						return nil, fmt.Errorf("删除物理文件记录失败: %s", err)
					}
				} else {
					err = txFileRepo.DecrPhyFileRefCount(phyFile.ID, 1)
					if err != nil {
						return nil, fmt.Errorf("更新物理文件引用数失败: %s", err)
					}
				}
			}

			if !child.DeletedAt.Valid {
				err = txUserRepo.DecrUserSpace(userID, child.FileSize)
				if err != nil {
					return nil, fmt.Errorf("更新用户空间失败: %s", err)
				}
			}
		}
	}

	err = txFileRepo.HardDeleteUserFile(userID, folderID)
	if err != nil {
		return nil, fmt.Errorf("删除文件夹记录失败: %s", err)
	}

	return cleanup, nil
}

func (fds *FolderService) RenameFolder(userID, userFolderID uint64, newFolderName string) (*api.FolderRenameResponse, error) {
	// 验证文件夹名
	if err := util.ValidateName(newFolderName); err != nil {
		return nil, util.BadRequest(fmt.Sprintf("校验FolderName失败: %s", err.Error()))
	}

	// 是否存在该文件夹
	userFolder, err := fds.fileRepo.GetUserFolderByID(userID, userFolderID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, util.NotFound("不存在该文件夹")
		}
		return nil, util.Internal(fmt.Sprintf("获取文件夹信息失败: %s", err))
	}

	// 查询同一目录是否有同名文件夹
	finalName := newFolderName
	if util.IsNameExistsInFolder(fds.fileRepo, userID, userFolder.ParentID, newFolderName, userFolderID, true) {
		uniqueName, err := util.GenerateUniqueName(fds.fileRepo, userID, userFolder.ParentID, newFolderName, "", userFolderID, true)
		if err != nil {
			return nil, util.Internal(fmt.Sprintf("生成唯一文件夹名失败: %s", err.Error()))
		}
		finalName = uniqueName
	}

	// 更新数据库
	userFolder.FileName = finalName
	if err = fds.fileRepo.UpdateUserFile(userFolder); err != nil {
		return nil, util.Internal(fmt.Sprintf("更新文件夹名失败: %s", err))
	}

	// 返回响应
	return &api.FolderRenameResponse{
		UserFolderID: userFolder.ID,
		FolderName:   finalName,
	}, nil
}

func (fds *FolderService) MoveFolder(userID, userFolderID, targetParenID uint64) (*api.FolderMoveResponse, error) {
	userFolder, err := fds.fileRepo.GetUserFolderByID(userID, userFolderID)
	if err != nil {
		return nil, util.NotFound(fmt.Sprintf("获取文件夹信息失败: %s", err.Error()))
	}
	if !userFolder.IsDir {
		return nil, util.BadRequest("此项不是文件夹")
	}
	if userFolder.DeletedAt.Valid {
		return nil, util.Conflict("文件夹已在回收站，无法移动")
	}

	var targetPathStack string
	if targetParenID == 0 {
		targetPathStack = "/0"
	} else {
		targetFolder, err := fds.fileRepo.GetUserFolderByID(userID, targetParenID)
		if err != nil {
			return nil, util.NotFound(fmt.Sprintf("目标父目录不存在: %s", err.Error()))
		}
		targetPathStack = targetFolder.PathStack

		if strings.HasPrefix(targetPathStack+"/", userFolder.PathStack+"/") || targetPathStack == userFolder.PathStack {
			return nil, util.BadRequest("不能将文件夹移动到自身或其子目录中")
		}
	}

	if userFolder.ParentID == targetParenID {
		return nil, util.Conflict("目标目录与原目录相同")
	}

	finalName := userFolder.FileName
	var newPathStack string

	err = fds.txManager.Transaction(func(tx *gorm.DB) error {
		txFileRepo := fds.fileRepo.WithTx(tx)

		if util.IsNameExistsInFolder(txFileRepo, userID, targetParenID, userFolder.FileName, userFolderID, true) {
			uniqueName, err := util.GenerateUniqueName(txFileRepo, userID, targetParenID, userFolder.FileName, "", userFolderID, true)
			if err != nil {
				return fmt.Errorf("生成唯一文件夹名失败: %s", err.Error())
			}
			finalName = uniqueName
		}

		oldPathStack := userFolder.PathStack
		newPathStack = targetPathStack + "/" + strconv.FormatUint(userFolder.ID, 10)

		userFolder.ParentID = targetParenID
		userFolder.FileName = finalName
		userFolder.PathStack = newPathStack

		if err = txFileRepo.UpdateUserFile(userFolder); err != nil {
			return fmt.Errorf("更新文件夹失败: %s", err.Error())
		}

		if err = updateChildrenPathStack(txFileRepo, userID, userFolderID, oldPathStack, newPathStack); err != nil {
			return fmt.Errorf("更新子文件路径失败: %s", err.Error())
		}

		return nil
	})
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("移动文件夹失败: %s", err))
	}

	return &api.FolderMoveResponse{
		UserFolderID: userFolder.ID,
		FolderName:   finalName,
		NewParentID:  targetParenID,
		NewPathStack: newPathStack,
	}, nil
}

func updateChildrenPathStack(txfileRepo *repository.FileRepo, userID, parentID uint64, oldPrefix, newPrefix string) error {
	children, err := txfileRepo.GetChildrenFiles(userID, parentID)
	if err != nil {
		return err
	}
	for _, child := range children {
		newPath := strings.Replace(child.PathStack, oldPrefix, newPrefix, 1)
		if err := txfileRepo.UpdateUserFilePath(child.ID, newPath); err != nil {
			return err
		}
		if child.IsDir {
			if err := updateChildrenPathStack(txfileRepo, userID, child.ID, oldPrefix, newPrefix); err != nil {
				return err
			}
		}
	}
	return nil
}

func (fds *FolderService) FindOrCreateFolder(userID uint64, parentID uint64, relativePath string) (uint64, error) {
	if relativePath == "" {
		return parentID, nil
	}

	path := strings.TrimSuffix(relativePath, "/")
	parts := strings.Split(path, "/")

	var leafID uint64

	err := fds.txManager.Transaction(func(tx *gorm.DB) error {
		txFileRepo := fds.fileRepo.WithTx(tx)
		currentParentID := parentID

		for _, folderName := range parts {
			if folderName == "" {
				continue
			}

			existing, err := txFileRepo.GetUserFileByFolderName(userID, currentParentID, folderName)
			if err == nil {
				currentParentID = existing.ID
				continue
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}

			userFolder := &model.UserFile{
				UserID:     userID,
				PhysicalID: nil,
				ParentID:   currentParentID,
				FileName:   folderName,
				IsDir:      true,
			}

			if err := txFileRepo.CreateUserFile(userFolder); err != nil {
				return err
			}

			pathStack, err := util.BuildPathStack(txFileRepo, userID, currentParentID, userFolder.ID)
			if err != nil {
				return err
			}

			if err := txFileRepo.UpdateUserFilePath(userFolder.ID, pathStack); err != nil {
				return err
			}

			currentParentID = userFolder.ID
		}

		leafID = currentParentID
		return nil
	})

	return leafID, err
}

func softDeleteFolderRecursive(txFileRepo *repository.FileRepo, txUserRepo *repository.UserRepo, userID, folderID uint64) error {
	children, err := txFileRepo.GetChildrenFiles(userID, folderID)
	if err != nil {
		return fmt.Errorf("获取子文件失败: %s", err)
	}

	for _, child := range children {
		if child.IsDir {
			err = softDeleteFolderRecursive(txFileRepo, txUserRepo, userID, child.ID)
			if err != nil {
				return err
			}
		} else {
			err = txFileRepo.SoftDeleteUserItem(userID, child.ID)
			if err != nil {
				return fmt.Errorf("移入回收站失败: %w", err)
			}
			if child.FileSize > 0 {
				err = txUserRepo.DecrUserSpace(userID, child.FileSize)
				if err != nil {
					return fmt.Errorf("更新用户空间失败: %s", err)
				}
			}
		}
	}

	err = txFileRepo.SoftDeleteUserItem(userID, folderID)
	if err != nil {
		return fmt.Errorf("移入回收站失败: %w", err)
	}

	return nil
}

func (fds *FolderService) RestoreFolder(userID, folderID uint64) (*api.TrashRestoreResponse, error) {
	err := fds.txManager.Transaction(func(tx *gorm.DB) error {
		txFileRepo := fds.fileRepo.WithTx(tx)
		return restoreFolderRecursive(txFileRepo, userID, folderID)
	})
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("还原文件夹失败: %s", err))
	}
	return &api.TrashRestoreResponse{Message: "文件夹成功还原"}, nil
}

func restoreFolderRecursive(txFileRepo *repository.FileRepo, userID, folderID uint64) error {
	userFolder, err := txFileRepo.GetUserFileByIDAny(userID, folderID)
	if err != nil {
		return err
	}
	if !userFolder.DeletedAt.Valid {
		return errors.New("文件夹不在回收站")
	}

	if util.IsNameExistsInFolder(txFileRepo, userID, userFolder.ParentID, userFolder.FileName, folderID, true) {
		newName, err := util.GenerateUniqueName(txFileRepo, userID, userFolder.ParentID, userFolder.FileName, "", folderID, true)
		if err != nil {
			return err
		}
		userFolder.FileName = newName
		if err := txFileRepo.UpdateUserFile(userFolder); err != nil {
			return err
		}
	}

	if err := txFileRepo.RestoreUserFile(userID, folderID); err != nil {
		return err
	}

	children, err := txFileRepo.GetTrashChildrenFiles(userID, folderID)
	if err != nil {
		return err
	}

	for _, child := range children {
		if child.IsDir {
			if err := restoreFolderRecursive(txFileRepo, userID, child.ID); err != nil {
				return err
			}
		} else {
			if err := txFileRepo.RestoreUserFile(userID, child.ID); err != nil {
				return err
			}
		}
	}

	return nil
}
