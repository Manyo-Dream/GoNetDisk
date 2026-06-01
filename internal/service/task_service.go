package service

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"GoNetDisk/configs"
	"GoNetDisk/internal/api"
	"GoNetDisk/internal/model"
	"GoNetDisk/internal/repository"
	"GoNetDisk/internal/util"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"
)

type TaskService struct {
	userRepo      *repository.UserRepo
	fileRepo      *repository.FileRepo
	taskRepo      *repository.TaskRepo
	fileService   *FileService
	folderService *FolderService
	minioClient   *minio.Client
	jwtManager    *util.JWTManager
	config        *configs.Config
	txManager     *repository.TxManager
}

func NewTaskService(
	userRepo *repository.UserRepo,
	fileRepo *repository.FileRepo,
	taskRepo *repository.TaskRepo,
	fileService *FileService,
	folderService *FolderService,
	minioClient *minio.Client,
	jwtManager *util.JWTManager,
	config *configs.Config,
	txManager *repository.TxManager,
) *TaskService {
	return &TaskService{
		userRepo:      userRepo,
		fileRepo:      fileRepo,
		taskRepo:      taskRepo,
		fileService:   fileService,
		folderService: folderService,
		minioClient:   minioClient,
		jwtManager:    jwtManager,
		config:        config,
		txManager:     txManager,
	}
}

func (ts *TaskService) CreateBatchUploadTask(userID uint64, req api.BatchUploadRequest) (*api.BatchUploadResponse, error) {
	if len(req.Files) == 0 {
		return nil, util.BadRequest("任务文件为空")
	}

	var totalSize int64
	seen := make(map[int]bool, len(req.Files))
	for _, f := range req.Files {
		if seen[f.Index] {
			return nil, util.BadRequest(fmt.Sprintf("文件序号 %d 重复", f.Index))
		}
		seen[f.Index] = true

		if f.FileSize <= 0 {
			return nil, util.BadRequest(fmt.Sprintf("文件 %s 大小无效: %d", f.FileName, f.FileSize))
		}

		totalSize += f.FileSize
	}

	var taskID string
	err := ts.txManager.Transaction(func(tx *gorm.DB) error {
		txUserRepo := ts.userRepo.WithTx(tx)
		txTaskRepo := ts.taskRepo.WithTx(tx)

		userInfo, err := txUserRepo.GetUserByID(userID)
		if err != nil {
			return err
		}

		if totalSize > int64(userInfo.TotalSpace-userInfo.UsedSpace) {
			return fmt.Errorf("剩余空间不足，用户剩余空间: %d MB", int64(userInfo.TotalSpace-userInfo.UsedSpace)/1000)
		}

		taskID = uuid.New().String()

		task := &model.UploadTask{
			TaskID:    taskID,
			UserID:    userID,
			Status:    model.UploadTaskStatusProcessing,
			FileCount: len(req.Files),
			TotalSize: totalSize,
			ParentID:  req.ParentID,
		}

		err = txTaskRepo.CreateUploadTask(task)
		if err != nil {
			return fmt.Errorf("创建上传任务失败: %s", err)
		}

		records := make([]*model.UploadFileRecord, 0, len(req.Files))
		for _, f := range req.Files {
			records = append(records, &model.UploadFileRecord{
				TaskID:       taskID,
				UserID:       userID,
				FileIndex:    f.Index,
				FileName:     f.FileName,
				FileExt:      f.FileExt,
				FileSize:     f.FileSize,
				Status:       model.FileStatusWaiting,
				RelativePath: f.RelativePath,
				ErrorMsg:     "",
			})
		}
		err = txTaskRepo.BatchCreateFileRecords(records)
		if err != nil {
			return fmt.Errorf("批量创建文件记录失败: %s", err)
		}

		return nil
	})
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("创建批量上传任务失败: %s", err))
	}

	return &api.BatchUploadResponse{
		TaskID:    taskID,
		FileCount: len(req.Files),
	}, nil
}

