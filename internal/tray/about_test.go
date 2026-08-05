package tray

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAboutCommandDarwinUsesAppleScriptDialog(t *testing.T) {
	t.Parallel()

	cmd, err := aboutCommand("darwin", "20260805.43.5+g69d002bbb362")
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Len(t, cmd.Args, 3)
	assert.Equal(t, "osascript", cmd.Args[0])
	assert.Equal(t, "-e", cmd.Args[1])
	assert.Contains(t, cmd.Args[2], "display dialog")
	assert.Contains(t, cmd.Args[2], aboutMessage)
	assert.Contains(t, cmd.Args[2], aboutTitle)
	assert.Contains(t, cmd.Args[2], "20260805.43.5+g69d002bbb362")
}

func TestAboutCommandUnsupportedPlatformReturnsError(t *testing.T) {
	t.Parallel()

	cmd, err := aboutCommand("plan9", "dev")
	require.Error(t, err)
	assert.Nil(t, cmd)
	assert.Contains(t, err.Error(), "unsupported")
}

func TestEnsureAboutHTMLWritesExpectedMessage(t *testing.T) {
	t.Parallel()

	path, err := ensureAboutHTML("20260805.43.5+g69d002bbb362")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(os.TempDir(), "workmuch-about.html"), path)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(content), aboutMessage))
	assert.Contains(t, string(content), "20260805.43.5+g69d002bbb362")
}
