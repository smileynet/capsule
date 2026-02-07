#!/usr/bin/env bash
# Test script for issues.jsonl bead fixtures.
# Verifies: JSONL parsing, bd import, hierarchy validity.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
JSONL="$SCRIPT_DIR/issues.jsonl"
PASS=0
FAIL=0

pass() { echo "  PASS: $1"; PASS=$((PASS + 1)); }
fail() { echo "  FAIL: $1"; FAIL=$((FAIL + 1)); }

for cmd in jq git bd; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "ERROR: $cmd is required but not installed"
        exit 1
    fi
done

echo "=== Test: JSONL file exists ==="
if [ -f "$JSONL" ]; then
    pass "issues.jsonl exists"
else
    fail "issues.jsonl not found"
    echo "RESULTS: $PASS passed, $FAIL failed"
    exit 1
fi

echo "=== Test: JSONL has exactly 4 lines ==="
LINE_COUNT=$(wc -l < "$JSONL")
if [ "$LINE_COUNT" -eq 4 ]; then
    pass "4 lines (4 beads)"
else
    fail "expected 4 lines, got $LINE_COUNT"
fi

echo "=== Test: Each line is valid JSON ==="
LINE_NUM=0
ALL_VALID=true
while IFS= read -r line; do
    LINE_NUM=$((LINE_NUM + 1))
    if ! echo "$line" | jq empty 2>/dev/null; then
        fail "line $LINE_NUM is not valid JSON"
        ALL_VALID=false
    fi
done < "$JSONL"
if [ "$ALL_VALID" = true ]; then
    pass "all lines are valid JSON"
fi

echo "=== Test: Required fields present ==="
FIELDS_OK=true
while IFS= read -r line; do
    ID=$(echo "$line" | jq -r '.id')
    for field in id title description issue_type status priority dependencies; do
        if ! echo "$line" | jq -e ".$field" >/dev/null 2>&1; then
            fail "$ID missing field: $field"
            FIELDS_OK=false
        fi
    done
done < "$JSONL"
if [ "$FIELDS_OK" = true ]; then
    pass "all required fields present"
fi

echo "=== Test: Issue types correct ==="
EPIC_COUNT=$(jq -s '[.[] | select(.issue_type=="epic")] | length' "$JSONL")
FEATURE_COUNT=$(jq -s '[.[] | select(.issue_type=="feature")] | length' "$JSONL")
TASK_COUNT=$(jq -s '[.[] | select(.issue_type=="task")] | length' "$JSONL")
if [ "$EPIC_COUNT" -eq 1 ] && [ "$FEATURE_COUNT" -eq 1 ] && [ "$TASK_COUNT" -eq 2 ]; then
    pass "1 epic, 1 feature, 2 tasks"
else
    fail "expected 1 epic, 1 feature, 2 tasks; got $EPIC_COUNT epic, $FEATURE_COUNT feature, $TASK_COUNT tasks"
fi

echo "=== Test: Hierarchy via parent-child dependencies ==="
# Feature should have parent-child dep on epic (lookup by id, not position)
FEAT_PARENT=$(jq -s '[.[] | select(.id=="demo-1.1")] | .[0].dependencies[] | select(.type=="parent-child") | .depends_on_id' "$JSONL")
if [ "$FEAT_PARENT" = '"demo-1"' ]; then
    pass "feature parent-child -> epic"
else
    fail "feature parent-child dep expected demo-1, got $FEAT_PARENT"
fi

# Tasks should have parent-child dep on feature (lookup by id)
TASK1_PARENT=$(jq -s '[.[] | select(.id=="demo-1.1.1")] | .[0].dependencies[] | select(.type=="parent-child") | .depends_on_id' "$JSONL")
TASK2_PARENT=$(jq -s '[.[] | select(.id=="demo-1.1.2")] | .[0].dependencies[] | select(.type=="parent-child") | .depends_on_id' "$JSONL")
if [ "$TASK1_PARENT" = '"demo-1.1"' ] && [ "$TASK2_PARENT" = '"demo-1.1"' ]; then
    pass "tasks parent-child -> feature"
else
    fail "task parent-child deps expected demo-1.1, got $TASK1_PARENT and $TASK2_PARENT"
fi

echo "=== Test: Acceptance criteria in task descriptions ==="
TASK1_DESC=$(jq -s '[.[] | select(.id=="demo-1.1.1")] | .[0].description' "$JSONL")
TASK2_DESC=$(jq -s '[.[] | select(.id=="demo-1.1.2")] | .[0].description' "$JSONL")
if echo "$TASK1_DESC" | grep -q "Acceptance criteria" && echo "$TASK2_DESC" | grep -q "Acceptance criteria"; then
    pass "tasks have acceptance criteria in descriptions"
else
    fail "tasks missing acceptance criteria in descriptions"
fi

echo "=== Test: bd import in temp environment ==="
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
IMPORT_OUTPUT="$TMPDIR/bd-import-output.txt"
(
    cd "$TMPDIR"
    git init -q
    git config user.email "test@example.com"
    git config user.name "Test"
    git commit --allow-empty -m "init" -q
    bd init --prefix=demo -q 2>/dev/null || bd init --prefix=demo 2>/dev/null
    bd import -i "$JSONL" 2>&1
) > "$IMPORT_OUTPUT" 2>&1
IMPORT_EXIT=$?
if [ "$IMPORT_EXIT" -eq 0 ]; then
    pass "bd import succeeded"
else
    fail "bd import failed (exit $IMPORT_EXIT)"
    cat "$IMPORT_OUTPUT"
fi

# Verify imported beads (no subshell — cd in pipeline to avoid losing counters)
READY_OUTPUT=$(cd "$TMPDIR" && bd list 2>&1)
BEAD_COUNT=$(echo "$READY_OUTPUT" | grep -c "demo-" || true)
if [ "$BEAD_COUNT" -eq 4 ]; then
    pass "all 4 beads imported"
else
    fail "expected 4 beads after import, found $BEAD_COUNT"
    echo "$READY_OUTPUT"
fi

echo ""
echo "==========================================="
echo "RESULTS: $PASS passed, $FAIL failed"
echo "==========================================="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
