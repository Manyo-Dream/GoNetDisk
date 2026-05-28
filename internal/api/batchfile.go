package api

type BatchFileMeta struct {
	Index        int    `json:"index" binding:"required"`
	FileName     string `json:"file_name" binding:"required"`
	FileSize     int64  `json:"file_size" binding:"required"`
	FileExt      string `json:"file_ext"`
	RelativePath string `json:"relative_path"`
}

type BatchUploadRequest struct {
	ParentID uint64          `json:"parent_id"`
	Files    []BatchFileMeta `json:"files"`
}

type BatchUploadResponse struct {
	TaskID    string `json:"task_id"`
	FileCount int    `json:"file_count"`
}

type FileProgressItem struct {
	Index      int    `json:"index"`
	FileName   string `json:"file_name"`
	Status     int    `json:"status"`
	ErrorMsg   string `json:"error_msg,omitempty"`
	UserFileID uint64 `json:"userfile_id,omitempty"`
}

type TaskProgressResponse struct {
	TaskID       string             `json:"task_id"`
	Status       int                `json:"status"`
	FileCount    int                `json:"file_count"`
	SuccessCount int                `json:"success_count"`
	FailCount    int                `json:"fail_count"`
	ProgressPct  int                `json:"progress_pct"`
	ErrorMsg     string             `json:"error_msg"`
	Files        []FileProgressItem `json:"files"`
}
