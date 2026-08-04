package doctor

import (
	"context"
	"time"

	"workmuch-go/internal/backend"
	"workmuch-go/internal/status"
)

type PermissionState string

const (
	PermissionGranted       PermissionState = "granted"
	PermissionDenied        PermissionState = "denied"
	PermissionNotApplicable PermissionState = "not_applicable"
	PermissionUnknown       PermissionState = "unknown"
)

type LaunchAgentState string

const (
	LaunchAgentRunning       LaunchAgentState = "running"
	LaunchAgentLoaded        LaunchAgentState = "loaded"
	LaunchAgentNotLoaded     LaunchAgentState = "not_loaded"
	LaunchAgentMissing       LaunchAgentState = "missing"
	LaunchAgentError         LaunchAgentState = "error"
	LaunchAgentNotApplicable LaunchAgentState = "not_applicable"
)

type PermissionReport struct {
	Name   string
	State  PermissionState
	Detail string
	Error  string
}

type SampleReport struct {
	FrontmostApp         string
	WindowTitle          string
	WindowTitleAvailable bool
	IdleSeconds          float64
	Error                string
}

type LogDirectoryReport struct {
	Directory      string
	Writable       bool
	CurrentWorkLog string
	Error          string
}

type LaunchAgentReport struct {
	State  LaunchAgentState
	Detail string
	Error  string
}

type RuntimeStatusReport struct {
	Path    string
	Present bool
	Status  status.RuntimeStatus
	Error   string
}

type LinuxSessionSupport string

const (
	LinuxSessionSupported   LinuxSessionSupport = "supported"
	LinuxSessionUnsupported LinuxSessionSupport = "unsupported"
	LinuxSessionUnknown     LinuxSessionSupport = "unknown"
)

type LinuxSessionReport struct {
	Applicable     bool
	Type           string
	Support        LinuxSessionSupport
	X11Display     string
	WaylandDisplay string
	Detail         string
}

type X11ConnectionState string

const (
	X11ConnectionNotAttempted X11ConnectionState = "not attempted"
	X11ConnectionConnected    X11ConnectionState = "connected"
	X11ConnectionFailed       X11ConnectionState = "failed"
)

type X11SamplingState string

const (
	X11SamplingNotAttempted X11SamplingState = "not attempted"
	X11SamplingSuccessful   X11SamplingState = "successful"
	X11SamplingFailed       X11SamplingState = "failed"
)

type X11Report struct {
	Applicable      bool
	Display         string
	Connection      X11ConnectionState
	ConnectionError string
	Sampling        X11SamplingState
	SamplingError   string
}

type DoctorReport struct {
	SelectedBackend string
	ActiveBackend   string
	BackendError    string
	LinuxSession    LinuxSessionReport
	X11             X11Report
	Permission      PermissionReport
	Sample          SampleReport
	Logs            LogDirectoryReport
	LaunchAgent     LaunchAgentReport
	Runtime         RuntimeStatusReport
}

type BackendFactory func(platform string, backendName string) (backend.Backend, error)

type PermissionChecker interface {
	Check(ctx context.Context, platform string) PermissionReport
}

type PermissionCheckerFunc func(ctx context.Context, platform string) PermissionReport

func (f PermissionCheckerFunc) Check(ctx context.Context, platform string) PermissionReport {
	return f(ctx, platform)
}

type LaunchAgentChecker interface {
	Check(ctx context.Context) LaunchAgentReport
}

type LaunchAgentCheckerFunc func(ctx context.Context) LaunchAgentReport

func (f LaunchAgentCheckerFunc) Check(ctx context.Context) LaunchAgentReport {
	return f(ctx)
}

type RuntimeStatusStore interface {
	Read() (status.RuntimeStatus, error)
	Path() string
}

type LogDirReporter func(now time.Time) LogDirectoryReport

type SessionReporter func(platform string) LinuxSessionReport

type Collector struct {
	Platform          string
	SelectedBackend   string
	SessionReporter   SessionReporter
	NewBackend        BackendFactory
	PermissionChecker PermissionChecker
	LogDirReporter    LogDirReporter
	LaunchAgentChecker
	RuntimeStatusStore RuntimeStatusStore
	Now                func() time.Time
}
