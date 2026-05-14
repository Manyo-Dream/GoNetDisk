package util

import (
	"fmt"
	"strings"
	"time"
)

var sanitizeReplacer = strings.NewReplacer(
	"/", "_",
	"\\", "_",
	"<", "_",
	">", "_",
	":", "_",
	"\"", "_",
	"|", "_",
	"?", "_",
	"*", "_",
	"\x00", "",
)

func SanitizeFileName(name string) string {
	cleaned := sanitizeReplacer.Replace(name)

	var builder strings.Builder
	builder.Grow(len(cleaned))
	for _, r := range cleaned {
		if r < 32 {
			continue
		}
		builder.WriteRune(r)
	}
	result := builder.String()

	result = strings.TrimRight(result, " .")

	if result == "" || result == "." || result == ".." {
		result = fmt.Sprintf("file_%d", time.Now().UnixNano())
	}

	return result
}
