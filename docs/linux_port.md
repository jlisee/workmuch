# Linux X11 Port Record

This document began as the Linux Porting Plan. The planned Go X11 port is
complete and this record preserves its scope and validation results.

## Delivered behavior

- The Linux backend connects to `$DISPLAY`, reads the active X11 window, and
  records its title, `WM_CLASS` program name, and idle time.
- It uses the X11 Screen Saver protocol through the pure-Go
  `github.com/jezek/xgb` library, so no X11 development package is required.
- `./run.sh doctor` reports session, X11 connection, and sampling diagnostics.
- `./run.sh --no-tray` writes normal worklogs, while
  `./run.sh --qa-console` streams CSV rows without creating log files.
- The backend retries after a lost X11 connection, and samples with no app or
  window title are not written.

The current user-facing instructions are in [Linux usage](linux.md).

## Validation result (2026-08-04)

The completed port was validated on the Ubuntu GNOME X11 target:

- The unattended session remained locked. The live tray's exported D-Bus menu
  exercised About, Status, and Quit; Status used the collector's cached sample
  and Quit stopped the collector cleanly.
- Locked-screen samples without activity produced no empty CSV rows. Synthetic
  input reset idle time from about 2,782 seconds to 0.17 seconds, and valid
  activity resumed without restarting the collector.
- An actual host suspend/resume was not safe with the machine unattended. As
  an unattended substitute, a disposable Xvfb server was stopped and recreated
  to test recovery from complete X11 transport loss. The collector reported
  `EOF`, retried while the display was absent, and resumed after it returned.
- A 30-minute tray soak produced exactly 1,800 parseable six-column rows. The
  timestamps were strictly increasing, the mean interval was 1.000000 seconds
  (0.997668 minimum and 1.002136 maximum), and tray Quit left runtime status
  stopped. The worklog, status file, and error log remained mode `0600`.

## Later independent work

- Rotate worklogs at local midnight.
- Build and install a stable binary with XDG autostart.
- Add idle-aware sampling and buffered writes.
- Add Wayland support.
- Import CSV into the SQLite schema.
