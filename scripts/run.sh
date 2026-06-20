#!/usr/bin/env bash
# Get the absolute path to the root directory
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." >/dev/null 2>&1 && pwd)"

# Set DB_PATH to the weights.db in the project root directory
export DB_PATH="$DIR/weights.db"

echo "Starting Weight Tracker in development mode..."
echo "Database: $DB_PATH"
echo "Server:   http://localhost:8080"
echo "Press Ctrl+C to stop"

# Run the server from the src directory so that templates/static assets are resolved correctly
cd "$DIR/src" || exit 1
go run main.go
