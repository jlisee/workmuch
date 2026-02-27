#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/go"
export GOCACHE="${GOCACHE:-$(pwd)/../.gocache}"
mkdir -p "$GOCACHE"
go run ./cmd/workmuch-go "$@"
