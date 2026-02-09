#!/usr/bin/env bash
# Pre-commit hook: lint + test before allowing commit
# Called by beads hook chaining (chain_strategy: before)
set -euo pipefail

cd "$(git rev-parse --show-toplevel)" || exit 1

echo "Running golangci-lint..."
golangci-lint run ./...

echo "Running tests..."
go test ./...
