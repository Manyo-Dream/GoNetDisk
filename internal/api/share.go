package api

import "time"

type ShareCreateRequest struct {
	UserFileID uint64 `json:"user_file_id" binding:"required"`
	Code       string `json:"code"`
	ExpireDays int    `json:"expire_days"`
}

type ShareCreateResponse struct {
	ShareCode string     `json:"share_code"`
	Code      string     `json:"code"`
	ExpireAt  *time.Time `json:"expire_at"`
	CreatedAt time.Time  `json:"created_at"`
}

type ShareListResponse struct {
	List     []ShareItem `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

type ShareItem struct {
	ShareCode string     `json:"share_code"`
	FileName  string     `json:"file_name"`
	IsDir     bool       `json:"is_dir"`
	Code      string     `json:"code"`
	ExpireAt  *time.Time `json:"expire_at"`
	ViewCount uint64     `json:"view_count"`
	CreatedAt time.Time  `json:"created_at"`
}

type ShareInfoResponse struct {
	ShareCode string     `json:"share_code"`
	FileName  string     `json:"file_name"`
	FileExt   string     `json:"file_ext"`
	FileSize  int64      `json:"file_size"`
	IsDir     bool       `json:"is_dir"`
	ExpireAt  *time.Time `json:"expire_at"`
	HasCode   bool       `json:"has_code"`
}
