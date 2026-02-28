package backend

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	system "workmuch-go/internal"
)

var ErrNotImplemented = errors.New("backend not implemented")

var (
	lookupHostname = os.Hostname
	lookupUsername = system.Username
)

type UsageSample struct {
	Host        string
	User        string
	WindowTitle string
	ProgramName string
	IdleSeconds float64
}

type Backend interface {
	Name() string
	Sample(ctx context.Context) (UsageSample, error)
	Reset() error
	Close() error
}

const (
	BackendAuto            = "auto"
	BackendMacOSSubprocess = "macos-subprocess"
	BackendMacOSNative     = "macos-native"
	BackendLinux           = "linux"
)

func NewBackend(platform string, backendName string) (Backend, error) {
	normalizedPlatform := normalizePlatform(platform)
	normalizedBackend := normalizeBackendName(backendName)

	if normalizedBackend == BackendAuto {
		switch normalizedPlatform {
		case "darwin":
			normalizedBackend = BackendMacOSSubprocess
		case "linux":
			normalizedBackend = BackendLinux
		default:
			return nil, fmt.Errorf("unsupported platform for auto backend: %q", normalizedPlatform)
		}
	}

	switch normalizedBackend {
	case BackendMacOSSubprocess:
		if normalizedPlatform != "darwin" {
			return nil, fmt.Errorf("%s backend requires darwin platform", BackendMacOSSubprocess)
		}
		return NewMacOSSubprocessBackend(), nil
	case BackendMacOSNative:
		if normalizedPlatform != "darwin" {
			return nil, fmt.Errorf("%s backend requires darwin platform", BackendMacOSNative)
		}
		backend, err := NewMacOSNativeBackend()
		if err != nil {
			return nil, err
		}
		return backend, nil
	case BackendLinux:
		if normalizedPlatform != "linux" {
			return nil, fmt.Errorf("%s backend requires linux platform", BackendLinux)
		}
		backend, err := NewLinuxBackend()
		if err != nil {
			return nil, err
		}
		return backend, nil
	default:
		return nil, fmt.Errorf("unknown backend %q", normalizedBackend)
	}
}

func normalizePlatform(platform string) string {
	platform = strings.TrimSpace(strings.ToLower(platform))
	if platform == "" {
		return runtime.GOOS
	}
	if strings.HasPrefix(platform, "linux") {
		return "linux"
	}
	return platform
}

func normalizeBackendName(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	if name == "" {
		return BackendAuto
	}
	return name
}

func completeSample(sample UsageSample) (UsageSample, error) {
	var errs []error

	host, err := lookupHostname()
	if err != nil {
		errs = append(errs, fmt.Errorf("hostname lookup failed: %w", err))
	} else {
		sample.Host = host
	}

	username, err := lookupUsername()
	if err != nil {
		errs = append(errs, fmt.Errorf("username lookup failed: %w", err))
	} else {
		sample.User = username
	}

	return sample, errors.Join(errs...)
}
