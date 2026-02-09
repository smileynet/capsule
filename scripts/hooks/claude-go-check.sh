#!/bin/bash
# Claude Code PostToolUse hook: format and check Go files after Write/Edit
# Receives tool input JSON on stdin

INPUT=$(cat)
FILE_PATH=$(echo "$INPUT" | jq -r '.tool_input.file_path // empty')

# Exit early for non-Go files or missing files
[[ "$FILE_PATH" == *.go ]] && [[ -f "$FILE_PATH" ]] || exit 0

# Get project root (where go.mod lives)
PROJECT_ROOT=$(cd "$(dirname "$FILE_PATH")" && go env GOMOD 2>/dev/null | xargs dirname 2>/dev/null)
[[ -n "$PROJECT_ROOT" ]] || exit 0

cd "$PROJECT_ROOT" || exit 0

# Format and fix imports
goimports -w "$FILE_PATH" 2>&1

# Build check — catches type errors, missing methods, undefined references
go build ./... 2>&1

# Vet check — catches shadowed vars, printf mismatches, unreachable code
go vet ./... 2>&1
