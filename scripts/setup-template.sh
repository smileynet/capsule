#!/usr/bin/env bash
# setup-template.sh — Create a fresh test environment from the demo-capsule template.
#
# Usage: setup-template.sh [TARGET_DIR]
#   TARGET_DIR: optional directory to create the project in (default: mktemp -d)
#
# Prints the project directory path to stdout on success.
# Exits non-zero with an error message on any failure.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TEMPLATE_DIR="$REPO_ROOT/templates/demo-capsule"

# --- Prerequisite checks ---
for cmd in git bd; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "ERROR: $cmd is required but not installed" >&2
        exit 1
    fi
done

if [ ! -d "$TEMPLATE_DIR" ]; then
    echo "ERROR: template directory not found: $TEMPLATE_DIR" >&2
    exit 1
fi

# --- Create target directory ---
if [ $# -ge 1 ]; then
    TARGET_DIR="$1"
    if [ -d "$TARGET_DIR/.git" ]; then
        echo "ERROR: $TARGET_DIR already contains a git repository" >&2
        exit 1
    fi
    mkdir -p "$TARGET_DIR"
else
    TARGET_DIR=$(mktemp -d)
fi

# --- Initialize git repo ---
(
    cd "$TARGET_DIR"
    git init -q
    git config user.email "capsule-test@example.com"
    git config user.name "Capsule Test"
    git commit --allow-empty -q -m "Initial commit"
)

# --- Copy template files ---
cp -r "$TEMPLATE_DIR/src" "$TARGET_DIR/src"
cp "$TEMPLATE_DIR/CLAUDE.md" "$TARGET_DIR/CLAUDE.md"
cp "$TEMPLATE_DIR/README.md" "$TARGET_DIR/README.md"

# --- Initialize beads and import fixtures ---
(
    cd "$TARGET_DIR"

    bd init --prefix=demo >/dev/null 2>&1

    bd import -i "$TEMPLATE_DIR/issues.jsonl" >/dev/null 2>&1

    # Commit everything
    git add -A
    git commit -q -m "Add template project and bead fixtures"
)

# --- Output the project directory ---
echo "$TARGET_DIR"
