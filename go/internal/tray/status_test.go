package tray

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"workmuch-go/internal/doctor"
)

func TestEnsureStatusHTMLCreatesPrivateUniqueFile(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	report := doctor.DoctorReport{
		Sample: doctor.SampleReport{
			WindowTitle:          "Private document title",
			WindowTitleAvailable: true,
		},
	}

	firstPath, err := ensureStatusHTML(report)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(firstPath) })
	secondPath, err := ensureStatusHTML(report)
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Remove(secondPath) })

	assert.NotEqual(t, firstPath, secondPath)
	assert.Equal(t, os.FileMode(0o600), fileMode(t, firstPath))
	assert.Equal(t, os.FileMode(0o600), fileMode(t, secondPath))
	assert.Equal(t, os.TempDir(), filepath.Dir(firstPath))
}

func fileMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err)
	return info.Mode().Perm()
}
