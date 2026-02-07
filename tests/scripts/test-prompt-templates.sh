#!/usr/bin/env bash
# Test script for cap-prk: Prompt template content validation
# Validates: t-1.3.2 (test-writer prompt), t-1.3.3 (test-review prompt), and t-1.4.1 (execute prompt) specs
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
PROMPTS_DIR="$REPO_ROOT/prompts"

PASS=0
FAIL=0

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); echo "  FAIL: $1"; }

echo "=== Prompt template content validation ==="
echo ""

# ========== t-1.3.2: test-writer prompt ==========
echo "--- t-1.3.2: test-writer prompt ---"
echo ""
TW_PROMPT="$PROMPTS_DIR/test-writer.md"

# [1/6] File exists and is non-empty
echo "[1/6] test-writer prompt exists and is non-empty"
# Given: the prompts directory
# When: checking for test-writer.md
# Then: file exists and has content
if [ -f "$TW_PROMPT" ] && [ -s "$TW_PROMPT" ]; then
    pass "prompts/test-writer.md exists and is non-empty"
else
    fail "prompts/test-writer.md missing or empty"
fi

# [2/6] Instructs reading the worklog
echo "[2/6] test-writer instructs reading worklog"
# Given: the test-writer prompt
# When: checking content for worklog instruction
# Then: contains instruction to read worklog.md
if grep -qi "worklog" "$TW_PROMPT"; then
    pass "Contains instruction to read worklog"
else
    fail "Missing instruction to read worklog"
fi

# [3/6] Instructs writing failing tests
echo "[3/6] test-writer instructs writing failing tests"
# Given: the test-writer prompt
# When: checking content for RED/failing test instruction
# Then: contains instruction about failing tests
if grep -qi "fail" "$TW_PROMPT" && grep -qi "test" "$TW_PROMPT"; then
    pass "Contains instruction about failing tests"
else
    fail "Missing instruction about failing tests"
fi

# [4/6] Instructs worklog update
echo "[4/6] test-writer instructs worklog update"
# Given: the test-writer prompt
# When: checking content for worklog update instruction
# Then: contains instruction to update/append to worklog
if grep -qi "update.*worklog\|append.*worklog\|worklog.*phase" "$TW_PROMPT"; then
    pass "Contains instruction to update worklog"
else
    fail "Missing instruction to update worklog"
fi

# [5/6] Instructs JSON signal output
echo "[5/6] test-writer instructs JSON signal output"
# Given: the test-writer prompt
# When: checking content for signal contract
# Then: contains JSON signal format with status field
if grep -q '"status"' "$TW_PROMPT" && grep -qi "signal" "$TW_PROMPT"; then
    pass "Contains JSON signal contract with status field"
else
    fail "Missing JSON signal contract"
fi

# [6/6] References acceptance criteria
echo "[6/6] test-writer references acceptance criteria"
# Given: the test-writer prompt
# When: checking content for acceptance criteria reference
# Then: contains instruction to derive tests from acceptance criteria
if grep -qi "acceptance criteria" "$TW_PROMPT"; then
    pass "Contains reference to acceptance criteria"
else
    fail "Missing reference to acceptance criteria"
fi

# Edge cases for test-writer
echo ""
echo "=== test-writer edge cases ==="

echo "[E1] Does not assume specific test framework or states which one"
# Given: the test-writer prompt
# When: checking for framework-agnostic approach
# Then: references project conventions (CLAUDE.md) for framework choice
if grep -qi "CLAUDE.md\|project conventions\|conventions" "$TW_PROMPT"; then
    pass "References project conventions for test framework"
else
    fail "Missing reference to project conventions"
fi

echo "[E2] Handles existing test files (retry case)"
# Given: the test-writer prompt
# When: checking for retry/update handling
# Then: contains instruction to update existing files
if grep -qi "update\|existing\|retry\|already exist" "$TW_PROMPT"; then
    pass "Contains instruction for handling existing test files"
else
    fail "Missing instruction for handling existing files"
fi

echo "[E3] Instructs not to write implementation code"
# Given: the test-writer prompt
# When: checking for implementation restriction
# Then: explicitly states not to implement
if grep -qi "no implementation\|not.*implement\|do not.*implement\|only.*test" "$TW_PROMPT"; then
    pass "Contains instruction not to write implementation code"
else
    fail "Missing instruction restricting implementation code"
fi

echo "[E4] Specifies test file location"
# Given: the test-writer prompt
# When: checking for file placement instruction
# Then: specifies where to place test files
if grep -qi "test file.*location\|place.*test\|where.*test\|directory\|alongside" "$TW_PROMPT"; then
    pass "Contains instruction for test file placement"
else
    fail "Missing instruction for test file placement"
fi

echo ""

# ========== t-1.3.3: test-review prompt ==========
echo "--- t-1.3.3: test-review prompt ---"
echo ""
TR_PROMPT="$PROMPTS_DIR/test-review.md"

