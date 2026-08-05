#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)

export GOCACHE="${GOCACHE:-$SCRIPT_DIR/.gocache}"
mkdir -p "$GOCACHE"

BIN_DIR="$SCRIPT_DIR/bin"
BIN_PATH="$BIN_DIR/workmuch"
mkdir -p "$BIN_DIR"

echo "[Building...]"
cd "$SCRIPT_DIR"
go build -o "$BIN_PATH" ./cmd/workmuch-go

echo "Output: $BIN_PATH"