func (ts *TaskService) UploadTaskFile(userID uint64, taskID string, fileIndex int, fileHeader *multipart.FileHeader) error {
	task, err := ts.taskRepo.GetUploadTask(userID, taskID)
	if err != nil {
		return util.NotFound(fmt.Sprintf("任务不存在: %s", err))
	}
	if task.Status != model.UploadTaskStatusProcessing {
		return util.BadRequest("任务已结束，无法上传")
	}

	record, err := ts.taskRepo.GetFileRecordByTaskAndIndex(taskID, fileIndex)
	if err != nil {
		return util.NotFound(fmt.Sprintf("文件记录不存在: %s", err))
	}
	if record.Status != model.FileStatusWaiting {
		return util.BadRequest("该文件已上传，请勿重复上传")
	}

	// 创建目录（独立事务内，和上传解耦）
	targetParentID, err := ts.folderService.FindOrCreateFolder(userID, task.ParentID, record.RelativePath)
	if err != nil {
		ts.updateRecordFailed(taskID, fileIndex, "创建文件夹失败: "+err.Error())
		return err
	}

	src, err := fileHeader.Open()
	if err != nil {
		ts.updateRecordFailed(taskID, fileIndex, "打开文件流失败")
		return err
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		ts.updateRecordFailed(taskID, fileIndex, "读取文件失败")
		return err
	}

	physicalID, fileHash, fileExt, err := ts.fileService.findOrCreatePhysicalFile(data, record.FileName, record.FileSize)
	if err != nil {
		ts.updateRecordFailed(taskID, fileIndex, err.Error())
		return err
	}

	// DB 写（独立事务：CreateUserFile + UpdatePath + IncrSpace + UpdateRecord）
	err = ts.txManager.Transaction(func(tx *gorm.DB) error {
		txFileRepo := ts.fileRepo.WithTx(tx)
		txUserRepo := ts.userRepo.WithTx(tx)
		txTaskRepo := ts.taskRepo.WithTx(tx)

		respFileName := record.FileName
		existing, err := txFileRepo.GetUserFileByFileName(userID, targetParentID, record.FileName)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if existing != nil {
			ext := filepath.Ext(record.FileName)
			name := strings.TrimSuffix(record.FileName, ext)
			respFileName = fmt.Sprintf("%s_%d%s", name, time.Now().UnixNano(), ext)
		}

		userFile := &model.UserFile{
			UserID:     userID,
			PhysicalID: &physicalID,
			ParentID:   targetParentID,
			FileName:   respFileName,
			FileExt:    fileExt,
			FileSize:   record.FileSize,
			IsDir:      false,
		}

		if err := txFileRepo.CreateUserFile(userFile); err != nil {
			return err
		}

		pathStack, err := util.BuildPathStack(txFileRepo, userID, targetParentID, userFile.ID)
		if err != nil {
			return err
		}

		if err := txFileRepo.UpdateUserFilePath(userFile.ID, pathStack); err != nil {
			return err
		}

		if err := txUserRepo.IncrUserSpace(userID, record.FileSize); err != nil {
			return err
		}

		if err := txTaskRepo.UpdateFileRecord(taskID, fileIndex, &model.UploadFileRecord{
			Status:     model.FileStatusSuccess,
			FileHash:   fileHash,
			PhysicalID: physicalID,
			UserFileID: userFile.ID,
		}); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		ts.updateRecordFailed(taskID, fileIndex, err.Error())
		return err
	}

	ts.syncTaskProgress(taskID)
	return nil
}

func (ts *TaskService) updateRecordFailed(taskID string, fileIndex int, errMsg string) {
	_ = ts.taskRepo.UpdateFileRecord(taskID, fileIndex, &model.UploadFileRecord{
		Status:   model.FileStatusFailed,
		ErrorMsg: errMsg,
	})
	ts.syncTaskProgress(taskID)
}

func (ts *TaskService) syncTaskProgress(taskID string) {
	records, err := ts.taskRepo.GetFileRecordsByTaskID(taskID)
	if err != nil {
		return
	}

	var successCount, failCount int
	for _, r := range records {
		switch r.Status {
		case model.FileStatusSuccess:
			successCount++
		case model.FileStatusFailed:
			failCount++
		}
	}

	task, err := ts.taskRepo.GetAnyUploadTask(taskID)
	if err != nil {
		return
	}

	updates := &model.UploadTask{
		SuccessCount: successCount,
		FailCount:    failCount,
	}

	if successCount+failCount == task.FileCount {
		if failCount == 0 {
			updates.Status = model.UploadTaskStatusSuccess
		} else if successCount == 0 {
			updates.Status = model.UploadTaskStatusAllFailed
		} else {
			updates.Status = model.UploadTaskStatusFail
		}
	}

	_ = ts.taskRepo.UpdateUploadTask(taskID, updates)
}

func (ts *TaskService) GetTaskProgress(userID uint64, taskID string) (*api.TaskProgressResponse, error) {
	task, err := ts.taskRepo.GetUploadTask(userID, taskID)
	if err != nil {
		return nil, util.NotFound(fmt.Sprintf("任务不存在: %s", err))
	}

	records, err := ts.taskRepo.GetFileRecordsByTaskID(taskID)
	if err != nil {
		return nil, util.Internal(fmt.Sprintf("获取文件记录失败: %s", err))
	}

	files := make([]api.FileProgressItem, 0, len(records))
	for _, r := range records {
		files = append(files, api.FileProgressItem{
			Index:      r.FileIndex,
			FileName:   r.FileName,
			Status:     r.Status,
			ErrorMsg:   r.ErrorMsg,
			UserFileID: r.UserFileID,
		})
	}

	var progressPct int
	if task.FileCount > 0 {
		progressPct = (task.SuccessCount + task.FailCount) * 100 / task.FileCount
	}

	return &api.TaskProgressResponse{
		TaskID:       task.TaskID,
		Status:       task.Status,
		FileCount:    task.FileCount,
		SuccessCount: task.SuccessCount,
		FailCount:    task.FailCount,
		ProgressPct:  progressPct,
		ErrorMsg:     task.ErrorMsg,
		Files:        files,
	}, nil
}
