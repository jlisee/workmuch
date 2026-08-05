#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
cd "$SCRIPT_DIR"
export GOCACHE="${GOCACHE:-$SCRIPT_DIR/.gocache}"
mkdir -p "$GOCACHE"

unformatted="$(gofmt -l .)"
if [[ -n "$unformatted" ]]; then
  echo "Go files need formatting:" >&2
  echo "$unformatted" >&2
  exit 1
fi

go vet ./...
go test ./...
