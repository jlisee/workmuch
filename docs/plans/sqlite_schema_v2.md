# WorkMuch SQLite Schema v2

This document explains the repository-root `sqlite_schema_v2.sql` schema and
how to use it for:

- durable raw activity logging
- incremental (process-once) conversion into activity spans
- multi-host aggregation
- subsecond timestamp precision

## Design goals

1. Store every raw sample once in an append-only table.
2. Process raw samples incrementally so each sample is handled once in normal operation.
3. Support merging data from multiple machines/locations using `host`.
4. Preserve subsecond timing by storing timestamps as Unix epoch milliseconds (`*_ms` integer fields).

## Timestamp format

All primary timestamps are stored as `INTEGER` milliseconds since Unix epoch UTC.

- `1739700000123` means `2025-02-16T...Z` with millisecond precision.
- Human-readable UTC values are exposed through views using SQLite `datetime(..., 'unixepoch')`.

Use UTC for all ingest/processing and convert to local time only in UI/reporting.

## Tables

## `raw_samples`

Append-only source data from the collector.

Key columns:

- `id`: autoincrement primary key, monotonic ingest order
- `host`: machine/location identifier (for example `mbp-home`, `work-laptop`)
- `observed_at_ms`: event timestamp (ms)
- `program_name`: focused app/program
- `window_title`: focused window title
- `idle_ms`: idle duration at sample time
- `ingest_batch_id`: optional ingest metadata
- `created_at_ms`: insert time (ms)

Important indexes:

- `(host, observed_at_ms, id)` for time-based queries
- `(host, id)` for incremental per-host processing

## `activity_spans`

Derived contiguous stretches of similar activity.

A span usually changes when one of these changes:

- active/inactive state
- program name
- window title
- large sample gap

Key columns:

- `host`
- `state` (`active`, `inactive`, `unknown`)
- `program_name`, `window_title`
- `start_sample_id`, `end_sample_id`: traceability back to raw samples
- `start_at_ms`, `end_at_ms`, `duration_ms`
- `sample_count`
- `algo_version`: span builder version
- `created_at_ms`, `updated_at_ms`

Important index/constraint:

- unique index on `start_sample_id` to support idempotent span creation

## `span_processor_state`

Per-host checkpoint + open span state for process-once incremental rollups.

Key columns:

- `host` (primary key)
- `last_processed_sample_id`
- `open_span_id`
- `open_state`, `open_program_name`, `open_window_title`
- `open_start_at_ms`, `open_last_at_ms`, `open_sample_count`
- `updated_at_ms`

This table lets the worker resume after restart without rescanning old samples.

## Views

## `v_raw_samples`

Adds `observed_at_utc` (readable timestamp) for debugging and ad hoc queries.

## `v_activity_spans`

Adds `start_at_utc` / `end_at_utc` for reporting/debugging.

## Setup

Create a database and apply schema:

```bash
sqlite3 /path/to/workmuch.db ".read sqlite_schema_v2.sql"
```

Recommended connection pragmas in app code:

```sql
PRAGMA journal_mode = WAL;
PRAGMA synchronous = FULL;
PRAGMA foreign_keys = ON;
PRAGMA busy_timeout = 5000;
```

If you need slightly lower write latency and can tolerate small durability tradeoffs on power loss, use `synchronous = NORMAL`.

## Ingest raw samples

Example insert:

```sql
INSERT INTO raw_samples (
  host, observed_at_ms, program_name, window_title, idle_ms, ingest_batch_id
) VALUES (
  :host, :observed_at_ms, :program_name, :window_title, :idle_ms, :ingest_batch_id
);
```

## Process-once incremental rollup pattern

For each `host`:

1. Start a transaction.
2. Read `span_processor_state` for that host (create row if missing).
3. Fetch new samples:
   - `SELECT * FROM raw_samples WHERE host = ? AND id > ? ORDER BY id LIMIT ?`
4. For each sample:
   - derive `state` from `idle_ms` threshold
   - continue current open span or close/open a new span on boundary
5. Update `activity_spans` and `span_processor_state.last_processed_sample_id`.
6. Commit.

Because checkpoint updates and span writes are in one transaction, successful commits mean those samples are processed exactly once in normal operation.

## Typical boundary rules

Start a new span when any condition is true:

- state changes (`active` <-> `inactive`)
- `program_name` changes
- `window_title` changes
- `observed_at_ms` gap exceeds `max_gap_ms`

Optional smoothing rules:

- require N consecutive samples before state flip
- merge tiny spans under a threshold duration

When smoothing logic changes, bump `algo_version`.

## Query examples

Active time by host and program for a day:

```sql
SELECT
  host,
  program_name,
  SUM(duration_ms) / 1000.0 AS active_seconds
FROM activity_spans
WHERE state = 'active'
  AND start_at_ms >= :day_start_ms
  AND start_at_ms < :day_end_ms
GROUP BY host, program_name
ORDER BY active_seconds DESC;
```

Timeline view for one host:

```sql
SELECT
  start_at_utc,
  end_at_utc,
  state,
  program_name,
  window_title,
  duration_ms
FROM v_activity_spans
WHERE host = :host
ORDER BY start_at_ms;
```

Raw-to-span traceability:

```sql
SELECT
  s.id AS span_id,
  s.host,
  s.start_sample_id,
  s.end_sample_id,
  r1.observed_at_ms AS start_sample_time_ms,
  r2.observed_at_ms AS end_sample_time_ms
FROM activity_spans s
JOIN raw_samples r1 ON r1.id = s.start_sample_id
JOIN raw_samples r2 ON r2.id = s.end_sample_id
WHERE s.id = :span_id;
```

## Operational notes

- Keep `raw_samples` immutable; derive/rebuild spans from raw when needed.
- Run `PRAGMA integrity_check;` periodically.
- Back up with SQLite online backup API or file-level snapshots.
- If out-of-order samples are possible, either:
  - enforce ordered ingest per host, or
  - use a lateness/watermark strategy before finalizing spans.
