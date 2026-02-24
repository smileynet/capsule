#!/usr/bin/env bash
# Test script for demo-full issues.jsonl bead fixtures.
# Verifies: JSONL parsing, bd import, hierarchy validity, cross-feature deps.
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

echo "=== Test: JSONL has exactly 18 lines ==="
LINE_COUNT=$(wc -l < "$JSONL")
if [ "$LINE_COUNT" -eq 18 ]; then
    pass "18 lines (18 beads)"
else
    fail "expected 18 lines, got $LINE_COUNT"
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
if [ "$EPIC_COUNT" -eq 3 ] && [ "$FEATURE_COUNT" -eq 4 ] && [ "$TASK_COUNT" -eq 11 ]; then
    pass "3 epics, 4 features, 11 tasks"
else
    fail "expected 3 epics, 4 features, 11 tasks; got $EPIC_COUNT epics, $FEATURE_COUNT features, $TASK_COUNT tasks"
fi

echo "=== Test: Hierarchy via parent-child dependencies ==="
# Epic 1 features
FEAT11_PARENT=$(jq -s '[.[] | select(.id=="demo-1.1")] | .[0].dependencies[] | select(.type=="parent-child") | .depends_on_id' "$JSONL")
FEAT12_PARENT=$(jq -s '[.[] | select(.id=="demo-1.2")] | .[0].dependencies[] | select(.type=="parent-child") | .depends_on_id' "$JSONL")
if [ "$FEAT11_PARENT" = '"demo-1"' ] && [ "$FEAT12_PARENT" = '"demo-1"' ]; then
    pass "epic 1 features parent-child -> demo-1"
else
    fail "epic 1 feature parent-child deps expected demo-1, got $FEAT11_PARENT and $FEAT12_PARENT"
fi

# Epic 2 features
FEAT21_PARENT=$(jq -s '[.[] | select(.id=="demo-2.1")] | .[0].dependencies[] | select(.type=="parent-child") | .depends_on_id' "$JSONL")
FEAT22_PARENT=$(jq -s '[.[] | select(.id=="demo-2.2")] | .[0].dependencies[] | select(.type=="parent-child") | .depends_on_id' "$JSONL")
if [ "$FEAT21_PARENT" = '"demo-2"' ] && [ "$FEAT22_PARENT" = '"demo-2"' ]; then
    pass "epic 2 features parent-child -> demo-2"
else
    fail "epic 2 feature parent-child deps expected demo-2, got $FEAT21_PARENT and $FEAT22_PARENT"
fi

# Epic 1 tasks -> features
for task_id in demo-1.1.1 demo-1.1.2 demo-1.1.3; do
    PARENT=$(jq -s "[.[] | select(.id==\"$task_id\")] | .[0].dependencies[] | select(.type==\"parent-child\") | .depends_on_id" "$JSONL")
    if [ "$PARENT" = '"demo-1.1"' ]; then
        pass "$task_id parent-child -> demo-1.1"
    else
        fail "$task_id parent-child expected demo-1.1, got $PARENT"
    fi
done

for task_id in demo-1.2.1 demo-1.2.2; do
    PARENT=$(jq -s "[.[] | select(.id==\"$task_id\")] | .[0].dependencies[] | select(.type==\"parent-child\") | .depends_on_id" "$JSONL")
    if [ "$PARENT" = '"demo-1.2"' ]; then
        pass "$task_id parent-child -> demo-1.2"
    else
        fail "$task_id parent-child expected demo-1.2, got $PARENT"
    fi
done

# Epic 2 tasks -> features
for task_id in demo-2.1.1 demo-2.1.2; do
    PARENT=$(jq -s "[.[] | select(.id==\"$task_id\")] | .[0].dependencies[] | select(.type==\"parent-child\") | .depends_on_id" "$JSONL")
    if [ "$PARENT" = '"demo-2.1"' ]; then
        pass "$task_id parent-child -> demo-2.1"
    else
        fail "$task_id parent-child expected demo-2.1, got $PARENT"
    fi
done

for task_id in demo-2.2.1 demo-2.2.2; do
    PARENT=$(jq -s "[.[] | select(.id==\"$task_id\")] | .[0].dependencies[] | select(.type==\"parent-child\") | .depends_on_id" "$JSONL")
    if [ "$PARENT" = '"demo-2.2"' ]; then
        pass "$task_id parent-child -> demo-2.2"
    else
        fail "$task_id parent-child expected demo-2.2, got $PARENT"
    fi
done

# Epic 100 tasks -> epic
for task_id in demo-100.1 demo-100.2; do
    PARENT=$(jq -s "[.[] | select(.id==\"$task_id\")] | .[0].dependencies[] | select(.type==\"parent-child\") | .depends_on_id" "$JSONL")
    if [ "$PARENT" = '"demo-100"' ]; then
        pass "$task_id parent-child -> demo-100"
    else
        fail "$task_id parent-child expected demo-100, got $PARENT"
    fi
done

echo "=== Test: Cross-feature blocking dependencies ==="
# demo-2.1.2 blocked by demo-2.1.1
BLOCK_211=$(jq -s '[.[] | select(.id=="demo-2.1.2")] | .[0].dependencies[] | select(.type=="blocks") | .depends_on_id' "$JSONL")
if [ "$BLOCK_211" = '"demo-2.1.1"' ]; then
    pass "demo-2.1.2 blocked by demo-2.1.1"
else
    fail "demo-2.1.2 blocks dep expected demo-2.1.1, got $BLOCK_211"
fi

