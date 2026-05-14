package util

import (
	"fmt"
	"strconv"

	"github.com/manyodream/gonetdisk/internal/repository"
)

func BuildPathStack(repo *repository.FileRepo, userID, parentID, itemID uint64) (string, error) {
	if parentID == 0 {
		return fmt.Sprintf("/0/%d", itemID), nil
	}

	parentFolder, err := repo.GetUserFolderByID(userID, parentID)
	if err != nil {
		return "", fmt.Errorf("父目录不存在或不是当前用户目录: %w", err)
	}

	return parentFolder.PathStack + "/" + strconv.FormatUint(itemID, 10), nil
}
