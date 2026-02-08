#!/usr/bin/env bash
# Test script for scripts/lib/resolve-parent-chain.sh
# Validates: parent chain resolution via .parent field, dependency fallback, edge cases.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
LIB_SCRIPT="$REPO_ROOT/scripts/lib/resolve-parent-chain.sh"
SETUP_SCRIPT="$REPO_ROOT/scripts/setup-template.sh"

PASS=0
FAIL=0

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); echo "  FAIL: $1"; }

# --- Prerequisite checks ---
for cmd in jq bd; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "ERROR: $cmd is required but not installed" >&2
        exit 1
    fi
done

if [ ! -f "$LIB_SCRIPT" ]; then
    echo "ERROR: resolve-parent-chain.sh not found at $LIB_SCRIPT" >&2
    exit 1
fi

# --- Create test environment ---
echo "=== Setting up test environment ==="
PROJECT_DIR=$("$SETUP_SCRIPT")
trap 'rm -rf "$PROJECT_DIR"' EXIT
echo "  Test project: $PROJECT_DIR"

# Source the lib
source "$LIB_SCRIPT"

echo ""
echo "=== resolve-parent-chain.sh ==="
echo ""

# ---------- Test 1: Task → Feature → Epic (full chain via .parent field) ----------
echo "[1/5] Full chain: task → feature → epic"
# Given: a task bead (demo-1.1.1) with parent feature (demo-1.1) and grandparent epic (demo-1)
# When: resolve_parent_chain is called with the task's JSON
# Then: FEATURE_ID, FEATURE_TITLE, EPIC_ID, EPIC_TITLE are all populated
BEAD_JSON=$(cd "$PROJECT_DIR" && bd show demo-1.1.1 --json 2>/dev/null)
resolve_parent_chain "$PROJECT_DIR" "$BEAD_JSON"

CHAIN_OK=true
if [ "$FEATURE_ID" != "demo-1.1" ]; then
    fail "Expected FEATURE_ID=demo-1.1, got '$FEATURE_ID'"
    CHAIN_OK=false
fi
if [ -z "$FEATURE_TITLE" ]; then
    fail "FEATURE_TITLE is empty"
    CHAIN_OK=false
fi
if [ "$EPIC_ID" != "demo-1" ]; then
    fail "Expected EPIC_ID=demo-1, got '$EPIC_ID'"
    CHAIN_OK=false
fi
if [ -z "$EPIC_TITLE" ]; then
    fail "EPIC_TITLE is empty"
    CHAIN_OK=false
fi
if [ -z "$FEATURE_GOAL" ]; then
    fail "FEATURE_GOAL is empty"
    CHAIN_OK=false
fi
if [ -z "$EPIC_GOAL" ]; then
    fail "EPIC_GOAL is empty"
    CHAIN_OK=false
fi
if [ "$CHAIN_OK" = true ]; then
    pass "Full chain resolved: feature=$FEATURE_ID, epic=$EPIC_ID"
fi

# ---------- Test 2: Feature → Epic (one level) ----------
echo "[2/5] Feature parent is epic"
# Given: a feature bead (demo-1.1) whose parent is an epic (demo-1)
# When: resolve_parent_chain is called
# Then: EPIC_ID is set, FEATURE_ID is empty (parent is epic, not feature)
BEAD_JSON=$(cd "$PROJECT_DIR" && bd show demo-1.1 --json 2>/dev/null)
resolve_parent_chain "$PROJECT_DIR" "$BEAD_JSON"

if [ -z "$FEATURE_ID" ] && [ "$EPIC_ID" = "demo-1" ] && [ -n "$EPIC_TITLE" ]; then
    pass "Feature's parent resolved as epic=$EPIC_ID"
else
    fail "Expected EPIC_ID=demo-1, FEATURE_ID='', got epic='$EPIC_ID' feature='$FEATURE_ID'"
fi

