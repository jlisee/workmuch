package platform

import (
	"os"
	"path/filepath"
	"time"
)

func UserHomeDir() (string, error) {
	return os.UserHomeDir()
}

func LogDir(homeDir string) string {
	return filepath.Join(homeDir, ".workmuch")
}

func EnsureLogDir(homeDir string) (string, error) {
	logDir := LogDir(homeDir)
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		return "", err
	}
	return logDir, nil
}

func WorkLogPath(now time.Time, logDir string) string {
	fileName := now.Format("2006-01-02") + ".worklog"
	return filepath.Join(logDir, fileName)
}

func ErrorLogPath(logDir string) string {
	return filepath.Join(logDir, "error.log")
}

func StatusPath(logDir string) string {
	return filepath.Join(logDir, "status.json")
}
