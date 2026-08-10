package doctor

import (
	"context"
	"time"

	"workmuch-go/internal/backend"
	"workmuch-go/internal/macosapp"
	"workmuch-go/internal/status"
)

type PermissionState string

const (
	PermissionGranted       PermissionState = "granted"
	PermissionDenied        PermissionState = "denied"
	PermissionNotApplicable PermissionState = "not_applicable"
	PermissionUnknown       PermissionState = "unknown"
)

type LoginItemState = macosapp.LoginItemState

const (
	LoginItemNotRegistered    = macosapp.LoginItemNotRegistered
	LoginItemEnabled          = macosapp.LoginItemEnabled
	LoginItemRequiresApproval = macosapp.LoginItemRequiresApproval
	LoginItemNotFound         = macosapp.LoginItemNotFound
	LoginItemUnsupported      = macosapp.LoginItemUnsupported
	LoginItemNotApplicable    = macosapp.LoginItemNotApplicable
)

type PermissionReport struct {
	Name   string
	State  PermissionState
	Detail string
	Error  string
}

type SampleSource string

const (
	SampleSourceDiagnostic SampleSource = "live diagnostic probe"
	SampleSourceRuntime    SampleSource = "collector last successful sample"
)

type SampleReport struct {
	Source               SampleSource
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

type LoginItemReport struct {
	State  LoginItemState
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
	LoginItem       LoginItemReport
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

type LoginItemChecker interface {
	Check(ctx context.Context) LoginItemReport
}

type LoginItemCheckerFunc func(ctx context.Context) LoginItemReport

func (f LoginItemCheckerFunc) Check(ctx context.Context) LoginItemReport {
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
	UseRuntimeSample  bool
	SessionReporter   SessionReporter
	NewBackend        BackendFactory
	PermissionChecker PermissionChecker
	LogDirReporter    LogDirReporter
	LoginItemChecker
	RuntimeStatusStore RuntimeStatusStore
	Now                func() time.Time
}
