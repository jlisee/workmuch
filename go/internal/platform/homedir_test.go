package platform

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestEnsureLogDirUsesPrivatePermissions(t *testing.T) {
	homeDir := t.TempDir()
	logDir := LogDir(homeDir)
	require.NoError(t, os.Mkdir(logDir, 0o755))

	ensuredLogDir, err := EnsureLogDir(homeDir)
	require.NoError(t, err)
	assert.Equal(t, logDir, ensuredLogDir)

	info, err := os.Stat(logDir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), info.Mode().Perm())
}

func TestOpenPrivateAppendFileUsesPrivatePermissions(t *testing.T) {
	testCases := []struct {
		name     string
		existing bool
	}{
		{name: "new file"},
		{name: "existing file", existing: true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "activity.log")
			if testCase.existing {
				require.NoError(t, os.WriteFile(path, []byte("existing\n"), 0o644))
			}

			file, err := OpenPrivateAppendFile(path)
			require.NoError(t, err)
			require.NoError(t, file.Close())

			info, err := os.Stat(path)
			require.NoError(t, err)
			assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
		})
	}
}
