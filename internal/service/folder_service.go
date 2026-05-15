package service

import (
	"context"
	"errors"
	"fmt"
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

type FolderService struct {
	userRepo    *repository.UserRepo
	fileRepo    *repository.FileRepo
	jwtManger   *util.JWTManager
	minioClient *minio.Client
	config      *configs.Config
}

func NewFolderService(userRepo *repository.UserRepo, fileRepo *repository.FileRepo, jwtManger *util.JWTManager, minioClient *minio.Client, config *configs.Config) *FolderService {
	return &FolderService{userRepo: userRepo, fileRepo: fileRepo, jwtManger: jwtManger, minioClient: minioClient, config: config}
}

func (fds *FolderService) CreateFolder(email, folderName string, parentID uint64) (*dto.FolderResponse, error) {
	// 参数校验
	if err := util.ValidateName(folderName); err != nil {
		return nil, util.BadRequest(fmt.Sprintf("校验FolderName失败: %s", err.Error()))
	}

	// 获取userID
	userInfo, err := fds.userRepo.GetByEmail(email)
	if err != nil {
		return nil, util.NotFound(fmt.Sprintf("获取UserID失败: %s", err.Error()))
	}

	// 查询parentID对应的父文件夹是否存在(根目录0、其他目录分情况查询)
	if parentID != 0 {
		_, err := fds.fileRepo.GetParentFolderByParentID(userInfo.ID, parentID)
		if err != nil {
			return nil, util.NotFound(fmt.Sprintf("父目录不存在或不是当前用户目录: %s", err.Error()))
		}
	}

	// 查询用户文件表是否存在同名文件夹，存在则增加时间后缀
	_, err = fds.fileRepo.GetUserFileByFolderName(userInfo.ID, parentID, folderName)
	if err == nil {
		folderName = fmt.Sprintf("%s_%d", folderName, time.Now().UnixNano())
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, util.Conflict(fmt.Sprintf("检查文件夹重名失败: %s", err.Error()))
	}

	// 创建用户文件表
	userFolder := &model.UserFile{
		UserID:     userInfo.ID,
		PhysicalID: nil,
		ParentID:   parentID,
		FileName:   folderName,
		IsDir:      true,
	}
	err = fds.fileRepo.CreateUserFile(userFolder)
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("创建用户文件夹失败: %s", err))
	}

	// 构建pathStack
	pathStack, err := util.BuildPathStack(fds.fileRepo, userInfo.ID, parentID, userFolder.ID)
	if err != nil {
		return nil, util.NotFound(fmt.Sprintf("构建路径栈失败: %s", err.Error()))
	}

	// 更新用户文件表
	err = fds.fileRepo.UpdateUserFilePath(userFolder.ID, pathStack)
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("更新用户文件表失败: %s", err.Error()))
	}

	// 返回响应
	return &dto.FolderResponse{
		FolderName: userFolder.FileName,
		ParentID:   userFolder.ParentID,
		FolderID:   userFolder.ID,
	}, nil
}

func (fds *FolderService) MoveFolderToTrash(userID, userFileID uint64) (*dto.TrashDeleteResponse, error) {
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

	err = fds.softDeleteFolderRecursive(userID, userFileID)
	if err != nil {
		return nil, err
	}

	return &dto.TrashDeleteResponse{Message: "文件夹成功移入回收站"}, nil
}

func (fds *FolderService) RemoveFolder(userID, folderID uint64) (*dto.TrashDeleteResponse, error) {
	userFolder, err := fds.fileRepo.GetUserFileByIDAny(userID, folderID)
	if err != nil {
		return nil, util.NotFound(fmt.Sprintf("文件夹不存在: %s", err))
	}
	if !userFolder.IsDir {
		return nil, util.BadRequest("该项是文件，请调用文件接口")
	}

	err = fds.hardDeleteFolderRecursive(userID, folderID)
	if err != nil {
		return nil, err
	}

	return &dto.TrashDeleteResponse{Message: "文件夹已彻底删除"}, nil
}

