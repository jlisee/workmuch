package doctor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
		NewBackend:         backend.NewBackend,
		PermissionChecker:  NativePermissionChecker{},
		LogDirReporter:     DefaultLogDirReport,
		LaunchAgentChecker: NativeLaunchAgentChecker{Platform: runtime.GOOS},
		RuntimeStatusStore: store,
		Now:                time.Now,
	}
}

func (c Collector) Collect(ctx context.Context) DoctorReport {
	now := c.now()
	selectedBackend := strings.TrimSpace(c.SelectedBackend)
	if selectedBackend == "" {
		selectedBackend = backend.BackendAuto
	}

	report := DoctorReport{
		SelectedBackend: selectedBackend,
	}

	if c.PermissionChecker != nil {
		report.Permission = c.PermissionChecker.Check(ctx, c.platform())
	}
	if c.LogDirReporter != nil {
		report.Logs = c.LogDirReporter(now)
	}
	if c.LaunchAgentChecker != nil {
		report.LaunchAgent = c.LaunchAgentChecker.Check(ctx)
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

	newBackend := c.NewBackend
	if newBackend == nil {
		newBackend = backend.NewBackend
	}
	usageBackend, err := newBackend(c.platform(), selectedBackend)
	if err != nil {
		report.BackendError = err.Error()
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
	sample, sampleErr := usageBackend.Sample(ctx)
	if sampleErr != nil {
		report.Sample.Error = sampleErr.Error()
	}
	report.Sample.FrontmostApp = sample.ProgramName
	report.Sample.WindowTitle = sample.WindowTitle
	report.Sample.WindowTitleAvailable = sample.WindowTitle != ""
	report.Sample.IdleSeconds = sample.IdleSeconds

	return report
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

func launchAgentPlistPath(homeDir string) string {
	return filepath.Join(homeDir, "Library", "LaunchAgents", "com.jlisee.workmuch.plist")
}
