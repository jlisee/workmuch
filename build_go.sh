#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &>/dev/null && pwd)
GO_DIR="${SCRIPT_DIR}/go"

export GOCACHE="${GOCACHE:-$(pwd)/../.gocache}"
mkdir -p "$GOCACHE"

BIN_DIR="$GO_DIR/bin"
BIN_PATH="$BIN_DIR/workmuch"
mkdir -p "$BIN_DIR"

echo "[Building...]"
cd "$GO_DIR"
go build -o "$BIN_PATH" ./cmd/workmuch-go

echo "Output: $BIN_PATH"
