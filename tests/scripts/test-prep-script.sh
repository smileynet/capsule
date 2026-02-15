#!/usr/bin/env bash
# Test script for t-1.2.2: Create prep.sh script for worktree + worklog setup
# Validates: worktree creation, branch naming, worklog instantiation, error handling.
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
for cmd in git bd; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "ERROR: $cmd is required but not installed" >&2
        exit 1
    fi
done

if [ ! -f "$SETUP_SCRIPT" ]; then
    echo "ERROR: setup-template.sh not found at $SETUP_SCRIPT" >&2
    exit 1
fi

if [ ! -f "$PREP_SCRIPT" ]; then
    echo "ERROR: prep.sh not found at $PREP_SCRIPT" >&2
    exit 1
fi

# --- Create test environment ---
echo "=== Setting up test environment ==="
PROJECT_DIR=$("$SETUP_SCRIPT")
trap 'rm -rf "$PROJECT_DIR"' EXIT
echo "  Test project: $PROJECT_DIR"

BEAD_ID="demo-1.1.1"

echo ""
echo "=== t-1.2.2: prep.sh ==="
echo ""

# ---------- Test 1: Happy path - worktree created, worklog present ----------
echo "[1/6] Happy path: worktree created with worklog"
# Given: a valid template project and bead ID
# When: prep.sh is run with a valid bead
# Then: worktree directory and worklog.md are created
PREP_OUTPUT=$("$PREP_SCRIPT" "$BEAD_ID" --project-dir="$PROJECT_DIR" 2>&1) || {
    fail "prep.sh exited non-zero for valid bead"
    echo "  Output: $PREP_OUTPUT"
}

WORKTREE_DIR="$PROJECT_DIR/.capsule/worktrees/$BEAD_ID"
if [ -d "$WORKTREE_DIR" ]; then
    pass "Worktree directory created at .capsule/worktrees/$BEAD_ID"
else
    fail "Worktree directory not found at $WORKTREE_DIR"
fi

if [ -f "$WORKTREE_DIR/worklog.md" ]; then
    pass "worklog.md present in worktree"
else
    fail "worklog.md not found in worktree"
fi

# ---------- Test 2: Branch naming convention ----------
echo "[2/6] Worktree branch naming"
# Given: a worktree created by prep.sh
# When: checking the git branch name
# Then: branch is named capsule-<bead-id>
BRANCH=$(cd "$WORKTREE_DIR" && git branch --show-current 2>/dev/null || echo "")
EXPECTED_BRANCH="capsule-$BEAD_ID"
if [ "$BRANCH" = "$EXPECTED_BRANCH" ]; then
    pass "Branch named $EXPECTED_BRANCH"
else
    fail "Expected branch $EXPECTED_BRANCH, got '$BRANCH'"
fi

# ---------- Test 3: Worklog content has bead metadata ----------
echo "[3/6] Worklog content interpolated with bead metadata"
# Given: a worktree with rendered worklog.md
# When: checking worklog content for bead metadata
# Then: contains epic, feature, task titles and no leftover placeholders
WORKLOG=$(cat "$WORKTREE_DIR/worklog.md" 2>/dev/null || echo "")

CONTENT_OK=true
# Task title from demo-1.1.1
if ! echo "$WORKLOG" | grep -qF "Validate email format"; then
    fail "Worklog missing task title 'Validate email format'"
    CONTENT_OK=false
fi
# Feature title from demo-1.1
if ! echo "$WORKLOG" | grep -qF "Add input validation"; then
    fail "Worklog missing feature title 'Add input validation'"
    CONTENT_OK=false
fi
# Epic title from demo-1
if ! echo "$WORKLOG" | grep -qF "Demo Capsule Feature Set"; then
    fail "Worklog missing epic title 'Demo Capsule Feature Set'"
    CONTENT_OK=false
fi
# Task ID
if ! echo "$WORKLOG" | grep -qF "$BEAD_ID"; then
    fail "Worklog missing task ID $BEAD_ID"
    CONTENT_OK=false
fi
# No leftover placeholders
LEFTOVER=$(echo "$WORKLOG" | grep -c '{{' || true)
if [ "$LEFTOVER" -gt 0 ]; then
    fail "Worklog has $LEFTOVER lines with leftover {{ placeholders"
    CONTENT_OK=false
fi
if [ "$CONTENT_OK" = true ]; then
    pass "Worklog contains bead-specific epic/feature/task content with no leftover placeholders"
fi

# ---------- Test 4: Invalid bead-id ----------
echo "[4/6] Invalid bead-id rejected"
# Given: a bead ID that does not exist
# When: prep.sh is run with the invalid bead
# Then: exits non-zero with descriptive error
if INVALID_OUTPUT=$("$PREP_SCRIPT" "nonexistent-bead-999" --project-dir="$PROJECT_DIR" 2>&1); then
    fail "prep.sh should exit non-zero for invalid bead"
else
    # Verify error message mentions the bead or "not found"/"invalid"
    if echo "$INVALID_OUTPUT" | grep -qiE "not found|invalid|does not exist|error"; then
        pass "Invalid bead rejected with descriptive error"
    else
        fail "Invalid bead rejected but no descriptive error message"
        echo "  Output: $INVALID_OUTPUT"
    fi
fi

# ---------- Test 5: Duplicate worktree ----------
echo "[5/6] Duplicate worktree handled"
# Given: a worktree already exists for this bead
# When: prep.sh is run again with the same bead
# Then: idempotent skip or descriptive error
if DUP_OUTPUT=$("$PREP_SCRIPT" "$BEAD_ID" --project-dir="$PROJECT_DIR" 2>&1); then
    # Idempotent skip is acceptable
    if echo "$DUP_OUTPUT" | grep -qiE "already exists|skip"; then
        pass "Duplicate worktree: idempotent skip with message"
    else
        fail "Duplicate worktree succeeded but no 'already exists' message"
        echo "  Output: $DUP_OUTPUT"
    fi
