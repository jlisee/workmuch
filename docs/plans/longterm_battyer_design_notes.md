# Battery-Friendly Collector Design

NOTE:  Long term plans that we'll use as needed.

This document captures the energy and battery-life design for the Go collector,
especially on macOS laptops. It focuses on three areas:

- idle-aware sampling
- write batching
- OS/backend abstraction changes needed to support those behaviors cleanly

The goal is to keep WorkMuch useful as an always-on activity recorder without
making it feel like a background battery tax.

## Background

WorkMuch currently samples on a fixed interval. In the Go implementation, the
default rate is `1Hz`: once per second the collector asks the active backend for
the current window/app/idle state, writes one CSV row, flushes the CSV writer,
and sleeps until the next tick.

On macOS there are two backend styles:

- `macos-native`: uses AppKit, CoreGraphics, and Accessibility APIs directly.
- `macos-subprocess`: shells out to `osascript` and `ioreg`.

The native backend should be preferred for battery life. A native sample that
takes a few milliseconds at `1Hz` has a low CPU duty cycle, roughly:

```text
1ms per second = 0.1% of one CPU core
5ms per second = 0.5% of one CPU core
```

That duty-cycle estimate is useful, but it is not the whole battery story.
Apple's energy guidance emphasizes that frequent timers and small repeated I/O
can prevent the system from staying in lower-power states. WorkMuch should
therefore optimize for fewer wakeups and fewer writes when the extra precision
does not improve the dataset.

Useful Apple background:

- Energy Efficiency Guide for Mac Apps: minimize timer usage
  https://developer.apple.com/library/archive/documentation/Performance/Conceptual/power_efficiency_guidelines_osx/Timers.html
- Energy Efficiency Guide for Mac Apps: minimize I/O
  https://developer.apple.com/library/archive/documentation/Performance/Conceptual/power_efficiency_guidelines_osx/MinimizingIO.html
- Energy Efficiency Guide for Mac Apps: general best practices
  https://developer.apple.com/library/archive/documentation/Performance/Conceptual/power_efficiency_guidelines_osx/BestPractices.html

## Design Principles

1. Use native OS APIs for steady-state collection.
   Subprocess-based sampling is useful as a fallback, but it
   launches multiple processes per sample and should not be the long-running
   default on laptops.

2. Spend precision where the user is active.
   When the user is interacting with the machine, `1Hz` samples are valuable.
   After the machine has been idle for a while, repeated samples usually add
   little information beyond "still idle".

3. Prefer recoverable data loss over constant disk churn.
   It is acceptable to lose the last few seconds of samples on crash. It is not
   ideal for an always-on collector to force a write flush every second.

4. Keep raw data simple and append-only.
   Battery changes should not require changing the CSV row shape or future
   SQLite raw sample model. Adaptive behavior should affect when samples are
   taken and flushed, not what a sample means.

## Idle-Aware Sampling

The collector should adjust sampling rate based on observed idle time.

Suggested default policy:

```text
active or idle <= 60s:       1 sample / second
idle > 60s and <= 10m:       1 sample / 5 seconds
idle > 10m:                  1 sample / 30 seconds
idle > 60m:                  1 sample / 60 seconds
```

When user activity returns, the collector should resume the active rate as soon
as the next sample observes a low idle value.

This means return-from-idle detection can be delayed by the current idle sample
interval. With a maximum interval of 60 seconds, a long idle span may end up to
60 seconds late in raw data. That is probably acceptable for a personal usage
logger, but the interval should be configurable for users who want sharper
boundaries.

### Optional Event-Assisted Wakeups

Some platforms may be able to notify the collector when user activity resumes.
If available, this would let WorkMuch sample very slowly during long idle spans
while still resuming quickly.

This should be treated as an optional optimization. The portable baseline should
remain polling with adaptive intervals.

## Write Batching

The current Go collector calls `Flush()` after every row. This is simple and
reduces data loss, but it creates unnecessary repeated I/O work.

Suggested write policy:

```text
flush every 30 seconds
flush on shutdown
flush after N rows, for example 128
flush immediately in --qa-console mode
```

The writer does not need to call `fsync` for normal operation. The OS can manage
physical persistence. WorkMuch only needs to avoid losing too much buffered data
inside the process if it crashes.

For future SQLite storage, the same idea applies:

```text
collect samples into short batches
insert samples in one transaction
commit every 10-60 seconds or N rows
```

SQLite should use WAL mode for normal operation once direct SQLite logging is
implemented. The existing v2 schema already recommends WAL and a busy timeout.