# [1/6] File exists and is non-empty
echo "[1/6] test-review prompt exists and is non-empty"
# Given: the prompts directory
# When: checking for test-review.md
# Then: file exists and has content
if [ -f "$TR_PROMPT" ] && [ -s "$TR_PROMPT" ]; then
    pass "prompts/test-review.md exists and is non-empty"
else
    fail "prompts/test-review.md missing or empty"
fi

# [2/6] Contains quality checklist
echo "[2/6] test-review contains quality checklist"
# Given: the test-review prompt
# When: checking content for quality criteria
# Then: references test isolation, naming, assertions, coverage
QUALITY_OK=true
if ! grep -qi "isolation\|isolated" "$TR_PROMPT"; then
    fail "Missing quality criterion: test isolation"
    QUALITY_OK=false
fi
if ! grep -qi "naming\|name" "$TR_PROMPT"; then
    fail "Missing quality criterion: test naming"
    QUALITY_OK=false
fi
if ! grep -qi "assertion" "$TR_PROMPT"; then
    fail "Missing quality criterion: assertions"
    QUALITY_OK=false
fi
if ! grep -qi "coverage\|acceptance criteria" "$TR_PROMPT"; then
    fail "Missing quality criterion: coverage"
    QUALITY_OK=false
fi
if [ "$QUALITY_OK" = true ]; then
    pass "Contains quality checklist (isolation, naming, assertions, coverage)"
fi

# [3/6] Instructs PASS/NEEDS_WORK decision
echo "[3/6] test-review instructs PASS/NEEDS_WORK decision"
# Given: the test-review prompt
# When: checking content for decision criteria
# Then: contains both PASS and NEEDS_WORK with decision guidance
if grep -q "PASS" "$TR_PROMPT" && grep -q "NEEDS_WORK" "$TR_PROMPT"; then
    pass "Contains PASS and NEEDS_WORK decision criteria"
else
    fail "Missing PASS/NEEDS_WORK decision criteria"
fi

# [4/6] Instructs feedback on NEEDS_WORK
echo "[4/6] test-review instructs actionable feedback on NEEDS_WORK"
# Given: the test-review prompt
# When: checking content for feedback instruction
# Then: contains instruction to provide actionable feedback
if grep -qi "feedback\|actionable\|specific.*issue\|suggest" "$TR_PROMPT"; then
    pass "Contains instruction for actionable feedback"
else
    fail "Missing instruction for actionable feedback"
fi

# [5/6] Instructs signal output
echo "[5/6] test-review instructs JSON signal output"
# Given: the test-review prompt
# When: checking content for signal contract
# Then: contains JSON signal format
if grep -q '"status"' "$TR_PROMPT" && grep -qi "signal" "$TR_PROMPT"; then
    pass "Contains JSON signal contract"
else
    fail "Missing JSON signal contract"
fi

# [6/6] Instructs worklog update
echo "[6/6] test-review instructs worklog update"
# Given: the test-review prompt
# When: checking content for worklog update instruction
# Then: contains instruction to record review findings in worklog
if grep -qi "worklog" "$TR_PROMPT" && grep -qi "phase.*entry\|update.*worklog\|append\|record" "$TR_PROMPT"; then
    pass "Contains instruction to update worklog with review findings"
else
    fail "Missing instruction to update worklog"
fi

# Edge cases for test-review
echo ""
echo "=== test-review edge cases ==="

echo "[E1] Handles case where no test files exist"
# Given: the test-review prompt
# When: checking for missing-tests handling
# Then: addresses case where test-writer produced no files
if grep -qi "no test\|not found\|missing.*test\|NEEDS_WORK\|ERROR" "$TR_PROMPT"; then
    pass "Addresses missing test files scenario"
else
    fail "Missing handling for no test files"
fi

echo "[E2] Distinguishes quality issues from coverage gaps"
# Given: the test-review prompt
# When: checking for separate quality and coverage evaluation
# Then: has distinct sections for coverage and quality
if grep -qi "coverage" "$TR_PROMPT" && grep -qi "quality" "$TR_PROMPT"; then
    pass "Distinguishes between coverage and quality evaluation"
else
    fail "Missing distinction between coverage and quality"
fi

echo "[E3] Reviewer role is review-only (not fix)"
# Given: the test-review prompt
# When: checking that reviewer role is read-only
# Then: uses review/verify/assess language (not fix/modify)
if grep -qi "review\|verify\|assess\|evaluate" "$TR_PROMPT"; then
    pass "Reviewer role is review/verify/assess (not fix)"
else
    fail "Missing clear reviewer role definition"
fi

# ========== t-1.4.1: execute prompt ==========
echo "--- t-1.4.1: execute prompt ---"
echo ""
EX_PROMPT="$PROMPTS_DIR/execute.md"

