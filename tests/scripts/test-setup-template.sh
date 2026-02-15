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
echo "[1/7] Run with no arguments"
# Given: setup-template.sh exists and is executable
# When: invoked with no arguments
# Then: prints a valid directory path to stdout
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
echo "[2/7] Git repository with initial commit"
# Given: a project directory created by setup-template.sh
# When: checking for git log
# Then: at least one commit exists
COMMIT_COUNT=$(cd "$PROJECT_DIR" && git log --oneline 2>/dev/null | wc -l)
if [ "$COMMIT_COUNT" -ge 1 ]; then
    pass "git log shows $COMMIT_COUNT commit(s)"
else
    fail "git log shows no commits"
fi

# ---------- Test 3: bd ready lists the 2 task beads ----------
echo "[3/7] bd ready lists task beads"
# Given: a project directory created by setup-template.sh
# When: running bd ready in the project
# Then: 2 task beads are listed
READY_OUTPUT=$(cd "$PROJECT_DIR" && bd ready 2>&1)
TASK_COUNT=$(echo "$READY_OUTPUT" | grep -c "demo-1\.1\." || true)
if [ "$TASK_COUNT" -eq 2 ]; then
    pass "bd ready lists 2 task beads"
else
    fail "expected 2 task beads in bd ready, got $TASK_COUNT"
    echo "  Output: $READY_OUTPUT"
fi

# ---------- Test 4: bd show has title, description, acceptance criteria, parent ----------
echo "[4/7] bd show has full metadata"
# Given: a project directory with beads initialized
# When: running bd show on a task bead
# Then: output contains title, description, acceptance criteria, and parent reference
SHOW_OUTPUT=$(cd "$PROJECT_DIR" && bd show demo-1.1.1 2>&1)
SHOW_OK=true
for field in "Validate email format" "DESCRIPTION" "ACCEPTANCE CRITERIA" "ValidateEmail returns nil for valid emails" "demo-1.1"; do
    if ! echo "$SHOW_OUTPUT" | grep -q "$field"; then
        fail "bd show demo-1.1.1 missing expected content: $field"
        SHOW_OK=false
    fi
done
if [ "$SHOW_OK" = true ]; then
    pass "bd show has title, description, acceptance criteria, and parent reference"
fi

# ---------- Test 5: Deterministic state across runs ----------
echo "[5/7] Deterministic state"
# Given: two separate invocations of setup-template.sh
# When: comparing file trees and bd list output
# Then: both are identical
PROJECT_DIR_DET=$("$SETUP_SCRIPT" 2>/dev/null)
CLEANUP_DIRS+=("$PROJECT_DIR_DET")
FILES1=$(cd "$PROJECT_DIR" && find . -not -path './.git/*' -not -path './.beads/beads.db*' -not -name 'last-touched' | sort)
FILES2=$(cd "$PROJECT_DIR_DET" && find . -not -path './.git/*' -not -path './.beads/beads.db*' -not -name 'last-touched' | sort)
BD_LIST1=$(cd "$PROJECT_DIR" && bd list 2>&1)
BD_LIST2=$(cd "$PROJECT_DIR_DET" && bd list 2>&1)
if [ "$FILES1" = "$FILES2" ] && [ "$BD_LIST1" = "$BD_LIST2" ]; then
    pass "file tree and bd list are identical across runs"
else
    fail "state differs between runs"
    echo "  Files diff: $(diff <(echo "$FILES1") <(echo "$FILES2") || true)"
fi

# ---------- Test 6: Template files copied ----------
echo "[6/7] Template files present"
# Given: a project directory created by setup-template.sh
# When: checking for expected template files
# Then: src/main.go, src/go.mod, AGENTS.md, README.md all present
FILES_OK=true
for f in src/main.go src/go.mod AGENTS.md README.md; do
    if [ ! -f "$PROJECT_DIR/$f" ]; then
        fail "missing: $f"
        FILES_OK=false
    fi
done
if [ "$FILES_OK" = true ]; then
    pass "src/main.go, src/go.mod, AGENTS.md, README.md all present"
fi

# ---------- Test 7: Fails when bd is not installed ----------
echo "[7/7] Fails when bd not on PATH"
# Given: PATH modified to exclude bd
# When: setup-template.sh is invoked
# Then: exits with non-zero status
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
# Given: a previously created project directory
# When: setup-template.sh is invoked again
# Then: a new, different directory is created
PROJECT_DIR2=$("$SETUP_SCRIPT" 2>/dev/null)
CLEANUP_DIRS+=("$PROJECT_DIR2")
if [ "$PROJECT_DIR" != "$PROJECT_DIR2" ] && [ -d "$PROJECT_DIR2" ]; then
    pass "each run creates a new directory"
else
    fail "directories are the same or second dir missing"
fi

# Works from any working directory
echo "[E2] Works from any directory"
# Given: an arbitrary working directory
# When: setup-template.sh is invoked from that directory
# Then: a valid project directory is created
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