## OS Abstraction Changes

Today the backend abstraction is centered on `Sample(ctx)`. That is enough for
fixed-rate polling, but battery-friendly behavior needs more information about
cost, precision, and optional OS capabilities.

The collector should own the sampling policy, but the backend/OS layer should
expose enough hints to make good decisions.

### Backend Capability Hints

Add a capability method to the backend interface:

```go
type Capabilities struct {
    Native              bool
    ExpensiveSample     bool
    SupportsIdleSeconds bool
    SupportsWindowTitle bool
    SupportsProgramName bool
    SupportsWakeEvents  bool
}

type Backend interface {
    Name() string
    Capabilities() Capabilities
    Sample(ctx context.Context) (UsageSample, error)
    Reset() error
    Close() error
}
```

Example interpretation:

- `macos-native`: `Native=true`, `ExpensiveSample=false`
- `macos-subprocess`: `Native=false`, `ExpensiveSample=true`
- Linux X11 backend: direct X11 calls through the pure-Go X11 client

The collector can use `ExpensiveSample` to lower the default sample rate or emit
a warning when a user chooses a battery-costly backend.

### Power State Provider

Add an OS-level provider for power state. This should be separate from the
activity backend because battery/AC status is not the same concept as focused
window sampling.

```go
type PowerState struct {
    OnBattery       bool
    LowPowerMode    bool
    BatteryPercent  *int
}

type PowerProvider interface {
    CurrentPowerState(ctx context.Context) (PowerState, error)
}
```

On macOS this can eventually be implemented with IOKit or another native power
API. The first implementation can be conservative and return unknown values.

The sampling policy can then support profiles such as:

```text
AC power:       normal adaptive sampling
battery:        more aggressive idle slowdowns
low power mode: longest allowed intervals
```

### Sampling Policy Object

Move interval selection into a small policy object that can be unit tested
without OS APIs.

```go
type SamplingPolicy struct {
    ActiveInterval       time.Duration
    ShortIdleThreshold   time.Duration
    ShortIdleInterval    time.Duration
    LongIdleThreshold    time.Duration
    LongIdleInterval     time.Duration
    VeryLongIdleThreshold time.Duration
    VeryLongIdleInterval  time.Duration
}

func (p SamplingPolicy) NextInterval(sample backend.UsageSample, power PowerState) time.Duration
```

This keeps the runner loop simple:

1. sample backend
2. write sample
3. ask policy for next interval
4. sleep with context

### Optional Wake Event Source

Later, platforms can expose a wake/activity event channel:

```go
type ActivityWakeProvider interface {
    ActivityWakeEvents(ctx context.Context) (<-chan struct{}, error)
}
```

The runner could then sleep until either:

- the adaptive timer fires
- an activity event arrives
- the context is canceled

This should not be required for the first battery-friendly implementation.

## Configuration

Add user-facing flags only after the policy is working internally. Suggested
flags:

```text
--battery-friendly
--flush-interval <duration>
--active-rate <float>
--idle-rate <float>
--long-idle-rate <float>
--max-idle-interval <duration>
```

Keep defaults simple. A single `--battery-friendly` flag may be enough at first.

Possible default profiles:

```text
normal:
  active interval: 1s
  idle > 60s: 5s
  idle > 10m: 30s
  idle > 60m: 60s
  flush interval: 30s

battery-friendly:
  active interval: 2s
  idle > 60s: 10s
  idle > 10m: 60s
  idle > 60m: 120s
  flush interval: 60s
```

## Data Quality Impact

Adaptive idle sampling changes the density of raw samples, not their meaning.
Reports should calculate spans using observed timestamps rather than assuming a
fixed sample period.

Potential effects:

- Idle span end time may be delayed by up to the current idle interval.
- Very short app switches during sparse idle sampling may be missed.
- Active periods remain high fidelity if the active interval stays near `1Hz`.

For WorkMuch's purpose, this is likely a good tradeoff. The most important
high-resolution data is what happens while the user is active.

## Implementation Plan

1. Add unit tests for a pure sampling policy.
2. Implement adaptive intervals in the Go runner.
3. Add buffered CSV flushing with shutdown flush.
4. Add backend capability hints.
5. Make `macos-subprocess` report itself as expensive and warn or lower rate.
6. Add a placeholder power provider with unknown state.
7. Add macOS power-state support later.
8. Consider optional activity wake events only after the polling version is
   working well.

This keeps the first useful change small: adaptive idle sampling and batched
writes can be implemented without changing the storage format or requiring new
macOS permissions.
