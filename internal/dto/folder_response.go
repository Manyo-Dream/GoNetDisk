package dto

type FolderResponse struct {
	FolderName string `json:"folder_name" form:"folder_name" binding:"required"`
	ParentID   uint64 `json:"parent_id" form:"parent_id"`
	FolderID   uint64 `json:"folder_id" form:"folder_id"`
}

type FolderRenameResponse struct {
	UserFolderID uint64 `json:"userfolder_id"`
	FolderName   string `json:"folder_name"`
}

type FolderMoveResponse struct {
	UserFolderID uint64 `json:"userfolder_id"`
	FolderName   string `json:"folder_name"`
	NewParentID  uint64 `json:"new_parent_id"`
	NewPathStack string `json:"new_path_stack"`
}
