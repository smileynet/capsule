#!/usr/bin/env bash
# Test script for cap-8ax.6.2: teardown.sh cleanup script
# Validates: worktree removal, .capsule/output/ cleanup, log preservation,
#            git worktree prune, reporting, graceful when nothing to clean.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
SETUP_SCRIPT="$REPO_ROOT/scripts/setup-template.sh"
TEARDOWN_SCRIPT="$REPO_ROOT/scripts/teardown.sh"
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

for script in "$SETUP_SCRIPT" "$TEARDOWN_SCRIPT" "$PREP_SCRIPT"; do
    if [ ! -f "$script" ]; then
        echo "ERROR: $(basename "$script") not found at $script" >&2
        exit 1
    fi
done

echo "=== cap-8ax.6.2: teardown.sh tests ==="
echo ""

# =============================================================================
# Test 1: Graceful on clean project (nothing to tear down)
# =============================================================================
echo "[1/7] Graceful when nothing to clean"
# Given: a fresh template project with no worktrees or output
# When: teardown.sh is run
# Then: it exits 0 and reports nothing cleaned

PROJECT_DIR=$("$SETUP_SCRIPT")
trap 'rm -rf "$PROJECT_DIR"' EXIT

TEARDOWN_EXIT=0
TEARDOWN_OUTPUT=$("$TEARDOWN_SCRIPT" --project-dir="$PROJECT_DIR" 2>&1) || TEARDOWN_EXIT=$?

if [ "$TEARDOWN_EXIT" -eq 0 ]; then
    pass "Exits 0 on clean project"
else
    fail "Exited $TEARDOWN_EXIT on clean project (expected 0)"
    echo "  Output: $TEARDOWN_OUTPUT"
fi

# =============================================================================
# Test 2: Removes capsule worktrees
# =============================================================================
echo "[2/7] Removes capsule worktrees"
# Given: a project with an active capsule worktree
# When: teardown.sh is run
# Then: the worktree directory is removed and git worktree list shows no capsule entries

TEST_BEAD_ID="demo-1.1.1"
"$PREP_SCRIPT" "$TEST_BEAD_ID" --project-dir="$PROJECT_DIR" >/dev/null 2>&1

WORKTREE_DIR="$PROJECT_DIR/.capsule/worktrees/$TEST_BEAD_ID"
if [ ! -d "$WORKTREE_DIR" ]; then
    fail "Prerequisite: worktree not created by prep.sh"
else
    TEARDOWN_EXIT=0
    TEARDOWN_OUTPUT=$("$TEARDOWN_SCRIPT" --project-dir="$PROJECT_DIR" 2>&1) || TEARDOWN_EXIT=$?

    if [ "$TEARDOWN_EXIT" -eq 0 ]; then
        pass "Exits 0 after cleaning worktree"
    else
        fail "Exited $TEARDOWN_EXIT (expected 0)"
    fi

    if [ ! -d "$WORKTREE_DIR" ]; then
        pass "Worktree directory removed"
    else
        fail "Worktree directory still exists: $WORKTREE_DIR"
    fi

    CAPSULE_WORKTREE_COUNT=$(cd "$PROJECT_DIR" && git worktree list | grep -c "capsule-" || true)
    if [ "$CAPSULE_WORKTREE_COUNT" -eq 0 ]; then
        pass "No capsule worktrees in git worktree list"
    else
        fail "Found $CAPSULE_WORKTREE_COUNT capsule worktrees still registered"
    fi
fi

# =============================================================================
# Test 3: Cleans .capsule/output/ directory
# =============================================================================
echo "[3/7] Cleans .capsule/output/ directory"
# Given: a project with files in .capsule/output/
# When: teardown.sh is run
# Then: .capsule/output/ contents are removed

mkdir -p "$PROJECT_DIR/.capsule/output"
echo "test output" > "$PROJECT_DIR/.capsule/output/result.json"

TEARDOWN_EXIT=0
TEARDOWN_OUTPUT=$("$TEARDOWN_SCRIPT" --project-dir="$PROJECT_DIR" 2>&1) || TEARDOWN_EXIT=$?

if [ "$TEARDOWN_EXIT" -eq 0 ]; then
    pass "Exits 0 when cleaning output"
else
    fail "Exited $TEARDOWN_EXIT when cleaning output (expected 0)"
fi

if [ ! -f "$PROJECT_DIR/.capsule/output/result.json" ]; then
    pass ".capsule/output/ contents cleaned"
else
    fail ".capsule/output/result.json still exists"
fi

if [ -d "$PROJECT_DIR/.capsule/output" ]; then
    pass ".capsule/output/ directory preserved"
else
    fail ".capsule/output/ directory was deleted (should be preserved)"
fi

# =============================================================================
# Test 4: Preserves .capsule/logs/ directory
# =============================================================================
echo "[4/7] Preserves .capsule/logs/"
# Given: a project with archived logs in .capsule/logs/
# When: teardown.sh is run
# Then: .capsule/logs/ and its contents are preserved

mkdir -p "$PROJECT_DIR/.capsule/logs/some-bead"
echo "archived worklog" > "$PROJECT_DIR/.capsule/logs/some-bead/worklog.md"

TEARDOWN_EXIT=0
TEARDOWN_OUTPUT=$("$TEARDOWN_SCRIPT" --project-dir="$PROJECT_DIR" 2>&1) || TEARDOWN_EXIT=$?

