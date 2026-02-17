#!/usr/bin/env bash
# merge.sh — Agent-reviewed merge driver for the capsule pipeline.
#
# Usage: merge.sh <bead-id> [--project-dir=DIR]
#   bead-id:       The bead whose worktree will be merged to main
#   --project-dir: Project root directory (default: current directory)
#
# Steps:
#   1. Validate worktree exists and sign-off PASS is in worklog
#   2. Invoke claude merge agent (via run-phase.sh) to review and commit in worktree
#   3. Switch to main and merge --no-ff
#   4. Archive worklog and phase outputs to .capsule/logs/<bead-id>/
#   5. Remove worktree and delete branch
#   6. Close the bead
#
# Exit codes:
#   0 - Merge completed successfully
#   1 - Merge failed (agent returned NEEDS_WORK or precondition not met)
#   2 - Error (missing dependencies, invalid arguments)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
RUN_PHASE="$SCRIPT_DIR/run-phase.sh"

# --- Parse arguments ---
BEAD_ID=""
PROJECT_DIR="."
MAIN_BRANCH_ARG=""

while [ $# -gt 0 ]; do
    case "$1" in
        --project-dir=*)
            PROJECT_DIR="${1#--project-dir=}"
            shift
            ;;
        --main-branch=*)
            MAIN_BRANCH_ARG="${1#--main-branch=}"
            shift
            ;;
        -*)
            echo "ERROR: Unknown option: $1" >&2
            echo "Usage: merge.sh <bead-id> [--project-dir=DIR] [--main-branch=BRANCH]" >&2
            exit 2
            ;;
        *)
            if [ -z "$BEAD_ID" ]; then
                BEAD_ID="$1"
            else
                echo "ERROR: Unexpected argument: $1" >&2
                echo "Usage: merge.sh <bead-id> [--project-dir=DIR] [--main-branch=BRANCH]" >&2
                exit 2
            fi
            shift
            ;;
    esac
done

if [ -z "$BEAD_ID" ]; then
    echo "ERROR: bead-id is required" >&2
    echo "Usage: merge.sh <bead-id> [--project-dir=DIR] [--main-branch=BRANCH]" >&2
    exit 2
fi

# Resolve project directory to absolute path
PROJECT_DIR="$(cd "$PROJECT_DIR" && pwd)"

# --- Prerequisite checks ---
for cmd in git bd jq claude; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "ERROR: $cmd is required but not installed" >&2
        exit 2
    fi
done

if [ ! -f "$RUN_PHASE" ]; then
    echo "ERROR: run-phase.sh not found at $RUN_PHASE" >&2
    exit 2
fi

# --- Validate worktree exists ---
WORKTREE_DIR="$PROJECT_DIR/.capsule/worktrees/$BEAD_ID"
BRANCH_NAME="capsule-$BEAD_ID"

if [ ! -d "$WORKTREE_DIR" ]; then
    echo "ERROR: Worktree does not exist: $WORKTREE_DIR" >&2
    echo "" >&2
    echo "To fix: Run prep first, or use the full pipeline:" >&2
    echo "  scripts/prep.sh $BEAD_ID --project-dir=$PROJECT_DIR" >&2
    echo "  scripts/run-pipeline.sh $BEAD_ID --project-dir=$PROJECT_DIR" >&2
    exit 1
fi

# --- Validate sign-off PASS in worklog ---
WORKLOG="$WORKTREE_DIR/worklog.md"
if [ ! -f "$WORKLOG" ]; then
    echo "ERROR: worklog.md not found in worktree" >&2
    exit 1
fi

if ! grep -q 'Verdict: PASS' "$WORKLOG"; then
    echo "ERROR: Sign-off PASS not found in worklog.md. Cannot merge without sign-off." >&2
    echo "" >&2
    echo "To fix: Run the pipeline to get sign-off, or inspect the worklog:" >&2
    echo "  scripts/run-pipeline.sh $BEAD_ID --project-dir=$PROJECT_DIR" >&2
    echo "  cat $WORKLOG | grep -A2 'Sign-off'" >&2
    exit 1
fi

# --- Get bead info for commit message ---
BEAD_JSON=$(cd "$PROJECT_DIR" && bd show "$BEAD_ID" --json 2>/dev/null) || {
    echo "ERROR: Could not read bead '$BEAD_ID'" >&2
    exit 2
}
TASK_TITLE=$(printf '%s\n' "$BEAD_JSON" | jq -r '.[0].title // empty')
COMMIT_MSG="$BEAD_ID: $TASK_TITLE"

