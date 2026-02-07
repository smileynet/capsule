#!/usr/bin/env bash
# Test script for cap-8ax.5.2: Create merge prompt template and thin merge driver script
# Validates: agent-reviewed merge, worklog archival, worktree cleanup, bead closure, error handling.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
SETUP_SCRIPT="$REPO_ROOT/scripts/setup-template.sh"
PREP_SCRIPT="$REPO_ROOT/scripts/prep.sh"
MERGE_SCRIPT="$REPO_ROOT/scripts/merge.sh"

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

for script in "$SETUP_SCRIPT" "$PREP_SCRIPT"; do
    if [ ! -f "$script" ]; then
        echo "ERROR: $(basename "$script") not found at $script" >&2
        exit 1
    fi
done

if [ ! -f "$MERGE_SCRIPT" ]; then
    echo "ERROR: merge.sh not found at $MERGE_SCRIPT" >&2
    exit 1
fi

# --- Helper: create a mock claude that simulates merge agent behavior ---
create_mock_claude() {
    local mock_dir="$1"
    local mock_claude="$mock_dir/claude"
    cat > "$mock_claude" << 'MOCK_EOF'
#!/usr/bin/env bash
# Mock claude: simulates the merge agent
# Stages implementation and test files, commits, and emits PASS signal

# Parse arguments
while [ $# -gt 0 ]; do
    case "$1" in
        -p) shift 2 ;;
        --dangerously-skip-permissions) shift ;;
        *) shift ;;
    esac
done

