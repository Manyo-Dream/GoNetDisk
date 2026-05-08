package util

import "errors"

func ValidatePassword(password string) error {
	if len(password) < 6 {
		return errors.New("密码长度不能少于6位，且必须包含字母和数字")
	}

	hasLetter := false
	hasDigit := false
	for _, c := range password {
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' {
			hasLetter = true
		}
		if c >= '0' && c <= '9' {
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return errors.New("密码长度不能少于6位，且必须包含字母和数字")
	}

	return nil
}
