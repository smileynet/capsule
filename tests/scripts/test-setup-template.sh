#!/usr/bin/env bash
# Test script for t-1.1.3: Create setup-template.sh initialization script
# Validates: temp dir creation, git init, bd state, template files, error handling.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
SETUP_SCRIPT="$REPO_ROOT/scripts/setup-template.sh"

PASS=0
FAIL=0

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); echo "  FAIL: $1"; }

# Prerequisites: required tools installed
for cmd in git bd; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "ERROR: $cmd is required but not installed"
        exit 1
    fi
done

# Prerequisite: setup script exists
if [ ! -f "$SETUP_SCRIPT" ]; then
    echo "ERROR: scripts/setup-template.sh not found"
    exit 1
fi

if [ ! -x "$SETUP_SCRIPT" ]; then
    echo "ERROR: scripts/setup-template.sh is not executable"
    exit 1
fi

echo "=== t-1.1.3: setup-template.sh ==="
echo ""

# ---------- Test 1: Run with no arguments, creates project in temp dir ----------
echo "[1/5] Run with no arguments"
PROJECT_DIR=$("$SETUP_SCRIPT" 2>/dev/null)
if [ -n "$PROJECT_DIR" ] && [ -d "$PROJECT_DIR" ]; then
    pass "Script prints path to stdout and directory exists"
else
    fail "Script did not output a valid directory path (got: '$PROJECT_DIR')"
    echo "RESULTS: $PASS passed, $FAIL failed"
    exit 1
fi
# Cleanup on exit — track all created temp dirs
CLEANUP_DIRS=("$PROJECT_DIR")
cleanup() { for d in "${CLEANUP_DIRS[@]}"; do rm -rf "$d" 2>/dev/null; done; }
trap cleanup EXIT

# ---------- Test 2: Git repository initialized with at least one commit ----------
echo "[2/5] Git repository with initial commit"
COMMIT_COUNT=$(cd "$PROJECT_DIR" && git log --oneline 2>/dev/null | wc -l)
if [ "$COMMIT_COUNT" -ge 1 ]; then
    pass "git log shows $COMMIT_COUNT commit(s)"
else
    fail "git log shows no commits"
fi

# ---------- Test 3: bd ready lists the 2 task beads ----------
echo "[3/5] bd ready lists task beads"
READY_OUTPUT=$(cd "$PROJECT_DIR" && bd ready 2>&1)
TASK_COUNT=$(echo "$READY_OUTPUT" | grep -c "demo-1\.1\." || true)
if [ "$TASK_COUNT" -eq 2 ]; then
    pass "bd ready lists 2 task beads"
else
    fail "expected 2 task beads in bd ready, got $TASK_COUNT"
    echo "  Output: $READY_OUTPUT"
fi

# ---------- Test 4: Template files copied ----------
echo "[4/5] Template files present"
FILES_OK=true
for f in src/main.go src/go.mod CLAUDE.md README.md; do
    if [ ! -f "$PROJECT_DIR/$f" ]; then
        fail "missing: $f"
        FILES_OK=false
    fi
done
if [ "$FILES_OK" = true ]; then
    pass "src/main.go, src/go.mod, CLAUDE.md, README.md all present"
fi

# ---------- Test 5: Fails when bd is not installed ----------
echo "[5/5] Fails when bd not on PATH"
# Build PATH with bd's directory removed
BD_DIR="$(dirname "$(command -v bd)")"
NO_BD_PATH=$(echo "$PATH" | tr ':' '\n' | grep -Fxv "$BD_DIR" | tr '\n' ':' | sed 's/:$//')
if env PATH="$NO_BD_PATH" "$SETUP_SCRIPT" >/dev/null 2>&1; then
    fail "script should fail when bd is not installed"
else
    pass "script fails with non-zero exit when bd is missing"
fi

# ---------- Edge Cases ----------
echo ""
echo "=== Edge Cases ==="

# Idempotent-safe: fresh temp dir each time
echo "[E1] Fresh directory each run"
PROJECT_DIR2=$("$SETUP_SCRIPT" 2>/dev/null)
CLEANUP_DIRS+=("$PROJECT_DIR2")
if [ "$PROJECT_DIR" != "$PROJECT_DIR2" ] && [ -d "$PROJECT_DIR2" ]; then
    pass "each run creates a new directory"
else
    fail "directories are the same or second dir missing"
fi

# Works from any working directory
echo "[E2] Works from any directory"
OTHER_DIR=$(mktemp -d)
CLEANUP_DIRS+=("$OTHER_DIR")
PROJECT_DIR3=$(cd "$OTHER_DIR" && "$SETUP_SCRIPT" 2>/dev/null)
CLEANUP_DIRS+=("$PROJECT_DIR3")
if [ -n "$PROJECT_DIR3" ] && [ -d "$PROJECT_DIR3" ]; then
    pass "works when run from arbitrary directory"
else
    fail "failed when run from $OTHER_DIR"
fi

echo ""
echo "==========================================="
echo "RESULTS: $PASS passed, $FAIL failed"
echo "==========================================="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
