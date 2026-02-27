# Systray / Menu Bar Plan (Planned)

This document outlines future systray work. No tray integration is implemented in this change.

## Goals

- Offer quick visibility into collector state.
- Provide lightweight controls without requiring a terminal.
- Keep tray behavior separate from collector core loop.

## Candidate Go libraries

- [`fyne-io/systray`](https://github.com/fyne-io/systray): mature cross-platform tray abstraction.
- [`getlantern/systray`](https://github.com/getlantern/systray): widely used minimal tray package.

Selection criteria:

- macOS stability in long-running sessions
- simple menu event handling
- clean build story with Go toolchain

## Proposed v1 tray menu

- Status: Running / Stopped (read-only text item)
- Start Logging
- Stop Logging
- Open Log Folder (`~/.workmuch`)
- Quit

## Integration approach

- Keep collector loop in a reusable package and control it via start/stop channels.
- Keep tray process as the primary app for user interaction.
- Run collector in-process first; evaluate helper process model later if needed.

## macOS constraints to account for

- Some tray libs require running UI loop on the main thread.
- Menu updates must remain responsive while sampling loop runs.
- Permission prompts and tray UX should avoid deadlocks with backend calls.

## Later enhancements

- Sample rate controls
- Backend selection UI
- “Open latest log file” action
- Recent error indicator in menu
