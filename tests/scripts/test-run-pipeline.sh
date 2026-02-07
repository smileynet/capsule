#!/usr/bin/env bash
# Test script for cap-8ax.6.1: Create run-pipeline.sh orchestration script
# Validates: full pipeline orchestration, phase pair loops, retry logic, error handling.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
SETUP_SCRIPT="$REPO_ROOT/scripts/setup-template.sh"
PIPELINE_SCRIPT="$REPO_ROOT/scripts/run-pipeline.sh"

PASS=0
FAIL=0

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); echo "  FAIL: $1"; }

# --- Prerequisite checks ---
for cmd in git bd jq; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "ERROR: $cmd is required but not installed" >&2
        exit 1
    fi
done

for script in "$SETUP_SCRIPT" "$PIPELINE_SCRIPT"; do
    if [ ! -f "$script" ]; then
        echo "ERROR: $(basename "$script") not found at $script" >&2
        exit 1
    fi
done

# --- Helper: create a mock claude that always returns PASS ---
# Handles sign-off by appending Verdict: PASS to worklog.md
# Handles merge by staging and committing implementation files
create_mock_claude_pass() {
    local mock_dir="$1"
    local mock_claude="$mock_dir/claude"
    cat > "$mock_claude" << 'MOCK_EOF'
#!/usr/bin/env bash
PROMPT=""
while [ $# -gt 0 ]; do
    case "$1" in
        -p) PROMPT="$2"; shift 2 ;;
        --dangerously-skip-permissions) shift ;;
        *) shift ;;
    esac
done

FIRST_LINE=$(printf '%s\n' "$PROMPT" | head -1)

# Sign-off phase: append verdict to worklog
if printf '%s\n' "$FIRST_LINE" | grep -qi 'sign-off\|signoff'; then
    if [ -f worklog.md ]; then
        cat >> worklog.md << 'SIGNOFF_EOF'

### Phase 5: sign-off

_Status: complete_

**Verdict: PASS**

All tests pass. Implementation meets acceptance criteria.
SIGNOFF_EOF
        git add worklog.md
        git commit -q -m "Add sign-off verdict" 2>/dev/null || true
    fi
fi

