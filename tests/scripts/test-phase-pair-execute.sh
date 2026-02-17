#!/usr/bin/env bash
# Test script for cap-8ax.4.3: execute/execute-review pair E2E
# Validates: full chain from setup-template → prep → execute → execute-review → retry
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
echo "=== cap-8ax.4.3: execute/execute-review pair ==="
echo ""

# ---------- Test 1: execute PASS signal and output logging ----------
echo "[1/5] execute creates implementation files and returns PASS"
# Given: a worktree created by setup-template + prep, with mock claude configured
RESPONSE_DIR="$PROJECT_DIR/.capsule/mock-responses"
mkdir -p "$RESPONSE_DIR"
cat > "$RESPONSE_DIR/execute-pass.txt" << 'RESP_EOF'
Reading worklog.md for task context...
Reading AGENTS.md for project conventions...
Confirming RED state - running tests...
Tests fail as expected (5 failing).

Implementing minimal code to pass all tests...

Created src/validate.go with ValidateEmail function.
All 5 tests now pass (GREEN state achieved).
Updated worklog.md with Phase 3 entry.

{"status":"PASS","feedback":"Implemented ValidateEmail in src/validate.go. All 5 tests pass. Minimal implementation: regex-based email validation with proper error messages.","files_changed":["src/validate.go","worklog.md"],"summary":"Implementation complete - all tests pass (GREEN)"}
RESP_EOF

export MOCK_RESPONSE_FILE="$RESPONSE_DIR/execute-pass.txt"

# When: run-phase.sh execute is invoked on the worktree
EXIT_CODE=0
PHASE_OUTPUT=$("$RUN_PHASE" execute "$WORKTREE_DIR" 2>/dev/null) || EXIT_CODE=$?

# Then: exit code is 0 (PASS), signal is valid, output is logged
if [ "$EXIT_CODE" -eq 0 ]; then
    pass "execute phase completed with exit code 0 (PASS)"
else
    fail "execute phase returned exit code $EXIT_CODE, expected 0"
    echo "  Output: $PHASE_OUTPUT"
fi

# Verify signal JSON was returned
SIGNAL_STATUS=$(printf '%s\n' "$PHASE_OUTPUT" | jq -r '.status' 2>/dev/null || echo "")
if [ "$SIGNAL_STATUS" = "PASS" ]; then
    pass "execute returned valid PASS signal"
else
    fail "execute signal status is '$SIGNAL_STATUS', expected 'PASS'"
    echo "  Output: $PHASE_OUTPUT"
fi

# Verify output was logged
LOG_DIR="$WORKTREE_DIR/.capsule/output"
if [ -d "$LOG_DIR" ]; then
    LOG_FILE=$(find "$LOG_DIR" -name "execute-*.log" ! -name "*.stderr" ! -name "execute-review-*.log" 2>/dev/null | head -1)
    if [ -n "$LOG_FILE" ] && grep -qF "ValidateEmail" "$LOG_FILE"; then
        pass "execute output captured to log file"
    else
        fail "execute log file missing or missing expected content"
    fi
else
    fail "Output log directory not found at $LOG_DIR"
fi

unset MOCK_RESPONSE_FILE

# ---------- Test 2: execute-review evaluates implementation ----------
echo ""
echo "[2/5] execute-review evaluates and returns structured verdict"
# Given: execute has run (logs persist from Test 1)
cat > "$RESPONSE_DIR/execute-review-pass.txt" << 'RESP_EOF'
Reading worklog.md for task context...
Reading implementation files from execute phase...
Running tests to verify they pass...

All 5 tests pass. Reviewing implementation quality...

Correctness: All acceptance criteria implemented correctly.
Scope: Only acceptance-criteria-scoped changes present.
Quality: Code follows project conventions, clean and readable.
No test modifications detected.

