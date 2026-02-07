#!/usr/bin/env bash
# Test script for cap-8ax.3.5: test-writer/test-review pair E2E
# Validates: full chain from setup-template → prep → test-writer → test-review → retry
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
SETUP_SCRIPT="$REPO_ROOT/scripts/setup-template.sh"
PREP_SCRIPT="$REPO_ROOT/scripts/prep.sh"
RUN_PHASE="$REPO_ROOT/scripts/run-phase.sh"

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

for script in "$SETUP_SCRIPT" "$PREP_SCRIPT" "$RUN_PHASE"; do
    if [ ! -f "$script" ]; then
        echo "ERROR: $(basename "$script") not found at $script" >&2
        exit 1
    fi
    if [ ! -x "$script" ]; then
        echo "ERROR: $(basename "$script") is not executable" >&2
        exit 1
    fi
done

# --- Create test environment ---
echo "=== Setting up test environment ==="
PROJECT_DIR=$("$SETUP_SCRIPT")
trap 'rm -rf "$PROJECT_DIR"' EXIT
echo "  Test project: $PROJECT_DIR"

BEAD_ID="demo-1.1.1"

# Run prep to create worktree
echo "  Running prep.sh for $BEAD_ID..."
PREP_OUTPUT=$("$PREP_SCRIPT" "$BEAD_ID" --project-dir="$PROJECT_DIR" 2>&1) || {
    echo "ERROR: prep.sh failed: $PREP_OUTPUT" >&2
    exit 1
}

WORKTREE_DIR="$PROJECT_DIR/.capsule/worktrees/$BEAD_ID"
if [ ! -d "$WORKTREE_DIR" ]; then
    echo "ERROR: Worktree not created at $WORKTREE_DIR" >&2
    exit 1
fi
echo "  Worktree: $WORKTREE_DIR"

# --- Create mock claude binary ---
MOCK_DIR="$PROJECT_DIR/.capsule/mock-bin"
mkdir -p "$MOCK_DIR"

# Mock claude script: supports both response mode and prompt-capture mode.
# Set MOCK_CAPTURE_FILE to capture the prompt passed via -p.
# Set MOCK_CLAUDE_RESPONSE for static response.
# Set MOCK_RESPONSE_FILE to a file containing the response (for multi-line).
# Set MOCK_CLAUDE_EXIT for a custom exit code.
cat > "$MOCK_DIR/claude" << 'MOCK_EOF'
#!/usr/bin/env bash
# If capture mode is enabled, save the -p argument
if [ -n "${MOCK_CAPTURE_FILE:-}" ]; then
    CAPTURE_NEXT=false
    for arg in "$@"; do
        if [ "$CAPTURE_NEXT" = true ]; then
            echo "$arg" > "$MOCK_CAPTURE_FILE"
            CAPTURE_NEXT=false
        fi
        if [ "$arg" = "-p" ]; then
            CAPTURE_NEXT=true
        fi
    done
fi
# File-based response (for multi-line output)
if [ -n "${MOCK_RESPONSE_FILE:-}" ] && [ -f "$MOCK_RESPONSE_FILE" ]; then
    cat "$MOCK_RESPONSE_FILE"
    exit "${MOCK_CLAUDE_EXIT:-0}"
fi
# Output the configured response
if [ -n "${MOCK_CLAUDE_RESPONSE:-}" ]; then
    echo "$MOCK_CLAUDE_RESPONSE"
else
    echo "Mock claude: no response configured"
fi
exit "${MOCK_CLAUDE_EXIT:-0}"
MOCK_EOF
chmod +x "$MOCK_DIR/claude"

# Put mock claude first on PATH
export PATH="$MOCK_DIR:$PATH"
echo "  Mock claude: $MOCK_DIR/claude"

echo ""
echo "=== cap-8ax.3.5: test-writer/test-review pair ==="
echo ""

# ---------- Test 1: test-writer PASS signal and output logging ----------
echo "[1/5] test-writer creates test files and returns PASS"
RESPONSE_DIR="$PROJECT_DIR/.capsule/mock-responses"
mkdir -p "$RESPONSE_DIR"
cat > "$RESPONSE_DIR/test-writer-pass.txt" << 'RESP_EOF'
Reading worklog.md for task context...
Reading CLAUDE.md for project conventions...
Writing failing tests for ValidateEmail...

I've created test files and updated the worklog.

{"status":"PASS","feedback":"Created validate_test.go with 5 failing tests covering all acceptance criteria for ValidateEmail","files_changed":["src/validate_test.go","worklog.md"],"summary":"Failing tests written for ValidateEmail"}
RESP_EOF

