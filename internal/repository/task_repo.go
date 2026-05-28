package repository

import (
	"GoNetDisk/internal/model"

	"gorm.io/gorm"
)

type TaskRepo struct {
	db *gorm.DB
}

func NewTaskRepo(db *gorm.DB) *TaskRepo {
	return &TaskRepo{db: db}
}

func (tr *TaskRepo) CreateUploadTask(task *model.UploadTask) error {
	return tr.db.Create(task).Error
}

func (tr *TaskRepo) GetUploadTask(userID uint64, taskID string) (*model.UploadTask, error) {
	var result model.UploadTask

	err := tr.db.Model(&model.UploadTask{}).
		Where("user_id = ? AND task_id = ?", userID, taskID).
		First(&result).Error

	return &result, err
}

func (tr *TaskRepo) UpdateUploadTask(taskID string, updates *model.UploadTask) error {
	return tr.db.Model(&model.UploadTask{}).
		Where("task_id = ?", taskID).
		Updates(updates).Error
}

func (tr *TaskRepo) BatchCreateFileRecords(fileRecords []*model.UploadFileRecord) error {
	return tr.db.Create(fileRecords).Error
}

func (tr *TaskRepo) GetFileRecordsByTaskID(taskID string) ([]*model.UploadFileRecord, error) {
	var fileRecords []*model.UploadFileRecord

	err := tr.db.Model(&model.UploadFileRecord{}).
		Where("task_id = ?", taskID).
		Order("file_index ASC").
		Find(&fileRecords).Error
	if err != nil {
		return nil, err
	}

	return fileRecords, nil
}

func (tr *TaskRepo) UpdateFileRecord(taskID string, fileIndex int, update *model.UploadFileRecord) error {
	return tr.db.Model(&model.UploadFileRecord{}).
		Where("task_id = ? AND file_index = ?", taskID, fileIndex).
		Updates(update).Error
}

func (tr *TaskRepo) GetFileRecordByTaskAndIndex(taskID string, fileIndex int) (*model.UploadFileRecord, error) {
	var record model.UploadFileRecord
	err := tr.db.Model(&model.UploadFileRecord{}).
		Where("task_id = ? AND file_index = ?", taskID, fileIndex).
		First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (tr *TaskRepo) GetAnyUploadTask(taskID string) (*model.UploadTask, error) {
	var result model.UploadTask
	err := tr.db.Model(&model.UploadTask{}).
		Where("task_id = ?", taskID).
		First(&result).Error
	return &result, err
}
