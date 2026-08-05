# Python Retirement Plan — Completed Record

This document began as the Python Retirement Plan. The plan was completed on
2026-08-05; it is retained as a short record of the migration rather than as
current implementation guidance.

## Result

- The Go module, command, and internal packages now live at the repository
  root, with `workmuch-go` retained as the module path.
- `build.sh`, `run.sh`, `test.sh`, and `lint.sh` are the supported repository
  entrypoints.
- The retired Python application, its test suite, project metadata, and the
  macOS comparison benchmark were removed.
- Documentation now describes the Go collector only. The Go
  `macos-subprocess` backend remains an independent macOS fallback.

## Completed commits

1. `447b02a` — moved the Go module to the repository root.
2. `ff8c731` — made the Go shell entrypoints primary.
3. `87a3507` — removed the retired Python collector and tooling.
4. This documentation cleanup completes the editorial portion of the plan.

## Validation

The automated retirement checks are:

```bash
./test.sh
./lint.sh
```

Platform-dependent manual checks still need an appropriate GUI session:

```bash
./run.sh doctor
DISPLAY=:invalid ./run.sh doctor
./run.sh --qa-console
./run.sh --no-tray
```

On Linux, also confirm X11 activity and idle sampling. On macOS, confirm native
sampling and the Go `macos-subprocess` fallback where it is needed.
