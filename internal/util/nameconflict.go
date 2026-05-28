package util

import (
	"fmt"
	"regexp"
	"time"

	"GoNetDisk/internal/model"
	"GoNetDisk/internal/repository"
)

func IsNameExistsInFolder(repo *repository.FileRepo, userID, parentID uint64, name string, excludeID uint64, isDir bool) bool {
	var result *model.UserFile
	var err error

	if isDir {
		result, err = repo.GetUserFolderByFileName(userID, parentID, name)
	} else {
		result, err = repo.GetUserFileByFileName(userID, parentID, name)
	}

	if err != nil {
		return false
	}

	return result != nil && !result.DeletedAt.Valid && result.ID != excludeID
}

func GenerateUniqueName(repo *repository.FileRepo, userID, parentID uint64, baseName, ext string, excludeID uint64, isDir bool) (string, error) {
	const maxAttempts = 9999

	pattern := regexp.MustCompile(`^(.+?)(?:\((\d+)\))?$`)
	matches := pattern.FindStringSubmatch(baseName)

	var namePart string
	if matches != nil {
		namePart = matches[1]
	} else {
		namePart = baseName
	}

	for i := 1; i <= maxAttempts; i++ {
		candidateName := fmt.Sprintf("%s(%d)%s", namePart, i, ext)
		if !IsNameExistsInFolder(repo, userID, parentID, candidateName, excludeID, isDir) {
			return candidateName, nil
		}
	}

	uniqueName := fmt.Sprintf("%s_%d%s", namePart, time.Now().UnixNano(), ext)
	return uniqueName, nil
}
