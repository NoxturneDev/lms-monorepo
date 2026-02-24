#!/bin/sh
set -e

# 1. Force Go to use go.mod (ignore any go.work files)
export GOWORK=off

echo "1. Resolving dependencies..."
go mod tidy

echo "2. Creating tmp directory..."
mkdir -p ./tmp

echo "3. Pre-building binary to check for errors..."
if ! go build -o ./tmp/main .; then
    echo "ERROR: Failed to build. Showing errors:"
    go build -o ./tmp/main . 2>&1
    exit 1
fi

echo "4. Starting Air (Hot Reload)..."
# We use exec so Air becomes the main process (PID 1)
exec air -c .air.toml