# --- Invoke merge agent ---
echo "Running merge agent..."
export CAPSULE_COMMIT_MSG="$COMMIT_MSG"

PHASE_EXIT=0
PHASE_OUTPUT=$("$RUN_PHASE" merge "$WORKTREE_DIR" 2>&1) || PHASE_EXIT=$?

if [ "$PHASE_EXIT" -eq 2 ]; then
    echo "ERROR: Merge agent failed" >&2
    printf '%s\n' "$PHASE_OUTPUT" >&2
    exit 2
fi

if [ "$PHASE_EXIT" -eq 1 ]; then
    echo "NEEDS_WORK: Merge agent found issues" >&2
    printf '%s\n' "$PHASE_OUTPUT" >&2
    exit 1
fi

echo "Merge agent: PASS"

# --- Determine main branch ---
if [ -n "$MAIN_BRANCH_ARG" ]; then
    MAIN_BRANCH="$MAIN_BRANCH_ARG"
else
    MAIN_BRANCH=$(cd "$PROJECT_DIR" && git symbolic-ref refs/remotes/origin/HEAD 2>/dev/null | sed 's@^refs/remotes/origin/@@' || true)
    if [ -z "$MAIN_BRANCH" ]; then
        # No remote HEAD, try local branches
        if (cd "$PROJECT_DIR" && git rev-parse --verify main >/dev/null 2>&1); then
            MAIN_BRANCH="main"
        elif (cd "$PROJECT_DIR" && git rev-parse --verify master >/dev/null 2>&1); then
            MAIN_BRANCH="master"
        else
            echo "ERROR: Could not determine main branch" >&2
            exit 2
        fi
    fi
fi

# --- Merge to main ---
(
    cd "$PROJECT_DIR"
    git checkout "$MAIN_BRANCH" -q

    # Merge with --no-ff to preserve branch history
    git merge --no-ff "$BRANCH_NAME" -m "Merge $COMMIT_MSG" -q || {
        echo "ERROR: Merge conflict merging $BRANCH_NAME into $MAIN_BRANCH." >&2
        git merge --abort 2>/dev/null || true
        echo "" >&2
        echo "To fix: Resolve the conflict manually:" >&2
        echo "  cd $PROJECT_DIR" >&2
        echo "  git checkout $MAIN_BRANCH" >&2
        echo "  git merge --no-ff $BRANCH_NAME" >&2
        echo "  # resolve conflicts, then:" >&2
        echo "  git add . && git commit" >&2
        exit 1
    }

    echo "Merged $BRANCH_NAME to $MAIN_BRANCH"
)

# --- Archive worklog and phase outputs (after successful merge) ---
ARCHIVE_DIR="$PROJECT_DIR/.capsule/logs/$BEAD_ID"
mkdir -p "$ARCHIVE_DIR"

# Archive worklog
cp "$WORKLOG" "$ARCHIVE_DIR/worklog.md"

# Archive phase outputs if they exist
if [ -d "$WORKTREE_DIR/.capsule/output" ]; then
    cp -r "$WORKTREE_DIR/.capsule/output/"* "$ARCHIVE_DIR/" 2>/dev/null || true
fi

echo "Archived worklog to $ARCHIVE_DIR/"

# --- Remove worktree ---
if ! (cd "$PROJECT_DIR" && git worktree remove "$WORKTREE_DIR" --force 2>/dev/null); then
    # Fallback: manual removal
    rm -rf "$WORKTREE_DIR"
    (cd "$PROJECT_DIR" && git worktree prune 2>/dev/null) || true
fi
echo "Removed worktree"

# --- Delete branch ---
if ! (cd "$PROJECT_DIR" && git branch -d "$BRANCH_NAME" 2>/dev/null); then
    (cd "$PROJECT_DIR" && git branch -D "$BRANCH_NAME" 2>/dev/null) || true
fi
echo "Deleted branch $BRANCH_NAME"

# --- Close bead ---
BEAD_STATUS=$(cd "$PROJECT_DIR" && bd show "$BEAD_ID" --json 2>/dev/null | jq -r '.[0].status // empty' 2>/dev/null) || true
if [ "$BEAD_STATUS" = "closed" ]; then
    echo "Bead $BEAD_ID already closed, skipping"
else
    if ! (cd "$PROJECT_DIR" && bd close "$BEAD_ID" 2>/dev/null); then
        echo "WARNING: Could not close bead $BEAD_ID" >&2
    else
        echo "Closed bead $BEAD_ID"
    fi
fi
echo ""
echo "Merge complete: $COMMIT_MSG"
