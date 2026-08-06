package tray

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"workmuch-go/internal/app"
	"workmuch-go/internal/doctor"
)

func showStatusScreen(opts app.Options) error {
	report := collectStatusReport(opts)
	target, err := ensureStatusHTML(report)
	if err != nil {
		return err
	}
	cmd, err := openFileCommand(runtime.GOOS, target)
	if err != nil {
		return err
	}
	return cmd.Start()
}

func collectStatusReport(opts app.Options) doctor.DoctorReport {
	return doctor.NewStatusCollector(opts.Backend).Collect(context.Background())
}

func ensureStatusHTML(report doctor.DoctorReport) (string, error) {
	file, err := os.CreateTemp("", "workmuch-status-*.html")
	if err != nil {
		return "", fmt.Errorf("create private status page: %w", err)
	}
	path := file.Name()
	keepFile := false
	defer func() {
		if !keepFile {
			_ = os.Remove(path)
		}
	}()

	if _, err := file.WriteString(doctor.RenderHTML(report)); err != nil {
		_ = file.Close()
		return "", fmt.Errorf("write status page %s: %w", path, err)
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("close status page %s: %w", path, err)
	}
	keepFile = true
	return path, nil
}

func openFileCommand(goos string, target string) (*exec.Cmd, error) {
	switch goos {
	case "darwin":
		return exec.Command("open", target), nil
	case "linux":
		page, err := os.ReadFile(target)
		if err != nil {
			return nil, fmt.Errorf("read browser page %s: %w", target, err)
		}
		pageURL, err := serveBrowserPage(page)
		if err != nil {
			return nil, err
		}
		return exec.Command("xdg-open", pageURL), nil
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", target), nil
	default:
		return nil, fmt.Errorf("open file unsupported on %s", goos)
	}
}
