package api

import "GoNetDisk/internal/model"

type FileUploadRequest struct {
	ParentID uint64 `form:"parent_id"`
}

type FileDownloadRequest struct {
	UserFileID uint64 `uri:"userfile_id" binding:"required,min=1"`
}

type FileListRequest struct {
	ParentID uint64 `form:"parent_id"` // 父目录ID，0表示根目录
	Page     int    `form:"page"`      // 页码，默认1
	PageSize int    `form:"page_size"` // 每页数量，默认5
	SortBy   string `form:"sort_by"`   // 排序字段: name/size/updated_at
	OrderBy  string `form:"order_by"`  // 排序方向: asc/desc
}

type FileRenameRequest struct {
	UserFileID  uint64 `json:"user_file_id" form:"user_file_id" binding:"required,min=1"`
	NewFileName string `json:"new_file_name" form:"new_file_name" binding:"required"`
}

type FileMoveRequest struct {
	UserFileID     uint64 `json:"user_file_id" form:"user_file_id" binding:"required,min=1"`
	TargetParentID uint64 `json:"target_parent_id" form:"target_parent_id"`
}

type FileUploadResponse struct {
	UserFileID uint64 `json:"userfile_id"`
	FileName   string `json:"file_name"`
	FileExt    string `json:"file_ext"`
	FIleSize   int64  `json:"file_size"`
	ParentID   uint64 `json:"parent_id"`
	FilePath   string `json:"file_path"`
}

type FileDownloadResponse struct {
	FileName    string `json:"file_name"`
	StorageType string `json:"storage_type"`
	FileExt     string `json:"file_ext"`
	FileSize    int64  `json:"file_size"`
}

type FileListResponse struct {
	List     []model.UserFile `json:"list"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

type FileRenameResponse struct {
	UserFileID uint64 `json:"userfile_id"`
	FileName   string `json:"file_name"`
	FileExt    string `json:"file_ext"`
}

type FileMoveResponse struct {
	UserFileID   uint64 `json:"userfile_id"`
	FileName     string `json:"file_name"`
	NewParentID  uint64 `json:"new_parent_id"`
	NewPathStack string `json:"new_path_stack"`
}
