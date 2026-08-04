package doctor

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLaunchAgentCheckerReportsCommandTimeoutAsError(t *testing.T) {
	homeDir := createLaunchAgentPlist(t)
	checker := NativeLaunchAgentChecker{
		Platform: "darwin",
		HomeDir:  func() (string, error) { return homeDir, nil },
		RunLaunchctl: func(context.Context, string) ([]byte, error) {
			return nil, context.DeadlineExceeded
		},
	}

	report := checker.Check(context.Background())

	assert.Equal(t, LaunchAgentError, report.State)
	assert.ErrorContains(t, errors.New(report.Error), context.DeadlineExceeded.Error())
}

func TestLaunchAgentCheckerReportsMissingLaunchctlAsError(t *testing.T) {
	homeDir := createLaunchAgentPlist(t)
	checker := NativeLaunchAgentChecker{
		Platform: "darwin",
		HomeDir:  func() (string, error) { return homeDir, nil },
		RunLaunchctl: func(context.Context, string) ([]byte, error) {
			return nil, os.ErrNotExist
		},
	}

	report := checker.Check(context.Background())

	assert.Equal(t, LaunchAgentError, report.State)
	assert.ErrorContains(t, errors.New(report.Error), os.ErrNotExist.Error())
}

func createLaunchAgentPlist(t *testing.T) string {
	t.Helper()
	homeDir := t.TempDir()
	plistPath := launchAgentPlistPath(homeDir)
	require.NoError(t, os.MkdirAll(filepath.Dir(plistPath), 0o755))
	require.NoError(t, os.WriteFile(plistPath, []byte("plist"), 0o600))
	return homeDir
}