{"status":"PASS","feedback":"Implementation passes all checks. All 5 tests pass, code quality is acceptable, changes are properly scoped, no test files modified. ValidateEmail uses regex validation with clear error messages.","files_changed":["worklog.md"],"summary":"Implementation review passed - all checks green"}
RESP_EOF

export MOCK_RESPONSE_FILE="$RESPONSE_DIR/execute-review-pass.txt"

# When: run-phase.sh execute-review is invoked on the worktree
EXIT_CODE=0
REVIEW_OUTPUT=$("$RUN_PHASE" execute-review "$WORKTREE_DIR" 2>/dev/null) || EXIT_CODE=$?

# Then: exit code is 0 (PASS), feedback is present, output is logged
if [ "$EXIT_CODE" -eq 0 ]; then
    pass "execute-review phase completed with exit code 0 (PASS)"
else
    fail "execute-review phase returned exit code $EXIT_CODE, expected 0"
    echo "  Output: $REVIEW_OUTPUT"
fi

# Verify review signal
REVIEW_STATUS=$(printf '%s\n' "$REVIEW_OUTPUT" | jq -r '.status' 2>/dev/null || echo "")
if [ "$REVIEW_STATUS" = "PASS" ]; then
    pass "execute-review returned valid PASS signal with feedback"
else
    fail "execute-review signal status is '$REVIEW_STATUS', expected 'PASS'"
    echo "  Output: $REVIEW_OUTPUT"
fi

# Verify feedback is present and meaningful
REVIEW_FEEDBACK=$(printf '%s\n' "$REVIEW_OUTPUT" | jq -r '.feedback' 2>/dev/null || echo "")
if [ -n "$REVIEW_FEEDBACK" ] && [ "$REVIEW_FEEDBACK" != "null" ]; then
    pass "execute-review feedback is present and non-empty"
else
    fail "execute-review feedback is missing or empty"
fi

# Verify review output was logged
REVIEW_LOG=$(find "$LOG_DIR" -name "execute-review-*.log" ! -name "*.stderr" 2>/dev/null | head -1)
if [ -n "$REVIEW_LOG" ] && grep -qF "acceptance criteria" "$REVIEW_LOG"; then
    pass "execute-review output captured to log file"
else
    fail "execute-review log file missing or missing expected content"
fi

unset MOCK_RESPONSE_FILE

# ---------- Test 3: NEEDS_WORK triggers retry with feedback ----------
echo ""
echo "[3/5] NEEDS_WORK signal triggers retry with feedback"
# Given: clean output state
# Clean previous output logs for a fresh start
rm -rf "$WORKTREE_DIR/.capsule/output"

# Given: execute-review configured to return NEEDS_WORK
cat > "$RESPONSE_DIR/execute-review-needs-work.txt" << 'RESP_EOF'
Running tests... all pass.
Reviewing implementation...

Issue found: ValidateEmail swallows the error from regexp.Compile instead of propagating it. This is fragile if the regex pattern is changed later.

{"status":"NEEDS_WORK","feedback":"Function ValidateEmail in src/validate.go swallows error from regexp.Compile (line 12). Propagate the error to the caller instead of silently returning false. Also add a package-level compiled regex to avoid recompilation on every call.","files_changed":["worklog.md"],"summary":"Implementation needs work - error handling and regex compilation issues"}
RESP_EOF

export MOCK_RESPONSE_FILE="$RESPONSE_DIR/execute-review-needs-work.txt"

# When: execute-review is invoked
EXIT_CODE=0
NEEDS_WORK_OUTPUT=$("$RUN_PHASE" execute-review "$WORKTREE_DIR" 2>/dev/null) || EXIT_CODE=$?

# Then: exit code is 1 (NEEDS_WORK)
if [ "$EXIT_CODE" -eq 1 ]; then
    pass "NEEDS_WORK signal returns exit code 1"
else
    fail "Expected exit code 1 for NEEDS_WORK, got $EXIT_CODE"
    echo "  Output: $NEEDS_WORK_OUTPUT"
