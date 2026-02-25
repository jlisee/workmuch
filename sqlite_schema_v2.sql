-- WorkMuch v2 schema for SQLite
-- Goals:
-- 1) Append-only raw samples
-- 2) Incremental "process once" span rollups
-- 3) Multi-host support
-- 4) Subsecond timestamps (stored as Unix epoch milliseconds)

PRAGMA foreign_keys = ON;

-- Optional runtime settings (apply at connection open time in app code):
-- PRAGMA journal_mode = WAL;
-- PRAGMA synchronous = FULL;
-- PRAGMA busy_timeout = 5000;

BEGIN;

CREATE TABLE IF NOT EXISTS raw_samples (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    host TEXT NOT NULL,
    observed_at_ms INTEGER NOT NULL CHECK (observed_at_ms >= 0),
    program_name TEXT NOT NULL DEFAULT '',
    window_title TEXT NOT NULL DEFAULT '',
    idle_ms INTEGER NOT NULL CHECK (idle_ms >= 0),
    ingest_batch_id TEXT,
    created_at_ms INTEGER NOT NULL DEFAULT (
        CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER)
    )
);

CREATE INDEX IF NOT EXISTS idx_raw_samples_host_time
    ON raw_samples(host, observed_at_ms, id);

CREATE INDEX IF NOT EXISTS idx_raw_samples_host_id
    ON raw_samples(host, id);

-- Optional de-dupe guard if your collector can retry inserts.
-- Keep disabled by default to avoid dropping legitimate duplicate observations.
-- CREATE UNIQUE INDEX IF NOT EXISTS idx_raw_samples_dedupe
--     ON raw_samples(host, observed_at_ms, program_name, window_title, idle_ms);

CREATE TABLE IF NOT EXISTS activity_spans (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    host TEXT NOT NULL,
    state TEXT NOT NULL CHECK (state IN ('active', 'inactive', 'unknown')),
    program_name TEXT NOT NULL DEFAULT '',
    window_title TEXT NOT NULL DEFAULT '',
    start_sample_id INTEGER NOT NULL REFERENCES raw_samples(id),
    end_sample_id INTEGER NOT NULL REFERENCES raw_samples(id),
    start_at_ms INTEGER NOT NULL CHECK (start_at_ms >= 0),
    end_at_ms INTEGER NOT NULL CHECK (end_at_ms >= start_at_ms),
    duration_ms INTEGER NOT NULL CHECK (duration_ms >= 0),
    sample_count INTEGER NOT NULL CHECK (sample_count > 0),
    algo_version TEXT NOT NULL DEFAULT 'v1',
    created_at_ms INTEGER NOT NULL DEFAULT (
        CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER)
    ),
    updated_at_ms INTEGER NOT NULL DEFAULT (
        CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER)
    )
);

-- Guarantees each sample starts at most one span (idempotent rollup).
CREATE UNIQUE INDEX IF NOT EXISTS idx_activity_spans_start_sample
    ON activity_spans(start_sample_id);

CREATE INDEX IF NOT EXISTS idx_activity_spans_host_time
    ON activity_spans(host, start_at_ms, end_at_ms);

CREATE INDEX IF NOT EXISTS idx_activity_spans_host_state_time
    ON activity_spans(host, state, start_at_ms);

-- Per-host incremental rollup state.
-- The processor reads raw_samples where id > last_processed_sample_id for a host.
CREATE TABLE IF NOT EXISTS span_processor_state (
    host TEXT PRIMARY KEY,
    last_processed_sample_id INTEGER NOT NULL DEFAULT 0,
    open_span_id INTEGER REFERENCES activity_spans(id),
    open_state TEXT CHECK (open_state IN ('active', 'inactive', 'unknown')),
    open_program_name TEXT,
    open_window_title TEXT,
    open_start_at_ms INTEGER CHECK (open_start_at_ms >= 0),
    open_last_at_ms INTEGER CHECK (open_last_at_ms >= 0),
    open_sample_count INTEGER NOT NULL DEFAULT 0 CHECK (open_sample_count >= 0),
    updated_at_ms INTEGER NOT NULL DEFAULT (
        CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER)
    )
);

-- Helpful views for debugging and analytics.
CREATE VIEW IF NOT EXISTS v_raw_samples AS
SELECT
    id,
    host,
    observed_at_ms,
    datetime(observed_at_ms / 1000.0, 'unixepoch') AS observed_at_utc,
    program_name,
    window_title,
    idle_ms
FROM raw_samples;

CREATE VIEW IF NOT EXISTS v_activity_spans AS
SELECT
    id,
    host,
    state,
    program_name,
    window_title,
    start_sample_id,
    end_sample_id,
    start_at_ms,
    datetime(start_at_ms / 1000.0, 'unixepoch') AS start_at_utc,
    end_at_ms,
    datetime(end_at_ms / 1000.0, 'unixepoch') AS end_at_utc,
    duration_ms,
    sample_count,
    algo_version
FROM activity_spans;

COMMIT;
