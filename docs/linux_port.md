# Linux Porting Plan

The Python backend already uses the X11 Screen Saver extension. It loads
`libXss` and calls `XScreenSaverQueryInfo` to obtain the idle time.

"Extension" here means an X11 protocol capability provided by the X server,
not a GNOME extension or an additional WorkMuch plugin. The Go implementation
will speak the same protocol through a pure-Go library, avoiding `libxss-dev`
and CGO.

Each step below must remain green, receive its own manual check, and stop for
review before the next step begins.

## Step 1: Implement basic Linux X11 sampling

Use red/green TDD with `testify` to replace the Linux backend stub.

- Connect to `$DISPLAY`.
- Read the active X11 window.
- Read its title and `WM_CLASS` program name.
- Query idle milliseconds using the same XScreenSaver mechanism as Python.
- Add basic `Reset()` and `Close()` behavior.
- Preserve the current CSV fields and backend interface.
- Do not change storage, tray behavior, permissions, or installation yet.

Run the automated checks:

```bash
./test_go.sh
./lint.sh
```

Then run the manual checks:

```bash
./run_go.sh doctor
./run_go.sh --qa-console
```

Switch among several applications and verify that program, title, and idle
time change. This step proves the backend without writing activity files.

## Step 2: Enable safe persistent collection

Once sampling is correct:

- Add `--no-tray` for foreground collection into normal worklogs.
- Make `~/.workmuch` private with mode `0700`.
- Make worklogs and `error.log` private with mode `0600`.
- Add tests for run-mode selection and permissions.

Run the automated checks, then start a manual collection:

```bash
./test_go.sh
./lint.sh
./run_go.sh --no-tray
```

Run it for several minutes, stop it with Ctrl+C, and validate the resulting
daily CSV. This is the milestone at which daily Linux collection can begin.

## Step 3: Improve Linux diagnostics and documentation

After real data is being collected:

- Make `doctor` clearly report X11 connection and sampling failures.
- Detect Wayland and explain that only X11/Xorg is supported initially.
- Document the CSV column order, units, location, and privacy implications.
- Document `--qa-console`, `--no-tray`, and tray usage.

Run `doctor` normally and with an invalid `$DISPLAY`. Confirm that failures are
clear and do not produce empty worklog rows.

## Step 4: Validate tray and recovery behavior

Keep reliability testing separate from the first usable collector:

- Exercise the existing Linux tray integration.
- Test lock/unlock and suspend/resume.
- Verify that the backend reconnects after transient X11 failures.
- Perform a 30-minute soak test.
- Fix only problems found during this pass.

Manual checks include About, Status, Quit, idle reset after input, recovery
after unlock, CSV parsing, timestamp ordering, and approximately one row per
second.

## Later independent steps

- Rotate worklogs at local midnight.
- Build and install a stable binary with XDG autostart.
- Add idle-aware sampling and buffered writes.
- Add Wayland support.
- Import CSV into the existing SQLite schema.

## Assumptions

- The first release targets the current Ubuntu X11 session.
- No `libxss-dev` installation is required with the pure-Go approach.
- SQLite ingestion and reporting are outside the first usable Linux slice.
