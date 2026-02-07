#!/usr/bin/env bash
# teardown.sh — Clean up capsule worktrees and output from a project.
#
# Usage: teardown.sh [--project-dir=DIR]
#   --project-dir: Project root directory (default: current directory)
#
# Cleans:
#   .capsule/worktrees/*/  — removes all capsule worktrees (git worktree remove + prune)
#   .capsule/output/       — removes contents (but preserves the directory)
#
# Preserves:
#   .capsule/logs/         — archived worklogs are never deleted
#
# Exit codes:
#   0 - Cleanup completed (or nothing to clean)
set -euo pipefail

# --- Parse arguments ---
PROJECT_DIR="."

for arg in "$@"; do
    case "$arg" in
        --project-dir=*)
            PROJECT_DIR="${arg#--project-dir=}"
            ;;
        -*)
            echo "ERROR: Unknown option: $arg" >&2
            echo "Usage: teardown.sh [--project-dir=DIR]" >&2
            exit 1
            ;;
        *)
            echo "ERROR: Unexpected argument: $arg" >&2
            echo "Usage: teardown.sh [--project-dir=DIR]" >&2
            exit 1
            ;;
    esac
done

# Resolve project directory to absolute path
PROJECT_DIR="$(cd "$PROJECT_DIR" && pwd)"

WORKTREES_DIR="$PROJECT_DIR/.capsule/worktrees"
OUTPUT_DIR="$PROJECT_DIR/.capsule/output"

CLEANED_WORKTREES=0
CLEANED_OUTPUT=0

# --- Remove capsule worktrees ---
if [ -d "$WORKTREES_DIR" ]; then
    for wt in "$WORKTREES_DIR"/*/; do
        [ -d "$wt" ] || continue
        wt_name="$(basename "$wt")"

        # Try git worktree remove first, fall back to manual removal
        if ! (cd "$PROJECT_DIR" && git worktree remove "$wt" --force 2>/dev/null); then
            rm -rf "$wt"
        fi

        # Delete the capsule branch if it exists
        (cd "$PROJECT_DIR" && git branch -D "capsule-$wt_name" 2>/dev/null) || true

        CLEANED_WORKTREES=$((CLEANED_WORKTREES + 1))
        echo "Removed worktree: $wt_name"
    done

    # Prune stale worktree metadata
    (cd "$PROJECT_DIR" && git worktree prune 2>/dev/null) || true
fi

# --- Clean .capsule/output/ ---
if [ -d "$OUTPUT_DIR" ]; then
    FILE_COUNT=$(find "$OUTPUT_DIR" -type f 2>/dev/null | wc -l)
    if [ "$FILE_COUNT" -gt 0 ]; then
        rm -rf "$OUTPUT_DIR"/*
        CLEANED_OUTPUT=$FILE_COUNT
        echo "Cleaned output: $FILE_COUNT file(s)"
    fi
fi

# --- Report ---
if [ "$CLEANED_WORKTREES" -eq 0 ] && [ "$CLEANED_OUTPUT" -eq 0 ]; then
    echo "Nothing to clean."
else
    echo ""
    echo "Teardown complete:"
    if [ "$CLEANED_WORKTREES" -gt 0 ]; then
        echo "  Worktrees removed: $CLEANED_WORKTREES"
    fi
    if [ "$CLEANED_OUTPUT" -gt 0 ]; then
        echo "  Output files cleaned: $CLEANED_OUTPUT"
    fi
fi