# ---------- Test 3: Epic has no parent ----------
echo "[3/5] Epic has no parent"
# Given: an epic bead (demo-1) with no parent
# When: resolve_parent_chain is called
# Then: both FEATURE_ID and EPIC_ID are empty
BEAD_JSON=$(cd "$PROJECT_DIR" && bd show demo-1 --json 2>/dev/null)
resolve_parent_chain "$PROJECT_DIR" "$BEAD_JSON"

if [ -z "$FEATURE_ID" ] && [ -z "$EPIC_ID" ]; then
    pass "Epic root: no parent resolved (feature='', epic='')"
else
    fail "Expected empty feature and epic for root, got feature='$FEATURE_ID' epic='$EPIC_ID'"
fi

# ---------- Test 4: Dependency fallback when .parent is absent ----------
echo "[4/5] Dependency fallback when .parent is absent"
# Given: bd show JSON where .parent is null but dependencies[] has parent-child entry
# When: _extract_parent_id is called
# Then: parent ID is extracted from dependencies array

# Simulate JSON with null .parent but valid dependencies
FALLBACK_JSON='[{
  "id": "test-1.1.1",
  "title": "Test task",
  "parent": null,
  "dependencies": [
    {
      "id": "test-1.1",
      "title": "Test feature",
      "status": "open",
      "dependency_type": "parent-child"
    }
  ]
}]'

EXTRACTED=$(_extract_parent_id "$FALLBACK_JSON")
if [ "$EXTRACTED" = "test-1.1" ]; then
    pass "Dependency fallback extracted parent ID 'test-1.1' from dependencies array"
else
    fail "Expected 'test-1.1' from dependency fallback, got '$EXTRACTED'"
fi

# ---------- Test 5: No parent and no dependencies ----------
echo "[5/5] No parent and no dependencies"
# Given: bd show JSON with null .parent and empty dependencies
# When: _extract_parent_id is called
# Then: returns empty string

NO_PARENT_JSON='[{
  "id": "orphan-1",
  "title": "Orphan task",
  "parent": null,
  "dependencies": []
}]'

EXTRACTED=$(_extract_parent_id "$NO_PARENT_JSON")
if [ -z "$EXTRACTED" ]; then
    pass "No parent or dependencies: returned empty"
else
    fail "Expected empty for orphan, got '$EXTRACTED'"
fi

# ---------- Edge Cases ----------
echo ""
echo "=== Edge Cases ==="

# E1: Dependency fallback with missing .parent field entirely
echo "[E1] Dependency fallback when .parent field missing entirely"
# Given: JSON with no .parent key at all (not null, absent)
# When: _extract_parent_id is called
# Then: falls back to dependencies array
MISSING_PARENT_JSON='[{
  "id": "legacy-1",
  "title": "Legacy task",
  "dependencies": [
    {
      "id": "legacy-parent",
      "title": "Legacy parent",
      "dependency_type": "parent-child"
    }
  ]
}]'

EXTRACTED=$(_extract_parent_id "$MISSING_PARENT_JSON")
if [ "$EXTRACTED" = "legacy-parent" ]; then
    pass "Missing .parent field: fallback extracted 'legacy-parent'"
else
    fail "Expected 'legacy-parent', got '$EXTRACTED'"
fi

# E2: Primary .parent takes precedence over dependencies
echo "[E2] Primary .parent takes precedence over dependencies"
# Given: JSON with both .parent and dependencies[] parent-child
# When: _extract_parent_id is called
# Then: .parent is used (not dependencies)
BOTH_JSON='[{
  "id": "both-1",
  "title": "Both sources",
  "parent": "primary-parent",
  "dependencies": [
    {
      "id": "dep-parent",
      "dependency_type": "parent-child"
    }
  ]
}]'

EXTRACTED=$(_extract_parent_id "$BOTH_JSON")
if [ "$EXTRACTED" = "primary-parent" ]; then
    pass "Primary .parent takes precedence over dependencies"
else
    fail "Expected 'primary-parent', got '$EXTRACTED'"
fi

echo ""
echo "==========================================="
echo "RESULTS: $PASS passed, $FAIL failed"
echo "==========================================="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
