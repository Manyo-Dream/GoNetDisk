package dto

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
