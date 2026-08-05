package backend

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	appNameCommand = []string{
		"osascript",
		"-e",
		"tell application \"System Events\" to get name of first application process whose frontmost is true",
	}
	windowTitleCommand = []string{
		"osascript",
		"-e",
		"tell application \"System Events\" to tell (first application process whose frontmost is true) to get value of attribute \"AXTitle\" of front window",
	}
	idleTimeCommand = []string{"ioreg", "-c", "IOHIDSystem"}
)

var hidIdlePattern = regexp.MustCompile(`"HIDIdleTime"\s*=\s*([0-9]+)`) // nanoseconds

type commandExecutor interface {
	Run(ctx context.Context, timeout time.Duration, name string, args ...string) (string, error)
}

type osCommandExecutor struct{}

func (osCommandExecutor) Run(ctx context.Context, timeout time.Duration, name string, args ...string) (string, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, name, args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

type MacOSSubprocessBackend struct {
	exec commandExecutor
}

func NewMacOSSubprocessBackend() *MacOSSubprocessBackend {
	return newMacOSSubprocessBackend(osCommandExecutor{})
}

func newMacOSSubprocessBackend(exec commandExecutor) *MacOSSubprocessBackend {
	if exec == nil {
		exec = osCommandExecutor{}
	}
	return &MacOSSubprocessBackend{exec: exec}
}

func (b *MacOSSubprocessBackend) Name() string {
	return BackendMacOSSubprocess
}

func (b *MacOSSubprocessBackend) Sample(ctx context.Context) (UsageSample, error) {
	sample := UsageSample{}
	var errs []error

	progName, err := b.runAndTrim(ctx, appNameCommand)
	if err != nil {
		errs = append(errs, fmt.Errorf("program name query failed: %w", err))
	}
	sample.ProgramName = progName

	windowTitle, err := b.runAndTrim(ctx, windowTitleCommand)
	if err != nil {
		errs = append(errs, fmt.Errorf("window title query failed: %w", err))
	}
	sample.WindowTitle = windowTitle

	idleOutput, err := b.runAndTrim(ctx, idleTimeCommand)
	if err != nil {
		errs = append(errs, fmt.Errorf("idle time query failed: %w", err))
	}

	idleSeconds, parseErr := ParseIdleSeconds(idleOutput)
	if parseErr != nil {
		errs = append(errs, fmt.Errorf("idle time parse failed: %w", parseErr))
	} else {
		sample.IdleSeconds = idleSeconds
	}

	sample, err = completeSample(sample)
	if err != nil {
		errs = append(errs, err)
	}

	return sample, errors.Join(errs...)
}

func (b *MacOSSubprocessBackend) Reset() error {
	return nil
}

func (b *MacOSSubprocessBackend) Close() error {
	return nil
}

func (b *MacOSSubprocessBackend) runAndTrim(ctx context.Context, command []string) (string, error) {
	if len(command) == 0 {
		return "", errors.New("empty command")
	}

	output, err := b.exec.Run(ctx, time.Second, command[0], command[1:]...)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

func ParseIdleSeconds(output string) (float64, error) {
	match := hidIdlePattern.FindStringSubmatch(output)
	if len(match) != 2 {
		return 0.0, errors.New("HIDIdleTime field not found")
	}

	nanoseconds, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return 0.0, err
	}
	return nanoseconds / 1e9, nil
}
