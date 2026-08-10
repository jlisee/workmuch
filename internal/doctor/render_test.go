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
			Source:               SampleSourceDiagnostic,
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
		LoginItem: LoginItemReport{State: LoginItemNotRegistered},
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
	assert.Contains(t, output, "Sample source: live diagnostic probe")
	assert.Contains(t, output, "Frontmost app: Safari")
	assert.Contains(t, output, "Focused window title: Docs")
	assert.Contains(t, output, "Idle seconds: 1.25")
	assert.Contains(t, output, "Log directory: /Users/test/.workmuch (writable)")
	assert.Contains(t, output, "Current work log: /Users/test/.workmuch/2026-07-08.worklog")
	assert.Contains(t, output, "Login Item: not_registered")
	assert.Contains(t, output, "Runtime status: running")
	assert.Contains(t, output, "Runtime selected backend: auto")
	assert.Contains(t, output, "Runtime active backend: macos-native")
	assert.Contains(t, output, "Sample count: 4")
}

func TestRenderTextExplainsWaylandAndX11Failures(t *testing.T) {
	t.Parallel()

	report := DoctorReport{
		LinuxSession: LinuxSessionReport{
			Applicable:     true,
			Type:           "wayland",
			Support:        LinuxSessionUnsupported,
			X11Display:     ":1",
			WaylandDisplay: "wayland-0",
			Detail:         "Wayland detected; WorkMuch currently supports only X11/Xorg sessions.",
		},
		X11: X11Report{
			Applicable:      true,
			Display:         ":1",
			Connection:      X11ConnectionFailed,
			ConnectionError: "connect to X11 display \" :1\": connection refused",
			Sampling:        X11SamplingNotAttempted,
		},
	}

	output := RenderText(report)

	assert.Contains(t, output, "Desktop session: wayland")
	assert.Contains(t, output, "Linux session support: unsupported")
	assert.Contains(t, output, "only X11/Xorg sessions")
	assert.Contains(t, output, "X11 display: :1")
	assert.Contains(t, output, "Wayland display: wayland-0")
	assert.Contains(t, output, "X11 connection: failed")
	assert.Contains(t, output, "X11 connection error: connect to X11 display")
	assert.Contains(t, output, "X11 sampling: not attempted")
}

func TestRenderHTMLShowsX11SamplingFailure(t *testing.T) {
	t.Parallel()

	report := DoctorReport{
		LinuxSession: LinuxSessionReport{
			Applicable: true,
			Type:       "x11",
			Support:    LinuxSessionSupported,
			X11Display: ":0",
		},
		X11: X11Report{
			Applicable:    true,
			Display:       ":0",
			Connection:    X11ConnectionConnected,
			Sampling:      X11SamplingFailed,
			SamplingError: "idle time query failed",
		},
	}

	output := RenderHTML(report)

	assert.Contains(t, output, "X11 connection")
	assert.Contains(t, output, "connected")
	assert.Contains(t, output, "X11 sampling")
	assert.Contains(t, output, "idle time query failed")
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

func TestRenderTextShowsEveryLoginItemState(t *testing.T) {
	t.Parallel()

	states := []LoginItemState{
		LoginItemNotRegistered,
		LoginItemEnabled,
		LoginItemRequiresApproval,
		LoginItemNotFound,
		LoginItemUnsupported,
	}
	for _, state := range states {
		output := RenderText(DoctorReport{LoginItem: LoginItemReport{State: state}})

		assert.Contains(t, output, "Login Item: "+string(state))
		assert.NotContains(t, output, "LaunchAgent")
	}
}