fi

# Given: feedback extracted from NEEDS_WORK signal for retry
RETRY_FEEDBACK=$(printf '%s\n' "$NEEDS_WORK_OUTPUT" | jq -r '.feedback' 2>/dev/null || echo "")

# When: execute is retried with --feedback
CAPTURE_FILE="$PROJECT_DIR/.capsule/captured-prompt.txt"
export MOCK_CAPTURE_FILE="$CAPTURE_FILE"

cat > "$RESPONSE_DIR/execute-retry.txt" << 'RESP_EOF'
Received feedback from execute-review. Addressing issues...
Fixed error propagation in ValidateEmail.
Moved regex compilation to package level.

{"status":"PASS","feedback":"Fixed error handling: regexp.Compile error now propagated to caller. Moved regex to package-level var for single compilation. All tests still pass.","files_changed":["src/validate.go","worklog.md"],"summary":"Implementation fixed per reviewer feedback - error handling corrected"}
RESP_EOF

export MOCK_RESPONSE_FILE="$RESPONSE_DIR/execute-retry.txt"

EXIT_CODE=0
RETRY_OUTPUT=$("$RUN_PHASE" execute "$WORKTREE_DIR" --feedback="$RETRY_FEEDBACK" 2>/dev/null) || EXIT_CODE=$?

if [ "$EXIT_CODE" -eq 0 ]; then
    pass "execute retry completed successfully"
else
    fail "execute retry returned exit code $EXIT_CODE, expected 0"
    echo "  Output: $RETRY_OUTPUT"
fi

# Then: retry succeeds and prompt contains base template + injected feedback
if [ -f "$CAPTURE_FILE" ]; then
    CAPTURED=$(cat "$CAPTURE_FILE")
    FEEDBACK_OK=true

    if ! echo "$CAPTURED" | grep -qF "swallows error from regexp.Compile"; then
        fail "Review feedback not found in retry prompt"
        FEEDBACK_OK=false
    fi

    if ! echo "$CAPTURED" | grep -qF "Previous Feedback"; then
        fail "Feedback section header not found in retry prompt"
        FEEDBACK_OK=false
    fi

    # Base prompt should still be present
    if ! echo "$CAPTURED" | grep -qF "implementation agent"; then
        fail "Base execute prompt not found in retry prompt"
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
# Given: execute and execute-review have both run (logs from Test 3)
# When: output directory is inspected
LOG_DIR="$WORKTREE_DIR/.capsule/output"
# Then: log files exist for both phase types
if [ -d "$LOG_DIR" ]; then
    LOG_COUNT=$(find "$LOG_DIR" -name "*.log" ! -name "*.stderr" 2>/dev/null | wc -l)
    if [ "$LOG_COUNT" -ge 2 ]; then
        pass "Multiple phase log files present ($LOG_COUNT logs)"
    else
        fail "Expected at least 2 log files, found $LOG_COUNT"
    fi

    EX_LOGS=$(find "$LOG_DIR" -name "execute-*.log" ! -name "*.stderr" ! -name "execute-review-*.log" 2>/dev/null | wc -l)
    ER_LOGS=$(find "$LOG_DIR" -name "execute-review-*.log" ! -name "*.stderr" 2>/dev/null | wc -l)
    if [ "$EX_LOGS" -ge 1 ] && [ "$ER_LOGS" -ge 1 ]; then
        pass "Both execute and execute-review logs present"
    else
        fail "Expected logs from both phases (execute: $EX_LOGS, execute-review: $ER_LOGS)"
    fi
else
    fail "Output log directory not found"
fi

# ---------- Test 5: Full PASS path - no retry ----------
echo ""
echo "[5/5] Full PASS path: execute-review passes on first attempt"
# Given: clean output state
# Clean state for a fresh run
rm -rf "$WORKTREE_DIR/.capsule/output"

