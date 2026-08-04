# Linux usage

The Go collector currently supports X11/Xorg sessions. It does not support
Wayland sessions yet. Some Wayland desktops expose an XWayland `$DISPLAY`, but
that view can omit native Wayland applications, so WorkMuch does not treat it
as a supported session.

Run the diagnostics before starting collection:

```bash
./run_go.sh doctor
```

On Linux, `doctor` reports the detected desktop session, whether that session
is supported, the X11 display, the X11 connection result, and the sampling
result. It also shows a live app, window title, and idle time when sampling
succeeds. To check the connection error shown for an unusable display, run:

```bash
DISPLAY=:invalid ./run_go.sh doctor
```

`doctor` samples only for its report. It does not append a worklog row. The
collector also exits before opening a worklog when it cannot connect to X11,
and it skips samples where neither an app nor a window title is available.

## Collection modes

Run without options to start the tray collector:

```bash
./run_go.sh
```

Collection starts immediately and writes the daily worklog. The tray menu has
`About`, `Status`, and `Quit`. `Status` opens the same health information as
`doctor`; `Quit` stops collection and exits the tray.

To collect in the foreground without a tray, use:

```bash
./run_go.sh --no-tray
```

This writes the normal daily worklog until Ctrl+C stops the process. For
manual QA, stream rows to standard output instead:

```bash
./run_go.sh --qa-console
```

QA console mode does not create or append a worklog or `error.log`. Redirecting
its standard output will still save the activity data wherever the shell is
directed.

## Worklog format

Normal tray and `--no-tray` collection append headerless CSV rows to:

```text
~/.workmuch/YYYY-MM-DD.worklog
```

The date in the file name is the local date when collection starts. Each row
has these columns in order:

| Column | Meaning | Unit or format |
| --- | --- | --- |
| `host` | Local host name | Text |
| `user` | Local user name | Text |
| `window_title` | Active window title | Text |
| `program_name` | Active program from X11 `WM_CLASS` | Text |
| `idle_seconds` | Time since the last user input | Seconds, decimal |
| `timestamp_seconds` | Time when WorkMuch recorded the sample | Unix epoch seconds, decimal |

Fields follow normal CSV quoting rules. Both numeric fields are currently
written with six digits after the decimal point.

## Privacy

Worklogs can reveal document names, sites, applications, user and host names,
idle periods, and daily work patterns. WorkMuch makes `~/.workmuch` accessible
only to its owner (`0700`) and creates worklogs and `error.log` with mode
`0600`. These permissions do not protect copies made by backups, shell
redirection, synchronization tools, or users with administrative access.

Review how those files are stored and retained before collecting sensitive
work. The tray's `Status` page can also display the current window title, so
avoid sharing it when private material is active.
