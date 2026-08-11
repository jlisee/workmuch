package doctor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"workmuch-go/internal/backend"
	"workmuch-go/internal/platform"
	"workmuch-go/internal/status"
)

func NewCollector(selectedBackend string) Collector {
	selectedBackend = strings.TrimSpace(selectedBackend)
	if selectedBackend == "" {
		selectedBackend = backend.BackendAuto
	}

	store := defaultRuntimeStatusStore()
	return Collector{
		Platform:           runtime.GOOS,
		SelectedBackend:    selectedBackend,
		SessionReporter:    func(platformName string) LinuxSessionReport { return DetectLinuxSession(platformName, os.Getenv) },
		NewBackend:         backend.NewBackend,
		PermissionChecker:  NativePermissionChecker{},
		LogDirReporter:     DefaultLogDirReport,
		LoginItemChecker:   NativeLoginItemChecker{Platform: runtime.GOOS},
		RuntimeStatusStore: store,
		Now:                time.Now,
	}
}

func NewStatusCollector(selectedBackend string) Collector {
	collector := NewCollector(selectedBackend)
	collector.UseRuntimeSample = true
	return collector
}

func (c Collector) Collect(ctx context.Context) DoctorReport {
	now := c.now()
	platformName := c.platform()
	selectedBackend := strings.TrimSpace(c.SelectedBackend)
	if selectedBackend == "" {
		selectedBackend = backend.BackendAuto
	}

	report := DoctorReport{
		GeneratedAt:     now,
		SelectedBackend: selectedBackend,
		Sample: SampleReport{
			Source: SampleSourceDiagnostic,
		},
	}
	if c.SessionReporter != nil {
		report.LinuxSession = c.SessionReporter(platformName)
	}
	if platformName == "linux" && (selectedBackend == backend.BackendAuto || selectedBackend == backend.BackendLinux) {
		report.X11 = X11Report{
			Applicable: true,
			Display:    report.LinuxSession.X11Display,
			Connection: X11ConnectionNotAttempted,
			Sampling:   X11SamplingNotAttempted,
		}
	}

	if c.PermissionChecker != nil {
		report.Permission = c.PermissionChecker.Check(ctx, platformName)
	}
	if c.LogDirReporter != nil {
		report.Logs = c.LogDirReporter(now)
	}
	if c.LoginItemChecker != nil {
		report.LoginItem = c.LoginItemChecker.Check(ctx)
	}
	if c.RuntimeStatusStore != nil {
		report.Runtime.Path = c.RuntimeStatusStore.Path()
		runtimeStatus, err := c.RuntimeStatusStore.Read()
		switch {
		case err == nil:
			report.Runtime.Present = true
			report.Runtime.Status = runtimeStatus
		case errors.Is(err, status.ErrNotFound):
			report.Runtime.Present = false
		default:
			report.Runtime.Error = err.Error()
		}
	}
	if c.UseRuntimeSample {
		report.Sample.Source = SampleSourceRuntime
		populateRuntimeSample(&report)
		return report
	}

	newBackend := c.NewBackend
	if newBackend == nil {
		newBackend = backend.NewBackend
	}
	usageBackend, err := newBackend(platformName, selectedBackend)
	if err != nil {
		report.BackendError = err.Error()
		if report.X11.Applicable {
			report.X11.Connection = X11ConnectionFailed
			report.X11.ConnectionError = err.Error()
		}
		return report
	}
	defer func() {
		if closeErr := usageBackend.Close(); closeErr != nil {
			if report.BackendError == "" {
				report.BackendError = fmt.Sprintf("close backend %s: %v", usageBackend.Name(), closeErr)
			} else {
				report.BackendError += "; " + fmt.Sprintf("close backend %s: %v", usageBackend.Name(), closeErr)
			}
		}
	}()

	report.ActiveBackend = usageBackend.Name()
	if report.X11.Applicable && report.ActiveBackend == backend.BackendLinux {
		report.X11.Connection = X11ConnectionConnected
	}
	sample, sampleErr := usageBackend.Sample(ctx)
	if sampleErr == nil && report.X11.Applicable && !sample.HasActivity() {
		sampleErr = errors.New("X11 sample returned neither a program name nor a window title")
	}
	if sampleErr != nil {
		report.Sample.Error = sampleErr.Error()
		if report.X11.Applicable {
			report.X11.Sampling = X11SamplingFailed
			report.X11.SamplingError = sampleErr.Error()
		}
	} else if report.X11.Applicable {
		report.X11.Sampling = X11SamplingSuccessful
	}
	report.Sample.FrontmostApp = sample.ProgramName
	report.Sample.WindowTitle = sample.WindowTitle
	report.Sample.WindowTitleAvailable = sample.WindowTitle != ""
	report.Sample.IdleSeconds = sample.IdleSeconds

	return report
}

