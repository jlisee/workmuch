package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"workmuch-go/internal/app"
	"workmuch-go/internal/platform"
)

func main() {
	exitCode := run()
	os.Exit(exitCode)
}

func run() int {
	opts, showHelp, err := app.ParseOptions(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n\n", err)
		fmt.Fprint(os.Stderr, app.HelpText(filepath.Base(os.Args[0])))
		return 1
	}
	if showHelp {
		fmt.Print(app.HelpText(filepath.Base(os.Args[0])))
		return 0
	}

	logger, closeLogger := configureLogger(opts.QAConsole)
	defer closeLogger()

	logger.Printf("Program started")
	logger.Printf("Recording at %fHz", opts.Rate)
	logger.Printf("Waiting %f seconds before starting logging", opts.StartDelay)
	logger.Printf("CSV columns: window_title, program_name, idle_seconds, timestamp_seconds")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := app.Run(ctx, opts, logger); err != nil {
		logger.Printf("fatal error: %v", err)
		return 1
	}

	logger.Printf("Program shutdown complete")
	return 0
}

func configureLogger(qaConsole bool) (*log.Logger, func()) {
	logWriter := io.Writer(os.Stderr)
	cleanup := func() {}

	if !qaConsole && !isTerminal(os.Stdout) {
		homeDir, err := platform.UserHomeDir()
		if err == nil {
			if logDir, ensureErr := platform.EnsureLogDir(homeDir); ensureErr == nil {
				errorLogPath := platform.ErrorLogPath(logDir)
				if file, openErr := os.OpenFile(errorLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644); openErr == nil {
					logWriter = file
					cleanup = func() {
						_ = file.Close()
					}
				}
			}
		}
	}

	return log.New(logWriter, "", log.LstdFlags), cleanup
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}
