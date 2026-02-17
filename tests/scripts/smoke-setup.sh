#!/usr/bin/env bash
# Smoke test for setup-template.sh — happy-path only.
# Validates: creates project dir, git repo initialized, beads imported, template files present.
# Target: under 10 seconds.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
SETUP_SCRIPT="$REPO_ROOT/scripts/setup-template.sh"

PASS=0
FAIL=0

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); echo "  FAIL: $1"; }

# --- Prerequisite checks ---
for cmd in git bd; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "ERROR: $cmd is required but not installed" >&2
        exit 1
    fi
done

if [ ! -x "$SETUP_SCRIPT" ]; then
    echo "ERROR: setup-template.sh not found or not executable at $SETUP_SCRIPT" >&2
    exit 1
fi

echo "=== Smoke: setup-template.sh ==="
echo ""

# --- Run setup-template.sh ---
# Given: setup-template.sh exists and prerequisites are met
# When: invoked with no arguments
# Then: creates a valid project directory
PROJECT_DIR=$("$SETUP_SCRIPT" 2>/dev/null) || {
    echo "FAIL: setup-template.sh exited non-zero" >&2
    exit 1
}
trap 'rm -rf "$PROJECT_DIR"' EXIT

echo "[1/4] Project directory created"
if [ -d "$PROJECT_DIR" ]; then
    pass "directory exists: $PROJECT_DIR"
else
    fail "directory not created"
fi

echo "[2/4] Git repository initialized"
# Given: setup-template.sh has run
# When: checking for git commits
# Then: at least two commits exist (initial + template)
COMMIT_COUNT=$(cd "$PROJECT_DIR" && git log --oneline 2>/dev/null | wc -l)
if [ "$COMMIT_COUNT" -ge 2 ]; then
    pass "git log shows $COMMIT_COUNT commits"
else
    fail "expected at least 2 commits, got $COMMIT_COUNT"
fi

echo "[3/4] Beads imported"
# Given: setup-template.sh has run
# When: running bd ready in the project
# Then: task beads are listed
READY_OUTPUT=$(cd "$PROJECT_DIR" && bd ready 2>&1)
TASK_COUNT=$(echo "$READY_OUTPUT" | grep -c "demo-" || true)
if [ "$TASK_COUNT" -ge 1 ]; then
    pass "bd ready lists $TASK_COUNT bead(s)"
else
    fail "bd ready shows no beads"
    echo "  Output: $READY_OUTPUT"
fi

echo "[4/4] Template files present"
# Given: setup-template.sh has run
# When: checking for expected template files
# Then: AGENTS.md and src/ directory exist
FILES_OK=true
for f in AGENTS.md src/main.go; do
    if [ ! -f "$PROJECT_DIR/$f" ]; then
        fail "missing: $f"
        FILES_OK=false
    fi
done
if [ "$FILES_OK" = true ]; then
    pass "template files present (AGENTS.md, src/main.go)"
fi

echo ""
echo "==========================================="
echo "RESULTS: $PASS passed, $FAIL failed"
echo "==========================================="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
