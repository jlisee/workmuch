package platform

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLogDirAndPaths(t *testing.T) {
	home := "/Users/tester"
	logDir := LogDir(home)
	expectedLogDir := filepath.Join(home, ".workmuch")
	if logDir != expectedLogDir {
		t.Fatalf("expected log dir %q, got %q", expectedLogDir, logDir)
	}

	now := time.Date(2026, time.February, 27, 14, 0, 0, 0, time.UTC)
	workLogPath := WorkLogPath(now, logDir)
	expectedWorkLogPath := filepath.Join(expectedLogDir, "2026-02-27.worklog")
	if workLogPath != expectedWorkLogPath {
		t.Fatalf("expected work log path %q, got %q", expectedWorkLogPath, workLogPath)
	}

	errorLogPath := ErrorLogPath(logDir)
	expectedErrorLogPath := filepath.Join(expectedLogDir, "error.log")
	if errorLogPath != expectedErrorLogPath {
		t.Fatalf("expected error log path %q, got %q", expectedErrorLogPath, errorLogPath)
	}
}