export MOCK_RESPONSE_FILE="$RESPONSE_DIR/test-writer-pass.txt"

EXIT_CODE=0
PHASE_OUTPUT=$("$RUN_PHASE" test-writer "$WORKTREE_DIR" 2>/dev/null) || EXIT_CODE=$?

if [ "$EXIT_CODE" -eq 0 ]; then
    pass "test-writer phase completed with exit code 0 (PASS)"
else
    fail "test-writer phase returned exit code $EXIT_CODE, expected 0"
    echo "  Output: $PHASE_OUTPUT"
fi

# Verify signal JSON was returned
SIGNAL_STATUS=$(echo "$PHASE_OUTPUT" | jq -r '.status' 2>/dev/null || echo "")
if [ "$SIGNAL_STATUS" = "PASS" ]; then
    pass "test-writer returned valid PASS signal"
else
    fail "test-writer signal status is '$SIGNAL_STATUS', expected 'PASS'"
    echo "  Output: $PHASE_OUTPUT"
fi

# Verify output was logged
LOG_DIR="$WORKTREE_DIR/.capsule/output"
if [ -d "$LOG_DIR" ]; then
    LOG_FILE=$(find "$LOG_DIR" -name "test-writer-*.log" ! -name "*.stderr" 2>/dev/null | head -1)
    if [ -n "$LOG_FILE" ] && grep -qF "ValidateEmail" "$LOG_FILE"; then
        pass "test-writer output captured to log file"
    else
        fail "test-writer log file missing or missing expected content"
    fi
else
    fail "Output log directory not found at $LOG_DIR"
fi

unset MOCK_RESPONSE_FILE

# ---------- Test 2: test-review evaluates test-writer output ----------
echo ""
echo "[2/5] test-review evaluates and returns structured verdict"

cat > "$RESPONSE_DIR/test-review-pass.txt" << 'RESP_EOF'
Reading worklog.md for task context...
Reading test files created by test-writer...
Checking coverage against acceptance criteria...

All acceptance criteria have corresponding tests. Tests fail due to missing implementation (correct RED phase behavior).

{"status":"PASS","feedback":"All 5 acceptance criteria covered by tests. Tests fail because ValidateEmail is not implemented (correct TDD RED phase). Test quality is good - clear names, one assertion per test, proper isolation.","files_changed":["worklog.md"],"summary":"Tests pass review - all acceptance criteria covered, failing for correct reason"}
RESP_EOF

export MOCK_RESPONSE_FILE="$RESPONSE_DIR/test-review-pass.txt"

EXIT_CODE=0
REVIEW_OUTPUT=$("$RUN_PHASE" test-review "$WORKTREE_DIR" 2>/dev/null) || EXIT_CODE=$?

if [ "$EXIT_CODE" -eq 0 ]; then
    pass "test-review phase completed with exit code 0 (PASS)"
else
    fail "test-review phase returned exit code $EXIT_CODE, expected 0"
    echo "  Output: $REVIEW_OUTPUT"
fi

# Verify review signal
REVIEW_STATUS=$(echo "$REVIEW_OUTPUT" | jq -r '.status' 2>/dev/null || echo "")
if [ "$REVIEW_STATUS" = "PASS" ]; then
    pass "test-review returned valid PASS signal with feedback"
else
    fail "test-review signal status is '$REVIEW_STATUS', expected 'PASS'"
    echo "  Output: $REVIEW_OUTPUT"
fi

# Verify feedback is present and meaningful
REVIEW_FEEDBACK=$(echo "$REVIEW_OUTPUT" | jq -r '.feedback' 2>/dev/null || echo "")
if [ -n "$REVIEW_FEEDBACK" ] && [ "$REVIEW_FEEDBACK" != "null" ]; then
    pass "test-review feedback is present and non-empty"
else
    fail "test-review feedback is missing or empty"
fi

# Verify review output was logged
REVIEW_LOG=$(find "$LOG_DIR" -name "test-review-*.log" ! -name "*.stderr" 2>/dev/null | head -1)
if [ -n "$REVIEW_LOG" ] && grep -qF "acceptance criteria" "$REVIEW_LOG"; then
    pass "test-review output captured to log file"
else
    fail "test-review log file missing or missing expected content"
fi

unset MOCK_RESPONSE_FILE

# ---------- Test 3: NEEDS_WORK triggers retry with feedback ----------
echo ""
echo "[3/5] NEEDS_WORK signal triggers retry with feedback"

# Clean previous output logs for a fresh start
rm -rf "$WORKTREE_DIR/.capsule/output"

