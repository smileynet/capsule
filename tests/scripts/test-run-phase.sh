#!/usr/bin/env bash
# Test script for cap-8ax.3.4: run-phase.sh
# Validates: prompt loading, claude invocation, signal parsing, exit codes,
#            feedback flag, error handling.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
RUN_PHASE="$REPO_ROOT/scripts/run-phase.sh"

PASS=0
FAIL=0

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); echo "  FAIL: $1"; }

# --- Prerequisite checks ---
if ! command -v jq >/dev/null 2>&1; then
    echo "ERROR: jq is required but not installed" >&2
    exit 1
fi

if [ ! -f "$RUN_PHASE" ]; then
    echo "ERROR: run-phase.sh not found at $RUN_PHASE" >&2
    exit 1
fi

if [ ! -x "$RUN_PHASE" ]; then
    echo "ERROR: run-phase.sh is not executable" >&2
    exit 1
fi

# --- Create test environment ---
echo "=== Setting up test environment ==="
WORK_DIR=$(mktemp -d)
trap 'rm -rf "$WORK_DIR"' EXIT

# Create a minimal worktree directory with worklog.md and CLAUDE.md
WORKTREE="$WORK_DIR/worktree"
mkdir -p "$WORKTREE"
echo "# Worklog" > "$WORKTREE/worklog.md"
echo "# Project" > "$WORKTREE/CLAUDE.md"

# Create mock claude binary that returns configurable signals
MOCK_DIR="$WORK_DIR/mock-bin"
mkdir -p "$MOCK_DIR"

# Mock claude script: supports both response mode and prompt-capture mode.
# Set MOCK_CAPTURE_FILE to capture the prompt passed via -p.
# Set MOCK_CLAUDE_RESPONSE for the response to output.
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

echo "  Work dir: $WORK_DIR"
echo "  Worktree: $WORKTREE"
echo "  Mock claude: $MOCK_DIR/claude"

echo ""
echo "=== cap-8ax.3.4: run-phase.sh ==="
echo ""

# ---------- Test 1: PASS signal → exit code 0 ----------
echo "[1/9] PASS signal returns exit code 0"
# Given: mock claude returns a PASS signal
export MOCK_CLAUDE_RESPONSE='Reading worklog.md...
Writing tests...
{"status":"PASS","feedback":"All tests written","files_changed":["src/test.go"],"summary":"Tests created"}'

# When: run-phase.sh is invoked
EXIT_CODE=0
OUTPUT=$("$RUN_PHASE" test-writer "$WORKTREE" 2>&1) || EXIT_CODE=$?
# Then: exit code is 0
if [ "$EXIT_CODE" -eq 0 ]; then
    pass "PASS signal → exit code 0"
else
    fail "Expected exit code 0 for PASS signal, got $EXIT_CODE"
    echo "  Output: $OUTPUT"
fi

# ---------- Test 2: NEEDS_WORK signal → exit code 1 ----------
echo "[2/9] NEEDS_WORK signal returns exit code 1"
# Given: mock claude returns a NEEDS_WORK signal
export MOCK_CLAUDE_RESPONSE='Reviewing tests...
{"status":"NEEDS_WORK","feedback":"Missing edge case test","files_changed":["worklog.md"],"summary":"Tests need improvement"}'

# When: run-phase.sh is invoked
EXIT_CODE=0
OUTPUT=$("$RUN_PHASE" test-writer "$WORKTREE" 2>&1) || EXIT_CODE=$?
# Then: exit code is 1
if [ "$EXIT_CODE" -eq 1 ]; then
    pass "NEEDS_WORK signal → exit code 1"
else
    fail "Expected exit code 1 for NEEDS_WORK signal, got $EXIT_CODE"
    echo "  Output: $OUTPUT"
fi

# ---------- Test 3: ERROR signal → exit code 2 ----------
echo "[3/9] ERROR signal returns exit code 2"
# Given: mock claude returns an ERROR signal
export MOCK_CLAUDE_RESPONSE='Something went wrong
{"status":"ERROR","feedback":"Could not read worklog","files_changed":[],"summary":"Phase failed"}'

# When: run-phase.sh is invoked
EXIT_CODE=0
OUTPUT=$("$RUN_PHASE" test-writer "$WORKTREE" 2>&1) || EXIT_CODE=$?
# Then: exit code is 2
if [ "$EXIT_CODE" -eq 2 ]; then
    pass "ERROR signal → exit code 2"
else
    fail "Expected exit code 2 for ERROR signal, got $EXIT_CODE"
    echo "  Output: $OUTPUT"