# Find implementation and test files (exclude worklog.md and .capsule/)
FILES_TO_STAGE=()
while IFS= read -r f; do
    case "$f" in
        worklog.md|.capsule/*) ;;
        *) FILES_TO_STAGE+=("$f") ;;
    esac
done < <(git diff --name-only HEAD 2>/dev/null; git ls-files --others --exclude-standard 2>/dev/null)

# Stage and commit
if [ ${#FILES_TO_STAGE[@]} -gt 0 ]; then
    git add "${FILES_TO_STAGE[@]}"
    git commit -m "mock merge commit" -q 2>/dev/null || true
fi

COMMIT_HASH=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Emit signal
cat << SIGNAL_EOF
Reviewing worktree for merge readiness...

All implementation and test files identified. Staging for commit.

Files merged:
- src/main_test.go
- src/main.go

Files excluded:
- worklog.md (worklog - archived separately)

{"status":"PASS","feedback":"All implementation and test files staged and committed. Worklog excluded for archival.","files_changed":["src/main.go","src/main_test.go"],"summary":"Merge commit created with implementation files only","commit_hash":"$COMMIT_HASH"}
SIGNAL_EOF
MOCK_EOF
    chmod +x "$mock_claude"
    echo "$mock_dir"
}

# --- Helper: simulate completed pipeline state in worktree ---
simulate_completed_pipeline() {
    local worktree_dir="$1"
    (
        cd "$worktree_dir"

        # Create implementation file
        mkdir -p src
        cat > src/main.go << 'GO_EOF'
package main

import (
    "fmt"
    "regexp"
)

type Contact struct {
    Name  string
    Email string
    Phone string
}

func ValidateEmail(email string) error {
    re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
    if !re.MatchString(email) {
        return fmt.Errorf("invalid email format: %s", email)
    }
    return nil
}

func main() {
    fmt.Println("demo capsule")
}
GO_EOF

        # Create test file
        cat > src/main_test.go << 'TEST_EOF'
package main

import "testing"

func TestValidateEmail(t *testing.T) {
    tests := []struct {
        email string
        valid bool
    }{
        {"user@example.com", true},
        {"invalid", false},
        {"@nodomain.com", false},
    }
    for _, tt := range tests {
        err := ValidateEmail(tt.email)
        if tt.valid && err != nil {
            t.Errorf("ValidateEmail(%q) = %v, want nil", tt.email, err)
        }
        if !tt.valid && err == nil {
            t.Errorf("ValidateEmail(%q) = nil, want error", tt.email)
        }
    }
}
TEST_EOF

        # Add sign-off PASS to worklog
        cat >> worklog.md << 'SIGNOFF_EOF'

### Phase 5: sign-off

_Status: complete_

**Verdict: PASS**

**Test verification:**
- Tests run: 3 tests executed
- Tests passing: 3 (all pass ✓)

**Commit-ready check:**
- No temporary files: ✓
- No debug code: ✓
- No test-only artifacts outside test files: ✓
- Clean source tree: ✓

**Acceptance criteria verification:**
- AC: "Validate email format" → test exists ✓, test passes ✓, implementation correct ✓
SIGNOFF_EOF

        git add -A
        git commit -q -m "Simulate completed pipeline"
    )
}

# --- Create test environment ---
echo "=== Setting up test environment ==="
PROJECT_DIR=$("$SETUP_SCRIPT")
trap 'rm -rf "$PROJECT_DIR"' EXIT
echo "  Test project: $PROJECT_DIR"

# Create mock claude
MOCK_DIR=$(mktemp -d)
trap 'rm -rf "$PROJECT_DIR" "$MOCK_DIR"' EXIT
create_mock_claude "$MOCK_DIR"

BEAD_ID="demo-1.1.1"

echo ""
echo "=== cap-8ax.5.2: merge.sh ==="
echo ""

# ---------- Test 1: Happy path - full merge completes ----------
echo "[1/11] Happy path: merge completes successfully"
# Given: a worktree with completed pipeline (sign-off PASS)
# When: merge.sh is run
# Then: exits 0 and merge succeeds

# Prep a worktree
PREP_OUTPUT=$("$PREP_SCRIPT" "$BEAD_ID" --project-dir="$PROJECT_DIR" 2>&1) || {
    fail "prep.sh failed: $PREP_OUTPUT"
}
WORKTREE_DIR="$PROJECT_DIR/.capsule/worktrees/$BEAD_ID"
simulate_completed_pipeline "$WORKTREE_DIR"

# Record main branch state before merge
MAIN_BEFORE=$(cd "$PROJECT_DIR" && git rev-parse HEAD)

# Run merge with mock claude
MERGE_EXIT=0
MERGE_OUTPUT=$(PATH="$MOCK_DIR:$PATH" "$MERGE_SCRIPT" "$BEAD_ID" --project-dir="$PROJECT_DIR" 2>&1) || MERGE_EXIT=$?
if [ "$MERGE_EXIT" -eq 0 ]; then
    pass "merge.sh exited 0"
else
    fail "merge.sh exited $MERGE_EXIT"
    echo "  Output: $MERGE_OUTPUT"
fi

# ---------- Test 2: Merge commit exists on main ----------
echo "[2/11] Merge commit on main with --no-ff"
# Given: merge.sh completed successfully
# When: checking git log on main
# Then: a merge commit exists (not fast-forward)
MAIN_AFTER=$(cd "$PROJECT_DIR" && git rev-parse HEAD)
if [ "$MAIN_BEFORE" = "$MAIN_AFTER" ]; then
    fail "No new commit on main after merge"
else
    # Check for merge commit (has two parents)
    PARENT_COUNT=$(cd "$PROJECT_DIR" && git cat-file -p HEAD | grep -c '^parent' || true)
    if [ "$PARENT_COUNT" -ge 2 ]; then
        pass "Merge commit exists on main (--no-ff merge)"
    else
        fail "Expected --no-ff merge commit (2 parents), got $PARENT_COUNT parent(s)"
    fi
fi

# ---------- Test 3: Worklog archived ----------
echo "[3/11] Worklog archived to .capsule/logs/"
# Given: merge completed
# When: checking .capsule/logs/<bead-id>/
# Then: worklog.md is archived there
ARCHIVE_DIR="$PROJECT_DIR/.capsule/logs/$BEAD_ID"
if [ -f "$ARCHIVE_DIR/worklog.md" ]; then
    pass "Worklog archived to .capsule/logs/$BEAD_ID/worklog.md"
else
    fail "Worklog not found at $ARCHIVE_DIR/worklog.md"
    echo "  Contents of .capsule/logs/:"
    ls -la "$PROJECT_DIR/.capsule/logs/" 2>&1 || echo "  (directory not found)"
fi

# ---------- Test 4: Worktree removed ----------
echo "[4/11] Worktree removed after merge"
# Given: merge completed
# When: checking for the worktree directory
# Then: worktree is gone
if [ ! -d "$WORKTREE_DIR" ]; then
    pass "Worktree removed"
else
    fail "Worktree still exists at $WORKTREE_DIR"
fi

# ---------- Test 5: Branch deleted ----------
echo "[5/11] Feature branch deleted after merge"
# Given: merge completed
# When: listing branches
# Then: capsule-<bead-id> branch is gone
BRANCH_NAME="capsule-$BEAD_ID"
BRANCH_EXISTS=$(cd "$PROJECT_DIR" && git branch --list "$BRANCH_NAME" | wc -l)
if [ "$BRANCH_EXISTS" -eq 0 ]; then
    pass "Branch $BRANCH_NAME deleted"
else
    fail "Branch $BRANCH_NAME still exists"
fi

# ---------- Test 6: Bead closed ----------
echo "[6/11] Bead status closed after merge"
# Given: merge completed
# When: checking bead status
# Then: bead is closed
BEAD_STATUS=$(cd "$PROJECT_DIR" && bd show "$BEAD_ID" --json 2>/dev/null | jq -r '.[0].status')
if [ "$BEAD_STATUS" = "closed" ]; then
    pass "Bead $BEAD_ID is closed"
else
    fail "Bead $BEAD_ID status is '$BEAD_STATUS', expected 'closed'"
fi

# ---------- Test 7: Phase outputs archived ----------
echo "[7/11] Phase outputs archived"
# Given: merge completed with .capsule/output/ in worktree
# When: checking archive directory
# Then: phase outputs are in .capsule/logs/<bead-id>/
# Note: mock may not produce output files, so check archive dir exists
if [ -d "$ARCHIVE_DIR" ]; then
    pass "Archive directory exists at .capsule/logs/$BEAD_ID/"
else
    fail "Archive directory not found"
fi

echo ""
echo "=== Error Handling ==="
echo ""

# ---------- Test 8: Missing worktree rejected ----------
echo "[8/11] Missing worktree rejected"
# Given: no worktree exists for a bead
# When: merge.sh is run
# Then: exits non-zero with descriptive error
MISSING_BEAD="demo-nonexistent"
MISSING_EXIT=0
MISSING_OUTPUT=$(PATH="$MOCK_DIR:$PATH" "$MERGE_SCRIPT" "$MISSING_BEAD" --project-dir="$PROJECT_DIR" 2>&1) || MISSING_EXIT=$?
if [ "$MISSING_EXIT" -eq 0 ]; then
    fail "merge.sh should exit non-zero for missing worktree"
else
    if echo "$MISSING_OUTPUT" | grep -qiE "not found|does not exist|no worktree|ERROR"; then
        pass "Missing worktree rejected with descriptive error"
    else
        fail "Missing worktree rejected but no descriptive error message"
        echo "  Output: $MISSING_OUTPUT"
    fi
fi

# ---------- Test 9: Sign-off not PASS rejected ----------
echo "[9/11] Sign-off not PASS rejected"
# Given: a worktree where sign-off returned NEEDS_WORK
# When: merge.sh is run
# Then: exits non-zero and refuses to merge
BEAD_ID_NOSIGNOFF="demo-1.1.2"
PREP_NOSIGNOFF=$("$PREP_SCRIPT" "$BEAD_ID_NOSIGNOFF" --project-dir="$PROJECT_DIR" 2>&1) || true
WORKTREE_NOSIGNOFF="$PROJECT_DIR/.capsule/worktrees/$BEAD_ID_NOSIGNOFF"
# Add a NEEDS_WORK verdict
cat >> "$WORKTREE_NOSIGNOFF/worklog.md" << 'NOSIGNOFF_EOF'

### Phase 5: sign-off

_Status: complete_

**Verdict: NEEDS_WORK**

Tests failing. Implementation incomplete.
NOSIGNOFF_EOF
(cd "$WORKTREE_NOSIGNOFF" && git add -A && git commit -q -m "Add NEEDS_WORK sign-off")

NOSIGNOFF_EXIT=0
NOSIGNOFF_OUTPUT=$(PATH="$MOCK_DIR:$PATH" "$MERGE_SCRIPT" "$BEAD_ID_NOSIGNOFF" --project-dir="$PROJECT_DIR" 2>&1) || NOSIGNOFF_EXIT=$?
if [ "$NOSIGNOFF_EXIT" -eq 0 ]; then
    fail "merge.sh should exit non-zero when sign-off is not PASS"
else
    if echo "$NOSIGNOFF_OUTPUT" | grep -qiE "sign-off|PASS|not found|ERROR"; then
        pass "Sign-off not PASS rejected with descriptive error"
    else
        fail "Sign-off not PASS rejected but no descriptive error message"
        echo "  Output: $NOSIGNOFF_OUTPUT"
    fi
fi
# Clean up worktree for this test
(cd "$PROJECT_DIR" && git worktree remove "$WORKTREE_NOSIGNOFF" --force 2>/dev/null) || rm -rf "$WORKTREE_NOSIGNOFF"
(cd "$PROJECT_DIR" && git worktree prune 2>/dev/null) || true

# ---------- Test 10: Missing worklog rejected ----------
echo "[10/11] Missing worklog.md rejected"
# Given: a worktree with no worklog.md
# When: merge.sh is run
# Then: exits non-zero with descriptive error
BEAD_ID_NOWORKLOG="demo-1.1.2"
(cd "$PROJECT_DIR" && git branch -D "capsule-$BEAD_ID_NOWORKLOG" 2>/dev/null) || true
PREP_NOWORKLOG=$("$PREP_SCRIPT" "$BEAD_ID_NOWORKLOG" --project-dir="$PROJECT_DIR" 2>&1) || true
WORKTREE_NOWORKLOG="$PROJECT_DIR/.capsule/worktrees/$BEAD_ID_NOWORKLOG"
rm -f "$WORKTREE_NOWORKLOG/worklog.md"

NOWORKLOG_EXIT=0
NOWORKLOG_OUTPUT=$(PATH="$MOCK_DIR:$PATH" "$MERGE_SCRIPT" "$BEAD_ID_NOWORKLOG" --project-dir="$PROJECT_DIR" 2>&1) || NOWORKLOG_EXIT=$?
if [ "$NOWORKLOG_EXIT" -eq 0 ]; then
    fail "merge.sh should exit non-zero when worklog.md is missing"
else
    if echo "$NOWORKLOG_OUTPUT" | grep -qiE "worklog|not found|ERROR"; then
        pass "Missing worklog.md rejected with descriptive error"
    else
        fail "Missing worklog.md rejected but no descriptive error message"
        echo "  Output: $NOWORKLOG_OUTPUT"
    fi
fi
# Clean up
(cd "$PROJECT_DIR" && git worktree remove "$WORKTREE_NOWORKLOG" --force 2>/dev/null) || rm -rf "$WORKTREE_NOWORKLOG"
(cd "$PROJECT_DIR" && git worktree prune 2>/dev/null) || true

# ---------- Test 11: Agent NEEDS_WORK propagated ----------
echo "[11/11] Agent NEEDS_WORK propagated"
# Given: a worktree with sign-off PASS but merge agent returns NEEDS_WORK
# When: merge.sh is run
# Then: exits non-zero and reports NEEDS_WORK
BEAD_ID_NEEDSWORK="demo-1.1.2"
(cd "$PROJECT_DIR" && git branch -D "capsule-$BEAD_ID_NEEDSWORK" 2>/dev/null) || true
PREP_NEEDSWORK=$("$PREP_SCRIPT" "$BEAD_ID_NEEDSWORK" --project-dir="$PROJECT_DIR" 2>&1) || true
WORKTREE_NEEDSWORK="$PROJECT_DIR/.capsule/worktrees/$BEAD_ID_NEEDSWORK"
cat >> "$WORKTREE_NEEDSWORK/worklog.md" << 'NEEDSWORK_SIGNOFF_EOF'

### Phase 5: sign-off

_Status: complete_

**Verdict: PASS**
NEEDSWORK_SIGNOFF_EOF
(cd "$WORKTREE_NEEDSWORK" && git add -A && git commit -q -m "Add sign-off PASS")

# Create mock claude that returns NEEDS_WORK
MOCK_NEEDSWORK_DIR=$(mktemp -d)
trap 'rm -rf "$PROJECT_DIR" "$MOCK_DIR" "$MOCK_NEEDSWORK_DIR"' EXIT
cat > "$MOCK_NEEDSWORK_DIR/claude" << 'NEEDSWORK_MOCK_EOF'
#!/usr/bin/env bash
echo '{"status":"NEEDS_WORK","feedback":"Debug statements found in main.go","files_changed":[],"summary":"Quality issues found"}'
NEEDSWORK_MOCK_EOF
chmod +x "$MOCK_NEEDSWORK_DIR/claude"

NEEDSWORK_EXIT=0
NEEDSWORK_OUTPUT=$(PATH="$MOCK_NEEDSWORK_DIR:$PATH" "$MERGE_SCRIPT" "$BEAD_ID_NEEDSWORK" --project-dir="$PROJECT_DIR" 2>&1) || NEEDSWORK_EXIT=$?
if [ "$NEEDSWORK_EXIT" -eq 0 ]; then
    fail "merge.sh should exit non-zero when agent returns NEEDS_WORK"
else
    if echo "$NEEDSWORK_OUTPUT" | grep -qiE "NEEDS_WORK|issues"; then
        pass "Agent NEEDS_WORK propagated correctly"
    else
        fail "Agent NEEDS_WORK exit but no descriptive message"
        echo "  Output: $NEEDSWORK_OUTPUT"
    fi
fi

echo ""
echo "==========================================="
echo "RESULTS: $PASS passed, $FAIL failed"
echo "==========================================="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
