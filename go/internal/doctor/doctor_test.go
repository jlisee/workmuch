package doctor

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"workmuch-go/internal/backend"
	"workmuch-go/internal/status"
)

type fakeBackend struct {
	name      string
	sample    backend.UsageSample
	sampleErr error
	closed    bool
}

func (f *fakeBackend) Name() string {
	return f.name
}

func (f *fakeBackend) Sample(context.Context) (backend.UsageSample, error) {
	return f.sample, f.sampleErr
}

func (f *fakeBackend) Reset() error {
	return nil
}

func (f *fakeBackend) Close() error {
	f.closed = true
	return nil
}

type fakeRuntimeStatusStore struct {
	path   string
	value  status.RuntimeStatus
	err    error
	called bool
}

func (s *fakeRuntimeStatusStore) Read() (status.RuntimeStatus, error) {
	s.called = true
	return s.value, s.err
}

func (s *fakeRuntimeStatusStore) Path() string {
	return s.path
}

func TestCollectorBuildsDoctorReport(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.July, 8, 10, 0, 0, 0, time.UTC)
	startedAt := now.Add(-time.Minute)
	statusStore := &fakeRuntimeStatusStore{
		path: "/Users/test/.workmuch/status.json",
		value: status.RuntimeStatus{
			StartedAt:          &startedAt,
			SampleCount:        12,
			SelectedBackend:    "auto",
			ActiveBackend:      "macos-native",
			CurrentWorkLogPath: "/Users/test/.workmuch/2026-07-08.worklog",
		},
	}
	usageBackend := &fakeBackend{
		name: backend.BackendMacOSNative,
		sample: backend.UsageSample{
			ProgramName: "Safari",
			WindowTitle: "Docs",
			IdleSeconds: 2.5,
		},
	}
	var requestedPlatform string
	var requestedBackend string
	collector := Collector{
		Platform:        "darwin",
		SelectedBackend: backend.BackendAuto,
		NewBackend: func(platform string, backendName string) (backend.Backend, error) {
			requestedPlatform = platform
			requestedBackend = backendName
			return usageBackend, nil
		},
		PermissionChecker: PermissionCheckerFunc(func(context.Context, string) PermissionReport {
			return PermissionReport{Name: "Accessibility", State: PermissionGranted}
		}),
		LogDirReporter: func(time.Time) LogDirectoryReport {
			return LogDirectoryReport{
				Directory:      "/Users/test/.workmuch",
				Writable:       true,
				CurrentWorkLog: "/Users/test/.workmuch/2026-07-08.worklog",
			}
		},
		LaunchAgentChecker: LaunchAgentCheckerFunc(func(context.Context) LaunchAgentReport {
			return LaunchAgentReport{State: LaunchAgentMissing}
		}),
		RuntimeStatusStore: statusStore,
		Now:                func() time.Time { return now },
	}

	report := collector.Collect(context.Background())

	assert.Equal(t, "darwin", requestedPlatform)
	assert.Equal(t, backend.BackendAuto, requestedBackend)
	assert.Equal(t, backend.BackendAuto, report.SelectedBackend)
	assert.Equal(t, backend.BackendMacOSNative, report.ActiveBackend)
	assert.Equal(t, PermissionGranted, report.Permission.State)
	assert.Equal(t, "Safari", report.Sample.FrontmostApp)
	assert.Equal(t, "Docs", report.Sample.WindowTitle)
	assert.True(t, report.Sample.WindowTitleAvailable)
	assert.Equal(t, 2.5, report.Sample.IdleSeconds)
	assert.Equal(t, "/Users/test/.workmuch", report.Logs.Directory)
	assert.True(t, report.Logs.Writable)
	assert.Equal(t, LaunchAgentMissing, report.LaunchAgent.State)
	require.True(t, report.Runtime.Present)
	assert.Equal(t, int64(12), report.Runtime.Status.SampleCount)
	assert.True(t, statusStore.called)
	assert.True(t, usageBackend.closed)
}

