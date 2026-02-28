package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"time"

	"workmuch-go/internal/backend"
	"workmuch-go/internal/logging"
	"workmuch-go/internal/platform"
)

const backendResetInterval = 5 * time.Second

func Run(ctx context.Context, opts Options, logger *log.Logger) error {
	if logger == nil {
		logger = log.New(os.Stderr, "", log.LstdFlags)
	}

	if opts.StartDelay > 0 {
		delay := time.Duration(opts.StartDelay * float64(time.Second))
		if err := sleepWithContext(ctx, delay); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}
		logger.Printf("Delay complete, logging commencing")
	}

	usageBackend, err := backend.NewBackend(runtime.GOOS, opts.Backend)
	if err != nil {
		return err
	}
	logBackendSelection(logger, opts.Backend, usageBackend.Name())
	defer func() {
		if closeErr := usageBackend.Close(); closeErr != nil {
			logger.Printf("backend close failed: %v", closeErr)
		}
	}()

	initialSample, sampleErr := usageBackend.Sample(ctx)
	if sampleErr != nil {
		logger.Printf("initial backend sample warning: %v", sampleErr)
	}
	if initialSample.WindowTitle == "" {
		logger.Printf("warning: system does not currently supply window titles")
	}
	if initialSample.ProgramName == "" {
		logger.Printf("warning: system does not currently supply program names")
	}

	output, closeOutput, err := openCSVOutput(opts)
	if err != nil {
		return err
	}
	defer closeOutput()

	csvWriter := logging.NewCSVWriter(output)
	period := time.Duration(float64(time.Second) / opts.Rate)
	nextWakeAt := time.Now().Add(period)
	resetAt := time.Now().Add(backendResetInterval)

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		now := time.Now()
		if !now.Before(resetAt) {
			if err := usageBackend.Reset(); err != nil {
				logger.Printf("backend reset failed: %v", err)
			}
			resetAt = time.Now().Add(backendResetInterval)
		}

		sample, sampleErr := usageBackend.Sample(ctx)
		if sampleErr != nil {
			logger.Printf("backend sample warning: %v", sampleErr)
		}

		if err := csvWriter.WriteSample(sample, platform.NowUnixSeconds()); err != nil {
			return fmt.Errorf("write csv record: %w", err)
		}
		if err := csvWriter.Flush(); err != nil {
			return fmt.Errorf("flush csv record: %w", err)
		}

		sleepDuration, updatedWakeAt := ComputeNextSleep(time.Now(), nextWakeAt, period)
		nextWakeAt = updatedWakeAt

		if err := sleepWithContext(ctx, sleepDuration); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		}
	}
}

func ComputeNextSleep(now time.Time, wakeAt time.Time, period time.Duration) (time.Duration, time.Time) {
	if period <= 0 {
		return 0, wakeAt
	}

	for wakeAt.Before(now) {
		wakeAt = wakeAt.Add(period)
	}

	return wakeAt.Sub(now), wakeAt.Add(period)
}

func logBackendSelection(logger *log.Logger, requested string, active string) {
	logger.Printf("backend requested=%s active=%s", requested, active)
}

func openCSVOutput(opts Options) (io.Writer, func(), error) {
	if opts.QAConsole {
		return os.Stdout, func() {}, nil
	}

	homeDir, err := platform.UserHomeDir()
	if err != nil {
		return nil, nil, fmt.Errorf("resolve home directory: %w", err)
	}
	logDir, err := platform.EnsureLogDir(homeDir)
	if err != nil {
		return nil, nil, fmt.Errorf("ensure log directory: %w", err)
	}

	workLogPath := platform.WorkLogPath(time.Now(), logDir)
	workLogFile, err := os.OpenFile(workLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, fmt.Errorf("open work log %s: %w", workLogPath, err)
	}

	closeFn := func() {
		_ = workLogFile.Close()
	}
	return workLogFile, closeFn, nil
}

func sleepWithContext(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}

	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
