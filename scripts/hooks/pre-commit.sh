#!/usr/bin/env bash
# Pre-commit hook: incremental lint + test for staged Go packages
# Called by bd hook chaining (.git/hooks/pre-commit.old)
set -euo pipefail

cd "$(git rev-parse --show-toplevel)" || exit 1

# Detect staged Go files and exit early if none
staged=$(git diff --cached --name-only -- '*.go')
if [ -z "$staged" ]; then
  exit 0
fi

# Map staged files to affected packages (./dir/...)
packages=$(echo "$staged" \
  | while IFS= read -r f; do dirname "$f"; done \
  | sort -u \
  | sed 's|^\.$|.|; s|^\([^.]\)|./\1|; s|$|/...|')

echo "Running golangci-lint (incremental)..."
# shellcheck disable=SC2086
golangci-lint run --new-from-rev=HEAD $packages

echo "Running tests (affected packages, -short)..."
# shellcheck disable=SC2086
go test -short $packages
