#!/usr/bin/env bash
# Test script for run-summary.sh
# Validates: argument parsing, context gathering, output destinations, graceful degradation.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
SUMMARY_SCRIPT="$REPO_ROOT/scripts/run-summary.sh"
SETUP_SCRIPT="$REPO_ROOT/scripts/setup-template.sh"

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

for script in "$SUMMARY_SCRIPT" "$SETUP_SCRIPT"; do
    if [ ! -f "$script" ]; then
        echo "ERROR: $(basename "$script") not found at $script" >&2
        exit 1
    fi
done

# =============================================================================
echo "=== Setting up test environment ==="
PROJECT_DIR=$("$SETUP_SCRIPT")
trap 'rm -rf "$PROJECT_DIR"' EXIT
echo "  Test project: $PROJECT_DIR"

MOCK_DIR=$(mktemp -d)
trap 'rm -rf "$PROJECT_DIR" "$MOCK_DIR"' EXIT

BEAD_ID="demo-1.1.1"

echo ""
echo "=== run-summary.sh tests ==="
echo ""

# ---------- Test 1: Script exists and is executable ----------
echo "[1/8] Script exists and is executable"
if [ -x "$SUMMARY_SCRIPT" ]; then
    pass "run-summary.sh exists and is executable"
else
    fail "run-summary.sh is not executable"
fi

# ---------- Test 2: Missing bead-id rejected ----------
echo "[2/8] Missing bead-id rejected"
MISSING_EXIT=0
MISSING_OUTPUT=$("$SUMMARY_SCRIPT" 2>&1) || MISSING_EXIT=$?
if [ "$MISSING_EXIT" -ne 0 ]; then
    if echo "$MISSING_OUTPUT" | grep -qiE 'bead-id.*required|usage'; then
        pass "Missing bead-id exits non-zero with usage message"
    else
        fail "Missing bead-id exits non-zero but no usage message"
        echo "  Output: $MISSING_OUTPUT"
    fi
else
    fail "Missing bead-id should exit non-zero"
fi

# ---------- Test 3: Summary generates output on success path ----------
echo "[3/8] Summary generates output on success path (mock claude)"

# Create a worklog in the archive location (post-merge)
ARCHIVE_DIR="$PROJECT_DIR/.capsule/logs/$BEAD_ID"
mkdir -p "$ARCHIVE_DIR"
cat > "$ARCHIVE_DIR/worklog.md" << 'EOF'
# Worklog: demo-1.1.1 — Validate email format

## Phase 1: test-writer
_Status: complete_
Created test file: src/validation_test.go

## Phase 5: sign-off
_Status: complete_
**Verdict: PASS**
Tests run: 3 tests executed
EOF

