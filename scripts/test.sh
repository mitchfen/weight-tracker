#!/usr/bin/env bash
# Get the absolute path to the root directory
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." >/dev/null 2>&1 && pwd)"

echo "Running Weight Tracker tests with code coverage..."
cd "$DIR" || exit 1
go test -v -cover ./...
