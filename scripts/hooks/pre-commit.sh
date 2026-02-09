#!/bin/bash
# Pre-commit hook: lint + test before allowing commit
# Called by beads hook chaining (chain_strategy: before)
set -e

echo "Running golangci-lint..."
golangci-lint run ./...

echo "Running tests..."
go test ./...