func populateRuntimeSample(report *DoctorReport) {
	if report == nil || !report.Runtime.Present {
		return
	}

	runtimeStatus := report.Runtime.Status
	report.ActiveBackend = runtimeStatus.ActiveBackend
	if report.X11.Applicable && runtimeStatus.ActiveBackend == backend.BackendLinux &&
		runtimeStatus.StartedAt != nil && runtimeStatus.StoppedAt == nil {
		report.X11.Connection = X11ConnectionConnected
	}
	if runtimeStatus.LastSuccessfulSample == nil {
		return
	}

	sample := runtimeStatus.LastSuccessfulSample
	report.Sample.FrontmostApp = sample.ProgramName
	report.Sample.WindowTitle = sample.WindowTitle
	report.Sample.WindowTitleAvailable = sample.WindowTitle != ""
	report.Sample.IdleSeconds = sample.IdleSeconds
	if report.X11.Applicable && runtimeStatus.ActiveBackend == backend.BackendLinux {
		report.X11.Sampling = X11SamplingSuccessful
	}
}

func (c Collector) platform() string {
	if strings.TrimSpace(c.Platform) == "" {
		return runtime.GOOS
	}
	return c.Platform
}

func (c Collector) now() time.Time {
	if c.Now == nil {
		return time.Now()
	}
	return c.Now()
}

func DefaultLogDirReport(now time.Time) LogDirectoryReport {
	homeDir, err := platform.UserHomeDir()
	if err != nil {
		return LogDirectoryReport{Error: fmt.Sprintf("resolve home directory: %v", err)}
	}

	logDir := platform.LogDir(homeDir)
	report := LogDirectoryReport{
		Directory:      logDir,
		CurrentWorkLog: platform.WorkLogPath(now, logDir),
	}
	if _, err := platform.EnsureLogDir(homeDir); err != nil {
		report.Error = fmt.Sprintf("ensure log directory %s: %v", logDir, err)
		return report
	}

	tempFile, err := os.CreateTemp(logDir, ".doctor-write-*.tmp")
	if err != nil {
		report.Error = fmt.Sprintf("check log directory writability %s: %v", logDir, err)
		return report
	}
	tempPath := tempFile.Name()
	closeErr := tempFile.Close()
	removeErr := os.Remove(tempPath)
	if closeErr != nil {
		report.Error = fmt.Sprintf("close log directory writability probe %s: %v", tempPath, closeErr)
		return report
	}
	if removeErr != nil {
		report.Error = fmt.Sprintf("remove log directory writability probe %s: %v", tempPath, removeErr)
		return report
	}
	report.Writable = true
	return report
}

func defaultRuntimeStatusStore() RuntimeStatusStore {
	homeDir, err := platform.UserHomeDir()
	if err != nil {
		return errorRuntimeStatusStore{err: fmt.Errorf("resolve home directory: %w", err)}
	}
	logDir := platform.LogDir(homeDir)
	return status.NewStore(platform.StatusPath(logDir))
}

type errorRuntimeStatusStore struct {
	err error
}

func (s errorRuntimeStatusStore) Read() (status.RuntimeStatus, error) {
	return status.RuntimeStatus{}, s.err
}

func (s errorRuntimeStatusStore) Path() string {
	return ""
}

func permissionPlatform(platformName string) string {
	platformName = strings.TrimSpace(strings.ToLower(platformName))
	if platformName == "" {
		return runtime.GOOS
	}
	return platformName
}
