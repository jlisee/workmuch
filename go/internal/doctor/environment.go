package doctor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"workmuch-go/internal/backend"
	"workmuch-go/internal/platform"
)

const launchAgentLabel = "com.jlisee.workmuch"

type NativePermissionChecker struct{}

func (NativePermissionChecker) Check(_ context.Context, platformName string) PermissionReport {
	report := PermissionReport{Name: "Accessibility"}
	if permissionPlatform(platformName) != "darwin" {
		report.State = PermissionNotApplicable
		return report
	}

	trusted, err := backend.IsMacOSAccessibilityTrusted()
	if err != nil {
		report.State = PermissionUnknown
		report.Error = err.Error()
		return report
	}
	if trusted {
		report.State = PermissionGranted
		return report
	}
	report.State = PermissionDenied
	report.Detail = "Grant Accessibility permission to the WorkMuch executable in System Settings."
	return report
}

type NativeLaunchAgentChecker struct {
	Platform     string
	HomeDir      func() (string, error)
	RunLaunchctl func(ctx context.Context, target string) ([]byte, error)
}

func (c NativeLaunchAgentChecker) Check(ctx context.Context) LaunchAgentReport {
	platformName := strings.TrimSpace(c.Platform)
	if platformName == "" {
		platformName = runtime.GOOS
	}
	if platformName != "darwin" {
		return LaunchAgentReport{State: LaunchAgentNotApplicable}
	}

	homeDirLookup := c.HomeDir
	if homeDirLookup == nil {
		homeDirLookup = platform.UserHomeDir
	}
	homeDir, err := homeDirLookup()
	if err != nil {
		return LaunchAgentReport{State: LaunchAgentError, Error: fmt.Sprintf("resolve home directory: %v", err)}
	}
	plistPath := launchAgentPlistPath(homeDir)
	if _, err := os.Stat(plistPath); err != nil {
		if os.IsNotExist(err) {
			return LaunchAgentReport{State: LaunchAgentMissing, Detail: plistPath}
		}
		return LaunchAgentReport{State: LaunchAgentError, Detail: plistPath, Error: fmt.Sprintf("stat launch agent plist: %v", err)}
	}

	commandCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	target := fmt.Sprintf("gui/%s/%s", strconv.Itoa(os.Getuid()), launchAgentLabel)
	runLaunchctl := c.RunLaunchctl
	if runLaunchctl == nil {
		runLaunchctl = func(ctx context.Context, target string) ([]byte, error) {
			return exec.CommandContext(ctx, "launchctl", "print", target).CombinedOutput()
		}
	}
	output, err := runLaunchctl(commandCtx, target)
	if err != nil {
		report := LaunchAgentReport{Detail: strings.TrimSpace(string(output)), Error: err.Error()}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			report.State = LaunchAgentNotLoaded
			return report
		}
		report.State = LaunchAgentError
		return report
	}

	text := string(output)
	if strings.Contains(text, "state = running") || strings.Contains(text, "pid = ") {
		return LaunchAgentReport{State: LaunchAgentRunning}
	}
	return LaunchAgentReport{State: LaunchAgentLoaded}
}
