package api

// ========== ChunkRequest ==========
type ChunkInitRequest struct {
	FileName  string `json:"file_name" binding:"required"`
	FileSize  int64  `json:"file_size" binding:"required"`
	FileHash  string `json:"file_hash" binding:"required"`
	ParentID  uint   `json:"parent_id"`
	ChunkSize int64  `json:"chunk_size" binding:"required"`
}

type ChunkUploadRequest struct {
	UploadID   string `json:"upload_id" binding:"required"`
	ChunkIndex int    `json:"chunk_index" binding:"required"`
	ChunkHash  string `json:"chunk_hash" binding:"required"`
}

type ChunkCompleteRequest struct {
	UploadID string `json:"upload_id" binding:"required"`
}

type ChunkStatusRequest struct {
	UploadID string `form:"upload_id" binding:"required"`
}

// ========== ChunkResponse ==========
type ChunkInitResponse struct {
	UploadID      string `json:"upload_id"`
	ChunkSize     int64  `json:"chunk_size"`
	ChunkCount    int    `json:"chunk_count"`
	InstantUpload bool   `json:"instant_upload"`
	UserFileID    uint64   `json:"user_file_id,omitempty"`
}

type ChunkUploadResponse struct {
	UploadID   string `json:"upload_id"`
	ChunkIndex int    `json:"chunk_index"`
}

type ChunkCompleteResponse struct {
	UserFileID uint64 `json:"user_file_id"`
	FileName   string `json:"file_name"`
	FileExt    string `json:"file_ext"`
	FileSize   int64  `json:"file_size"`
	ParentID   uint64 `json:"parent_id"`
}

type ChunkStatusResponse struct {
	UploadID       string `json:"upload_id"`
	FileName       string `json:"file_name"`
	ChunkSize      int64  `json:"chunk_size"`
	ChunkCount     int    `json:"chunk_count"`
	Status         int    `json:"status"`
	UploadedChunks []int  `json:"uploaded_chunks"`
}
