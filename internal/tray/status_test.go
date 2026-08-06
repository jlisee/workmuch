package tray

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"workmuch-go/internal/app"
	"workmuch-go/internal/backend"
	"workmuch-go/internal/doctor"
	"workmuch-go/internal/platform"
	"workmuch-go/internal/status"
)

func TestCollectStatusReportUsesPersistedCollectorSample(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("DISPLAY", "")
	logDir, err := platform.EnsureLogDir(homeDir)
	require.NoError(t, err)
	store := status.NewStore(platform.StatusPath(logDir))
	startedAt := time.Date(2026, time.August, 4, 11, 0, 0, 0, time.UTC)
	require.NoError(t, store.Write(status.RuntimeStatus{
		StartedAt:       &startedAt,
		SelectedBackend: backend.BackendAuto,
		ActiveBackend:   backend.BackendLinux,
		LastSuccessfulSample: &status.ActivitySample{
			ProgramName: "Firefox",
			WindowTitle: "WorkMuch status",
			IdleSeconds: 2.5,
		},
	}))

	report := collectStatusReport(app.DefaultOptions())

	assert.Empty(t, report.BackendError)
	assert.Equal(t, doctor.SampleSourceRuntime, report.Sample.Source)
	assert.Equal(t, "Firefox", report.Sample.FrontmostApp)
	assert.Equal(t, "WorkMuch status", report.Sample.WindowTitle)
	html := doctor.RenderHTML(report)
	assert.Contains(t, html, "collector last successful sample")
	assert.Contains(t, html, "Firefox")
	assert.Contains(t, html, "WorkMuch status")
}

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

func TestOpenFileCommandLinuxUsesBrowserURL(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "status page.html")
	require.NoError(t, os.WriteFile(path, []byte("<h1>Status</h1>"), 0o600))

	cmd, err := openFileCommand("linux", path)
	require.NoError(t, err)
	require.NotNil(t, cmd)
	require.Len(t, cmd.Args, 2)
	assert.Equal(t, "xdg-open", cmd.Args[0])
	target, err := url.Parse(cmd.Args[1])
	require.NoError(t, err)
	assert.Equal(t, "http", target.Scheme)
	assert.Equal(t, "127.0.0.1", target.Hostname())
	response, err := http.Get(target.String())
	require.NoError(t, err)
	t.Cleanup(func() { _ = response.Body.Close() })
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)
	assert.Equal(t, "<h1>Status</h1>", string(body))
}

func fileMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	require.NoError(t, err)
	return info.Mode().Perm()
}
