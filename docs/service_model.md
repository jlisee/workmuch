# Service Model (Planned)

This document captures implementation notes for turning the Go collector into a service.
Service setup is intentionally not implemented in this change.

## Goals

- Start and keep the collector running automatically for a logged-in user.
- Keep service lifecycle independent from manual terminal sessions.
- Preserve existing output format and log destination (`~/.workmuch/*.worklog`).

## macOS options

### LaunchAgent (recommended)

Use a per-user `launchd` agent (`~/Library/LaunchAgents/...plist`) for activity collection.
This runs in a user session where focused app/window context is available.

Suggested label: `com.jlisee.workmuch`

Suggested command:

- program: absolute path to built Go binary
- arguments: include collector flags (for example `--backend macos-subprocess`)

Suggested plist keys:

- `Label`
- `ProgramArguments`
- `RunAtLoad = true`
- `KeepAlive = true`
- `StandardOutPath` and `StandardErrorPath` to `~/.workmuch/`

### LaunchDaemon (not recommended)

A system daemon is not a good fit for this workload because foreground window/session data is user-session scoped.

## Install / update / uninstall flow (planned)

1. Build and place binary in a stable location.
2. Write/update LaunchAgent plist in `~/Library/LaunchAgents`.
3. Load with `launchctl bootstrap gui/$UID <plist>`.
4. Verify with `launchctl print gui/$UID/<label>`.
5. On updates, unload/reload or kickstart service.
6. On uninstall, unload service and remove plist/binary.

## Permissions and failure modes

- macOS accessibility permission may be required for reliable window title collection.
- If permission is missing, service should keep running and collect partial data.
- Transient command failures from subprocess backend should not stop service.

## Observability notes

- Keep persistent stderr logs in `~/.workmuch/error.log`.
- Add simple health checks later (last sample timestamp, last backend error).
- Consider a status command for service diagnostics.
