# Systray / Menu Bar Integration

Systray support is now implemented for the Go app, using
`github.com/tailscale/systray`.

## Current behavior

- `./run_go.sh` launches the tray app by default.
- Logging starts immediately on launch.
- Tray menu items in v1:
  - `About` (opens a basic about screen)
  - `Status` (opens backend, sampling, log, and runtime diagnostics)
  - `Quit`
- `--qa-console` bypasses tray mode and runs the foreground collector with CSV
  output to stdout.
- `--no-tray` bypasses tray mode but retains the normal daily worklog output.

## Implementation notes

- Tray orchestration lives in `go/internal/tray`.
- Collector lifecycle control is handled by `go/internal/app.Controller`.
- The app loop still runs in-process.
- The tray icon is embedded from:
  - `go/internal/tray/assets/icon32x32.png` (active tray icon)
  - `go/internal/tray/assets/icon16x16.png` (reserved for future use)

## Shutdown behavior

- Selecting `Quit` exits the tray and shuts down the collector.
- Selecting `About` opens a basic screen with:
  - `Current app and window title based activity tracker`
- Selecting `Status` opens a private temporary page with the current doctor
  report.
- External cancellation (for example SIGTERM/interrupt in terminal-launched
  sessions) also shuts down the collector and exits tray mode.

## Follow-up ideas

- Add tray controls for start/stop.
- Add "Open log folder" and "Open latest work log".