# test-review returns NEEDS_WORK
cat > "$RESPONSE_DIR/test-review-needs-work.txt" << 'RESP_EOF'
Reviewing tests...
Missing edge case test for empty string input.

{"status":"NEEDS_WORK","feedback":"Missing test for empty string email. Add TestValidateEmail_EmptyString that verifies empty input returns descriptive error.","files_changed":["worklog.md"],"summary":"Tests need improvement - missing empty string edge case"}
RESP_EOF

export MOCK_RESPONSE_FILE="$RESPONSE_DIR/test-review-needs-work.txt"

EXIT_CODE=0
NEEDS_WORK_OUTPUT=$("$RUN_PHASE" test-review "$WORKTREE_DIR" 2>/dev/null) || EXIT_CODE=$?

if [ "$EXIT_CODE" -eq 1 ]; then
    pass "NEEDS_WORK signal returns exit code 1"
else
    fail "Expected exit code 1 for NEEDS_WORK, got $EXIT_CODE"
    echo "  Output: $NEEDS_WORK_OUTPUT"
fi

# Extract feedback for retry
RETRY_FEEDBACK=$(echo "$NEEDS_WORK_OUTPUT" | jq -r '.feedback' 2>/dev/null || echo "")

# Now retry test-writer with feedback
CAPTURE_FILE="$PROJECT_DIR/.capsule/captured-prompt.txt"
export MOCK_CAPTURE_FILE="$CAPTURE_FILE"

cat > "$RESPONSE_DIR/test-writer-retry.txt" << 'RESP_EOF'
Received feedback from test-review. Addressing issues...
Added TestValidateEmail_EmptyString test case.

{"status":"PASS","feedback":"Added missing empty string test case as requested by reviewer","files_changed":["src/validate_test.go","worklog.md"],"summary":"Tests updated with empty string edge case"}
RESP_EOF

export MOCK_RESPONSE_FILE="$RESPONSE_DIR/test-writer-retry.txt"

EXIT_CODE=0
RETRY_OUTPUT=$("$RUN_PHASE" test-writer "$WORKTREE_DIR" --feedback="$RETRY_FEEDBACK" 2>/dev/null) || EXIT_CODE=$?

if [ "$EXIT_CODE" -eq 0 ]; then
    pass "test-writer retry completed successfully"
else
    fail "test-writer retry returned exit code $EXIT_CODE, expected 0"
    echo "  Output: $RETRY_OUTPUT"
fi

# Verify feedback was injected into the prompt
if [ -f "$CAPTURE_FILE" ]; then
    CAPTURED=$(cat "$CAPTURE_FILE")
    FEEDBACK_OK=true

    if ! echo "$CAPTURED" | grep -qF "Missing test for empty string email"; then
        fail "Review feedback not found in retry prompt"
        FEEDBACK_OK=false
    fi

    if ! echo "$CAPTURED" | grep -qF "Previous Feedback"; then
        fail "Feedback section header not found in retry prompt"
        FEEDBACK_OK=false
    fi

    # Base prompt should still be present
    if ! echo "$CAPTURED" | grep -qF "test-writing agent"; then
        fail "Base test-writer prompt not found in retry prompt"
        FEEDBACK_OK=false
    fi

    if [ "$FEEDBACK_OK" = true ]; then
        pass "Retry prompt contains both base template and injected feedback"
    fi
else
    fail "Prompt capture file not created during retry"
fi

rm -f "$CAPTURE_FILE"
unset MOCK_CAPTURE_FILE
unset MOCK_RESPONSE_FILE

# ---------- Test 4: Both phases produce output logs ----------
echo ""
echo "[4/5] Both phase types produce output logs"
LOG_DIR="$WORKTREE_DIR/.capsule/output"
if [ -d "$LOG_DIR" ]; then
    LOG_COUNT=$(find "$LOG_DIR" -name "*.log" ! -name "*.stderr" 2>/dev/null | wc -l)
    if [ "$LOG_COUNT" -ge 2 ]; then
        pass "Multiple phase log files present ($LOG_COUNT logs)"
    else
        fail "Expected at least 2 log files, found $LOG_COUNT"
    fi

    TW_LOGS=$(find "$LOG_DIR" -name "test-writer-*.log" ! -name "*.stderr" 2>/dev/null | wc -l)
    TR_LOGS=$(find "$LOG_DIR" -name "test-review-*.log" ! -name "*.stderr" 2>/dev/null | wc -l)
    if [ "$TW_LOGS" -ge 1 ] && [ "$TR_LOGS" -ge 1 ]; then
        pass "Both test-writer and test-review logs present"
    else
        fail "Expected logs from both phases (test-writer: $TW_LOGS, test-review: $TR_LOGS)"
    fi
