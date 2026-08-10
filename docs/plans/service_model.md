# Service Model

WorkMuch uses the supported macOS 13+ main-app Login Item model. The packaged
`WorkMuch.app` calls `SMAppService.mainAppService`; it does not install a plist
under `~/Library/LaunchAgents` and does not invoke `launchctl`.

## Implemented behavior

- The signed app has a stable path at `/Applications/WorkMuch.app`, bundle
  identifier `com.jlisee.workmuch`, and executable name `workmuch`.
- A bundled tray launch outside `/Applications` explains that the app must be
  moved and exits before requesting permission or registering startup.
- The first installed tray launch registers the main app only while its status
  is `not_registered`.
- `enabled` is successful. `requires_approval` belongs to the user and is never
  overridden. Framework failures are logged without stopping collection.
- Doctor and tray Status report the Login Item state from ServiceManagement.
- Installed tray launches request Accessibility when needed and keep collecting
  partial data while approval is pending.
- Checkout use through `./run.sh`, `doctor`, `--qa-console`, and `--no-tray`
  remains independent from automatic registration and prompting.

Build, installation, permission, update, and uninstall operations are
documented in
[macOS installation and local releases](../explanations/macos-release.md).

## Remaining operational work

- Complete manual acceptance on Apple Silicon and Intel hardware.
- Measure idle-aware sampling and status-write behavior before changing the
  collector's timing or buffering policy.
- Add user-facing startup controls only if the System Settings Login Item
  control proves insufficient.