func TestCollectorKeepsPartialReportWhenBackendFails(t *testing.T) {
	t.Parallel()

	backendErr := errors.New("backend unavailable")
	collector := Collector{
		Platform:        "linux",
		SelectedBackend: backend.BackendLinux,
		SessionReporter: func(string) LinuxSessionReport {
			return LinuxSessionReport{
				Applicable: true,
				Type:       "x11",
				Support:    LinuxSessionSupported,
				X11Display: ":99",
			}
		},
		NewBackend: func(string, string) (backend.Backend, error) {
			return nil, backendErr
		},
		PermissionChecker: PermissionCheckerFunc(func(context.Context, string) PermissionReport {
			return PermissionReport{Name: "Accessibility", State: PermissionNotApplicable}
		}),
		LogDirReporter: func(time.Time) LogDirectoryReport {
			return LogDirectoryReport{Directory: "/tmp/workmuch", Writable: true}
		},
		LaunchAgentChecker: LaunchAgentCheckerFunc(func(context.Context) LaunchAgentReport {
			return LaunchAgentReport{State: LaunchAgentNotApplicable}
		}),
		RuntimeStatusStore: &fakeRuntimeStatusStore{err: status.ErrNotFound},
	}

	report := collector.Collect(context.Background())

	assert.Equal(t, backend.BackendLinux, report.SelectedBackend)
	assert.Empty(t, report.ActiveBackend)
	assert.Contains(t, report.BackendError, "backend unavailable")
	assert.True(t, report.X11.Applicable)
	assert.Equal(t, ":99", report.X11.Display)
	assert.Equal(t, X11ConnectionFailed, report.X11.Connection)
	assert.Contains(t, report.X11.ConnectionError, "backend unavailable")
	assert.Equal(t, X11SamplingNotAttempted, report.X11.Sampling)
	assert.Equal(t, PermissionNotApplicable, report.Permission.State)
	assert.False(t, report.Runtime.Present)
	assert.Empty(t, report.Runtime.Error)
}

func TestCollectorReportsX11SamplingFailure(t *testing.T) {
	t.Parallel()

	sampleErr := errors.New("idle time query failed")
	collector := Collector{
		Platform:        "linux",
		SelectedBackend: backend.BackendLinux,
		SessionReporter: func(string) LinuxSessionReport {
			return LinuxSessionReport{
				Applicable: true,
				Type:       "x11",
				Support:    LinuxSessionSupported,
				X11Display: ":0",
			}
		},
		NewBackend: func(string, string) (backend.Backend, error) {
			return &fakeBackend{
				name:      backend.BackendLinux,
				sampleErr: sampleErr,
			}, nil
		},
		PermissionChecker: PermissionCheckerFunc(func(context.Context, string) PermissionReport {
			return PermissionReport{Name: "Accessibility", State: PermissionNotApplicable}
		}),
		LogDirReporter: func(time.Time) LogDirectoryReport {
			return LogDirectoryReport{}
		},
		LaunchAgentChecker: LaunchAgentCheckerFunc(func(context.Context) LaunchAgentReport {
			return LaunchAgentReport{State: LaunchAgentNotApplicable}
		}),
		RuntimeStatusStore: &fakeRuntimeStatusStore{err: status.ErrNotFound},
	}

	report := collector.Collect(context.Background())

	assert.Equal(t, X11ConnectionConnected, report.X11.Connection)
	assert.Equal(t, X11SamplingFailed, report.X11.Sampling)
	assert.Contains(t, report.X11.SamplingError, "idle time query failed")
}

func TestCollectorReportsEmptyX11ActivityAsSamplingFailure(t *testing.T) {
	t.Parallel()

	collector := Collector{
		Platform:        "linux",
		SelectedBackend: backend.BackendLinux,
		SessionReporter: func(string) LinuxSessionReport {
			return LinuxSessionReport{
				Applicable: true,
				Type:       "x11",
				Support:    LinuxSessionSupported,
				X11Display: ":0",
			}
		},
		NewBackend: func(string, string) (backend.Backend, error) {
			return &fakeBackend{
				name: backend.BackendLinux,
				sample: backend.UsageSample{
					IdleSeconds: 1.25,
				},
			}, nil
		},
		PermissionChecker: PermissionCheckerFunc(func(context.Context, string) PermissionReport {
			return PermissionReport{Name: "Accessibility", State: PermissionNotApplicable}
		}),
		LogDirReporter: func(time.Time) LogDirectoryReport {
			return LogDirectoryReport{}
		},
		LaunchAgentChecker: LaunchAgentCheckerFunc(func(context.Context) LaunchAgentReport {
			return LaunchAgentReport{State: LaunchAgentNotApplicable}
		}),
		RuntimeStatusStore: &fakeRuntimeStatusStore{err: status.ErrNotFound},
	}

	report := collector.Collect(context.Background())

	assert.Equal(t, X11ConnectionConnected, report.X11.Connection)
	assert.Equal(t, X11SamplingFailed, report.X11.Sampling)
	assert.Contains(t, report.X11.SamplingError, "neither a program name nor a window title")
	assert.Contains(t, report.Sample.Error, "neither a program name nor a window title")
}

