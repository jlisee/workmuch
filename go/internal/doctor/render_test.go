package doctor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"workmuch-go/internal/status"
)

func TestRenderTextUsesStableLabels(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, time.July, 8, 10, 0, 0, 0, time.UTC)
	report := DoctorReport{
		SelectedBackend: "auto",
		ActiveBackend:   "macos-native",
		Permission:      PermissionReport{Name: "Accessibility", State: PermissionGranted},
		Sample: SampleReport{
			FrontmostApp:         "Safari",
			WindowTitle:          "Docs",
			WindowTitleAvailable: true,
			IdleSeconds:          1.25,
		},
		Logs: LogDirectoryReport{
			Directory:      "/Users/test/.workmuch",
			Writable:       true,
			CurrentWorkLog: "/Users/test/.workmuch/2026-07-08.worklog",
		},
		LaunchAgent: LaunchAgentReport{State: LaunchAgentMissing},
		Runtime: RuntimeStatusReport{
			Path:    "/Users/test/.workmuch/status.json",
			Present: true,
			Status: status.RuntimeStatus{
				StartedAt:          &startedAt,
				SampleCount:        4,
				SelectedBackend:    "auto",
				ActiveBackend:      "macos-native",
				CurrentWorkLogPath: "/Users/test/.workmuch/2026-07-08.worklog",
			},
		},
	}

	output := RenderText(report)

	assert.Contains(t, output, "WorkMuch Doctor")
	assert.Contains(t, output, "Selected backend: auto")
	assert.Contains(t, output, "Active backend: macos-native")
	assert.Contains(t, output, "Accessibility permission: granted")
	assert.Contains(t, output, "Frontmost app: Safari")
	assert.Contains(t, output, "Focused window title: Docs")
	assert.Contains(t, output, "Idle seconds: 1.25")
	assert.Contains(t, output, "Log directory: /Users/test/.workmuch (writable)")
	assert.Contains(t, output, "Current work log: /Users/test/.workmuch/2026-07-08.worklog")
	assert.Contains(t, output, "LaunchAgent: missing")
	assert.Contains(t, output, "Runtime status: running")
	assert.Contains(t, output, "Runtime selected backend: auto")
	assert.Contains(t, output, "Runtime active backend: macos-native")
	assert.Contains(t, output, "Sample count: 4")
}

func TestRenderHTMLEscapesReportValues(t *testing.T) {
	t.Parallel()

	report := DoctorReport{
		SelectedBackend: "auto",
		ActiveBackend:   "macos-native",
		Permission:      PermissionReport{Name: "Accessibility", State: PermissionDenied},
		Sample: SampleReport{
			FrontmostApp: "<Safari>",
			WindowTitle:  "Docs & Notes",
		},
	}

	output := RenderHTML(report)

	assert.Contains(t, output, "<title>WorkMuch Status</title>")
	assert.Contains(t, output, "&lt;Safari&gt;")
	assert.Contains(t, output, "Docs &amp; Notes")
	assert.NotContains(t, output, "<Safari>")
}

func TestRenderHTMLShowsRuntimeBackendSelection(t *testing.T) {
	t.Parallel()

	report := DoctorReport{
		Runtime: RuntimeStatusReport{
			Present: true,
			Status: status.RuntimeStatus{
				SelectedBackend: "macos-subprocess",
				ActiveBackend:   "macos-subprocess",
			},
		},
	}

	output := RenderHTML(report)

	assert.Contains(t, output, "Runtime selected backend")
	assert.Contains(t, output, "macos-subprocess")
	assert.Contains(t, output, "Runtime active backend")
}
