#!/usr/bin/env bash
# Test script for t-1.1.1: Create template project source files and CLAUDE.md
# Validates the template project structure, buildability, and documentation.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TEMPLATE_DIR="$REPO_ROOT/templates/demo-capsule"

PASS=0
FAIL=0

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); echo "  FAIL: $1"; }

echo "=== t-1.1.1: Template Source Files ==="
echo ""

# Test 1: templates/demo-capsule/ exists with expected structure
echo "[1/6] Directory structure"
if [ -d "$TEMPLATE_DIR" ] && [ -d "$TEMPLATE_DIR/src" ]; then
  pass "templates/demo-capsule/ and src/ exist"
else
  fail "templates/demo-capsule/ or src/ missing"
fi

# Test 2: go.mod exists with module declaration
echo "[2/6] go.mod validity"
if [ -f "$TEMPLATE_DIR/src/go.mod" ] && grep -q "^module " "$TEMPLATE_DIR/src/go.mod"; then
  pass "go.mod exists with module declaration"
else
  fail "go.mod missing or no module declaration"
fi

# Test 3: main.go exists with package main
echo "[3/6] main.go validity"
if [ -f "$TEMPLATE_DIR/src/main.go" ] && grep -q "^package main" "$TEMPLATE_DIR/src/main.go"; then
  pass "main.go exists with package main"
else
  fail "main.go missing or no package main"
fi

# Test 4: go build succeeds
echo "[4/6] go build"
TMPBIN=$(mktemp -d)
if BUILD_OUTPUT=$( cd "$TEMPLATE_DIR/src" && go build -o "$TMPBIN/" ./... 2>&1 ); then
  pass "go build succeeds"
else
  fail "go build failed: $BUILD_OUTPUT"
fi
rm -rf "$TMPBIN"

# Test 5: CLAUDE.md exists with content
echo "[5/6] CLAUDE.md"
if [ -f "$TEMPLATE_DIR/CLAUDE.md" ] && [ -s "$TEMPLATE_DIR/CLAUDE.md" ]; then
  pass "CLAUDE.md exists with content"
else
  fail "CLAUDE.md missing or empty"
fi

# Test 6: README.md exists with content
echo "[6/6] README.md"
if [ -f "$TEMPLATE_DIR/README.md" ] && [ -s "$TEMPLATE_DIR/README.md" ]; then
  pass "README.md exists with content"
else
  fail "README.md missing or empty"
fi

# Edge case checks
echo ""
echo "=== Edge Cases ==="

# go.mod module path does not conflict with real modules
echo "[E1] Module path safety"
if grep -q "example.com/" "$TEMPLATE_DIR/src/go.mod" 2>/dev/null; then
  pass "Module path uses example.com (safe namespace)"
else
  fail "Module path may conflict with real modules"
fi

# main.go has feature gap marked with FEATURE_GAP comment
echo "[E2] Feature gap present"
if grep -q "FEATURE_GAP" "$TEMPLATE_DIR/src/main.go" 2>/dev/null; then
  pass "Feature gap marked in main.go"
else
  fail "No FEATURE_GAP marker in main.go"
fi

# CLAUDE.md does not reference paths outside template
echo "[E3] CLAUDE.md path references"
if ! grep -qE "(^|[^.])/home/|/usr/|/tmp/" "$TEMPLATE_DIR/CLAUDE.md" 2>/dev/null; then
  pass "CLAUDE.md has no absolute path references"
else
  fail "CLAUDE.md references paths outside template"
fi

# No hardcoded absolute paths in template files
echo "[E4] No hardcoded absolute paths"
ABSOLUTE_REFS=$(grep -rlE "(^|[^.])/home/|/usr/|/tmp/" "$TEMPLATE_DIR/" 2>/dev/null || true)
if [ -z "$ABSOLUTE_REFS" ]; then
  pass "No hardcoded absolute paths in template files"
else
  fail "Hardcoded absolute paths found in: $ABSOLUTE_REFS"
fi

echo ""
echo "=== Results: $PASS passed, $FAIL failed ==="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
