package dto

type ShareCreateRequest struct {
	UserFileID uint64 `json:"user_file_id" binding:"required"`
	Code       string `json:"code"`
	ExpireDays int    `json:"expire_days"`
}