# When: execute then execute-review both return PASS
export MOCK_RESPONSE_FILE="$RESPONSE_DIR/execute-pass.txt"
EX_EXIT=0
"$RUN_PHASE" execute "$WORKTREE_DIR" >/dev/null 2>/dev/null || EX_EXIT=$?

# execute-review PASS
export MOCK_RESPONSE_FILE="$RESPONSE_DIR/execute-review-pass.txt"
ER_EXIT=0
"$RUN_PHASE" execute-review "$WORKTREE_DIR" >/dev/null 2>/dev/null || ER_EXIT=$?

# Then: both exit with 0, no retry needed
if [ "$EX_EXIT" -eq 0 ] && [ "$ER_EXIT" -eq 0 ]; then
    pass "Full PASS path: execute (exit 0) → execute-review (exit 0), no retry needed"
else
    fail "Full PASS path: execute exit=$EX_EXIT, execute-review exit=$ER_EXIT (both should be 0)"
fi

unset MOCK_RESPONSE_FILE

# Note: Spec test cases 4 (verify worklog entries) and 5 (verify no test file
# modifications) are not feasible with mock-based testing since mock claude does
# not create actual files or modify worklog.md. These behaviors are validated by
# real claude runs in the full integration test suite.

# ---------- Edge Cases ----------
echo ""
echo "=== Edge Cases ==="

# E1: execute ERROR signal propagates correctly
echo "[E1] execute ERROR propagates exit code 2"
# Given: mock claude returns an ERROR signal for execute
cat > "$RESPONSE_DIR/execute-error.txt" << 'RESP_EOF'
Running tests... tests already pass without implementation. This is an error - RED state not confirmed.

{"status":"ERROR","feedback":"Tests already pass before implementation. RED state not confirmed - test-writer phase may have produced incorrect tests.","files_changed":[],"summary":"Phase failed - tests already passing (no RED state)"}
RESP_EOF

export MOCK_RESPONSE_FILE="$RESPONSE_DIR/execute-error.txt"
# When: run-phase.sh execute is invoked
EXIT_CODE=0
"$RUN_PHASE" execute "$WORKTREE_DIR" >/dev/null 2>/dev/null || EXIT_CODE=$?
# Then: exit code is 2
if [ "$EXIT_CODE" -eq 2 ]; then
    pass "execute ERROR → exit code 2"
else
    fail "Expected exit code 2 for ERROR signal, got $EXIT_CODE"
fi

unset MOCK_RESPONSE_FILE

# E2: execute-review ERROR signal propagates correctly
echo "[E2] execute-review ERROR propagates exit code 2"
# Given: mock claude returns an ERROR signal for execute-review
cat > "$RESPONSE_DIR/execute-review-error.txt" << 'RESP_EOF'
No implementation files found in worktree. Cannot review.

{"status":"ERROR","feedback":"No implementation files found to review. Execute phase may not have run.","files_changed":[],"summary":"Phase failed - no implementation files"}
RESP_EOF

export MOCK_RESPONSE_FILE="$RESPONSE_DIR/execute-review-error.txt"
# When: run-phase.sh execute-review is invoked
EXIT_CODE=0
"$RUN_PHASE" execute-review "$WORKTREE_DIR" >/dev/null 2>/dev/null || EXIT_CODE=$?
# Then: exit code is 2
if [ "$EXIT_CODE" -eq 2 ]; then
    pass "execute-review ERROR → exit code 2"
else
    fail "Expected exit code 2 for ERROR signal, got $EXIT_CODE"
fi

unset MOCK_RESPONSE_FILE

# E3: execute-review finds scope creep -- returns NEEDS_WORK
echo "[E3] execute-review detects scope creep and returns NEEDS_WORK"
# Given: mock claude returns NEEDS_WORK due to scope creep
cat > "$RESPONSE_DIR/execute-review-scope-creep.txt" << 'RESP_EOF'
Running tests... all pass.
Reviewing implementation...

