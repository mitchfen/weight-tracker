#!/usr/bin/env bash
set -e

# Get the absolute path to the root directory
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." >/dev/null 2>&1 && pwd)"
cd "$DIR" || exit 1

echo "Downloading dependencies..."
go mod download

echo "Running go vet..."
go vet ./src/...

echo "Building project..."
go build -v -o server ./src

echo "Running tests..."
./scripts/test.sh
