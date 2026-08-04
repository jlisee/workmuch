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

type DoctorReport struct {
	SelectedBackend string
	ActiveBackend   string
	BackendError    string
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

type Collector struct {
	Platform          string
	SelectedBackend   string
	NewBackend        BackendFactory
	PermissionChecker PermissionChecker
	LogDirReporter    LogDirReporter
	LaunchAgentChecker
	RuntimeStatusStore RuntimeStatusStore
	Now                func() time.Time
}