else
    # Error exit is also acceptable
    if echo "$DUP_OUTPUT" | grep -qiE "already exists|duplicate"; then
        pass "Duplicate worktree: rejected with descriptive error"
    else
        fail "Duplicate worktree rejected but no descriptive error message"
        echo "  Output: $DUP_OUTPUT"
    fi
fi

# ---------- Test 6: Worktree on separate branch from main ----------
echo "[6/6] Worktree on separate branch from main"
# Given: a worktree created by prep.sh
# When: comparing worktree branch to main project branch
# Then: branches are different
MAIN_BRANCH=$(cd "$PROJECT_DIR" && git branch --show-current 2>/dev/null || echo "")
if [ "$BRANCH" != "$MAIN_BRANCH" ] && [ -n "$BRANCH" ]; then
    pass "Worktree branch ($BRANCH) is separate from main ($MAIN_BRANCH)"
else
    fail "Worktree branch ($BRANCH) should differ from main ($MAIN_BRANCH)"
fi

# ---------- Edge Cases ----------
echo ""
echo "=== Edge Cases ==="

# E1: No arguments shows usage error
echo "[E1] No arguments shows usage error"
# Given: no bead-id argument provided
# When: prep.sh is run without bead-id
# Then: exits non-zero with usage message
if NO_ARGS_OUTPUT=$("$PREP_SCRIPT" --project-dir="$PROJECT_DIR" 2>&1); then
    fail "prep.sh should exit non-zero with no bead-id argument"
else
    if echo "$NO_ARGS_OUTPUT" | grep -qiE "usage|bead-id|required"; then
        pass "No arguments: rejected with usage message"
    else
        fail "No arguments: rejected but no usage message"
        echo "  Output: $NO_ARGS_OUTPUT"
    fi
fi

# E2: prep.sh creates .capsule/logs directory
echo "[E2] prep.sh creates .capsule/logs directory"
# Given: a project where prep.sh has been run
# When: checking for .capsule/logs directory
# Then: directory exists
if [ -d "$PROJECT_DIR/.capsule/logs" ]; then
    pass ".capsule/logs directory exists"
else
    fail ".capsule/logs directory not created"
fi

# E3: Second bead creates separate worktree
echo "[E3] Second bead creates separate worktree"
# Given: a project with one existing worktree
# When: prep.sh is run with a different bead
# Then: creates a separate worktree with bead-specific content
BEAD_ID_2="demo-1.1.2"
BEAD2_OUTPUT=$("$PREP_SCRIPT" "$BEAD_ID_2" --project-dir="$PROJECT_DIR" 2>&1) || {
    fail "prep.sh exited non-zero for second bead: $BEAD2_OUTPUT"
}
WORKTREE_DIR_2="$PROJECT_DIR/.capsule/worktrees/$BEAD_ID_2"
if [ -d "$WORKTREE_DIR_2" ] && [ -f "$WORKTREE_DIR_2/worklog.md" ]; then
    WORKLOG_2=$(cat "$WORKTREE_DIR_2/worklog.md")
    if echo "$WORKLOG_2" | grep -qF "Validate phone format"; then
        pass "Second bead has its own worktree with bead-specific content"
    else
        fail "Second bead worktree missing bead-specific content"
    fi
else
    fail "Second bead worktree not created at $WORKTREE_DIR_2"
fi

# E4: Template instantiation failure cleans up worktree
echo "[E4] Template failure cleans up worktree"
# Given: a valid bead and a fresh test environment
# When: the worklog template is unreadable (simulating awk failure)
# Then: prep.sh exits non-zero and the worktree directory is removed
#
# Uses a separate test environment so template chmod doesn't affect other tests.
# Subshell + EXIT trap guarantees template permissions are restored.
# Note: assumes non-root execution (chmod 000 must block reads).
TEMPLATE_FILE="$REPO_ROOT/templates/worklog.md.template"

E4_PROJECT=$("$SETUP_SCRIPT")
E4_WORKTREE="$E4_PROJECT/.capsule/worktrees/$BEAD_ID"
E4_BRANCH="capsule-$BEAD_ID"

# Run in subshell to guarantee template permission restore
E4_RESULT=$(
    trap 'chmod 644 "$TEMPLATE_FILE"; rm -rf "$E4_PROJECT"' EXIT
    chmod 000 "$TEMPLATE_FILE"
    if "$PREP_SCRIPT" "$BEAD_ID" --project-dir="$E4_PROJECT" >/dev/null 2>&1; then
        echo "UNEXPECTED_SUCCESS"
    elif [ -d "$E4_WORKTREE" ]; then
        echo "WORKTREE_LEFT_BEHIND"
    elif (cd "$E4_PROJECT" && git rev-parse --verify "$E4_BRANCH" >/dev/null 2>&1); then
        echo "BRANCH_LEFT_BEHIND"
    else
        echo "CLEANED_UP"
    fi
)

case "$E4_RESULT" in
    CLEANED_UP)
        pass "Worktree and branch cleaned up after template failure"
        ;;
    WORKTREE_LEFT_BEHIND)
        fail "Worktree directory should be removed on template failure, but still exists"
        ;;
    BRANCH_LEFT_BEHIND)
        fail "Branch should be removed on template failure, but still exists"
        ;;
    UNEXPECTED_SUCCESS)
        fail "prep.sh should exit non-zero when template is unreadable"
        ;;
    *)
        fail "Unexpected E4 result: $E4_RESULT"
        ;;
esac

echo ""
echo "==========================================="
echo "RESULTS: $PASS passed, $FAIL failed"
echo "==========================================="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
