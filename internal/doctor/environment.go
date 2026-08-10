package doctor

import (
	"context"
	"os"
	"runtime"
	"strings"

	"workmuch-go/internal/backend"
	"workmuch-go/internal/macosapp"
)

type NativePermissionChecker struct{}

func DetectLinuxSession(platformName string, getenv func(string) string) LinuxSessionReport {
	if permissionPlatform(platformName) != "linux" {
		return LinuxSessionReport{}
	}
	if getenv == nil {
		getenv = os.Getenv
	}

	report := LinuxSessionReport{
		Applicable:     true,
		Type:           strings.ToLower(strings.TrimSpace(getenv("XDG_SESSION_TYPE"))),
		Support:        LinuxSessionUnknown,
		X11Display:     strings.TrimSpace(getenv("DISPLAY")),
		WaylandDisplay: strings.TrimSpace(getenv("WAYLAND_DISPLAY")),
	}
	if report.Type == "" {
		switch {
		case report.WaylandDisplay != "":
			report.Type = "wayland"
		case report.X11Display != "":
			report.Type = "x11"
		}
	}

	switch report.Type {
	case "x11", "xorg":
		report.Support = LinuxSessionSupported
	case "wayland":
		report.Support = LinuxSessionUnsupported
		report.Detail = "Wayland detected; WorkMuch currently supports only X11/Xorg sessions."
	default:
		report.Detail = "WorkMuch currently supports only X11/Xorg sessions."
	}
	return report
}

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

type NativeLoginItemChecker struct {
	Platform string
	Service  macosapp.LoginItem
}

func (c NativeLoginItemChecker) Check(_ context.Context) LoginItemReport {
	platformName := strings.TrimSpace(c.Platform)
	if platformName == "" {
		platformName = runtime.GOOS
	}
	if platformName != "darwin" {
		return LoginItemReport{State: LoginItemNotApplicable}
	}

	service := c.Service
	if service == nil {
		service = macosapp.NewMainAppService()
	}
	state, err := service.Status()
	if err != nil {
		return LoginItemReport{State: LoginItemUnsupported, Error: err.Error()}
	}
	return LoginItemReport{State: state}
}