func TestCollectorUsesRunningCollectorSampleForStatus(t *testing.T) {
	t.Parallel()

	startedAt := time.Date(2026, time.August, 4, 11, 0, 0, 0, time.UTC)
	collector := Collector{
		Platform:         "linux",
		SelectedBackend:  backend.BackendLinux,
		UseRuntimeSample: true,
		SessionReporter: func(string) LinuxSessionReport {
			return LinuxSessionReport{
				Applicable: true,
				Type:       "x11",
				Support:    LinuxSessionSupported,
				X11Display: ":0",
			}
		},
		NewBackend: func(string, string) (backend.Backend, error) {
			t.Fatal("status should not open a separate backend")
			return nil, nil
		},
		PermissionChecker: PermissionCheckerFunc(func(context.Context, string) PermissionReport {
			return PermissionReport{Name: "Accessibility", State: PermissionNotApplicable}
		}),
		LogDirReporter: func(time.Time) LogDirectoryReport {
			return LogDirectoryReport{}
		},
		LaunchAgentChecker: LaunchAgentCheckerFunc(func(context.Context) LaunchAgentReport {
			return LaunchAgentReport{State: LaunchAgentNotApplicable}
		}),
		RuntimeStatusStore: &fakeRuntimeStatusStore{
			value: status.RuntimeStatus{
				StartedAt:       &startedAt,
				SelectedBackend: backend.BackendAuto,
				ActiveBackend:   backend.BackendLinux,
				LastSuccessfulSample: &status.ActivitySample{
					ProgramName: "Gnome-terminal",
					WindowTitle: "Terminal",
					IdleSeconds: 3.5,
				},
			},
		},
	}

	report := collector.Collect(context.Background())

	assert.Equal(t, backend.BackendLinux, report.ActiveBackend)
	assert.Equal(t, SampleSourceRuntime, report.Sample.Source)
	assert.Equal(t, "Gnome-terminal", report.Sample.FrontmostApp)
	assert.Equal(t, "Terminal", report.Sample.WindowTitle)
	assert.True(t, report.Sample.WindowTitleAvailable)
	assert.Equal(t, 3.5, report.Sample.IdleSeconds)
	assert.Empty(t, report.Sample.Error)
	assert.Equal(t, X11ConnectionConnected, report.X11.Connection)
	assert.Equal(t, X11SamplingSuccessful, report.X11.Sampling)
}

func TestCollectorRecordsSampleWarnings(t *testing.T) {
	t.Parallel()

	sampleErr := errors.New("window query failed")
	collector := Collector{
		Platform:        "darwin",
		SelectedBackend: backend.BackendMacOSNative,
		NewBackend: func(string, string) (backend.Backend, error) {
			return &fakeBackend{
				name: backend.BackendMacOSNative,
				sample: backend.UsageSample{
					ProgramName: "Terminal",
				},
				sampleErr: sampleErr,
			}, nil
		},
		PermissionChecker: PermissionCheckerFunc(func(context.Context, string) PermissionReport {
			return PermissionReport{Name: "Accessibility", State: PermissionDenied}
		}),
		LogDirReporter: func(time.Time) LogDirectoryReport {
			return LogDirectoryReport{}
		},
		LaunchAgentChecker: LaunchAgentCheckerFunc(func(context.Context) LaunchAgentReport {
			return LaunchAgentReport{}
		}),
		RuntimeStatusStore: &fakeRuntimeStatusStore{err: status.ErrNotFound},
	}

	report := collector.Collect(context.Background())

	assert.Equal(t, "Terminal", report.Sample.FrontmostApp)
	assert.Contains(t, report.Sample.Error, "window query failed")
	assert.False(t, report.Sample.WindowTitleAvailable)
}
