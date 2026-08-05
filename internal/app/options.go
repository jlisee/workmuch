package app

import (
	"fmt"
	"strconv"
	"strings"

	"workmuch-go/internal/backend"
)

type Options struct {
	Rate       float64
	StartDelay float64
	QAConsole  bool
	NoTray     bool
	Backend    string
}

func DefaultOptions() Options {
	return Options{
		Rate:       1.0,
		StartDelay: 0.0,
		QAConsole:  false,
		NoTray:     false,
		Backend:    backend.BackendAuto,
	}
}

func ParseOptions(args []string) (Options, bool, error) {
	opts := DefaultOptions()
	showHelp := false

	for i := 0; i < len(args); i++ {
		arg := args[i]

		switch {
		case arg == "-h" || arg == "--help":
			showHelp = true
		case arg == "--qa-console":
			opts.QAConsole = true
		case arg == "--no-tray":
			opts.NoTray = true
		case arg == "-r" || arg == "--rate":
			value, next, err := readFloatArg(args, i, arg)
			if err != nil {
				return Options{}, false, err
			}
			opts.Rate = value
			i = next
		case strings.HasPrefix(arg, "--rate="):
			value, err := strconv.ParseFloat(strings.TrimPrefix(arg, "--rate="), 64)
			if err != nil {
				return Options{}, false, fmt.Errorf("invalid --rate value: %w", err)
			}
			opts.Rate = value
		case arg == "-d" || arg == "--start-delay":
			value, next, err := readFloatArg(args, i, arg)
			if err != nil {
				return Options{}, false, err
			}
			opts.StartDelay = value
			i = next
		case strings.HasPrefix(arg, "--start-delay="):
			value, err := strconv.ParseFloat(strings.TrimPrefix(arg, "--start-delay="), 64)
			if err != nil {
				return Options{}, false, fmt.Errorf("invalid --start-delay value: %w", err)
			}
			opts.StartDelay = value
		case arg == "--backend":
			if i+1 >= len(args) {
				return Options{}, false, fmt.Errorf("missing value for %s", arg)
			}
			i++
			opts.Backend = args[i]
		case strings.HasPrefix(arg, "--backend="):
			opts.Backend = strings.TrimPrefix(arg, "--backend=")
		case strings.HasPrefix(arg, "-"):
			return Options{}, false, fmt.Errorf("unknown flag: %s", arg)
		default:
			return Options{}, false, fmt.Errorf("unexpected argument: %s", arg)
		}
	}

	opts.Backend = strings.ToLower(strings.TrimSpace(opts.Backend))
	if opts.Backend == "" {
		opts.Backend = backend.BackendAuto
	}

	if opts.Rate <= 0 {
		return Options{}, false, fmt.Errorf("rate must be greater than zero")
	}
	if opts.StartDelay < 0 {
		return Options{}, false, fmt.Errorf("start-delay must be >= 0")
	}

	switch opts.Backend {
	case backend.BackendAuto, backend.BackendMacOSSubprocess, backend.BackendMacOSNative, backend.BackendLinux:
		return opts, showHelp, nil
	default:
		return Options{}, false, fmt.Errorf("unsupported backend %q", opts.Backend)
	}
}

func HelpText(program string) string {
	return fmt.Sprintf(`Usage: %s [command] [options]

Commands:
  doctor                    Print backend, permission, log, service, and runtime diagnostics

Default behavior:
  Launch the tray icon and log activity in the background.

Options:
  -r, --rate <float>          Samples per second (default: 1.0)
  -d, --start-delay <float>   Seconds to wait before logging (default: 0.0)
      --qa-console            Disable tray mode and write CSV to stdout
      --no-tray               Disable tray mode and write CSV to the daily worklog
      --backend <name>        One of: auto, macos-subprocess, macos-native, linux
  -h, --help                  Show this help
`, program)
}

func readFloatArg(args []string, index int, flagName string) (float64, int, error) {
	if index+1 >= len(args) {
		return 0, index, fmt.Errorf("missing value for %s", flagName)
	}
	value, err := strconv.ParseFloat(args[index+1], 64)
	if err != nil {
		return 0, index, fmt.Errorf("invalid %s value: %w", flagName, err)
	}
	return value, index + 1, nil
}
