package api

import "GoNetDisk/internal/model"

type TrashListRequest struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

type TrashListResponse struct {
	List     []model.UserFile `json:"list"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

type TrashDeleteResponse struct {
	Message string `json:"message"`
}

type TrashRestoreResponse struct {
	Message string `json:"message"`
}
