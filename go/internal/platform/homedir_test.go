package platform

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLogDirAndPaths(t *testing.T) {
	home := "/Users/tester"
	logDir := LogDir(home)
	expectedLogDir := filepath.Join(home, ".workmuch")
	assert.Equal(t, expectedLogDir, logDir)

	now := time.Date(2026, time.February, 27, 14, 0, 0, 0, time.UTC)
	workLogPath := WorkLogPath(now, logDir)
	expectedWorkLogPath := filepath.Join(expectedLogDir, "2026-02-27.worklog")
	assert.Equal(t, expectedWorkLogPath, workLogPath)

	errorLogPath := ErrorLogPath(logDir)
	expectedErrorLogPath := filepath.Join(expectedLogDir, "error.log")
	assert.Equal(t, expectedErrorLogPath, errorLogPath)
}
