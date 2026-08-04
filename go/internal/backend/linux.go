package backend

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
)

type linuxX11Client interface {
	ActiveWindow() (uint32, error)
	WindowTitle(window uint32) (string, error)
	ProgramName(window uint32) (string, error)
	IdleMilliseconds() (uint32, error)
	Close() error
}

type linuxX11Connector func(display string) (linuxX11Client, error)

type LinuxBackend struct {
	mu      sync.Mutex
	display string
	connect linuxX11Connector
	client  linuxX11Client
}

func NewLinuxBackend() (Backend, error) {
	return newLinuxBackend(os.Getenv("DISPLAY"), connectLinuxX11)
}

func newLinuxBackend(display string, connect linuxX11Connector) (*LinuxBackend, error) {
	if strings.TrimSpace(display) == "" {
		return nil, errors.New("DISPLAY is not set")
	}
	if connect == nil {
		return nil, errors.New("X11 connector is unavailable")
	}

	client, err := connect(display)
	if err != nil {
		return nil, fmt.Errorf("connect to X11 display %q: %w", display, err)
	}

	return &LinuxBackend{
		display: display,
		connect: connect,
		client:  client,
	}, nil
}

func (b *LinuxBackend) Name() string {
	return BackendLinux
}

func (b *LinuxBackend) Sample(_ context.Context) (UsageSample, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	sample := UsageSample{}
	var errs []error

	if b.client == nil {
		errs = append(errs, errors.New("X11 client is not connected"))
	} else {
		window, err := b.client.ActiveWindow()
		if err != nil {
			errs = append(errs, fmt.Errorf("active window query failed: %w", err))
		} else if window == 0 {
			errs = append(errs, errors.New("active window query returned no window"))
		} else {
			title, titleErr := b.client.WindowTitle(window)
			if titleErr != nil {
				errs = append(errs, fmt.Errorf("window title query failed: %w", titleErr))
			} else {
				sample.WindowTitle = title
			}

			program, programErr := b.client.ProgramName(window)
			if programErr != nil {
				errs = append(errs, fmt.Errorf("program name query failed: %w", programErr))
			} else {
				sample.ProgramName = program
			}
		}

		idleMillis, idleErr := b.client.IdleMilliseconds()
		if idleErr != nil {
			errs = append(errs, fmt.Errorf("idle time query failed: %w", idleErr))
		} else {
			sample.IdleSeconds = float64(idleMillis) / 1000.0
		}
	}

	completed, err := completeSample(sample)
	if err != nil {
		errs = append(errs, err)
	}

	return completed, errors.Join(errs...)
}

func (b *LinuxBackend) Reset() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	var errs []error
	if b.client != nil {
		if err := b.client.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close X11 connection: %w", err))
		}
		b.client = nil
	}

	client, err := b.connect(b.display)
	if err != nil {
		errs = append(errs, fmt.Errorf("reconnect to X11 display %q: %w", b.display, err))
	} else {
		b.client = client
	}

	return errors.Join(errs...)
}

func (b *LinuxBackend) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.client == nil {
		return nil
	}

	client := b.client
	b.client = nil
	return client.Close()
}

func parseWMClass(value []byte) string {
	parts := strings.Split(strings.TrimRight(string(value), "\x00"), "\x00")
	if len(parts) > 1 && parts[1] != "" {
		return parts[1]
	}
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func findTopLevelWindow(
	focus uint32,
	root uint32,
	parentOf func(window uint32) (uint32, error),
	titleOf func(window uint32) (string, error),
) (uint32, error) {
	window := focus
	for {
		parent, err := parentOf(window)
		if err != nil {
			return 0, err
		}
		if parent == 0 || parent == root {
			return window, nil
		}
		if parent == window {
			return 0, fmt.Errorf("window %#x is its own parent", window)
		}

		window = parent
		title, err := titleOf(window)
		if err != nil {
			return 0, err
		}
		if title != "" {
			return window, nil
		}
	}
}
