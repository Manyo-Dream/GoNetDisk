package dto

type FolderRequest struct {
	FolderName string `json:"folder_name" form:"folder_name" binding:"required"`
	ParentID   uint64 `json:"parent_id" form:"parent_id"`
}

type FolderRenameRequest struct {
	UserFolderID  uint64 `json:"user_folder_id" form:"user_folder_id" binding:"required,min=1"`
	NewFolderName string `json:"new_folder_name" form:"new_folder_name" binding:"required"`
}

type FolderMoveRequest struct {
	UserFolderID  uint64 `json:"user_folder_id" form:"user_folder_id" binding:"required,min=1"`
	TargetParentID uint64 `json:"target_parent_id" form:"target_parent_id"`
}
