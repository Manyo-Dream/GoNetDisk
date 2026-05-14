package dto

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
