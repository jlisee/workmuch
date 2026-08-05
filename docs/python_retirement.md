# Python Retirement Plan

WorkMuch now has Go implementations for Linux X11, native macOS sampling, and
the macOS subprocess fallback. The Python collector is no longer required by
the Go build or runtime, so this plan retires the parallel Python application
and makes Go the only supported implementation.

The retirement also removes the extra `go` directory. The Go module will move
to the repository root; the Go source itself will not be deleted. The existing
The Go behavior is now the primary `run.sh` and `test.sh` behavior. The
repository has no `_go.sh` wrappers.

Each step must remain green, receive its own review, and stop before the next
step begins. Use red/green TDD with `testify` for any behavior change. Pure file
moves do not require new tests, but the existing suite must pass before and
after each move.

## Target layout

The important repository paths should end in this shape:

```text
workmuch/
├── cmd/workmuch-go/
├── internal/
├── go.mod
├── go.sum
├── build.sh
├── run.sh
├── test.sh
├── lint.sh
├── docs/
├── assets/
├── README
└── sqlite_schema_v2.sql
```

The target has no `go/` directory, Python package, Python test tree, `uv`
metadata, or `_go.sh` wrappers.

## Step 1: Establish the retirement baseline

Before moving or deleting files:

- Confirm that Go remains the supported collector on Linux and macOS.
- Confirm that no external installation, automation, or user workflow still
  depends on `./run.sh`, the Python `workmuch` console script, or imports from
  the `workmuch` Python package.
- Run the current Go test and lint entrypoints.

```bash
./test.sh
./lint.sh
```

The Python test suite is not a retirement gate. It currently belongs to the
code being removed and does not need to be repaired solely to delete it.

## Step 2: Move the Go module to the repository root

Move the Go module without changing its module name or behavior:

- Move `go/go.mod` to `go.mod`.
- Move `go/go.sum` to `go.sum`.
- Move `go/cmd` to `cmd`.
- Move `go/internal` to `internal`.
- Preserve embedded tray assets under `internal/tray/assets`.
- Update `run.sh`, `test.sh`, and `lint.sh` so they run from the
  repository root and no longer `cd` into `go`.
- Update `.gitignore` entries that still assume build output is below `go/`.
- Remove the empty `go` directory.

Use history-preserving moves such as `git mv` during implementation. Keep the
module path `workmuch-go` unchanged so existing internal imports continue to
resolve.

Run:

```bash
./test.sh
./lint.sh
go test ./...
go vet ./...
```

This step is complete only when no source, test, script, or documentation path
requires the old `go/` directory.

## Step 3: Promote the Go shell entrypoints

Make the existing primary script names launch and test Go:

- Replace the Python contents of `run.sh` with the adjusted `run_go.sh`
  behavior.
- Replace the Python contents of `test.sh` with the adjusted `test_go.sh`
  behavior.
- Preserve executable permissions on both primary scripts.
- Remove `run_go.sh` and `test_go.sh` after the primary scripts work.
- Promote `build_go.sh` to `build.sh` and remove the final `_go.sh` wrapper.
- Update `AGENTS.md`, `README`, and all documents to use `./run.sh` and
  `./test.sh`.
- Keep `./lint.sh` as the formatting, vet, and full-test entrypoint.

Run:

```bash
./test.sh
./lint.sh
./run.sh --help
```

Then perform non-writing diagnostics:

```bash
./run.sh doctor
./run.sh --qa-console
```

Confirm that help, backend selection, tray selection, `doctor`,
`--qa-console`, and `--no-tray` have the same behavior as their former
`_go.sh` entrypoints.

## Step 4: Remove the Python application

Delete the retired Python-only implementation and tooling:

- Remove the `workmuch/` Python package.
- Remove the Python `tests/` directory.
- Remove `pyproject.toml` and `uv.lock`.
- Remove `scripts/benchmark_macos_backends.py`.
- Remove `scripts/` if it is empty afterward.
- Remove Python-only ignores such as `.venv/`, `__pycache__/`, `.pycache/`,
  and `workmuch.egg-info/` when they are no longer useful.
- Remove Python dependency, installation, backend-selection, benchmark, and
  test instructions from `README`.
- Rewrite historical Python comparisons in documentation when they imply that
  Python remains a supported collector.

Do not remove the Go `macos-subprocess` backend. It is an independent Go
fallback and does not require Python.

Search for stale references after deletion:

```bash
rg -n --hidden -g '!.git/**' \
  'python|pytest|pyobjc|python-xlib|uv |run_go\.sh|test_go\.sh|go/'
```

Review each match rather than requiring zero matches. Historical explanations
may still be useful, but commands and current-state documentation must describe
the Go-only repository.

## Step 5: Final validation

Run the complete automated checks from a clean checkout:

```bash
./test.sh
./lint.sh
```

Perform the supported-platform manual checks:

```bash
./run.sh doctor
DISPLAY=:invalid ./run.sh doctor
./run.sh --qa-console
./run.sh --no-tray
```

Also verify tray About, Status, and Quit on a GUI session. On Linux, confirm
X11 activity and idle sampling. On macOS, confirm native sampling and the Go
`macos-subprocess` fallback as applicable.

Finish with these repository checks:

- `git status` contains only the intended retirement changes.
- No tracked Python source, Python dependency lock, or `_go.sh` wrapper
  remains.
- No tracked path remains below `go/`.
- `README`, `AGENTS.md`, and docs show only valid commands and paths.

## Suggested commits

Keep the structural work reviewable with one commit per completed step:

1. `[layout] move Go module to repository root`
2. `[scripts] make Go the primary entrypoint`
3. `[python] remove retired Python collector`
4. `[docs] finish Go-only documentation cleanup`

The exact split can change if a move and its script update must be atomic, but
avoid combining unrelated collector behavior changes with the retirement.
