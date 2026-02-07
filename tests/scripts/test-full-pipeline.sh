#!/usr/bin/env bash
# Test script for cap-8ax.6.4: Full pipeline E2E smoke test
# Validates: setup → run-pipeline → assertions on main branch state.
# This is the Epic 1 E2E smoke test.
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

# --- Helper: create mock claude that simulates full E2E pipeline ---
# Phases:
#   test-writer: creates src/validate_test.go with tests
#   execute: creates src/validate.go with implementation
#   sign-off: appends Verdict: PASS to worklog
#   merge: stages and commits implementation files
create_mock_claude_e2e() {
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

# Test-writer phase: create test file
if printf '%s\n' "$FIRST_LINE" | grep -qi 'test-writer\|test.writer'; then
    cat > src/validate_test.go << 'TESTEOF'
package main

import "testing"

func TestValidateEmail_Valid(t *testing.T) {
	if err := ValidateEmail("user@example.com"); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidateEmail_MissingAt(t *testing.T) {
	if err := ValidateEmail("userexample.com"); err == nil {
		t.Error("expected error for missing @")
	}
}

func TestValidateEmail_Empty(t *testing.T) {
	if err := ValidateEmail(""); err == nil {
		t.Error("expected error for empty string")
	}
}
TESTEOF
    git add src/validate_test.go
    git commit -q -m "Add validation tests"
fi

# Execute phase: create implementation (exclude execute-review)
if printf '%s\n' "$FIRST_LINE" | grep -qi '^# execute phase'; then
    cat > src/validate.go << 'IMPLEOF'
package main

import (
	"fmt"
	"strings"
)

// ValidateEmail checks that the email has @ and a domain with a dot.
func ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("email cannot be empty")
	}
	at := strings.Index(email, "@")
	if at < 0 {
		return fmt.Errorf("email must contain @")
	}
	domain := email[at+1:]
	if !strings.Contains(domain, ".") {
		return fmt.Errorf("email domain must contain a dot")
	}
	return nil
}
IMPLEOF
    git add src/validate.go
    git commit -q -m "Add email validation implementation"
fi

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

printf '{"status":"PASS","feedback":"All checks passed.","files_changed":["src/validate.go"],"summary":"Phase completed successfully"}\n'
MOCK_EOF
    chmod +x "$mock_claude"
}

# =============================================================================
echo "=== Setting up E2E test environment ==="
PROJECT_DIR=$("$SETUP_SCRIPT")
trap 'rm -rf "$PROJECT_DIR"' EXIT
echo "  Test project: $PROJECT_DIR"

MOCK_DIR=$(mktemp -d)
trap 'rm -rf "$PROJECT_DIR" "$MOCK_DIR"' EXIT

BEAD_ID="demo-1.1.1"
create_mock_claude_e2e "$MOCK_DIR"

echo ""
echo "=== cap-8ax.6.4: Full pipeline E2E smoke test ==="
echo ""

# ---------- Run pipeline ----------
echo "[1/7] Run full pipeline"
# Given: a fresh template project with mock claude
# When: run-pipeline.sh is invoked with a task bead
# Then: pipeline completes successfully (exit 0)
PIPELINE_EXIT=0
PIPELINE_OUTPUT=$(PATH="$MOCK_DIR:$PATH" "$PIPELINE_SCRIPT" "$BEAD_ID" --project-dir="$PROJECT_DIR" 2>&1) || PIPELINE_EXIT=$?
if [ "$PIPELINE_EXIT" -eq 0 ]; then
    pass "Pipeline exits 0 (success)"
else
    fail "Pipeline exited $PIPELINE_EXIT (expected 0)"
    echo "  Output: $PIPELINE_OUTPUT"
    echo ""
    echo "Pipeline failed — remaining assertions will be skipped."
    echo ""
    echo "==========================================="
    echo "RESULTS: $PASS passed, $FAIL failed"
    echo "==========================================="
    exit 1
