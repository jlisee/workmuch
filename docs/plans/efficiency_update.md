# Battery and Disk Efficiency Plan

This plan targets the avoidable work in the collector before adding power
profiles or platform-specific APIs. The first useful milestone is simple:
sample accurately while the user is active, then progressively stop waking and
writing so often while the user remains idle.

## Current Cost

The runner currently does more steady-state work than the data requires:

- It samples the backend at a fixed rate, `1Hz` by default, even after hours of
  idle time.
- It calls `csv.Writer.Flush()` after every recorded row.
- It rewrites `status.json` after every sample. That update creates a temporary
  file, closes it, and renames it over the old status file once per second.
- Every five seconds it calls `Backend.Reset()`. On Linux that closes and
  reconnects the X11 client even when the connection is healthy.

The status rewrite and X11 reconnect are especially wasteful because they do
filesystem metadata and connection work in addition to ordinary sampling.

## First Milestone

Implement one portable policy in `internal/app`. Do not add backend capability
hints, a power-state provider, or platform-specific activity notifications.
Every current backend already reports idle seconds, which is enough for the
first version.

Use this default sampling schedule:

```text
idle <= 1 minute:       existing active interval (`--rate`, normally 1s)
idle > 1 minute:        5s
idle > 10 minutes:      30s
idle > 1 hour:          60s
```

Never make an interval shorter than the interval selected by `--rate`. This
preserves an explicitly slower user setting.

When a sample reports low idle time again, select the active interval for the
very next sleep. With polling alone, the first resumed sample can be delayed by
the current idle interval, so the worst case with the defaults is 60 seconds.
The collector returns to full speed immediately after that sample detects
activity; there should be no gradual ramp back up.

That detection delay is the one limitation the portable first version cannot
remove. If testing later shows that a delay of up to 60 seconds is unacceptable,
add a narrowly scoped activity-wakeup source on platforms that provide one.
That is the only current reason to introduce a platform abstraction, and it
does not justify a general `Capabilities` structure.

## Configuration

Keep the initial command-line surface small:

```text
--rate <float>                 active samples per second; existing option
--max-idle-interval <duration> cap adaptive idle sleeps; default 1m
--flush-interval <duration>    maximum CSV/status buffer age; default 30s
```

Parse the new values as Go durations, such as `10s` or `1m`. A zero flush
interval means immediate flushes for users who prefer the current durability
behavior. Reject negative values. `--qa-console` always flushes each row so
interactive output remains immediate, regardless of the configured interval.

The idle thresholds and intermediate `5s`/`30s` intervals should remain fixed
for the first implementation. More flags would create configuration and test
surface before there is evidence that those values need tuning.

## Sampling Policy

Add a small pure policy under `internal/app`; it only needs the active interval,
maximum idle interval, and the latest `IdleSeconds` value. It should return the
next interval and have no dependency on an OS backend.

Use the sample's positive idle value even when another part of the sample
returned a warning. For example, a locked desktop may fail to return a window
while still returning valid idle time. If idle time is zero or unavailable,
fall back to the active interval so failures recover quickly.

Replace the fixed-period schedule in the runner with the policy result after
each sample. Preserve context-aware sleep and calculate each deadline from the
current time when the interval changes; old fixed-rate deadlines should not
cause catch-up samples after a long idle period.

## Batched Writes

Normal worklogs should:

- buffer CSV rows for the configured flush interval;
- flush before a sleep that would carry buffered data past that interval;
- flush when the daily writer rotates files;
- flush and report errors during clean shutdown; and
- continue to avoid `fsync`, leaving physical write scheduling to the OS.

Flush opportunistically from the sampling loop instead of adding a second
periodic timer. If the next adaptive sleep would cross the flush deadline,
flush just before sleeping. During active use this batches roughly 30 seconds
of rows. During long idle periods it can flush the single new row immediately
and then sleep for a minute, avoiding a separate flush-only wakeup.

The runtime status tracker should update its in-memory value for every sample
but persist `status.json` on the same bounded schedule as the CSV writer.
Lifecycle changes and new warnings or fatal errors should still be persisted
immediately. The tray Status view may therefore show a healthy sample that is
up to one flush interval old, which is preferable to replacing the file every
second.

Change the output close path to return errors rather than discarding the
`DailyCSVWriter.Close()` result. A promised shutdown flush is only useful if a
failure reaches the log and process result.

## Backend Reset

Remove the unconditional five-second reset from the runner. The Linux backend
already reconnects when a sample detects a lost X11 connection, while both
macOS reset methods are no-ops. If a generic retry remains necessary, trigger
it only after a sample failure and apply a bounded backoff; a healthy backend
must never be reset on a timer.

This change needs no capability query because the common backend contract
already exposes `Reset()` and `Sample()` errors.

## TDD Implementation Order

1. Add table-driven policy tests for every idle boundary, a custom active rate,
   the maximum-idle cap, and the immediate return to the active interval.
2. Implement the pure adaptive policy and option parsing/help tests for the two
   duration flags.
3. Add fake-writer tests proving normal rows are not flushed individually,
   `--qa-console` rows are immediate, a deadline-crossing sleep flushes first,
   and shutdown/rotation flush errors are returned.
4. Refactor the runner to use adaptive deadlines and opportunistic flushes.
   Keep timing decisions behind pure helpers so unit tests do not sleep.
5. Add status-tracker tests proving samples are coalesced while lifecycle and
   error updates remain immediate, then align status persistence with the CSV
   flush schedule.
6. Add a Linux regression test showing successful samples do not call
   `Reset()`, while the existing connection-loss path still reconnects.
7. Run the full tests and lint, then manually observe active, short-idle, and
   long-idle transitions in `--qa-console` and normal tray modes.

Each step should follow red/green TDD and land with the CSV schema unchanged.

## Acceptance Criteria

With default settings:

- Active collection remains at `1Hz` and keeps the current row format.
- After one hour idle, there is at most one backend sample and one activity row
  per minute.
- Activity sampling returns to `1Hz` immediately after a poll observes resumed
  input.
- Healthy normal collection flushes CSV and persists status no more than about
  once every 30 seconds while active, and does not create extra flush-only
  wakeups while idle.
- `--qa-console` still displays every row immediately.
- Clean shutdown and daily rotation leave no buffered rows behind.
- A healthy Linux run no longer tears down its X11 connection every five
  seconds.

## Deferred Work

Measure this portable version before expanding it. Defer all of the following
until a measured problem requires one of them:

- backend capability hints;
- battery/AC or low-power-mode providers;
- different defaults for native and subprocess backends;
- platform activity-wakeup events; and
- SQLite batching or WAL tuning.

If near-instant resume becomes a firm requirement, first prototype only the
activity-wakeup event needed to interrupt a long idle sleep. Do not make that
experiment a prerequisite for adaptive polling and batched writes.