# [1/6] File exists and is non-empty
echo "[1/6] execute prompt exists and is non-empty"
# Given: the prompts directory
# When: checking for execute.md
# Then: file exists and has content
if [ -f "$EX_PROMPT" ] && [ -s "$EX_PROMPT" ]; then
    pass "prompts/execute.md exists and is non-empty"
else
    fail "prompts/execute.md missing or empty"
fi

# [2/6] Instructs reading worklog and test entries
echo "[2/6] execute instructs reading worklog and test entries"
# Given: the execute prompt
# When: checking content for worklog and test reading instruction
# Then: contains instruction to read worklog.md and test files from prior phases
if grep -qi "worklog" "$EX_PROMPT" && grep -qi "test" "$EX_PROMPT"; then
    pass "Contains instruction to read worklog and test entries"
else
    fail "Missing instruction to read worklog and test entries"
fi

# [3/6] Instructs confirming RED state (tests fail)
echo "[3/6] execute instructs confirming RED state"
# Given: the execute prompt
# When: checking content for RED state confirmation
# Then: contains instruction to run tests and verify they fail before implementing
if grep -qi "fail\|RED\|failing" "$EX_PROMPT" && grep -qi "run.*test\|test.*run\|confirm\|verify" "$EX_PROMPT"; then
    pass "Contains instruction to confirm RED state before implementing"
else
    fail "Missing instruction to confirm RED state"
fi

# [4/6] Instructs implementing minimal GREEN code
echo "[4/6] execute instructs implementing minimal GREEN code"
# Given: the execute prompt
# When: checking content for GREEN implementation instruction
# Then: contains instruction to implement minimum code to pass tests
if grep -qi "implement\|GREEN\|pass" "$EX_PROMPT" && grep -qi "minimal\|minimum" "$EX_PROMPT"; then
    pass "Contains instruction for minimal GREEN implementation"
else
    fail "Missing instruction for minimal GREEN implementation"
fi

# [5/6] Instructs worklog update with execute phase entry
echo "[5/6] execute instructs worklog update"
# Given: the execute prompt
# When: checking content for worklog update instruction
# Then: contains instruction to update worklog with execute phase entry
if grep -qi "worklog" "$EX_PROMPT" && grep -qi "phase.*3\|execute.*phase\|Phase 3\|phase.*entry\|update.*worklog\|append" "$EX_PROMPT"; then
    pass "Contains instruction to update worklog with execute phase entry"
else
    fail "Missing instruction to update worklog with execute phase entry"
fi

# [6/6] Instructs JSON signal output
echo "[6/6] execute instructs JSON signal output"
# Given: the execute prompt
# When: checking content for signal contract
# Then: contains JSON signal format with status field
if grep -q '"status"' "$EX_PROMPT" && grep -qi "signal" "$EX_PROMPT"; then
    pass "Contains JSON signal contract with status field"
else
    fail "Missing JSON signal contract"
fi

# Edge cases for execute
echo ""
echo "=== execute edge cases ==="

echo "[E1] Handles case where tests already pass (should ERROR)"
# Given: the execute prompt
# When: checking for already-passing test handling
# Then: contains instruction to signal ERROR if tests already pass
if grep -qi "already pass\|tests pass\|not fail\|ERROR" "$EX_PROMPT"; then
    pass "Addresses case where tests already pass"
else
    fail "Missing handling for tests already passing"
fi

echo "[E2] Instructs not to modify test files"
# Given: the execute prompt
# When: checking for test file modification restriction
# Then: explicitly states not to modify test files
if grep -qi "not.*modify.*test\|do not.*change.*test\|not.*edit.*test\|leave.*test\|not.*alter.*test" "$EX_PROMPT"; then
    pass "Contains instruction not to modify test files"
else
    fail "Missing instruction restricting test file modification"
fi

echo "[E3] Instructs refactoring only after GREEN"
# Given: the execute prompt
# When: checking for refactor ordering instruction
# Then: contains instruction to refactor only after tests pass
if grep -qi "refactor" "$EX_PROMPT" && grep -qi "after.*pass\|after.*GREEN\|GREEN.*refactor\|tests pass.*refactor\|once.*pass" "$EX_PROMPT"; then
    pass "Contains instruction to refactor only after GREEN"
else
    fail "Missing instruction to refactor after GREEN"
fi

echo "[E4] References CLAUDE.md for project conventions"
# Given: the execute prompt
# When: checking for project convention reference
# Then: references CLAUDE.md for coding conventions
if grep -qi "CLAUDE.md\|project conventions\|conventions" "$EX_PROMPT"; then
    pass "References project conventions (CLAUDE.md)"
else
    fail "Missing reference to project conventions"
fi

echo ""
echo "==========================================="
echo "RESULTS: $PASS passed, $FAIL failed"
echo "==========================================="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
