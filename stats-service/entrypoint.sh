#!/bin/sh
set -e

export GOWORK=off

echo "1. Resolving dependencies..."
go mod tidy

echo "2. Starting Air (Hot Reload)..."
exec air -c .air.toml
