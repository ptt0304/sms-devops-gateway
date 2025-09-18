package handler

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// nowLocal trả về thời gian hiện tại theo timezone của container
func nowLocal() time.Time {
	return time.Now().Local()
}

func defaultIfEmpty(value, defaultValue string) string {
	if value == "" {
		return defaultValue
	}
	return value
}

func matchWithWildcard(pattern, value string) bool {
	if pattern == "*" {
		return true
	}
	return strings.EqualFold(pattern, value)
}

func logToFile(logFile *os.File, msg string) {
	logFile.WriteString(fmt.Sprintf("[%s] %s\n", nowLocal().Format(time.RFC3339), msg))
}