# Merge phase: stage and commit implementation files
if printf '%s\n' "$FIRST_LINE" | grep -qi 'merge'; then
    FILES_TO_STAGE=()
    while IFS= read -r f; do
        case "$f" in
            worklog.md|.capsule/*) ;;
            *) FILES_TO_STAGE+=("$f") ;;
        esac
    done < <(git diff --name-only HEAD 2>/dev/null; git ls-files --others --exclude-standard 2>/dev/null)
    if [ ${#FILES_TO_STAGE[@]} -gt 0 ]; then
        git add "${FILES_TO_STAGE[@]}"
        git commit -m "mock merge commit" -q 2>/dev/null || true
    fi
fi

printf '{"status":"PASS","feedback":"All checks passed.","files_changed":["src/main.go"],"summary":"Phase completed successfully"}\n'
MOCK_EOF
    chmod +x "$mock_claude"
}

# --- Helper: create mock claude that returns NEEDS_WORK from review phases N times then PASS ---
# Writer phases always return PASS. Only review/sign-off phases use the counter.
create_mock_claude_retry() {
    local mock_dir="$1"
    local fail_count="$2"
    local counter_file="$mock_dir/.review_count"
    printf '0' > "$counter_file"
    local mock_claude="$mock_dir/claude"
    cat > "$mock_claude" << MOCK_EOF
#!/usr/bin/env bash
COUNTER_FILE="$counter_file"

PROMPT=""
while [ \$# -gt 0 ]; do
    case "\$1" in
        -p) PROMPT="\$2"; shift 2 ;;
        --dangerously-skip-permissions) shift ;;
        *) shift ;;
    esac
done

# Detect phase from prompt's first line heading
FIRST_LINE=\$(printf '%s\n' "\$PROMPT" | head -1)

# Sign-off phase: append verdict to worklog on PASS
if printf '%s\n' "\$FIRST_LINE" | grep -qi 'sign-off\|signoff'; then
    COUNT=\$(cat "\$COUNTER_FILE")
    COUNT=\$((COUNT + 1))
    printf '%s' "\$COUNT" > "\$COUNTER_FILE"
    if [ "\$COUNT" -gt "$fail_count" ] && [ -f worklog.md ]; then
        cat >> worklog.md << 'SIGNOFF_EOF'

### Phase 5: sign-off

_Status: complete_

**Verdict: PASS**

All tests pass. Implementation meets acceptance criteria.
SIGNOFF_EOF
        git add worklog.md
        git commit -q -m "Add sign-off verdict" 2>/dev/null || true
    fi
    if [ "\$COUNT" -le "$fail_count" ]; then
        printf '{"status":"NEEDS_WORK","feedback":"Issues found on review attempt %s.","files_changed":[],"summary":"Review failed"}\n' "\$COUNT"
        exit 0
    fi
    printf '{"status":"PASS","feedback":"All checks passed.","files_changed":["src/main.go"],"summary":"Phase completed successfully"}\n'
    exit 0
fi

# Review phase: use counter (match heading like "# Test-Review Phase" or "# Execute-Review Phase")
if printf '%s\n' "\$FIRST_LINE" | grep -qi 'review'; then
    COUNT=\$(cat "\$COUNTER_FILE")
    COUNT=\$((COUNT + 1))
    printf '%s' "\$COUNT" > "\$COUNTER_FILE"
    if [ "\$COUNT" -le "$fail_count" ]; then
        printf '{"status":"NEEDS_WORK","feedback":"Issues found on review attempt %s.","files_changed":[],"summary":"Review failed"}\n' "\$COUNT"
        exit 0
    fi
fi

# Merge phase: stage and commit implementation files
if printf '%s\n' "\$FIRST_LINE" | grep -qi 'merge'; then
    FILES_TO_STAGE=()
    while IFS= read -r f; do
        case "\$f" in
            worklog.md|.capsule/*) ;;
            *) FILES_TO_STAGE+=("\$f") ;;
        esac
    done < <(git diff --name-only HEAD 2>/dev/null; git ls-files --others --exclude-standard 2>/dev/null)
    if [ \${#FILES_TO_STAGE[@]} -gt 0 ]; then
        git add "\${FILES_TO_STAGE[@]}"
        git commit -m "mock merge commit" -q 2>/dev/null || true
    fi
fi

# Default: PASS (writer phases, merge, etc.)
printf '{"status":"PASS","feedback":"All checks passed.","files_changed":["src/main.go"],"summary":"Phase completed successfully"}\n'
MOCK_EOF
    chmod +x "$mock_claude"
}

# --- Helper: create mock claude that always returns ERROR ---
create_mock_claude_error() {
    local mock_dir="$1"
    local mock_claude="$mock_dir/claude"
    cat > "$mock_claude" << 'MOCK_EOF'
#!/usr/bin/env bash
while [ $# -gt 0 ]; do shift; done
printf '{"status":"ERROR","feedback":"Something went wrong.","files_changed":[],"summary":"Phase failed"}\n'
MOCK_EOF
    chmod +x "$mock_claude"
}

# --- Helper: create extra test beads ---
create_test_bead() {
    local project_dir="$1"
    local title="$2"
    local bead_id
    bead_id=$(cd "$project_dir" && bd create --title="$title" --type=task --priority=0 2>/dev/null | grep -oP 'demo-\w+')
    (cd "$project_dir" && git add -A && git commit -q -m "Add bead $bead_id") >/dev/null 2>&1 || true
    echo "$bead_id"
}

# =============================================================================
echo "=== Setting up test environment ==="
PROJECT_DIR=$("$SETUP_SCRIPT")
trap 'rm -rf "$PROJECT_DIR"' EXIT
echo "  Test project: $PROJECT_DIR"

MOCK_DIR=$(mktemp -d)
trap 'rm -rf "$PROJECT_DIR" "$MOCK_DIR"' EXIT

BEAD_ID="demo-1.1.1"

# Create extra beads for tests that need separate IDs
BEAD_ID_RETRY=$(create_test_bead "$PROJECT_DIR" "Retry test task")
BEAD_ID_ERROR=$(create_test_bead "$PROJECT_DIR" "Error test task")
BEAD_ID_MAXRETRY=$(create_test_bead "$PROJECT_DIR" "Max retry test task")

echo "  Beads: $BEAD_ID, $BEAD_ID_RETRY, $BEAD_ID_ERROR, $BEAD_ID_MAXRETRY"

echo ""
echo "=== cap-8ax.6.1: run-pipeline.sh ==="
echo ""

# ---------- Test 1: Missing bead-id rejected ----------
echo "[1/9] Missing bead-id rejected"
MISSING_EXIT=0
MISSING_OUTPUT=$("$PIPELINE_SCRIPT" 2>&1) || MISSING_EXIT=$?
if [ "$MISSING_EXIT" -ne 0 ]; then
    if echo "$MISSING_OUTPUT" | grep -qiE 'bead-id.*required|usage'; then
        pass "Missing bead-id exits non-zero with usage message"
    else
        fail "Missing bead-id exits non-zero but no usage message"
        echo "  Output: $MISSING_OUTPUT"
    fi
else
    fail "Missing bead-id should exit non-zero"
fi

# ---------- Test 2: Unknown option rejected ----------
echo "[2/9] Unknown option rejected"
UNKNOWN_EXIT=0
UNKNOWN_OUTPUT=$("$PIPELINE_SCRIPT" "$BEAD_ID" --bogus 2>&1) || UNKNOWN_EXIT=$?
if [ "$UNKNOWN_EXIT" -ne 0 ]; then
    if echo "$UNKNOWN_OUTPUT" | grep -qiE 'unknown.*option|usage'; then
        pass "Unknown option rejected with descriptive error"
    else
        fail "Unknown option rejected but no descriptive error"
        echo "  Output: $UNKNOWN_OUTPUT"
    fi
else
    fail "Unknown option should exit non-zero"
fi

# ---------- Test 3: Happy path - full pipeline completes ----------
echo "[3/9] Happy path: full pipeline completes"
# Given: all phases return PASS (mock handles sign-off and merge)
# When: run-pipeline.sh is run
# Then: exits 0
create_mock_claude_pass "$MOCK_DIR"

PIPELINE_EXIT=0
PIPELINE_OUTPUT=$(PATH="$MOCK_DIR:$PATH" "$PIPELINE_SCRIPT" "$BEAD_ID" --project-dir="$PROJECT_DIR" 2>&1) || PIPELINE_EXIT=$?
if [ "$PIPELINE_EXIT" -eq 0 ]; then
    pass "Pipeline exits 0 on happy path"
else
    fail "Pipeline exited $PIPELINE_EXIT on happy path"
    echo "  Output: $PIPELINE_OUTPUT"
fi

# ---------- Test 4: Pipeline output contains phase progression ----------
echo "[4/9] Pipeline output shows phase progression"
PHASES_FOUND=0
for phase in "prep" "test-writer" "test-review" "execute" "execute-review" "sign-off" "merge"; do
    if echo "$PIPELINE_OUTPUT" | grep -qi "$phase"; then
        PHASES_FOUND=$((PHASES_FOUND + 1))
    fi
done
if [ "$PHASES_FOUND" -ge 5 ]; then
    pass "Pipeline output references at least 5 phases ($PHASES_FOUND found)"
else
    fail "Pipeline output only references $PHASES_FOUND phases (expected >= 5)"
    echo "  Output: $PIPELINE_OUTPUT"
fi

# ---------- Test 5: Pipeline prints summary ----------
echo "[5/9] Pipeline prints summary"
if echo "$PIPELINE_OUTPUT" | grep -qiE 'summary|complete|pipeline.*finish'; then
    pass "Pipeline prints summary"
else
    fail "Pipeline does not print summary"
    echo "  Output: $PIPELINE_OUTPUT"
fi

# ---------- Test 6: Retry on NEEDS_WORK in test-review ----------
echo "[6/9] Retry on NEEDS_WORK in test-review phase pair"
# Given: claude returns NEEDS_WORK twice, then PASS
# When: run-pipeline.sh is run
# Then: pipeline retries and eventually succeeds

MOCK_RETRY_DIR=$(mktemp -d)
trap 'rm -rf "$PROJECT_DIR" "$MOCK_DIR" "$MOCK_RETRY_DIR"' EXIT
create_mock_claude_retry "$MOCK_RETRY_DIR" 2

RETRY_EXIT=0
RETRY_OUTPUT=$(PATH="$MOCK_RETRY_DIR:$PATH" "$PIPELINE_SCRIPT" "$BEAD_ID_RETRY" --project-dir="$PROJECT_DIR" 2>&1) || RETRY_EXIT=$?
if [ "$RETRY_EXIT" -eq 0 ]; then
    pass "Pipeline succeeds after retries"
else
    fail "Pipeline failed after retries (exit $RETRY_EXIT)"
    echo "  Output: $RETRY_OUTPUT"
fi

# ---------- Test 7: ERROR aborts pipeline ----------
echo "[7/9] ERROR aborts pipeline immediately"
# Given: claude returns ERROR on first call
# When: run-pipeline.sh is run
# Then: pipeline aborts with non-zero exit

MOCK_ERROR_DIR=$(mktemp -d)
trap 'rm -rf "$PROJECT_DIR" "$MOCK_DIR" "$MOCK_RETRY_DIR" "$MOCK_ERROR_DIR"' EXIT
create_mock_claude_error "$MOCK_ERROR_DIR"

ERROR_EXIT=0
ERROR_OUTPUT=$(PATH="$MOCK_ERROR_DIR:$PATH" "$PIPELINE_SCRIPT" "$BEAD_ID_ERROR" --project-dir="$PROJECT_DIR" 2>&1) || ERROR_EXIT=$?
if [ "$ERROR_EXIT" -eq 2 ]; then
    pass "Pipeline aborts on ERROR with exit code 2"
else
    fail "Pipeline should abort on ERROR with exit code 2, got $ERROR_EXIT"
    echo "  Output: $ERROR_OUTPUT"
fi

# ---------- Test 8: Worktree preserved on error ----------
echo "[8/9] Worktree preserved on error"
WORKTREE_ERROR="$PROJECT_DIR/.capsule/worktrees/$BEAD_ID_ERROR"
if [ -d "$WORKTREE_ERROR" ]; then
    pass "Worktree preserved on error for debugging"
else
    fail "Worktree was removed despite error (should be preserved)"
fi
# Clean up for subsequent tests
(cd "$PROJECT_DIR" && git worktree remove "$WORKTREE_ERROR" --force 2>/dev/null) || rm -rf "$WORKTREE_ERROR"
(cd "$PROJECT_DIR" && git worktree prune 2>/dev/null) || true
(cd "$PROJECT_DIR" && git branch -D "capsule-$BEAD_ID_ERROR" 2>/dev/null) || true

# ---------- Test 9: Max retries exceeded aborts pipeline ----------
echo "[9/9] Max retries exceeded aborts pipeline"
# Given: claude always returns NEEDS_WORK (never PASS)
# When: run-pipeline.sh is run with default retries
# Then: pipeline aborts after max retries

MOCK_MAXRETRY_DIR=$(mktemp -d)
trap 'rm -rf "$PROJECT_DIR" "$MOCK_DIR" "$MOCK_RETRY_DIR" "$MOCK_ERROR_DIR" "$MOCK_MAXRETRY_DIR"' EXIT
# Set fail_count very high so it never passes
create_mock_claude_retry "$MOCK_MAXRETRY_DIR" 100

MAXRETRY_EXIT=0
MAXRETRY_OUTPUT=$(PATH="$MOCK_MAXRETRY_DIR:$PATH" "$PIPELINE_SCRIPT" "$BEAD_ID_MAXRETRY" --project-dir="$PROJECT_DIR" 2>&1) || MAXRETRY_EXIT=$?
if [ "$MAXRETRY_EXIT" -ne 0 ]; then
    if echo "$MAXRETRY_OUTPUT" | grep -qiE 'retries.*exhaust|max.*retries|too many|retry'; then
        pass "Max retries exceeded aborts with descriptive message"
    else
        pass "Max retries exceeded aborts pipeline (exit $MAXRETRY_EXIT)"
    fi
else
    fail "Pipeline should abort when max retries exceeded"
    echo "  Output: $MAXRETRY_OUTPUT"
fi
# Clean up
WORKTREE_MAXRETRY="$PROJECT_DIR/.capsule/worktrees/$BEAD_ID_MAXRETRY"
(cd "$PROJECT_DIR" && git worktree remove "$WORKTREE_MAXRETRY" --force 2>/dev/null) || rm -rf "$WORKTREE_MAXRETRY"
(cd "$PROJECT_DIR" && git worktree prune 2>/dev/null) || true
(cd "$PROJECT_DIR" && git branch -D "capsule-$BEAD_ID_MAXRETRY" 2>/dev/null) || true

echo ""
echo "==========================================="
echo "RESULTS: $PASS passed, $FAIL failed"
echo "==========================================="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
