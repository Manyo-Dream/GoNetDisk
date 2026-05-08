package dto

type FolderRequest struct {
	FolderName string `json:"folder_name" form:"folder_name" binding:"required"`
	ParentID   uint64 `json:"parent_id" form:"parent_id"`
}

type FolderRenameRequest struct {
	UserFolderID  uint64 `json:"userfolder_id" form:"userfolder_id" binding:"required,min=1"`
	NewFolderName string `json:"new_foldername" form:"new_foldername" binding:"required"`
}

type FolderMoveRequest struct {
	UserFolderID  uint64 `json:"userfolder_id" form:"userfolder_id" binding:"required,min=1"`
	TargetParenID uint64 `json:"target_parent_id" form:"target_parent_id"`
}
