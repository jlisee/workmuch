package app

import (
	"bytes"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"workmuch-go/internal/platform"
)

func TestComputeNextSleepOnSchedule(t *testing.T) {
	now := time.Date(2026, time.February, 27, 10, 0, 0, 0, time.UTC)
	wakeAt := now.Add(2 * time.Second)
	period := 2 * time.Second

	sleepDuration, nextWakeAt := ComputeNextSleep(now, wakeAt, period)
	assert.Equal(t, 2*time.Second, sleepDuration)
	assert.True(t, nextWakeAt.Equal(wakeAt.Add(period)))
}

func TestComputeNextSleepWhenBehind(t *testing.T) {
	now := time.Date(2026, time.February, 27, 10, 0, 7, 0, time.UTC)
	wakeAt := time.Date(2026, time.February, 27, 10, 0, 2, 0, time.UTC)
	period := 2 * time.Second

	sleepDuration, nextWakeAt := ComputeNextSleep(now, wakeAt, period)
	assert.GreaterOrEqual(t, sleepDuration, time.Duration(0))
	expectedCurrentWake := time.Date(2026, time.February, 27, 10, 0, 8, 0, time.UTC)
	assert.Equal(t, expectedCurrentWake.Sub(now), sleepDuration)
	assert.True(t, nextWakeAt.Equal(expectedCurrentWake.Add(period)))
}

func TestLogBackendSelection(t *testing.T) {
	var output bytes.Buffer
	logger := log.New(&output, "", 0)

	logBackendSelection(logger, "auto", "macos-native")

	assert.Equal(t, "backend requested=auto active=macos-native", strings.TrimSpace(output.String()))
}

func TestOpenCSVOutputNoTrayUsesPrivateDailyWorklog(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	opts := DefaultOptions()
	opts.NoTray = true

	output, err := openCSVOutput(opts)
	require.NoError(t, err)
	t.Cleanup(output.close)

	assert.Equal(t, platform.LogDir(homeDir), output.logDir)
	assert.Equal(t, filepath.Dir(output.workLogPath), output.logDir)
	assert.NotEqual(t, os.Stdout, output.writer)

	info, err := os.Stat(output.workLogPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())
}
