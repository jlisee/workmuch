#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
cd "$SCRIPT_DIR"
export GOCACHE="${GOCACHE:-$SCRIPT_DIR/.gocache}"
mkdir -p "$GOCACHE"
go test ./... "$@"
