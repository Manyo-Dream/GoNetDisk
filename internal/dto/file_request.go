package dto

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
	UserFileID  uint64 `json:"userfile_id" form:"userfile_id" binding:"required,min=1"`
	NewFileName string `json:"new_filename" form:"new_filename" binding:"required"`
}

type FileMoveRequest struct {
	UserFileID    uint64 `json:"userfile_id" form:"userfile_id" binding:"required,min=1"`
	TargetParenID uint64 `json:"target_parent_id" form:"target_parent_id"`
}