func (fds *FolderService) hardDeleteFolderRecursive(userID, folderID uint64) error {
	children, err := fds.fileRepo.GetTrashChildrenFiles(userID, folderID)
	if err != nil {
		return util.Internal(fmt.Sprintf("获取子文件失败: %s", err))
	}

	for _, child := range children {
		if child.IsDir {
			err = fds.hardDeleteFolderRecursive(userID, child.ID)
			if err != nil {
				return err
			}
		} else {
			physicalID := child.PhysicalID

			err = fds.fileRepo.HardDeleteUserFile(userID, child.ID)
			if err != nil {
				return util.Internal(fmt.Sprintf("删除文件记录失败: %s", err))
			}

			if physicalID != nil {
				phyFile, err := fds.fileRepo.GetPhyFileByID(*physicalID)
				if err != nil {
					return util.Internal(fmt.Sprintf("获取物理文件失败: %s", err))
				}

				if phyFile.RefCount <= 1 {
					err = fds.minioClient.RemoveObject(
						context.Background(),
						fds.config.Minio.Bucket,
						phyFile.FilePath,
						minio.RemoveObjectOptions{},
					)
					if err != nil {
						return util.Internal(fmt.Sprintf("从 MinIO 删除文件失败: %s", err))
					}
					err = fds.fileRepo.DeletePhysicalFile(phyFile.ID)
					if err != nil {
						return util.Internal(fmt.Sprintf("删除物理文件记录失败: %s", err))
					}
				} else {
					err = fds.fileRepo.DecrPhyFileRefCount(phyFile.ID, 1)
					if err != nil {
						return util.Internal(fmt.Sprintf("更新物理文件引用数失败: %s", err))
					}
				}
			}

			if !child.DeletedAt.Valid {
				err = fds.userRepo.DecrUserSpace(userID, child.FileSize)
				if err != nil {
					return util.Internal(fmt.Sprintf("更新用户空间失败: %s", err))
				}
			}
		}
	}

	err = fds.fileRepo.HardDeleteUserFile(userID, folderID)
	if err != nil {
		return util.Internal(fmt.Sprintf("删除文件夹记录失败: %s", err))
	}

	return nil
}

func (fds *FolderService) RenameFolder(userID, userFolderID uint64, newFolderName string) (*dto.FolderRenameResponse, error) {
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
	return &dto.FolderRenameResponse{
		UserFolderID: userFolder.ID,
		FolderName:   finalName,
	}, nil
}

func (fds *FolderService) MoveFolder(userID, userFolderID, targetParenID uint64) (*dto.FolderMoveResponse, error) {
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
	if util.IsNameExistsInFolder(fds.fileRepo, userID, targetParenID, userFolder.FileName, userFolderID, true) {
		uniqueName, err := util.GenerateUniqueName(fds.fileRepo, userID, targetParenID, userFolder.FileName, "", userFolderID, true)
		if err != nil {
			return nil, util.Internal(fmt.Sprintf("生成唯一文件夹名失败: %s", err.Error()))
		}
		finalName = uniqueName
	}

	oldPathStack := userFolder.PathStack
	newPathStack := targetPathStack + "/" + strconv.FormatUint(userFolder.ID, 10)

	userFolder.ParentID = targetParenID
	userFolder.FileName = finalName
	userFolder.PathStack = newPathStack
	if err = fds.fileRepo.UpdateUserFile(userFolder); err != nil {
		return nil, util.Internal(fmt.Sprintf("更新文件夹失败: %s", err.Error()))
	}

	if err = fds.updateChildrenPathStack(userID, userFolderID, oldPathStack, newPathStack); err != nil {
		return nil, util.Internal(fmt.Sprintf("更新子文件路径失败: %s", err.Error()))
	}

	return &dto.FolderMoveResponse{
		UserFolderID: userFolder.ID,
		FolderName:   finalName,
		NewParentID:  targetParenID,
		NewPathStack: newPathStack,
	}, nil
}

func (fds *FolderService) updateChildrenPathStack(userID, parentID uint64, oldPrefix, newPrefix string) error {
	children, err := fds.fileRepo.GetChildrenFiles(userID, parentID)
	if err != nil {
		return err
	}

	for _, child := range children {
		newPath := strings.Replace(child.PathStack, oldPrefix, newPrefix, 1)
		if err = fds.fileRepo.UpdateUserFilePath(child.ID, newPath); err != nil {
			return err
		}
		if child.IsDir {
			if err = fds.updateChildrenPathStack(userID, child.ID, oldPrefix, newPrefix); err != nil {
				return err
			}
		}
	}

	return nil
}

