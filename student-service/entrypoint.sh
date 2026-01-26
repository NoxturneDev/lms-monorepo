#!/bin/sh
set -e

# 1. Force Go to use go.mod (ignore any go.work files)
export GOWORK=off

echo "1. Resolving dependencies..."
# This is the magic command that fixes "missing go.sum entry"
go mod tidy

echo "2. Starting Air (Hot Reload)..."
# We use exec so Air becomes the main process (PID 1)
exec air -c .air.toml
