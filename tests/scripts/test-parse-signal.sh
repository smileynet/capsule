#!/usr/bin/env bash
# Test script for cap-8ax.3.1: parse-signal.sh
# Validates: JSON extraction from mixed output, validation, error handling.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
PARSE_SCRIPT="$REPO_ROOT/scripts/parse-signal.sh"

PASS=0
FAIL=0

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); echo "  FAIL: $1"; }

# --- Prerequisite checks ---
if ! command -v jq >/dev/null 2>&1; then
    echo "ERROR: jq is required but not installed" >&2
    exit 1
fi

if [ ! -f "$PARSE_SCRIPT" ]; then
    echo "ERROR: parse-signal.sh not found at $PARSE_SCRIPT" >&2
    exit 1
fi

if [ ! -x "$PARSE_SCRIPT" ]; then
    echo "ERROR: parse-signal.sh is not executable" >&2
    exit 1
fi

echo "=== cap-8ax.3.1: parse-signal.sh ==="
echo ""

# ---------- Test 1: Parse valid JSON signal from clean input ----------
echo "[1/8] Parse valid JSON signal from clean input"
VALID_JSON='{"status":"PASS","feedback":"All tests pass","files_changed":["src/main.go"],"summary":"Tests green"}'
RESULT=$(echo "$VALID_JSON" | "$PARSE_SCRIPT" 2>&1)
STATUS=$(echo "$RESULT" | jq -r '.status')
if [ "$STATUS" = "PASS" ]; then
    pass "Parsed valid PASS signal"
else
    fail "Expected status PASS, got '$STATUS'"
    echo "  Result: $RESULT"
fi

# ---------- Test 2: Parse JSON from mixed output (text before signal) ----------
echo "[2/8] Parse JSON from mixed output"
MIXED_OUTPUT="Reading worklog.md...
Found 3 acceptance criteria.
Writing test file: src/validation_test.go

