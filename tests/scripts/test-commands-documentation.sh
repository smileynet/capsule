#!/usr/bin/env bash
# Test script for cap-8ax.6.3: docs/commands.md documentation
# Validates: file existence, script coverage, signal contract, retry rules,
#            directory structure, cross-reference with actual scripts.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
DOCS_FILE="$REPO_ROOT/docs/commands.md"
SCRIPTS_DIR="$REPO_ROOT/scripts"

PASS=0
FAIL=0

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); echo "  FAIL: $1"; }

echo "=== cap-8ax.6.3: docs/commands.md tests ==="
echo ""

# =============================================================================
# Test 1: docs/commands.md exists and is non-empty
# =============================================================================
echo "[1/6] File exists and is non-empty"

# Given: the docs directory exists
# When: checking for docs/commands.md
# Then: file exists and has content

if [ ! -f "$DOCS_FILE" ]; then
    fail "docs/commands.md does not exist"
    echo ""
    echo "==========================================="
    echo "RESULTS: $PASS passed, $FAIL failed"
    echo "==========================================="
    exit 1
fi

LINE_COUNT=$(wc -l < "$DOCS_FILE")
if [ "$LINE_COUNT" -gt 0 ]; then
    pass "docs/commands.md exists ($LINE_COUNT lines)"
else
    fail "docs/commands.md is empty"
fi

# =============================================================================
# Test 2: All scripts from scripts/ have a corresponding section
# =============================================================================
echo ""
echo "[2/6] All scripts are documented"

# Given: scripts/ contains shell scripts
# When: searching docs/commands.md for each script name as a section header
# Then: every script has a ### heading

ALL_SCRIPTS_FOUND=true
for script in "$SCRIPTS_DIR"/*.sh; do
    SCRIPT_NAME="$(basename "$script")"
    if grep -q "^### $SCRIPT_NAME" "$DOCS_FILE"; then
        pass "$SCRIPT_NAME documented"
    else
        fail "$SCRIPT_NAME not documented (no ### $SCRIPT_NAME section)"
        ALL_SCRIPTS_FOUND=false
    fi
done

# =============================================================================
# Test 3: Signal contract documented with required fields
# =============================================================================
echo ""
echo "[3/6] Signal contract documented"

# Given: the pipeline uses a JSON signal contract
# When: reading docs/commands.md
# Then: status, feedback, files_changed, summary fields are documented

SIGNAL_OK=true
for field in status feedback files_changed summary; do
    if grep -q "$field" "$DOCS_FILE"; then
        pass "Signal field '$field' documented"
    else
        fail "Signal field '$field' not documented"
        SIGNAL_OK=false
    fi
done

# Check status values
for status in PASS NEEDS_WORK ERROR; do
    if grep -q "$status" "$DOCS_FILE"; then
        pass "Status value '$status' documented"
    else
        fail "Status value '$status' not documented"
        SIGNAL_OK=false
    fi
done

# =============================================================================
# Test 4: Retry rules documented
# =============================================================================
echo ""
echo "[4/6] Retry rules documented"

# Given: run-pipeline.sh implements retry logic
# When: reading docs/commands.md
# Then: max retries, feedback injection, and abort conditions are described

RETRY_OK=true

if grep -qi "max.retries\|max_retries\|--max-retries" "$DOCS_FILE"; then
    pass "Max retries documented"
else
    fail "Max retries not documented"
    RETRY_OK=false
fi

if grep -qi "feedback" "$DOCS_FILE" && grep -qi "retry\|re-run" "$DOCS_FILE"; then
    pass "Feedback injection documented"
else
    fail "Feedback injection not documented"
    RETRY_OK=false
fi

if grep -qi "abort\|retries exhausted" "$DOCS_FILE"; then
    pass "Abort conditions documented"
else
    fail "Abort conditions not documented"
    RETRY_OK=false
fi

# =============================================================================
# Test 5: Directory structure documented
# =============================================================================
echo ""
echo "[5/6] Directory structure documented"

# Given: the project has a defined directory layout
# When: reading docs/commands.md
# Then: key paths are described

DIRS_OK=true
for path in "scripts/" "prompts/" "templates/" ".capsule/" "worktrees/" "logs/"; do
    if grep -q "$path" "$DOCS_FILE"; then
        pass "Path '$path' documented"
    else
        fail "Path '$path' not documented"
        DIRS_OK=false
    fi
done

# =============================================================================
# Test 6: Cross-reference — documented scripts match actual files
# =============================================================================
echo ""
echo "[6/6] Cross-reference documented commands with actual scripts"

# Given: docs/commands.md documents scripts with ### headings
# When: extracting documented script names
# Then: every documented script exists in scripts/ (no phantom docs)

XREF_OK=true
DOCUMENTED_SCRIPTS=$(grep -oP '(?<=^### )\S+\.sh' "$DOCS_FILE" || true)

for doc_script in $DOCUMENTED_SCRIPTS; do
    if [ -f "$SCRIPTS_DIR/$doc_script" ]; then
        pass "$doc_script exists in scripts/"
    else
        fail "$doc_script documented but not found in scripts/ (phantom documentation)"
        XREF_OK=false
    fi
done

# Also check that no actual scripts are missing from documentation (reverse check)
for script in "$SCRIPTS_DIR"/*.sh; do
    SCRIPT_NAME="$(basename "$script")"
    if echo "$DOCUMENTED_SCRIPTS" | grep -q "^${SCRIPT_NAME}$"; then
        pass "$SCRIPT_NAME has matching documentation"
    else
        fail "$SCRIPT_NAME exists in scripts/ but not documented"
        XREF_OK=false
    fi
done

# =============================================================================
# Edge case checks
# =============================================================================
echo ""
echo "[edge] Additional quality checks"

# Exit code documentation
if grep -q "Exit code" "$DOCS_FILE" || grep -q "exit code" "$DOCS_FILE"; then
    pass "Exit code semantics documented"
else
    fail "Exit codes not documented"
fi

# Error handling
if grep -qi "error handling\|recovery" "$DOCS_FILE"; then
    pass "Error handling and recovery documented"
else
    fail "Error handling not documented"
fi

# Environment variables
if grep -qi "environment variable\|CAPSULE_COMMIT_MSG" "$DOCS_FILE"; then
    pass "Environment variables documented"
else
    fail "Environment variables not documented"
fi

# Epic 2 interface boundaries
if grep -qi "interface\|epic 2\|go rewrite" "$DOCS_FILE"; then
    pass "Epic 2 interface boundaries documented"
else
    fail "Epic 2 interface boundaries not documented"
fi

# =============================================================================
# Results
# =============================================================================
echo ""
echo "==========================================="
echo "RESULTS: $PASS passed, $FAIL failed"
echo "==========================================="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