fi

# ---------- Test 4: No JSON signal → exit code 2 ----------
echo "[4/9] No JSON signal returns exit code 2"
# Given: mock claude returns plain text with no JSON
export MOCK_CLAUDE_RESPONSE='Just some text output with no JSON'

# When: run-phase.sh is invoked
EXIT_CODE=0
OUTPUT=$("$RUN_PHASE" test-writer "$WORKTREE" 2>&1) || EXIT_CODE=$?
# Then: exit code is 2 (graceful degradation)
if [ "$EXIT_CODE" -eq 2 ]; then
    pass "No JSON signal → exit code 2 (graceful degradation)"
else
    fail "Expected exit code 2 for missing signal, got $EXIT_CODE"
    echo "  Output: $OUTPUT"
fi

# ---------- Test 5: --feedback flag appends to prompt ----------
echo "[5/9] --feedback flag appends feedback to prompt"
# Given: mock claude with prompt capture enabled
CAPTURE_FILE="$WORK_DIR/captured-prompt.txt"
export MOCK_CAPTURE_FILE="$CAPTURE_FILE"
export MOCK_CLAUDE_RESPONSE='{"status":"PASS","feedback":"done","files_changed":[],"summary":"ok"}'

# When: run-phase.sh is invoked with --feedback
"$RUN_PHASE" test-writer "$WORKTREE" --feedback="Missing edge case for empty input" >/dev/null 2>&1 || true

# Then: captured prompt contains both base template and feedback text
if [ -f "$CAPTURE_FILE" ]; then
    CAPTURED=$(cat "$CAPTURE_FILE")
    FEEDBACK_OK=true
    # Verify feedback text is present
    if ! echo "$CAPTURED" | grep -qF "Missing edge case for empty input"; then
        fail "Feedback text not found in prompt"
        echo "  Captured prompt: $CAPTURED"
        FEEDBACK_OK=false
    fi
    # Verify base prompt template content is also present
    if ! echo "$CAPTURED" | grep -qF "test-writer"; then
        fail "Base prompt template content not found in prompt"
        echo "  Captured prompt: $CAPTURED"
        FEEDBACK_OK=false
    fi
    if [ "$FEEDBACK_OK" = true ]; then
        pass "Feedback text appended to prompt alongside base template"
    fi
else
    fail "Prompt capture file not created"
fi
rm -f "$CAPTURE_FILE"
unset MOCK_CAPTURE_FILE

# ---------- Test 6: Invalid phase-name → error ----------
echo "[6/9] Invalid phase-name rejected"
# Given: a phase name with no matching prompt template
export MOCK_CLAUDE_RESPONSE='{"status":"PASS","feedback":"ok","files_changed":[],"summary":"ok"}'

# When: run-phase.sh is invoked with invalid phase
EXIT_CODE=0
OUTPUT=$("$RUN_PHASE" nonexistent-phase "$WORKTREE" 2>&1) || EXIT_CODE=$?
# Then: non-zero exit with descriptive error message
if [ "$EXIT_CODE" -ne 0 ]; then
    if echo "$OUTPUT" | grep -qiE "not found|unknown|invalid|no prompt"; then
        pass "Invalid phase-name rejected with descriptive error"
    else
        fail "Invalid phase-name rejected but no descriptive error message"
        echo "  Output: $OUTPUT"
    fi
else
    fail "Expected non-zero exit for invalid phase-name, got 0"
fi

# ---------- Test 7: Output captured to .capsule/output log ----------
echo "[7/9] Output captured to .capsule/output log"
# Given: mock claude returns output with a PASS signal
export MOCK_CLAUDE_RESPONSE='Phase output here
{"status":"PASS","feedback":"done","files_changed":[],"summary":"complete"}'

# Clean any logs from previous tests so we can identify this run's log
rm -rf "$WORKTREE/.capsule/output"

# When: run-phase.sh is invoked
"$RUN_PHASE" test-writer "$WORKTREE" >/dev/null 2>&1 || true

# Then: output is captured to .capsule/output log file
LOG_DIR="$WORKTREE/.capsule/output"
if [ -d "$LOG_DIR" ]; then
    LOG_FILE=$(find "$LOG_DIR" -name "test-writer-*.log" ! -name "*.stderr" 2>/dev/null | head -1)
    if [ -n "$LOG_FILE" ]; then
        if grep -qF "Phase output here" "$LOG_FILE"; then
            pass "Output captured to .capsule/output/test-writer-*.log"
        else
            fail "Log file exists but missing expected content"
        fi
    else
        fail "No test-writer log files found in $LOG_DIR"
    fi