{\"status\":\"NEEDS_WORK\",\"feedback\":\"Missing test for edge case\",\"files_changed\":[\"src/test.go\"],\"summary\":\"Tests incomplete\"}"
RESULT=$(echo "$MIXED_OUTPUT" | "$PARSE_SCRIPT" 2>&1)
STATUS=$(echo "$RESULT" | jq -r '.status')
FEEDBACK=$(echo "$RESULT" | jq -r '.feedback')
if [ "$STATUS" = "NEEDS_WORK" ] && [ "$FEEDBACK" = "Missing test for edge case" ]; then
    pass "Parsed signal from mixed text+JSON output"
else
    fail "Expected NEEDS_WORK with feedback, got status='$STATUS' feedback='$FEEDBACK'"
fi

# ---------- Test 3: Handle missing JSON (no JSON in output) ----------
echo "[3/8] Handle missing JSON - returns ERROR signal"
NO_JSON_OUTPUT="This is just plain text output
with no JSON anywhere
just logs and messages"
RESULT=$(echo "$NO_JSON_OUTPUT" | "$PARSE_SCRIPT" 2>&1) || true
STATUS=$(echo "$RESULT" | jq -r '.status')
if [ "$STATUS" = "ERROR" ]; then
    pass "Missing JSON returns ERROR status"
else
    fail "Expected ERROR for missing JSON, got '$STATUS'"
    echo "  Result: $RESULT"
fi

# ---------- Test 4: Extract last JSON block when multiple exist ----------
echo "[4/8] Extract last JSON block from multiple"
MULTI_JSON="Some log output
{\"status\":\"NEEDS_WORK\",\"feedback\":\"first attempt\",\"files_changed\":[],\"summary\":\"first\"}
More log output
{\"status\":\"PASS\",\"feedback\":\"second attempt works\",\"files_changed\":[\"a.go\"],\"summary\":\"final\"}"
RESULT=$(echo "$MULTI_JSON" | "$PARSE_SCRIPT" 2>&1)
STATUS=$(echo "$RESULT" | jq -r '.status')
SUMMARY=$(echo "$RESULT" | jq -r '.summary')
if [ "$STATUS" = "PASS" ] && [ "$SUMMARY" = "final" ]; then
    pass "Extracted last JSON block from multiple"
else
    fail "Expected last block (PASS/final), got status='$STATUS' summary='$SUMMARY'"
fi

# ---------- Test 5: Validate required fields - missing status ----------
echo "[5/8] Reject JSON missing required 'status' field"
BAD_JSON='{"feedback":"ok","files_changed":[],"summary":"done"}'
RESULT=$(echo "$BAD_JSON" | "$PARSE_SCRIPT" 2>&1) || true
STATUS=$(echo "$RESULT" | jq -r '.status')
if [ "$STATUS" = "ERROR" ]; then
    pass "Missing 'status' field returns ERROR"
else
    fail "Expected ERROR for missing status, got '$STATUS'"
fi

# ---------- Test 6: Validate status value - invalid status ----------
echo "[6/8] Reject invalid status value"
BAD_STATUS='{"status":"UNKNOWN","feedback":"ok","files_changed":[],"summary":"done"}'
RESULT=$(echo "$BAD_STATUS" | "$PARSE_SCRIPT" 2>&1) || true
STATUS=$(echo "$RESULT" | jq -r '.status')
if [ "$STATUS" = "ERROR" ]; then
    pass "Invalid status value returns ERROR"
else
    fail "Expected ERROR for invalid status, got '$STATUS'"
fi

# ---------- Test 7: Validate files_changed is array ----------
echo "[7/8] Reject non-array files_changed"
BAD_FILES='{"status":"PASS","feedback":"ok","files_changed":"not-an-array","summary":"done"}'
RESULT=$(echo "$BAD_FILES" | "$PARSE_SCRIPT" 2>&1) || true
STATUS=$(echo "$RESULT" | jq -r '.status')
if [ "$STATUS" = "ERROR" ]; then
    pass "Non-array files_changed returns ERROR"
else
    fail "Expected ERROR for non-array files_changed, got '$STATUS'"
fi

# ---------- Test 8: Handle empty input ----------
echo "[8/8] Handle empty input"
RESULT=$(echo "" | "$PARSE_SCRIPT" 2>&1) || true
STATUS=$(echo "$RESULT" | jq -r '.status')
if [ "$STATUS" = "ERROR" ]; then
    pass "Empty input returns ERROR status"
else
    fail "Expected ERROR for empty input, got '$STATUS'"
fi

# ---------- Edge Cases ----------
echo ""
echo "=== Edge Cases ==="

# E1: JSON embedded in markdown code block
echo "[E1] JSON in markdown code block"
MARKDOWN_OUTPUT='Here is the result:
```json
{"status":"PASS","feedback":"done","files_changed":["x.go"],"summary":"ok"}
```'
RESULT=$(echo "$MARKDOWN_OUTPUT" | "$PARSE_SCRIPT" 2>&1)
STATUS=$(echo "$RESULT" | jq -r '.status')
if [ "$STATUS" = "PASS" ]; then
    pass "Extracted JSON from markdown code block context"
else
    fail "Expected PASS from markdown context, got '$STATUS'"
    echo "  Result: $RESULT"
fi

# E2: ERROR status passes through
echo "[E2] ERROR status passes through"
ERROR_JSON='{"status":"ERROR","feedback":"claude crashed","files_changed":[],"summary":"phase failed"}'
RESULT=$(echo "$ERROR_JSON" | "$PARSE_SCRIPT" 2>&1)
STATUS=$(echo "$RESULT" | jq -r '.status')
FEEDBACK=$(echo "$RESULT" | jq -r '.feedback')
if [ "$STATUS" = "ERROR" ] && [ "$FEEDBACK" = "claude crashed" ]; then
    pass "ERROR status passes through with original feedback"
else
    fail "Expected ERROR pass-through, got status='$STATUS' feedback='$FEEDBACK'"
fi

# E3: files_changed with empty array
echo "[E3] Empty files_changed array accepted"
EMPTY_FILES='{"status":"PASS","feedback":"review only","files_changed":[],"summary":"no changes"}'
RESULT=$(echo "$EMPTY_FILES" | "$PARSE_SCRIPT" 2>&1)
STATUS=$(echo "$RESULT" | jq -r '.status')
FILES_COUNT=$(echo "$RESULT" | jq '.files_changed | length')
if [ "$STATUS" = "PASS" ] && [ "$FILES_COUNT" = "0" ]; then
    pass "Empty files_changed array accepted"
else
    fail "Expected PASS with 0 files, got status='$STATUS' count='$FILES_COUNT'"
fi

echo ""
echo "==========================================="
echo "RESULTS: $PASS passed, $FAIL failed"
echo "==========================================="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
