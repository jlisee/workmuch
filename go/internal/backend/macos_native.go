package backend

import (
	"context"
	"errors"
	"fmt"
)

type macOSNativeAPI interface {
	IsAccessibilityTrusted() bool
	GetFrontmostWindowInfo() (pid int, appName string, windowTitle string, err error)
	GetFrontmostApplication() (pid int, appName string, err error)
	GetFocusedWindowTitle(pid int) (string, error)
	GetIdleSeconds() (float64, error)
	Close() error
}

type MacOSNativeBackend struct {
	api       macOSNativeAPI
	axTrusted bool
}

func newMacOSNativeBackend(api macOSNativeAPI) *MacOSNativeBackend {
	backend := &MacOSNativeBackend{api: api}
	if api != nil {
		backend.axTrusted = api.IsAccessibilityTrusted()
	}
	return backend
}

func (b *MacOSNativeBackend) Name() string {
	return BackendMacOSNative
}

func (b *MacOSNativeBackend) Sample(_ context.Context) (UsageSample, error) {
	sample := UsageSample{}
	var errs []error

	if b.api == nil {
		errs = append(errs, fmt.Errorf("%w: %s api unavailable", ErrNotImplemented, BackendMacOSNative))
		sample, err := completeSample(sample)
		if err != nil {
			errs = append(errs, err)
		}
		return sample, errors.Join(errs...)
	}

	pid, appName, windowTitle, err := b.api.GetFrontmostWindowInfo()
	if err != nil {
		errs = append(errs, fmt.Errorf("frontmost window query failed: %w", err))
	}
	sample.ProgramName = appName
	sample.WindowTitle = windowTitle

	if pid <= 0 {
		pid, appName, err = b.api.GetFrontmostApplication()
		if err != nil {
			errs = append(errs, fmt.Errorf("frontmost application query failed: %w", err))
		} else {
			sample.ProgramName = appName
		}
	}

	if !b.axTrusted {
		b.axTrusted = b.api.IsAccessibilityTrusted()
	}

	if b.axTrusted && sample.WindowTitle == "" && pid > 0 {
		title, titleErr := b.api.GetFocusedWindowTitle(pid)
		if titleErr == nil {
			sample.WindowTitle = title
		}
	}

	idleSeconds, err := b.api.GetIdleSeconds()
	if err != nil {
		errs = append(errs, fmt.Errorf("idle time query failed: %w", err))
	} else if idleSeconds > 0 {
		sample.IdleSeconds = idleSeconds
	}

	sample, err = completeSample(sample)
	if err != nil {
		errs = append(errs, err)
	}

	return sample, errors.Join(errs...)
}

func (b *MacOSNativeBackend) Reset() error {
	return nil
}

func (b *MacOSNativeBackend) Close() error {
	if b.api == nil {
		return nil
	}
	return b.api.Close()
}
