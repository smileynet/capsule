#!/usr/bin/env bash
# setup-template.sh — Create a fresh test environment from a template.
#
# Usage: setup-template.sh [--template=NAME] [TARGET_DIR]
#   --template=NAME: template to use (default: demo-brownfield)
#   TARGET_DIR: optional directory to create the project in (default: mktemp -d)
#
# Prints the project directory path to stdout on success.
# Exits non-zero with an error message on any failure.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# --- Parse arguments ---
TEMPLATE_NAME="demo-brownfield"
TARGET_DIR=""

for arg in "$@"; do
    case "$arg" in
        --template=*)
            TEMPLATE_NAME="${arg#--template=}"
            ;;
        *)
            TARGET_DIR="$arg"
            ;;
    esac
done

TEMPLATE_DIR="$REPO_ROOT/templates/$TEMPLATE_NAME"

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
if [ -n "$TARGET_DIR" ]; then
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
cp "$TEMPLATE_DIR/AGENTS.md" "$TARGET_DIR/AGENTS.md"
[ -d "$TEMPLATE_DIR/src" ] && cp -r "$TEMPLATE_DIR/src" "$TARGET_DIR/src"
[ -f "$TEMPLATE_DIR/README.md" ] && cp "$TEMPLATE_DIR/README.md" "$TARGET_DIR/README.md"

# --- Initialize beads and import fixtures ---
(
    cd "$TARGET_DIR"

    BD_INIT_OUT=$(bd init --prefix=demo 2>&1) || {
        echo "WARNING: bd init failed (non-fatal): $BD_INIT_OUT" >&2
    }

    BD_IMPORT_OUT=$(bd import -i "$TEMPLATE_DIR/issues.jsonl" 2>&1) || {
        echo "WARNING: bd import failed (non-fatal): $BD_IMPORT_OUT" >&2
    }

    # Commit everything
    git add -A
    git commit -q -m "Add template project and bead fixtures"
)

# --- Output the project directory ---
echo "$TARGET_DIR"