else
    fail "Output log directory not found"
fi

# ---------- Test 5: Full PASS path - no retry ----------
echo ""
echo "[5/5] Full PASS path: test-review passes on first attempt"

# Clean state for a fresh run
rm -rf "$WORKTREE_DIR/.capsule/output"

# test-writer PASS
export MOCK_RESPONSE_FILE="$RESPONSE_DIR/test-writer-pass.txt"
TW_EXIT=0
"$RUN_PHASE" test-writer "$WORKTREE_DIR" >/dev/null 2>/dev/null || TW_EXIT=$?

# test-review PASS
export MOCK_RESPONSE_FILE="$RESPONSE_DIR/test-review-pass.txt"
TR_EXIT=0
"$RUN_PHASE" test-review "$WORKTREE_DIR" >/dev/null 2>/dev/null || TR_EXIT=$?

if [ "$TW_EXIT" -eq 0 ] && [ "$TR_EXIT" -eq 0 ]; then
    pass "Full PASS path: test-writer (exit 0) → test-review (exit 0), no retry needed"
else
    fail "Full PASS path: test-writer exit=$TW_EXIT, test-review exit=$TR_EXIT (both should be 0)"
fi

unset MOCK_RESPONSE_FILE

# Note: Spec test cases 4 (verify tests fail / RED phase) and 5 (verify worklog
# entries chronologically ordered) are not feasible with mock-based testing since
# mock claude does not create actual files or modify worklog.md. These behaviors
# are validated by real claude runs in the full integration test suite.

# ---------- Edge Cases ----------
echo ""
echo "=== Edge Cases ==="

# E1: test-writer ERROR signal propagates correctly
echo "[E1] test-writer ERROR propagates exit code 2"

cat > "$RESPONSE_DIR/test-writer-error.txt" << 'RESP_EOF'
Could not read worklog.md - file is missing or corrupted.

{"status":"ERROR","feedback":"worklog.md not found in worktree","files_changed":[],"summary":"Phase failed - missing worklog"}
RESP_EOF

export MOCK_RESPONSE_FILE="$RESPONSE_DIR/test-writer-error.txt"
EXIT_CODE=0
"$RUN_PHASE" test-writer "$WORKTREE_DIR" >/dev/null 2>/dev/null || EXIT_CODE=$?

if [ "$EXIT_CODE" -eq 2 ]; then
    pass "test-writer ERROR → exit code 2"
else
    fail "Expected exit code 2 for ERROR signal, got $EXIT_CODE"
fi

unset MOCK_RESPONSE_FILE

# E2: test-review ERROR signal propagates correctly
echo "[E2] test-review ERROR propagates exit code 2"

cat > "$RESPONSE_DIR/test-review-error.txt" << 'RESP_EOF'
No test files found in worktree. Cannot review.

{"status":"ERROR","feedback":"No test files found to review","files_changed":[],"summary":"Phase failed - no test files"}
RESP_EOF

export MOCK_RESPONSE_FILE="$RESPONSE_DIR/test-review-error.txt"
EXIT_CODE=0
"$RUN_PHASE" test-review "$WORKTREE_DIR" >/dev/null 2>/dev/null || EXIT_CODE=$?

if [ "$EXIT_CODE" -eq 2 ]; then
    pass "test-review ERROR → exit code 2"
else
    fail "Expected exit code 2 for ERROR signal, got $EXIT_CODE"
fi

unset MOCK_RESPONSE_FILE

# E3: Signal JSON has all required fields
echo "[E3] Signal JSON has all required fields"

export MOCK_RESPONSE_FILE="$RESPONSE_DIR/test-writer-pass.txt"
SIGNAL_OUTPUT=$("$RUN_PHASE" test-writer "$WORKTREE_DIR" 2>/dev/null) || true

FIELDS_OK=true
for field in status feedback files_changed summary; do
    if [ "$(echo "$SIGNAL_OUTPUT" | jq "has(\"$field\")")" != "true" ]; then
        fail "Signal missing required field: $field"
        FIELDS_OK=false
    fi
done
if [ "$FIELDS_OK" = true ]; then
    pass "Signal JSON contains all required fields (status, feedback, files_changed, summary)"
fi

IS_ARRAY=$(echo "$SIGNAL_OUTPUT" | jq '.files_changed | type == "array"')
if [ "$IS_ARRAY" = "true" ]; then
    pass "files_changed is an array"
else
    fail "files_changed should be an array"
fi

unset MOCK_RESPONSE_FILE

echo ""
echo "==========================================="
echo "RESULTS: $PASS passed, $FAIL failed"
echo "==========================================="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