fi

# ---------- Assert: on main branch after pipeline ----------

echo "[2/7] On main branch after pipeline"
# Given: pipeline completed successfully
# When: checking current branch
# Then: we should be on main (merge.sh switches to main)
CURRENT_BRANCH=$(cd "$PROJECT_DIR" && git branch --show-current)
if [ "$CURRENT_BRANCH" = "main" ] || [ "$CURRENT_BRANCH" = "master" ]; then
    pass "Project is on main branch ($CURRENT_BRANCH)"
else
    fail "Expected main branch, got: $CURRENT_BRANCH"
fi

# ---------- Assert: tests exist on main ----------

echo "[3/7] Tests exist on main branch"
# Given: pipeline merged test-writer output to main
# When: checking for test files
# Then: src/validate_test.go exists
if [ -f "$PROJECT_DIR/src/validate_test.go" ]; then
    pass "src/validate_test.go exists on main"
else
    fail "src/validate_test.go not found on main"
    echo "  Files in src/: $(ls "$PROJECT_DIR/src/" 2>/dev/null)"
fi

# ---------- Assert: implementation passes tests ----------

echo "[4/7] Implementation passes tests"
# Given: tests and implementation exist on main
# When: running go test
# Then: all tests pass
if [ -f "$PROJECT_DIR/src/validate_test.go" ] && [ -f "$PROJECT_DIR/src/validate.go" ]; then
    TEST_EXIT=0
    TEST_OUTPUT=$(cd "$PROJECT_DIR/src" && go test ./... 2>&1) || TEST_EXIT=$?
    if [ "$TEST_EXIT" -eq 0 ]; then
        pass "go test passes on main"
    else
        fail "go test failed (exit $TEST_EXIT)"
        echo "  Output: $TEST_OUTPUT"
    fi
else
    fail "Cannot run tests — source files missing"
fi

# ---------- Assert: worklog in .capsule/logs/<bead-id>/ ----------

echo "[5/7] Worklog archived"
# Given: pipeline completed and merged
# When: checking .capsule/logs/<bead-id>/
# Then: worklog.md exists in archive
ARCHIVE_DIR="$PROJECT_DIR/.capsule/logs/$BEAD_ID"
if [ -f "$ARCHIVE_DIR/worklog.md" ]; then
    pass "Worklog archived at .capsule/logs/$BEAD_ID/worklog.md"
else
    fail "Worklog not found at $ARCHIVE_DIR/worklog.md"
    echo "  Contents of .capsule/logs/: $(ls -R "$PROJECT_DIR/.capsule/logs/" 2>/dev/null)"
fi

# ---------- Assert: no worktree remains ----------

echo "[6/7] No worktree remains"
# Given: pipeline completed and cleaned up
# When: checking for worktree directory
# Then: .capsule/worktrees/<bead-id>/ does not exist
WORKTREE_DIR="$PROJECT_DIR/.capsule/worktrees/$BEAD_ID"
if [ ! -d "$WORKTREE_DIR" ]; then
    pass "Worktree removed: .capsule/worktrees/$BEAD_ID/"
else
    fail "Worktree still exists: $WORKTREE_DIR"
fi

# ---------- Assert: bead is closed ----------

echo "[7/7] Bead is closed"
# Given: pipeline completed successfully
# When: checking bead status
# Then: bead shows as closed
BEAD_STATUS=$(cd "$PROJECT_DIR" && bd show "$BEAD_ID" --json 2>/dev/null | jq -r '.[0].status // "unknown"')
if [ "$BEAD_STATUS" = "closed" ]; then
    pass "Bead $BEAD_ID is closed"
else
    fail "Bead $BEAD_ID status is '$BEAD_STATUS' (expected 'closed')"
fi

echo ""
echo "==========================================="
echo "RESULTS: $PASS passed, $FAIL failed"
echo "==========================================="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