# demo-2.2.2 blocked by demo-2.2.1
BLOCK_221=$(jq -s '[.[] | select(.id=="demo-2.2.2")] | .[0].dependencies[] | select(.type=="blocks") | .depends_on_id' "$JSONL")
if [ "$BLOCK_221" = '"demo-2.2.1"' ]; then
    pass "demo-2.2.2 blocked by demo-2.2.1"
else
    fail "demo-2.2.2 blocks dep expected demo-2.2.1, got $BLOCK_221"
fi

echo "=== Test: Acceptance criteria in task descriptions ==="
AC_OK=true
TASK_IDS=$(jq -s -r '[.[] | select(.issue_type=="task")] | .[].id' "$JSONL")
for task_id in $TASK_IDS; do
    TASK_DESC=$(jq -s "[.[] | select(.id==\"$task_id\")] | .[0].description" "$JSONL")
    if ! echo "$TASK_DESC" | grep -q "Acceptance criteria"; then
        fail "$task_id missing acceptance criteria in description"
        AC_OK=false
    fi
done
if [ "$AC_OK" = true ]; then
    pass "all tasks have acceptance criteria in descriptions"
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
if [ "$BEAD_COUNT" -eq 18 ]; then
    pass "all 18 beads imported"
else
    fail "expected 18 beads after import, found $BEAD_COUNT"
    echo "$READY_OUTPUT"
fi

echo "=== Test: Parent chain resolves after import ==="
# Tasks -> features
TASK111_PARENT=$(cd "$TMPDIR" && bd show demo-1.1.1 --json 2>/dev/null | jq -r '.[0].parent // empty')
if [ "$TASK111_PARENT" = "demo-1.1" ]; then
    pass "demo-1.1.1 parent resolves to demo-1.1"
else
    fail "demo-1.1.1 parent expected demo-1.1, got '$TASK111_PARENT'"
fi

TASK211_PARENT=$(cd "$TMPDIR" && bd show demo-2.1.1 --json 2>/dev/null | jq -r '.[0].parent // empty')
if [ "$TASK211_PARENT" = "demo-2.1" ]; then
    pass "demo-2.1.1 parent resolves to demo-2.1"
else
    fail "demo-2.1.1 parent expected demo-2.1, got '$TASK211_PARENT'"
fi

TASK1001_PARENT=$(cd "$TMPDIR" && bd show demo-100.1 --json 2>/dev/null | jq -r '.[0].parent // empty')
if [ "$TASK1001_PARENT" = "demo-100" ]; then
    pass "demo-100.1 parent resolves to demo-100"
else
    fail "demo-100.1 parent expected demo-100, got '$TASK1001_PARENT'"
fi

# Features -> epics
FEAT11_PARENT_IMPORTED=$(cd "$TMPDIR" && bd show demo-1.1 --json 2>/dev/null | jq -r '.[0].parent // empty')
if [ "$FEAT11_PARENT_IMPORTED" = "demo-1" ]; then
    pass "demo-1.1 parent resolves to demo-1"
else
    fail "demo-1.1 parent expected demo-1, got '$FEAT11_PARENT_IMPORTED'"
fi

FEAT21_PARENT_IMPORTED=$(cd "$TMPDIR" && bd show demo-2.1 --json 2>/dev/null | jq -r '.[0].parent // empty')
if [ "$FEAT21_PARENT_IMPORTED" = "demo-2" ]; then
    pass "demo-2.1 parent resolves to demo-2"
else
    fail "demo-2.1 parent expected demo-2, got '$FEAT21_PARENT_IMPORTED'"
fi

echo "=== Test: Children queries ==="
# Epic 1 should have 2 children (features)
EPIC1_CHILDREN=$(cd "$TMPDIR" && bd list --parent=demo-1 --all --json 2>/dev/null | jq 'length')
if [ "$EPIC1_CHILDREN" -eq 2 ]; then
    pass "demo-1 has 2 children (features)"
else
    fail "demo-1 expected 2 children, got $EPIC1_CHILDREN"
fi

# Epic 2 should have 2 children (features)
EPIC2_CHILDREN=$(cd "$TMPDIR" && bd list --parent=demo-2 --all --json 2>/dev/null | jq 'length')
if [ "$EPIC2_CHILDREN" -eq 2 ]; then
    pass "demo-2 has 2 children (features)"
else
    fail "demo-2 expected 2 children, got $EPIC2_CHILDREN"
fi

# Epic 100 should have 2 children (tasks, no features)
EPIC100_CHILDREN=$(cd "$TMPDIR" && bd list --parent=demo-100 --all --json 2>/dev/null | jq 'length')
if [ "$EPIC100_CHILDREN" -eq 2 ]; then
    pass "demo-100 has 2 children (tasks)"
else
    fail "demo-100 expected 2 children, got $EPIC100_CHILDREN"
fi

# Feature demo-1.1 should have 3 children
FEAT11_CHILDREN=$(cd "$TMPDIR" && bd list --parent=demo-1.1 --all --json 2>/dev/null | jq 'length')
if [ "$FEAT11_CHILDREN" -eq 3 ]; then
    pass "demo-1.1 has 3 children (CRUD tasks)"
else
    fail "demo-1.1 expected 3 children, got $FEAT11_CHILDREN"
fi

# Feature demo-2.1 should have 2 children
FEAT21_CHILDREN=$(cd "$TMPDIR" && bd list --parent=demo-2.1 --all --json 2>/dev/null | jq 'length')
if [ "$FEAT21_CHILDREN" -eq 2 ]; then
    pass "demo-2.1 has 2 children (serialization tasks)"
else
    fail "demo-2.1 expected 2 children, got $FEAT21_CHILDREN"
fi

echo ""
echo "==========================================="
echo "RESULTS: $PASS passed, $FAIL failed"
echo "==========================================="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
