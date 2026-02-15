#!/usr/bin/env bash
# Smoke test for prep.sh — happy-path only.
# Validates: worktree created, branch named correctly, worklog.md present with content.
# Target: under 10 seconds.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
SETUP_SCRIPT="$REPO_ROOT/scripts/setup-template.sh"
PREP_SCRIPT="$REPO_ROOT/scripts/prep.sh"

PASS=0
FAIL=0

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); echo "  FAIL: $1"; }

# --- Prerequisite checks ---
for cmd in git bd jq; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "ERROR: $cmd is required but not installed" >&2
        exit 1
    fi
done

for script in "$SETUP_SCRIPT" "$PREP_SCRIPT"; do
    if [ ! -x "$script" ]; then
        echo "ERROR: $(basename "$script") not found or not executable at $script" >&2
        exit 1
    fi
done

echo "=== Smoke: prep.sh ==="
echo ""

# --- Create test environment ---
# Given: setup-template.sh creates a valid project
# When: setting up the test environment
# Then: a project directory with beads is ready
PROJECT_DIR=$("$SETUP_SCRIPT" 2>/dev/null) || {
    echo "FAIL: setup-template.sh exited non-zero" >&2
    exit 1
}
trap 'rm -rf "$PROJECT_DIR"' EXIT
echo "  Test project: $PROJECT_DIR"

BEAD_ID="demo-1.1.1"

# --- Run prep.sh ---
echo "[1/4] Prep completes successfully"
# Given: a valid project with beads
# When: prep.sh is run with a valid bead ID
# Then: exits zero
PREP_OUTPUT=$("$PREP_SCRIPT" "$BEAD_ID" --project-dir="$PROJECT_DIR" 2>&1) || {
    fail "prep.sh exited non-zero"
    echo "  Output: $PREP_OUTPUT"
    echo ""
    echo "==========================================="
    echo "RESULTS: $PASS passed, $FAIL failed"
    echo "==========================================="
    exit 1
}
pass "prep.sh exited zero"

WORKTREE_DIR="$PROJECT_DIR/.capsule/worktrees/$BEAD_ID"

echo "[2/4] Worktree directory exists"
# Given: prep.sh has run
# When: checking for the worktree directory
# Then: .capsule/worktrees/<bead-id>/ exists
if [ -d "$WORKTREE_DIR" ]; then
    pass "worktree at .capsule/worktrees/$BEAD_ID"
else
    fail "worktree directory not found"
fi

echo "[3/4] Branch named correctly"
# Given: a worktree created by prep.sh
# When: checking the git branch
# Then: branch is capsule-<bead-id>
BRANCH=$(cd "$WORKTREE_DIR" && git branch --show-current 2>/dev/null || echo "")
EXPECTED="capsule-$BEAD_ID"
if [ "$BRANCH" = "$EXPECTED" ]; then
    pass "branch: $EXPECTED"
else
    fail "expected branch $EXPECTED, got '$BRANCH'"
fi

echo "[4/4] Worklog present with interpolated content"
# Given: a worktree created by prep.sh
# When: checking worklog.md
# Then: file exists, contains bead-specific content, no leftover placeholders
if [ ! -f "$WORKTREE_DIR/worklog.md" ]; then
    fail "worklog.md not found"
else
    WORKLOG=$(cat "$WORKTREE_DIR/worklog.md")
    CONTENT_OK=true
    if ! echo "$WORKLOG" | grep -qF "Validate email format"; then
        fail "worklog missing task title"
        CONTENT_OK=false
    fi
    LEFTOVER=$(echo "$WORKLOG" | grep -c '{{' || true)
    if [ "$LEFTOVER" -gt 0 ]; then
        fail "worklog has $LEFTOVER lines with leftover {{ placeholders"
        CONTENT_OK=false
    fi
    if [ "$CONTENT_OK" = true ]; then
        pass "worklog.md has task content, no leftover placeholders"
    fi
fi

echo ""
echo "==========================================="
echo "RESULTS: $PASS passed, $FAIL failed"
echo "==========================================="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