Scope issue: Implementation adds a Logger utility and HTTP client wrapper that are not required by the acceptance criteria or tests.

{"status":"NEEDS_WORK","feedback":"Scope creep detected: src/logger.go and src/http_client.go are not required by acceptance criteria. Remove these files and keep only ValidateEmail implementation in src/validate.go.","files_changed":["worklog.md"],"summary":"Implementation has scope creep - extra files not in acceptance criteria"}
RESP_EOF

export MOCK_RESPONSE_FILE="$RESPONSE_DIR/execute-review-scope-creep.txt"
EXIT_CODE=0
SCOPE_OUTPUT=$("$RUN_PHASE" execute-review "$WORKTREE_DIR" 2>/dev/null) || EXIT_CODE=$?
if [ "$EXIT_CODE" -eq 1 ]; then
    pass "execute-review scope creep → exit code 1 (NEEDS_WORK)"
else
    fail "Expected exit code 1 for scope creep NEEDS_WORK, got $EXIT_CODE"
fi

# Verify feedback mentions scope
SCOPE_FEEDBACK=$(printf '%s\n' "$SCOPE_OUTPUT" | jq -r '.feedback' 2>/dev/null || echo "")
if echo "$SCOPE_FEEDBACK" | grep -qi "scope"; then
    pass "Scope creep feedback contains 'scope' reference"
else
    fail "Scope creep feedback doesn't mention scope: $SCOPE_FEEDBACK"
fi

unset MOCK_RESPONSE_FILE

# E4: execute-review finds quality issues -- returns NEEDS_WORK
echo "[E4] execute-review detects quality issues and returns NEEDS_WORK"
# Given: mock claude returns NEEDS_WORK due to code quality
cat > "$RESPONSE_DIR/execute-review-quality.txt" << 'RESP_EOF'
Running tests... all pass.
Reviewing code quality...

Quality issues found in implementation.

{"status":"NEEDS_WORK","feedback":"Code quality issues: (1) Function ValidateEmail has a TODO comment indicating incomplete work. (2) Error messages use fmt.Sprintf but don't include the invalid input value. (3) Dead code: unused helper function isASCII on line 45.","files_changed":["worklog.md"],"summary":"Implementation needs quality fixes - TODO comments, error messages, dead code"}
RESP_EOF

export MOCK_RESPONSE_FILE="$RESPONSE_DIR/execute-review-quality.txt"
EXIT_CODE=0
QUALITY_OUTPUT=$("$RUN_PHASE" execute-review "$WORKTREE_DIR" 2>/dev/null) || EXIT_CODE=$?
if [ "$EXIT_CODE" -eq 1 ]; then
    pass "execute-review quality issues → exit code 1 (NEEDS_WORK)"
else
    fail "Expected exit code 1 for quality NEEDS_WORK, got $EXIT_CODE"
fi

unset MOCK_RESPONSE_FILE

# E5: Signal JSON has all required fields
echo "[E5] Signal JSON has all required fields"
# Given: mock claude returns a PASS signal
export MOCK_RESPONSE_FILE="$RESPONSE_DIR/execute-pass.txt"
# When: signal output is parsed
SIGNAL_OUTPUT=$("$RUN_PHASE" execute "$WORKTREE_DIR" 2>/dev/null) || true
# Then: all required fields are present and files_changed is an array

FIELDS_OK=true
for field in status feedback files_changed summary; do
    if [ "$(printf '%s\n' "$SIGNAL_OUTPUT" | jq "has(\"$field\")")" != "true" ]; then
        fail "Signal missing required field: $field"
        FIELDS_OK=false
    fi
done
if [ "$FIELDS_OK" = true ]; then
    pass "Signal JSON contains all required fields (status, feedback, files_changed, summary)"
fi

IS_ARRAY=$(printf '%s\n' "$SIGNAL_OUTPUT" | jq '.files_changed | type == "array"')
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
