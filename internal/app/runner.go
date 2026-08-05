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
	"workmuch-go/internal/status"
)

const backendResetInterval = 5 * time.Second

func Run(ctx context.Context, opts Options, logger *log.Logger) error {
	if errors.Is(ctx.Err(), context.Canceled) {
		return nil
	}

	controller := NewController(opts, logger, RunCollector)
	controller.Start(ctx)
	return controller.Wait()
}

func runCollector(ctx context.Context, opts Options, logger *log.Logger) error {
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
			logger.Print(formatLogMessage("warning", "backend close", closeErr, logField{key: "backend", value: usageBackend.Name()}))
		}
	}()

	initialSample, sampleErr := usageBackend.Sample(ctx)
	initialWarnings := []string{}
	if sampleErr != nil {
		message := formatLogMessage("warning", "initial backend sample", sampleErr, logField{key: "backend", value: usageBackend.Name()})
		initialWarnings = append(initialWarnings, message)
		logger.Print(message)
	}
	if initialSample.WindowTitle == "" {
		message := formatLogMessage("warning", "system does not currently supply window titles", nil, logField{key: "backend", value: usageBackend.Name()})
		initialWarnings = append(initialWarnings, message)
		logger.Print(message)
	}
	if initialSample.ProgramName == "" {
		message := formatLogMessage("warning", "system does not currently supply program names", nil, logField{key: "backend", value: usageBackend.Name()})
		initialWarnings = append(initialWarnings, message)
		logger.Print(message)
	}

	output, err := openCSVOutput(opts)
	if err != nil {
		return err
	}
	defer output.close()

	statusTracker := newCollectorStatusTracker(opts, usageBackend.Name(), output.workLogPath, output.logDir, logger)
	if statusTracker != nil {
		statusTracker.Start()
		defer statusTracker.Stop()
		for _, warning := range initialWarnings {
			statusTracker.RecordWarning(warning)
		}
	}

	csvWriter := logging.NewCSVWriter(output.writer)
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
				message := formatLogMessage("warning", "backend reset", err, logField{key: "backend", value: usageBackend.Name()})
				logger.Print(message)
				if statusTracker != nil {
					statusTracker.RecordWarning(message)
				}
			}
			resetAt = time.Now().Add(backendResetInterval)
		}

		sample, sampleErr := usageBackend.Sample(ctx)
		if sampleErr != nil {
			message := formatLogMessage("warning", "backend sample", sampleErr, logField{key: "backend", value: usageBackend.Name()})
			logger.Print(message)
			if statusTracker != nil {
				statusTracker.RecordWarning(message)
			}
		}

		written, err := writeActivitySample(csvWriter, sample, platform.NowUnixSeconds())
		if err != nil {
			wrappedErr := fmt.Errorf("write csv record: %w", err)
			if statusTracker != nil {
				statusTracker.RecordError(formatLogMessage("error", "write csv record", err, logField{key: "path", value: output.workLogPath}))
			}
			return wrappedErr
		}
		if written {
			if err := csvWriter.Flush(); err != nil {
				wrappedErr := fmt.Errorf("flush csv record: %w", err)
				if statusTracker != nil {
					statusTracker.RecordError(formatLogMessage("error", "flush csv record", err, logField{key: "path", value: output.workLogPath}))
				}
				return wrappedErr
			}
		}
		if statusTracker != nil {
			statusTracker.RecordSample(sample, sampleErr == nil && written)
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

func writeActivitySample(csvWriter *logging.CSVWriter, sample backend.UsageSample, timestampSeconds float64) (bool, error) {
	if !sample.HasActivity() {
		return false, nil
	}
	if err := csvWriter.WriteSample(sample, timestampSeconds); err != nil {
		return false, err
	}
	return true, nil
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

type csvOutput struct {
	writer      io.Writer
	close       func()
	logDir      string
	workLogPath string
}

func openCSVOutput(opts Options) (csvOutput, error) {
	if opts.QAConsole {
		return csvOutput{writer: os.Stdout, close: func() {}, workLogPath: "stdout"}, nil
	}

	homeDir, err := platform.UserHomeDir()
	if err != nil {
		return csvOutput{}, fmt.Errorf("resolve home directory: %w", err)
	}
	logDir, err := platform.EnsureLogDir(homeDir)
	if err != nil {
		return csvOutput{}, fmt.Errorf("ensure log directory %s: %w", platform.LogDir(homeDir), err)
	}

	workLogPath := platform.WorkLogPath(time.Now(), logDir)
	workLogFile, err := platform.OpenPrivateAppendFile(workLogPath)
	if err != nil {
		return csvOutput{}, fmt.Errorf("open work log %s: %w", workLogPath, err)
	}

	closeFn := func() {
		_ = workLogFile.Close()
	}
	return csvOutput{
		writer:      workLogFile,
		close:       closeFn,
		logDir:      logDir,
		workLogPath: workLogPath,
	}, nil
}

func newCollectorStatusTracker(opts Options, activeBackend string, workLogPath string, logDir string, logger *log.Logger) *runtimeStatusTracker {
	if opts.QAConsole || logDir == "" {
		return nil
	}
	store := status.NewStore(platform.StatusPath(logDir))
	return newRuntimeStatusTracker(store, logger, time.Now, opts.Backend, activeBackend, workLogPath)
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
