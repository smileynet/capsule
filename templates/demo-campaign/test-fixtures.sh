#!/usr/bin/env bash
# Test script for demo-campaign issues.jsonl bead fixtures.
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

echo "=== Test: JSONL has exactly 7 lines ==="
LINE_COUNT=$(wc -l < "$JSONL")
if [ "$LINE_COUNT" -eq 7 ]; then
    pass "7 lines (7 beads)"
else
    fail "expected 7 lines, got $LINE_COUNT"
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
if [ "$EPIC_COUNT" -eq 1 ] && [ "$FEATURE_COUNT" -eq 2 ] && [ "$TASK_COUNT" -eq 4 ]; then
    pass "1 epic, 2 features, 4 tasks"
else
    fail "expected 1 epic, 2 features, 4 tasks; got $EPIC_COUNT epic, $FEATURE_COUNT feature, $TASK_COUNT tasks"
fi

echo "=== Test: Hierarchy via parent-child dependencies ==="
# Features should have parent-child dep on epic
FEAT1_PARENT=$(jq -s '[.[] | select(.id=="demo-1.1")] | .[0].dependencies[] | select(.type=="parent-child") | .depends_on_id' "$JSONL")
FEAT2_PARENT=$(jq -s '[.[] | select(.id=="demo-1.2")] | .[0].dependencies[] | select(.type=="parent-child") | .depends_on_id' "$JSONL")
if [ "$FEAT1_PARENT" = '"demo-1"' ] && [ "$FEAT2_PARENT" = '"demo-1"' ]; then
    pass "features parent-child -> epic"
else
    fail "feature parent-child deps expected demo-1, got $FEAT1_PARENT and $FEAT2_PARENT"
fi

# Validation tasks should have parent-child dep on feature demo-1.1
TASK1_PARENT=$(jq -s '[.[] | select(.id=="demo-1.1.1")] | .[0].dependencies[] | select(.type=="parent-child") | .depends_on_id' "$JSONL")
TASK2_PARENT=$(jq -s '[.[] | select(.id=="demo-1.1.2")] | .[0].dependencies[] | select(.type=="parent-child") | .depends_on_id' "$JSONL")
if [ "$TASK1_PARENT" = '"demo-1.1"' ] && [ "$TASK2_PARENT" = '"demo-1.1"' ]; then
    pass "validation tasks parent-child -> feature demo-1.1"
else
    fail "validation task parent-child deps expected demo-1.1, got $TASK1_PARENT and $TASK2_PARENT"
fi

# Formatting tasks should have parent-child dep on feature demo-1.2
TASK3_PARENT=$(jq -s '[.[] | select(.id=="demo-1.2.1")] | .[0].dependencies[] | select(.type=="parent-child") | .depends_on_id' "$JSONL")
TASK4_PARENT=$(jq -s '[.[] | select(.id=="demo-1.2.2")] | .[0].dependencies[] | select(.type=="parent-child") | .depends_on_id' "$JSONL")
if [ "$TASK3_PARENT" = '"demo-1.2"' ] && [ "$TASK4_PARENT" = '"demo-1.2"' ]; then
    pass "formatting tasks parent-child -> feature demo-1.2"
else
    fail "formatting task parent-child deps expected demo-1.2, got $TASK3_PARENT and $TASK4_PARENT"
fi

echo "=== Test: Acceptance criteria in task descriptions ==="
AC_OK=true
for task_id in demo-1.1.1 demo-1.1.2 demo-1.2.1 demo-1.2.2; do
    TASK_DESC=$(jq -s "[.[] | select(.id==\"$task_id\")] | .[0].description" "$JSONL")
    if ! echo "$TASK_DESC" | grep -q "Acceptance criteria"; then
        fail "$task_id missing acceptance criteria in description"
        AC_OK=false
    fi
done
if [ "$AC_OK" = true ]; then
    pass "all tasks have acceptance criteria in descriptions"
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
if [ "$BEAD_COUNT" -eq 7 ]; then
    pass "all 7 beads imported"
else
    fail "expected 7 beads after import, found $BEAD_COUNT"
    echo "$READY_OUTPUT"
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

echo "=== Test: Parent chain resolves after import ==="
# Verify tasks have parent set to their feature
TASK1_PARENT_IMPORTED=$(cd "$TMPDIR" && bd show demo-1.1.1 --json 2>/dev/null | jq -r '.[0].parent // empty')
if [ "$TASK1_PARENT_IMPORTED" = "demo-1.1" ]; then
    pass "demo-1.1.1 parent resolves to demo-1.1"
else
    fail "demo-1.1.1 parent expected demo-1.1, got '$TASK1_PARENT_IMPORTED'"
fi

TASK3_PARENT_IMPORTED=$(cd "$TMPDIR" && bd show demo-1.2.1 --json 2>/dev/null | jq -r '.[0].parent // empty')
if [ "$TASK3_PARENT_IMPORTED" = "demo-1.2" ]; then
    pass "demo-1.2.1 parent resolves to demo-1.2"
else
    fail "demo-1.2.1 parent expected demo-1.2, got '$TASK3_PARENT_IMPORTED'"
fi

# Verify features have parent set to epic
FEAT1_PARENT_IMPORTED=$(cd "$TMPDIR" && bd show demo-1.1 --json 2>/dev/null | jq -r '.[0].parent // empty')
if [ "$FEAT1_PARENT_IMPORTED" = "demo-1" ]; then
    pass "demo-1.1 parent resolves to demo-1"
else
    fail "demo-1.1 parent expected demo-1, got '$FEAT1_PARENT_IMPORTED'"
fi

FEAT2_PARENT_IMPORTED=$(cd "$TMPDIR" && bd show demo-1.2 --json 2>/dev/null | jq -r '.[0].parent // empty')
if [ "$FEAT2_PARENT_IMPORTED" = "demo-1" ]; then
    pass "demo-1.2 parent resolves to demo-1"
else
    fail "demo-1.2 parent expected demo-1, got '$FEAT2_PARENT_IMPORTED'"
fi

echo "=== Test: Children queries ==="
# Epic should have 2 children (both features)
EPIC_CHILDREN=$(cd "$TMPDIR" && bd list --parent=demo-1 --all --json 2>/dev/null | jq 'length')
if [ "$EPIC_CHILDREN" -eq 2 ]; then
    pass "demo-1 has 2 children (features)"
else
    fail "demo-1 expected 2 children, got $EPIC_CHILDREN"
fi

# Each feature should have 2 children (tasks)
FEAT1_CHILDREN=$(cd "$TMPDIR" && bd list --parent=demo-1.1 --all --json 2>/dev/null | jq 'length')
if [ "$FEAT1_CHILDREN" -eq 2 ]; then
    pass "demo-1.1 has 2 children (validation tasks)"
else
    fail "demo-1.1 expected 2 children, got $FEAT1_CHILDREN"
fi

FEAT2_CHILDREN=$(cd "$TMPDIR" && bd list --parent=demo-1.2 --all --json 2>/dev/null | jq 'length')
if [ "$FEAT2_CHILDREN" -eq 2 ]; then
    pass "demo-1.2 has 2 children (formatting tasks)"
else
    fail "demo-1.2 expected 2 children, got $FEAT2_CHILDREN"
fi

echo ""
echo "==========================================="
echo "RESULTS: $PASS passed, $FAIL failed"
echo "==========================================="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