# Create mock claude that echoes a summary
cat > "$MOCK_DIR/claude" << 'MOCK_EOF'
#!/usr/bin/env bash
while [ $# -gt 0 ]; do shift; done
cat << 'SUMMARY'
## Pipeline Summary: demo-1.1.1

### What Was Accomplished
The pipeline implemented email validation.

### Challenges Encountered
None — pipeline completed on first attempt.

### End State
Bead closed, code merged to main. 3 tests passing.

### Feature & Epic Progress
1 of 2 tasks closed for feature demo-1.1.
SUMMARY
MOCK_EOF
chmod +x "$MOCK_DIR/claude"

# Remove any prior summary
rm -f "$ARCHIVE_DIR/summary.md"

SUCCESS_EXIT=0
SUCCESS_OUTPUT=$(PATH="$MOCK_DIR:$PATH" "$SUMMARY_SCRIPT" "$BEAD_ID" \
    --project-dir="$PROJECT_DIR" \
    --outcome=SUCCESS \
    --test-review-attempts=1 \
    --exec-review-attempts=1 \
    --signoff-attempts=1 \
    --max-retries=3 \
    --duration=120 2>&1) || SUCCESS_EXIT=$?

if [ "$SUCCESS_EXIT" -eq 0 ] && [ -n "$SUCCESS_OUTPUT" ]; then
    pass "Summary generates non-empty output on success path"
else
    fail "Summary failed on success path (exit $SUCCESS_EXIT)"
    echo "  Output: $SUCCESS_OUTPUT"
fi

# ---------- Test 4: Summary saved to archive directory ----------
echo "[4/8] Summary saved to archive directory"
if [ -f "$ARCHIVE_DIR/summary.md" ]; then
    SAVED_SIZE=$(wc -c < "$ARCHIVE_DIR/summary.md")
    if [ "$SAVED_SIZE" -gt 0 ]; then
        pass "Summary saved to $ARCHIVE_DIR/summary.md ($SAVED_SIZE bytes)"
    else
        fail "Summary file exists but is empty"
    fi
else
    fail "Summary not saved to archive directory"
fi

# ---------- Test 5: Summary generates output on failure path ----------
echo "[5/8] Summary generates output on failure path (worklog in worktree)"

# Create a worktree directory with worklog (simulating failure case)
FAIL_BEAD="demo-1.1.2"
WORKTREE_DIR="$PROJECT_DIR/.capsule/worktrees/$FAIL_BEAD"
mkdir -p "$WORKTREE_DIR"
cat > "$WORKTREE_DIR/worklog.md" << 'EOF'
# Worklog: demo-1.1.2 — Validate phone format

## Phase 1: test-writer
_Status: complete_
Created test file: src/phone_test.go
EOF

FAIL_ARCHIVE="$PROJECT_DIR/.capsule/logs/$FAIL_BEAD"
rm -rf "$FAIL_ARCHIVE"

FAIL_EXIT=0
FAIL_OUTPUT=$(PATH="$MOCK_DIR:$PATH" "$SUMMARY_SCRIPT" "$FAIL_BEAD" \
    --project-dir="$PROJECT_DIR" \
    --outcome=FAILED \
    --failed-stage="execute/execute-review" \
    --test-review-attempts=1 \
    --exec-review-attempts=3 \
    --signoff-attempts=0 \
    --max-retries=3 \
    --duration=90 2>&1) || FAIL_EXIT=$?

if [ "$FAIL_EXIT" -eq 0 ] && [ -n "$FAIL_OUTPUT" ]; then
    pass "Summary generates output on failure path"
else
    fail "Summary failed on failure path (exit $FAIL_EXIT)"
    echo "  Output: $FAIL_OUTPUT"
fi

# Check archive was created for failure case too
if [ -f "$FAIL_ARCHIVE/summary.md" ]; then
    pass "Summary archive created even on failure path"
else
    fail "Summary archive not created on failure path"
fi

# ---------- Test 6: Summary handles missing worklog gracefully ----------
echo "[6/8] Summary handles missing worklog gracefully"

# Use a bead that exists but has no worklog anywhere
NOWORKLOG_BEAD="demo-1.1"  # Feature-level, never has a worktree
NOWORKLOG_ARCHIVE="$PROJECT_DIR/.capsule/logs/$NOWORKLOG_BEAD"
rm -rf "$NOWORKLOG_ARCHIVE"

NOWORKLOG_EXIT=0
NOWORKLOG_OUTPUT=$(PATH="$MOCK_DIR:$PATH" "$SUMMARY_SCRIPT" "$NOWORKLOG_BEAD" \
    --project-dir="$PROJECT_DIR" \
    --outcome=FAILED \
    --failed-stage="prep" \
    --duration=5 2>&1) || NOWORKLOG_EXIT=$?

if [ "$NOWORKLOG_EXIT" -eq 0 ]; then
    pass "Summary succeeds with missing worklog"
else
    fail "Summary should handle missing worklog gracefully (exit $NOWORKLOG_EXIT)"
    echo "  Output: $NOWORKLOG_OUTPUT"
fi

# ---------- Test 7: Pipeline still succeeds if summary fails ----------
echo "[7/8] Pipeline still succeeds if summary script fails"

# Create a mock claude that fails
MOCK_FAIL_DIR=$(mktemp -d)
trap 'rm -rf "$PROJECT_DIR" "$MOCK_DIR" "$MOCK_FAIL_DIR"' EXIT
cat > "$MOCK_FAIL_DIR/claude" << 'MOCK_EOF'
#!/usr/bin/env bash
exit 1
MOCK_EOF
chmod +x "$MOCK_FAIL_DIR/claude"

FAILSUM_EXIT=0
FAILSUM_OUTPUT=$(PATH="$MOCK_FAIL_DIR:$PATH" "$SUMMARY_SCRIPT" "$BEAD_ID" \
    --project-dir="$PROJECT_DIR" \
    --outcome=SUCCESS \
    --duration=10 2>&1) || FAILSUM_EXIT=$?

# The summary script itself will exit non-zero, but the pipeline wraps it with || true
# So just verify the script exits non-zero when claude fails
if [ "$FAILSUM_EXIT" -ne 0 ]; then
    pass "Summary exits non-zero on claude failure (pipeline uses || true to guard)"
else
    fail "Summary should exit non-zero when claude fails"
fi

# ---------- Test 8: Prompt template exists ----------
echo "[8/8] Summary prompt template exists"
PROMPT_FILE="$REPO_ROOT/prompts/summary.md"
if [ -f "$PROMPT_FILE" ]; then
    if grep -q '{{CONTEXT}}' "$PROMPT_FILE"; then
        pass "Prompt template exists with {{CONTEXT}} placeholder"
    else
        fail "Prompt template exists but missing {{CONTEXT}} placeholder"
    fi
else
    fail "Prompt template not found at $PROMPT_FILE"
fi

echo ""
echo "==========================================="
echo "RESULTS: $PASS passed, $FAIL failed"
echo "==========================================="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
