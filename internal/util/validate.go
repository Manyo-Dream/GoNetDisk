package util

import (
	"errors"
	"regexp"
	"strings"
)

func ValidateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("名称不能为空")
	}

	if len(name) > 255 {
		return errors.New("名称过长")
	}

	illegal := `[<>:"/\\|?*]`
	matched, _ := regexp.MatchString(illegal, name)
	if matched {
		return errors.New("名称包含非法字符")
	}

	if name == "." || name == ".." {
		return errors.New("名称不能为 '.' 或 '..'")
	}

	return nil
}
