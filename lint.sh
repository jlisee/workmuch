#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/go"
export GOCACHE="${GOCACHE:-$(pwd)/../.gocache}"
mkdir -p "$GOCACHE"

unformatted="$(gofmt -l .)"
if [[ -n "$unformatted" ]]; then
  echo "Go files need formatting:" >&2
  echo "$unformatted" >&2
  exit 1
fi

go vet ./...
go test ./...
