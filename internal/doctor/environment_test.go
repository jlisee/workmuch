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

func TestDetectLinuxSessionReportsWaylandAsUnsupported(t *testing.T) {
	t.Parallel()

	environment := map[string]string{
		"XDG_SESSION_TYPE": "wayland",
		"DISPLAY":          ":1",
		"WAYLAND_DISPLAY":  "wayland-0",
	}

	report := DetectLinuxSession("linux", func(name string) string {
		return environment[name]
	})

	assert.True(t, report.Applicable)
	assert.Equal(t, "wayland", report.Type)
	assert.Equal(t, LinuxSessionUnsupported, report.Support)
	assert.Equal(t, ":1", report.X11Display)
	assert.Equal(t, "wayland-0", report.WaylandDisplay)
	assert.Contains(t, report.Detail, "only X11/Xorg")
}

func TestDetectLinuxSessionInfersX11FromDisplay(t *testing.T) {
	t.Parallel()

	report := DetectLinuxSession("linux", func(name string) string {
		if name == "DISPLAY" {
			return ":0"
		}
		return ""
	})

	assert.True(t, report.Applicable)
	assert.Equal(t, "x11", report.Type)
	assert.Equal(t, LinuxSessionSupported, report.Support)
	assert.Equal(t, ":0", report.X11Display)
}

func TestDetectLinuxSessionIsNotApplicableOutsideLinux(t *testing.T) {
	t.Parallel()

	report := DetectLinuxSession("darwin", func(string) string {
		return "unexpected"
	})

	assert.False(t, report.Applicable)
}

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
