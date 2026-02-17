#!/usr/bin/env bash
# Test script for demo-greenfield issues.jsonl bead fixtures.
# Verifies: JSONL parsing, bd import, hierarchy validity, parent-chain resolution.
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

echo "=== Test: JSONL has exactly 6 lines ==="
LINE_COUNT=$(wc -l < "$JSONL")
if [ "$LINE_COUNT" -eq 6 ]; then
    pass "6 lines (6 beads)"
else
    fail "expected 6 lines, got $LINE_COUNT"
fi

echo "=== Test: Each line is valid JSON ==="
LINE_NUM=0
ALL_VALID=true
while IFS= read -r line; do
    LINE_NUM=$((LINE_NUM + 1))
    if ! printf '%s\n' "$line" | jq empty 2>/dev/null; then
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
    ID=$(printf '%s\n' "$line" | jq -r '.id')
    for field in id title description issue_type status priority dependencies; do
        if ! printf '%s\n' "$line" | jq -e ".$field" >/dev/null 2>&1; then
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
if [ "$EPIC_COUNT" -eq 2 ] && [ "$FEATURE_COUNT" -eq 1 ] && [ "$TASK_COUNT" -eq 3 ]; then
    pass "2 epics, 1 feature, 3 tasks"
else
    fail "expected 2 epics, 1 feature, 3 tasks; got $EPIC_COUNT epics, $FEATURE_COUNT features, $TASK_COUNT tasks"
fi

echo "=== Test: Hierarchy via parent-child dependencies ==="
# Feature should have parent-child dep on epic
FEAT_PARENT=$(jq -s '[.[] | select(.id=="demo-001.1")] | .[0].dependencies[] | select(.type=="parent-child") | .depends_on_id' "$JSONL")
if [ "$FEAT_PARENT" = '"demo-001"' ]; then
    pass "feature parent-child -> epic"
else
    fail "feature parent-child dep expected demo-001, got $FEAT_PARENT"
fi

# Tasks should have parent-child dep on feature
TASK1_PARENT=$(jq -s '[.[] | select(.id=="demo-001.1.1")] | .[0].dependencies[] | select(.type=="parent-child") | .depends_on_id' "$JSONL")
TASK2_PARENT=$(jq -s '[.[] | select(.id=="demo-001.1.2")] | .[0].dependencies[] | select(.type=="parent-child") | .depends_on_id' "$JSONL")
if [ "$TASK1_PARENT" = '"demo-001.1"' ] && [ "$TASK2_PARENT" = '"demo-001.1"' ]; then
    pass "tasks parent-child -> feature"
else
    fail "task parent-child deps expected demo-001.1, got $TASK1_PARENT and $TASK2_PARENT"
fi

# Parking lot task should have parent-child dep on parking lot epic
PARK_PARENT=$(jq -s '[.[] | select(.id=="demo-100.1")] | .[0].dependencies[] | select(.type=="parent-child") | .depends_on_id' "$JSONL")
if [ "$PARK_PARENT" = '"demo-100"' ]; then
    pass "parking-lot task parent-child -> parking-lot epic"
else
    fail "parking-lot parent-child dep expected demo-100, got $PARK_PARENT"
fi

echo "=== Test: Blocking dependency ==="
# demo-001.1.2 should have blocks dep on demo-001.1.1
BLOCKS_DEP=$(jq -s '[.[] | select(.id=="demo-001.1.2")] | .[0].dependencies[] | select(.type=="blocks") | .depends_on_id' "$JSONL")
if [ "$BLOCKS_DEP" = '"demo-001.1.1"' ]; then
    pass "demo-001.1.2 blocks dep on demo-001.1.1"
else
    fail "blocks dep expected demo-001.1.1, got $BLOCKS_DEP"
fi

echo "=== Test: No shorthand parent/depends_on fields ==="
SHORTHAND_OK=true
while IFS= read -r line; do
    ID=$(printf '%s\n' "$line" | jq -r '.id')
    if printf '%s\n' "$line" | jq -e '.parent' >/dev/null 2>&1; then
        fail "$ID has shorthand 'parent' field"
        SHORTHAND_OK=false
    fi
    if printf '%s\n' "$line" | jq -e '.depends_on' >/dev/null 2>&1; then
        fail "$ID has shorthand 'depends_on' field"
        SHORTHAND_OK=false
    fi
done < "$JSONL"
if [ "$SHORTHAND_OK" = true ]; then
    pass "no shorthand parent/depends_on fields"
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

# Verify imported beads
READY_OUTPUT=$(cd "$TMPDIR" && bd list 2>&1)
BEAD_COUNT=$(echo "$READY_OUTPUT" | grep -c "demo-" || true)
if [ "$BEAD_COUNT" -eq 6 ]; then
    pass "all 6 beads imported"
else
    fail "expected 6 beads after import, found $BEAD_COUNT"
    echo "$READY_OUTPUT"
fi

echo "=== Test: Parent chain resolves after import ==="
# Verify task has parent set to feature
TASK_PARENT=$(cd "$TMPDIR" && bd show demo-001.1.1 --json 2>/dev/null | jq -r '.[0].parent // empty')
if [ "$TASK_PARENT" = "demo-001.1" ]; then
    pass "demo-001.1.1 parent resolves to demo-001.1"
else
    fail "demo-001.1.1 parent expected demo-001.1, got '$TASK_PARENT'"
fi

# Verify feature has parent set to epic
FEAT_PARENT_IMPORTED=$(cd "$TMPDIR" && bd show demo-001.1 --json 2>/dev/null | jq -r '.[0].parent // empty')
if [ "$FEAT_PARENT_IMPORTED" = "demo-001" ]; then
    pass "demo-001.1 parent resolves to demo-001"
else
    fail "demo-001.1 parent expected demo-001, got '$FEAT_PARENT_IMPORTED'"
fi

# Verify children query works
CHILDREN_COUNT=$(cd "$TMPDIR" && bd list --parent=demo-001.1 --all --json 2>/dev/null | jq 'length')
if [ "$CHILDREN_COUNT" -eq 2 ]; then
    pass "demo-001.1 has 2 children"
else
    fail "demo-001.1 expected 2 children, got $CHILDREN_COUNT"
fi

echo ""
echo "==========================================="
echo "RESULTS: $PASS passed, $FAIL failed"
echo "==========================================="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