else
    fail ".capsule/output directory not found at $LOG_DIR"
fi

# ---------- Test 8: Signal JSON accessible in output ----------
echo "[8/9] Signal JSON printed to stdout"
# Given: mock claude returns logs followed by a PASS signal
export MOCK_CLAUDE_RESPONSE='Some logs
{"status":"PASS","feedback":"all good","files_changed":["a.go"],"summary":"done"}'

# When: run-phase.sh is invoked
OUTPUT=$("$RUN_PHASE" test-writer "$WORKTREE" 2>/dev/null) || true
# Then: parsed signal JSON is printed to stdout
if echo "$OUTPUT" | jq -e '.status' >/dev/null 2>&1; then
    STATUS=$(echo "$OUTPUT" | jq -r '.status')
    if [ "$STATUS" = "PASS" ]; then
        pass "Parsed signal JSON accessible in stdout"
    else
        fail "Signal in stdout has unexpected status: $STATUS"
    fi
else
    fail "No valid JSON signal found in stdout"
    echo "  Output: $OUTPUT"
fi

# ---------- Test 9: Claude crash (non-zero exit) → exit code 2 ----------
echo "[9/9] Claude crash returns exit code 2"
# Given: mock claude crashes with exit code 1
export MOCK_CLAUDE_RESPONSE='Partial output before crash'
export MOCK_CLAUDE_EXIT=1

# When: run-phase.sh is invoked
EXIT_CODE=0
OUTPUT=$("$RUN_PHASE" test-writer "$WORKTREE" 2>&1) || EXIT_CODE=$?
# Then: exit code is 2
if [ "$EXIT_CODE" -eq 2 ]; then
    pass "Claude crash → exit code 2"
else
    fail "Expected exit code 2 for claude crash, got $EXIT_CODE"
    echo "  Output: $OUTPUT"
fi
unset MOCK_CLAUDE_EXIT

# ---------- Edge Cases ----------
echo ""
echo "=== Edge Cases ==="

# E1: No arguments shows usage error
echo "[E1] No arguments shows usage error"
# Given: no arguments provided
# When: run-phase.sh is invoked with no args
EXIT_CODE=0
OUTPUT=$("$RUN_PHASE" 2>&1) || EXIT_CODE=$?
# Then: non-zero exit with usage message
if [ "$EXIT_CODE" -ne 0 ]; then
    if echo "$OUTPUT" | grep -qiE "usage|phase-name|required"; then
        pass "No arguments: rejected with usage message"
    else
        fail "No arguments: rejected but no usage message"
        echo "  Output: $OUTPUT"
    fi
else
    fail "Expected non-zero exit with no arguments"
fi

# E2: Missing worktree-path argument
echo "[E2] Missing worktree-path shows error"
# Given: phase name provided but no worktree path
# When: run-phase.sh is invoked with only phase name
EXIT_CODE=0
OUTPUT=$("$RUN_PHASE" test-writer 2>&1) || EXIT_CODE=$?
# Then: non-zero exit with descriptive error
if [ "$EXIT_CODE" -ne 0 ]; then
    if echo "$OUTPUT" | grep -qiE "usage|worktree|required|path"; then
        pass "Missing worktree-path: rejected with error message"
    else
        fail "Missing worktree-path: rejected but no descriptive error"
        echo "  Output: $OUTPUT"
    fi
else
    fail "Expected non-zero exit with missing worktree-path"
fi

# E3: Non-existent worktree path
echo "[E3] Non-existent worktree path rejected"
# Given: a worktree path that does not exist
# When: run-phase.sh is invoked with non-existent path
EXIT_CODE=0
OUTPUT=$("$RUN_PHASE" test-writer "$WORK_DIR/nonexistent-worktree" 2>&1) || EXIT_CODE=$?
# Then: non-zero exit with descriptive error
if [ "$EXIT_CODE" -ne 0 ]; then
    if echo "$OUTPUT" | grep -qiE "not found|does not exist|no such|invalid"; then
        pass "Non-existent worktree path rejected with descriptive error"
    else
        fail "Non-existent worktree path rejected but no descriptive error"
        echo "  Output: $OUTPUT"
    fi
else
    fail "Expected non-zero exit for non-existent worktree path"
fi

echo ""
echo "==========================================="
echo "RESULTS: $PASS passed, $FAIL failed"
echo "==========================================="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
