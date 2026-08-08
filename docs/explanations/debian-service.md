# Debian user service

The Debian package globally enables `workmuch.service` as a systemd user unit.
Each logged-in desktop user gets a separate WorkMuch process and separate
worklogs under that user's `~/.workmuch`. Installation does not create a
system-wide collector or a privileged data directory.

The package starts or restarts the service for active user managers. If no
desktop user is logged in during installation, this succeeds without starting
anything. WorkMuch starts at the next graphical login instead.

The service uses WorkMuch's default tray mode. It starts collection and
registers a tray icon through the desktop session's StatusNotifier/AppIndicator
service. Desktops without a compatible system tray provider can still run the
collector, but cannot display its icon or menu.

## Inspect and control the service

Run these commands as the desktop user whose collector you want to inspect:

```bash
systemctl --user status workmuch.service
journalctl --user -u workmuch.service
```

Follow new journal messages with:

```bash
journalctl --user -u workmuch.service -f
```

Stop WorkMuch for the current login with `systemctl --user stop
workmuch.service`. To opt the current user out across future logins and package
upgrades, create a per-user mask:

```bash
systemctl --user mask workmuch.service
systemctl --user stop workmuch.service
```

Remove that opt-out and start collection again with:

```bash
systemctl --user unmask workmuch.service
systemctl --user start workmuch.service
```

Package removal stops active instances and removes package-owned global
enablement. It does not remove worklogs in `~/.workmuch` or a mask created by a
user in their own systemd configuration.

## Display support

WorkMuch requires a nonempty `DISPLAY` and an accessible X11 server. The user
unit skips startup when the graphical session has not exported `DISPLAY` to
the systemd user manager. A desktop can import it with its normal session
startup integration or with `systemctl --user import-environment DISPLAY`.

Xorg sessions are supported. XWayland can expose a usable X11 display, but it
cannot report native Wayland windows consistently. A pure Wayland session is
unsupported. See the main Linux documentation for diagnostics and privacy
details.