// FindOrCreateFolder 在指定父目录下逐层创建文件夹，返回叶子文件夹ID
// relativePath: "旅行照片/风景/" → ["旅行照片", "风景"]
// 如果某层已存在同名文件夹，直接复用
func (fds *FolderService) FindOrCreateFolder(userID uint64, parentID uint64, relativePath string) (uint64, error) {
	if relativePath == "" {
		return parentID, nil
	}

	// 去除末尾斜杠并按 / 切分
	path := strings.TrimSuffix(relativePath, "/")
	parts := strings.Split(path, "/")

	currentParentID := parentID

	for _, folderName := range parts {
		if folderName == "" {
			continue
		}

		// 查当前层级是否已存在同名文件夹
		existing, err := fds.fileRepo.GetUserFileByFolderName(userID, currentParentID, folderName)
		if err == nil {
			// 找到了，复用
			currentParentID = existing.ID
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, util.Internal(fmt.Sprintf("查询文件夹失败: %s", err))
		}

		// 不存在，创建
		userFolder := &model.UserFile{
			UserID:     userID,
			PhysicalID: nil,
			ParentID:   currentParentID,
			FileName:   folderName,
			IsDir:      true,
		}

		err = fds.fileRepo.CreateUserFile(userFolder)
		if err != nil {
			return 0, util.Internal(fmt.Sprintf("创建文件夹失败: %s", err))
		}

		// 构建 pathStack
		pathStack, err := util.BuildPathStack(fds.fileRepo, userID, currentParentID, userFolder.ID)
		if err != nil {
			return 0, util.Internal(fmt.Sprintf("构建路径栈失败: %s", err))
		}

		err = fds.fileRepo.UpdateUserFilePath(userFolder.ID, pathStack)
		if err != nil {
			return 0, util.Internal(fmt.Sprintf("更新文件路径失败: %s", err))
		}

		currentParentID = userFolder.ID
	}

	return currentParentID, nil
}

func (fds *FolderService) softDeleteFolderRecursive(userID, folderID uint64) error {
	children, err := fds.fileRepo.GetChildrenFiles(userID, folderID)
	if err != nil {
		return util.Internal(fmt.Sprintf("获取子文件失败: %s", err))
	}

	for _, child := range children {
		if child.IsDir {
			err = fds.softDeleteFolderRecursive(userID, child.ID)
			if err != nil {
				return err
			}
		} else {
			err = fds.fileRepo.SoftDeleteUserItem(userID, child.ID)
			if err != nil {
				return util.Internal(fmt.Sprintf("移入回收站失败: %s", err))
			}
			if child.FileSize > 0 {
				err = fds.userRepo.DecrUserSpace(userID, child.FileSize)
				if err != nil {
					return util.Internal(fmt.Sprintf("更新用户空间失败: %s", err))
				}
			}
		}
	}

	err = fds.fileRepo.SoftDeleteUserItem(userID, folderID)
	if err != nil {
		return util.Internal(fmt.Sprintf("移入回收站失败: %s", err))
	}

	return nil
}

func (fds *FolderService) RestoreFolder(userID, folderID uint64) (*dto.TrashRestoreResponse, error) {
	userFolder, err := fds.fileRepo.GetUserFileByIDAny(userID, folderID)
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("查询用户文件夹失败: %s", err))
	}
	if !userFolder.DeletedAt.Valid {
		return nil, util.Conflict("文件夹不在回收站")
	}

	// 检查目标目录是否已存在同名文件夹，如存在则生成唯一名称
	if util.IsNameExistsInFolder(fds.fileRepo, userID, userFolder.ParentID, userFolder.FileName, folderID, true) {
		newName, err := util.GenerateUniqueName(fds.fileRepo, userID, userFolder.ParentID, userFolder.FileName, "", folderID, true)
		if err != nil {
			return nil, util.Internal(fmt.Sprintf("生成文件名失败: %s", err))
		}
		userFolder.FileName = newName
		err = fds.fileRepo.UpdateUserFile(userFolder)
		if err != nil {
			return nil, util.Internal(fmt.Sprintf("更新用户文件夹失败: %s", err))
		}
	}

	err = fds.fileRepo.RestoreUserFile(userID, folderID)
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("还原文件夹失败: %s", err))
	}

	children, err := fds.fileRepo.GetTrashChildrenFiles(userID, folderID)
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("获取子文件失败: %s", err))
	}

	for _, child := range children {
		if child.IsDir {
			_, err = fds.RestoreFolder(userID, child.ID)
			if err != nil {
				return nil, util.Internal(fmt.Sprintf("还原文件夹失败: %s", err))
			}
		} else {
			err = fds.fileRepo.RestoreUserFile(userID, child.ID)
			if err != nil {
				return nil, util.Internal(fmt.Sprintf("还原文件失败: %s", err))
			}
		}
	}

	return &dto.TrashRestoreResponse{Message: "文件夹成功还原"}, nil
}
