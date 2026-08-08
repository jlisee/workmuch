# Service Model

This document tracks what remains to make the Go collector feel like a normal
macOS tool: install it once, start it automatically, inspect its status, and
remove it cleanly.

Service setup is not implemented yet. The collector itself is usable from the
checkout, but it still needs an installer/service layer before it is a
"just start running on this Mac" tool.

## Goals

- [ ] Start and keep the collector running automatically for a logged-in user.
- [ ] Keep service lifecycle independent from manual terminal sessions.
- [x] Preserve the existing CSV output format.
- [x] Preserve the existing work log destination (`~/.workmuch/*.worklog`).
- [x] Keep a manual QA mode that avoids writing log files.
- [ ] Provide a simple command-line lifecycle: install, start, stop, status,
  restart, uninstall.
- [ ] Provide clear diagnostics when macOS permissions prevent complete
  collection.

## Current state

- [x] Go collector entrypoint exists at `cmd/workmuch-go`.
- [x] `./run.sh` launches the Go app.
- [x] Tray mode is the default run mode.
- [x] Tray menu supports `About`, `Status`, and `Quit`.
- [x] `--qa-console` bypasses tray mode and streams CSV rows to stdout.
- [x] Normal mode writes activity rows under `~/.workmuch`.
- [x] Non-console logging can write persistent errors to
  `~/.workmuch/error.log`.
- [x] macOS native backend exists and is selected by `--backend auto` on
  Darwin.
- [x] macOS subprocess backend exists as a fallback/debug path.
- [x] Collector shutdown is wired through tray Quit, SIGINT, and SIGTERM.
- [x] Unit tests cover the core app, backend, logging, platform, and tray
  lifecycle behavior.
- [ ] No stable installed binary location exists yet.
- [ ] No LaunchAgent plist is generated yet.
- [ ] No install/update/uninstall commands exist yet.
- [x] `./run.sh doctor` reports backend, permission, log, service, and runtime
  diagnostics.
- [ ] No app bundle or stable executable identity exists yet for a smoother
  macOS permissions experience.

## macOS options

### LaunchAgent (recommended)

Use a per-user `launchd` agent (`~/Library/LaunchAgents/...plist`) for activity collection.
This runs in a user session where focused app/window context is available.

Suggested label: `com.jlisee.workmuch`

Suggested command:

- program: absolute path to built Go binary
- arguments: include collector flags, if any

Suggested plist keys:

- `Label`
- `ProgramArguments`
- `RunAtLoad = true`
- `KeepAlive = true`
- `StandardOutPath` and `StandardErrorPath` to `~/.workmuch/`

Implementation notes:

- The binary path should be stable across upgrades so macOS permissions remain
  attached to the same executable identity as much as possible.
- Prefer `--backend auto` for the installed default. The native backend should
  be the long-running macOS path; the subprocess backend is useful for
  debugging but is more expensive.
- The LaunchAgent should run in the user's GUI session with
  `launchctl bootstrap gui/$UID <plist>`.
- The LaunchAgent should not use `--qa-console`; normal service runs should
  write work logs.

### LaunchDaemon (not recommended)

A system daemon is not a good fit for this workload because foreground window/session data is user-session scoped.

## Install / update / uninstall checklist

### Already done

- [x] Buildable Go command exists.
- [x] Runtime defaults to tray mode when run directly.
- [x] Runtime can run as a foreground collector with `--qa-console`.
- [x] Runtime handles SIGTERM for clean shutdown.
- [x] Runtime creates `~/.workmuch` for normal work logs.

### Still needed

- [ ] Add a build/install command that compiles the Go binary.
- [ ] Choose the installed binary path. Suggested:
  `~/Library/Application Support/WorkMuch/workmuch`.
- [ ] Create the install directory with user-only write access.
- [ ] Copy or replace the binary atomically during install/update.
- [ ] Generate `~/Library/LaunchAgents/com.jlisee.workmuch.plist`.
- [ ] Load the LaunchAgent with `launchctl bootstrap gui/$UID <plist>`.
- [ ] Start or restart the service with `launchctl kickstart`.
- [ ] Stop the service with `launchctl bootout gui/$UID <plist>` or the label.
- [ ] Print service state with `launchctl print gui/$UID/com.jlisee.workmuch`.
- [ ] Uninstall by stopping the service, removing the plist, and removing the
  installed binary.
- [ ] Make install/update idempotent so repeated runs are safe.
- [ ] Document how to recover from stale LaunchAgent state.

Possible command surface:

```text
workmuch install
workmuch uninstall
workmuch start
workmuch stop
workmuch restart
workmuch status
workmuch doctor
workmuch run --qa-console
```

The exact CLI shape can change, but the service lifecycle should be available
without manually editing plists or remembering `launchctl` commands.

## Permissions and failure modes

- [x] Native backend checks Accessibility trust internally.
- [x] If permission is missing, the collector keeps running and collects
  partial data.
- [x] Transient backend sample failures are logged as warnings instead of
  stopping the collector.
- [x] `./run.sh doctor` reports the active backend, Accessibility permission,
  a diagnostic sample, log-directory writability, the LaunchAgent plist state,
  and runtime status. It does not yet manage a LaunchAgent.
- [ ] Add user-facing guidance for granting Accessibility permission to the
  installed binary or app bundle.
- [ ] Verify app/window capture from the installed service in a normal macOS
  GUI login session.
- [ ] Decide whether to ship a `.app` wrapper or signed/ad-hoc-signed binary to
  make macOS permission prompts and identity more predictable.

Known readiness gap: manual QA from a non-normal GUI/shell context can emit
CSV rows with host/user but empty app and window fields. Before treating the
service as ready for daily use, verify that the installed binary can read the
frontmost app, focused window title, and idle time from the intended user
session.

## Observability notes

- [x] Keep persistent stderr logs in `~/.workmuch/error.log` for non-console
  runs.
- [x] Keep activity samples in daily `~/.workmuch/*.worklog` files.
- [x] `doctor` and the tray Status view report runtime diagnostics.
- [x] Track the last successful sample timestamp and latest backend warning or
  error.
- [x] Track selected and active backends and the current worklog path.
- [ ] Report whether the LaunchAgent is loaded and whether launchd thinks it is
  healthy; the current report only checks the plist.

## Power and always-on behavior

- [x] Native macOS backend exists and should be the default long-running path.
- [ ] Implement idle-aware adaptive sampling with a one-minute maximum sleep.
- [ ] Batch CSV and runtime-status writes in normal mode.
- [ ] Keep immediate flush behavior in `--qa-console`.
- [ ] Avoid backend capability and power-provider abstractions until measured
  behavior shows that the portable idle policy is insufficient.

These items are not required to prove the LaunchAgent model, but they matter
before recommending WorkMuch as an always-on laptop tool.

## Suggested next implementation order

1. Add `doctor`/`status` diagnostics and verify macOS data capture from a
   normal GUI session.
2. Add install/start/stop/restart/uninstall commands around a per-user
   LaunchAgent.
3. Stabilize the installed executable identity with a fixed path and possibly a
   `.app` wrapper or signing step.
4. Add adaptive sampling and buffered CSV flushing.
5. Expand tray actions once the service lifecycle is reliable.
