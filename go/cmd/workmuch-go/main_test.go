package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"workmuch-go/internal/app"
	"workmuch-go/internal/backend"
)

func TestSelectRunModeDefaultsToTray(t *testing.T) {
	t.Parallel()

	assert.Equal(t, runModeTray, selectRunMode(app.DefaultOptions()))
}

func TestSelectRunModeUsesForegroundForQA(t *testing.T) {
	t.Parallel()

	opts := app.DefaultOptions()
	opts.QAConsole = true

	assert.Equal(t, runModeForeground, selectRunMode(opts))
}

func TestSelectRunModeUsesForegroundWithoutTray(t *testing.T) {
	t.Parallel()

	opts := app.DefaultOptions()
	opts.NoTray = true

	assert.Equal(t, runModeForeground, selectRunMode(opts))
}

func TestParseCommandDefaultsToRun(t *testing.T) {
	t.Parallel()

	cmd, showHelp, err := parseCommand([]string{"--qa-console"})
	require.NoError(t, err)

	assert.False(t, showHelp)
	assert.Equal(t, commandRun, cmd.kind)
	assert.True(t, cmd.opts.QAConsole)
}

func TestParseCommandAcceptsNoTray(t *testing.T) {
	t.Parallel()

	cmd, showHelp, err := parseCommand([]string{"--no-tray"})
	require.NoError(t, err)

	assert.False(t, showHelp)
	assert.Equal(t, commandRun, cmd.kind)
	assert.True(t, cmd.opts.NoTray)
}

func TestParseCommandRecognizesDoctor(t *testing.T) {
	t.Parallel()

	cmd, showHelp, err := parseCommand([]string{"doctor", "--backend", backend.BackendMacOSNative})
	require.NoError(t, err)

	assert.False(t, showHelp)
	assert.Equal(t, commandDoctor, cmd.kind)
	assert.Equal(t, backend.BackendMacOSNative, cmd.opts.Backend)
}