if [ "$TEARDOWN_EXIT" -eq 0 ]; then
    pass "Exits 0 when logs present"
else
    fail "Exited $TEARDOWN_EXIT with logs present (expected 0)"
fi

if [ -f "$PROJECT_DIR/.capsule/logs/some-bead/worklog.md" ]; then
    pass ".capsule/logs/ preserved"
else
    fail ".capsule/logs/some-bead/worklog.md was deleted"
fi

# =============================================================================
# Test 5: Reports what was cleaned
# =============================================================================
echo "[5/7] Reports what was cleaned"
# Given: a project with worktrees and output to clean
# When: teardown.sh is run
# Then: output contains report of cleaned items

# Set up a worktree again for reporting test
"$PREP_SCRIPT" "$TEST_BEAD_ID" --project-dir="$PROJECT_DIR" >/dev/null 2>&1
mkdir -p "$PROJECT_DIR/.capsule/output"
echo "output" > "$PROJECT_DIR/.capsule/output/data.json"

TEARDOWN_EXIT=0
TEARDOWN_OUTPUT=$("$TEARDOWN_SCRIPT" --project-dir="$PROJECT_DIR" 2>&1) || TEARDOWN_EXIT=$?

if echo "$TEARDOWN_OUTPUT" | grep -qi "worktree\|cleaned\|removed"; then
    pass "Output reports cleanup activity"
else
    fail "Output does not report cleanup: $TEARDOWN_OUTPUT"
fi

# =============================================================================
# Test 6: Handles multiple worktrees
# =============================================================================
echo "[6/7] Handles multiple worktrees"
# Given: a project with multiple capsule worktrees
# When: teardown.sh is run
# Then: all worktrees are removed

# Create two worktrees
"$PREP_SCRIPT" "$TEST_BEAD_ID" --project-dir="$PROJECT_DIR" >/dev/null 2>&1
"$PREP_SCRIPT" "demo-1.1.2" --project-dir="$PROJECT_DIR" >/dev/null 2>&1

TEARDOWN_EXIT=0
TEARDOWN_OUTPUT=$("$TEARDOWN_SCRIPT" --project-dir="$PROJECT_DIR" 2>&1) || TEARDOWN_EXIT=$?

WT1="$PROJECT_DIR/.capsule/worktrees/$TEST_BEAD_ID"
WT2="$PROJECT_DIR/.capsule/worktrees/demo-1.1.2"

ALL_REMOVED=true
if [ -d "$WT1" ]; then
    fail "First worktree still exists"
    ALL_REMOVED=false
fi
if [ -d "$WT2" ]; then
    fail "Second worktree still exists"
    ALL_REMOVED=false
fi
if [ "$ALL_REMOVED" = true ]; then
    pass "All worktrees removed"
fi

CAPSULE_WORKTREE_COUNT=$(cd "$PROJECT_DIR" && git worktree list | grep -c "capsule-" || true)
if [ "$CAPSULE_WORKTREE_COUNT" -eq 0 ]; then
    pass "No capsule worktrees registered after multi-cleanup"
else
    fail "Found $CAPSULE_WORKTREE_COUNT capsule worktrees still registered"
fi

# =============================================================================
# Test 7: --dry-run shows what would be cleaned without deleting
# =============================================================================
echo "[7/7] --dry-run previews without deleting"
# Given: a project with a worktree and output files
# When: teardown.sh --dry-run is run
# Then: output contains [dry-run], worktree still exists, output files still exist

"$PREP_SCRIPT" "$TEST_BEAD_ID" --project-dir="$PROJECT_DIR" >/dev/null 2>&1
mkdir -p "$PROJECT_DIR/.capsule/output"
echo "output" > "$PROJECT_DIR/.capsule/output/data.json"

WORKTREE_DIR="$PROJECT_DIR/.capsule/worktrees/$TEST_BEAD_ID"

TEARDOWN_EXIT=0
TEARDOWN_OUTPUT=$("$TEARDOWN_SCRIPT" --project-dir="$PROJECT_DIR" --dry-run 2>&1) || TEARDOWN_EXIT=$?

if [ "$TEARDOWN_EXIT" -eq 0 ]; then
    pass "--dry-run exits 0"
else
    fail "--dry-run exited $TEARDOWN_EXIT (expected 0)"
fi

if echo "$TEARDOWN_OUTPUT" | grep -q '\[dry-run\]'; then
    pass "--dry-run output contains [dry-run] marker"
else
    fail "--dry-run output missing [dry-run] marker: $TEARDOWN_OUTPUT"
fi

if [ -d "$WORKTREE_DIR" ]; then
    pass "--dry-run preserves worktree"
else
    fail "--dry-run deleted worktree (should preserve)"
fi

if [ -f "$PROJECT_DIR/.capsule/output/data.json" ]; then
    pass "--dry-run preserves output files"
else
    fail "--dry-run deleted output files (should preserve)"
fi

# Clean up for real after dry-run test
"$TEARDOWN_SCRIPT" --project-dir="$PROJECT_DIR" >/dev/null 2>&1 || true

# =============================================================================
echo ""
echo "==========================================="
echo "RESULTS: $PASS passed, $FAIL failed"
echo "==========================================="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
