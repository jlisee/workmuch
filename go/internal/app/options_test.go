package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOptionsDefaults(t *testing.T) {
	opts, showHelp, err := ParseOptions(nil)
	require.NoError(t, err)
	assert.False(t, showHelp)
	assert.Equal(t, 1.0, opts.Rate)
	assert.Equal(t, 0.0, opts.StartDelay)
}

func TestParseOptionsShortFlags(t *testing.T) {
	opts, showHelp, err := ParseOptions([]string{"-r", "2.5", "-d", "1.25", "--qa-console"})
	require.NoError(t, err)
	assert.False(t, showHelp)
	assert.Equal(t, 2.5, opts.Rate)
	assert.Equal(t, 1.25, opts.StartDelay)
	assert.True(t, opts.QAConsole)
}

func TestParseOptionsLongFlags(t *testing.T) {
	opts, _, err := ParseOptions([]string{"--rate=3", "--start-delay=4", "--backend", "macos-subprocess"})
	require.NoError(t, err)
	assert.Equal(t, 3.0, opts.Rate)
	assert.Equal(t, 4.0, opts.StartDelay)
	assert.Equal(t, "macos-subprocess", opts.Backend)
}

func TestParseOptionsHelp(t *testing.T) {
	_, showHelp, err := ParseOptions([]string{"--help"})
	require.NoError(t, err)
	assert.True(t, showHelp)
}

func TestHelpTextDocumentsTrayDefaultAndQAMode(t *testing.T) {
	t.Parallel()

	helpText := HelpText("workmuch-go")

	assert.Contains(t, helpText, "Launch the tray icon and log activity in the background.")
	assert.Contains(t, helpText, "--qa-console            Disable tray mode and write CSV to stdout")
}

func TestParseOptionsRateMustBePositive(t *testing.T) {
	_, _, err := ParseOptions([]string{"--rate", "0"})
	require.Error(t, err)
}

func TestParseOptionsUnknownFlag(t *testing.T) {
	_, _, err := ParseOptions([]string{"--nope"})
	require.Error(t, err)
}
