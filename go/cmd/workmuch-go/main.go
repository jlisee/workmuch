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
	"workmuch-go/internal/doctor"
	"workmuch-go/internal/platform"
	"workmuch-go/internal/tray"
)

type runMode int

const (
	runModeForeground runMode = iota
	runModeTray
)

type commandKind int

const (
	commandRun commandKind = iota
	commandDoctor
)

type command struct {
	kind commandKind
	opts app.Options
}

func main() {
	exitCode := run()
	os.Exit(exitCode)
}

func run() int {
	cmd, showHelp, err := parseCommand(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n\n", err)
		fmt.Fprint(os.Stderr, app.HelpText(filepath.Base(os.Args[0])))
		return 1
	}
	if showHelp {
		fmt.Print(app.HelpText(filepath.Base(os.Args[0])))
		return 0
	}

	if cmd.kind == commandDoctor {
		if err := executeDoctor(cmd.opts); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			return 1
		}
		return 0
	}

	logger, closeLogger := configureLogger(cmd.opts.QAConsole)
	defer closeLogger()

	logger.Printf("Program started")
	logger.Printf("Recording at %fHz", cmd.opts.Rate)
	logger.Printf("Waiting %f seconds before starting logging", cmd.opts.StartDelay)
	logger.Printf("CSV columns: host, user, window_title, program_name, idle_seconds, timestamp_seconds")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := executeRunMode(ctx, cmd.opts, logger); err != nil {
		logger.Printf("fatal error: %v", err)
		return 1
	}

	logger.Printf("Program shutdown complete")
	return 0
}

func parseCommand(args []string) (command, bool, error) {
	if len(args) > 0 {
		switch args[0] {
		case "doctor":
			opts, showHelp, err := app.ParseOptions(args[1:])
			return command{kind: commandDoctor, opts: opts}, showHelp, err
		}
	}

	opts, showHelp, err := app.ParseOptions(args)
	return command{kind: commandRun, opts: opts}, showHelp, err
}

func selectRunMode(opts app.Options) runMode {
	if opts.QAConsole || opts.NoTray {
		return runModeForeground
	}
	return runModeTray
}

func executeRunMode(ctx context.Context, opts app.Options, logger *log.Logger) error {
	switch selectRunMode(opts) {
	case runModeForeground:
		return app.Run(ctx, opts, logger)
	case runModeTray:
		return tray.RunWithContext(ctx, opts, logger)
	default:
		return fmt.Errorf("unsupported run mode: %d", selectRunMode(opts))
	}
}

func executeDoctor(opts app.Options) error {
	report := doctor.NewCollector(opts.Backend).Collect(context.Background())
	fmt.Print(doctor.RenderText(report))
	return nil
}

func configureLogger(qaConsole bool) (*log.Logger, func()) {
	logWriter := io.Writer(os.Stderr)
	cleanup := func() {}

	if !qaConsole && !isTerminal(os.Stdout) {
		homeDir, err := platform.UserHomeDir()
		if err == nil {
			if logDir, ensureErr := platform.EnsureLogDir(homeDir); ensureErr == nil {
				errorLogPath := platform.ErrorLogPath(logDir)
				if file, openErr := platform.OpenPrivateAppendFile(errorLogPath); openErr == nil {
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
