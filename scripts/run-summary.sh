#!/usr/bin/env bash
# run-summary.sh — Generate a narrative summary of a pipeline run.
#
# Usage: run-summary.sh <bead-id> [options]
#   bead-id:                  The bead the pipeline ran for
#   --project-dir=DIR:        Project root directory (default: current directory)
#   --outcome=SUCCESS|FAILED|ERROR: Pipeline outcome
#   --failed-stage=STAGE:     Stage where pipeline failed (if applicable)
#   --test-review-attempts=N: Number of test-review attempts
#   --exec-review-attempts=N: Number of execute-review attempts
#   --signoff-attempts=N:     Number of sign-off attempts
#   --max-retries=N:          Max retries configured for pipeline
#   --duration=SECONDS:       Total pipeline duration in seconds
#   --last-feedback=TEXT:     Last review feedback from failed phase
#
# Gathers structured context (worklog, retry counts, bead hierarchy progress)
# and invokes Claude to produce a narrative summary.
#
# Output destinations:
#   - stdout (always)
#   - .capsule/logs/<bead-id>/summary.md (saved to archive)
#   - bead comment via bd (if bd is available)
#
# Exit codes:
#   0 — Summary generated successfully
#   1 — Summary generation failed (non-fatal, never affects pipeline exit)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PROMPT_FILE="$REPO_ROOT/prompts/summary.md"

# --- Parse arguments ---
BEAD_ID=""
PROJECT_DIR="."
OUTCOME="UNKNOWN"
FAILED_STAGE=""
TEST_REVIEW_ATTEMPTS=0
EXEC_REVIEW_ATTEMPTS=0
SIGNOFF_ATTEMPTS=0
MAX_RETRIES=3
DURATION=0
LAST_FEEDBACK=""

for arg in "$@"; do
    case "$arg" in
        --project-dir=*)
            PROJECT_DIR="${arg#--project-dir=}"
            ;;
        --outcome=*)
            OUTCOME="${arg#--outcome=}"
            ;;
        --failed-stage=*)
            FAILED_STAGE="${arg#--failed-stage=}"
            ;;
        --test-review-attempts=*)
            TEST_REVIEW_ATTEMPTS="${arg#--test-review-attempts=}"
            ;;
        --exec-review-attempts=*)
            EXEC_REVIEW_ATTEMPTS="${arg#--exec-review-attempts=}"
            ;;
        --signoff-attempts=*)
            SIGNOFF_ATTEMPTS="${arg#--signoff-attempts=}"
            ;;
        --max-retries=*)
            MAX_RETRIES="${arg#--max-retries=}"
            ;;
        --duration=*)
            DURATION="${arg#--duration=}"
            ;;
        --last-feedback=*)
            LAST_FEEDBACK="${arg#--last-feedback=}"
            ;;
        -*)
            echo "ERROR: Unknown option: $arg" >&2
            exit 1
            ;;
        *)
            if [ -z "$BEAD_ID" ]; then
                BEAD_ID="$arg"
            else
                echo "ERROR: Unexpected argument: $arg" >&2
                exit 1
            fi
            ;;
    esac
done

if [ -z "$BEAD_ID" ]; then
    echo "ERROR: bead-id is required" >&2
    echo "Usage: run-summary.sh <bead-id> [--project-dir=DIR] [--outcome=SUCCESS|FAILED|ERROR] ..." >&2
    exit 1
fi

# Resolve project directory to absolute path
PROJECT_DIR="$(cd "$PROJECT_DIR" && pwd)"

# --- Prerequisite checks ---
if ! command -v claude >/dev/null 2>&1; then
    echo "ERROR: claude is required but not installed" >&2
    exit 1
fi

if [ ! -f "$PROMPT_FILE" ]; then
    echo "ERROR: summary prompt not found at $PROMPT_FILE" >&2
    exit 1
fi

# --- Locate worklog ---
WORKLOG=""
ARCHIVE_DIR="$PROJECT_DIR/.capsule/logs/$BEAD_ID"
WORKTREE_DIR="$PROJECT_DIR/.capsule/worktrees/$BEAD_ID"

# Try archive first (post-merge), then worktree (failure case)
if [ -f "$ARCHIVE_DIR/worklog.md" ]; then
    WORKLOG="$ARCHIVE_DIR/worklog.md"
elif [ -f "$WORKTREE_DIR/worklog.md" ]; then
    WORKLOG="$WORKTREE_DIR/worklog.md"
fi

WORKLOG_CONTENTS=""
if [ -n "$WORKLOG" ]; then
    WORKLOG_CONTENTS=$(cat "$WORKLOG")
fi

# --- Query bead hierarchy (graceful degradation if bd or jq missing) ---
if command -v bd >/dev/null 2>&1 && command -v jq >/dev/null 2>&1; then
    HAS_BD=true
else
    HAS_BD=false
fi

BEAD_TITLE=""
FEATURE_PROGRESS=""
EPIC_PROGRESS=""

if $HAS_BD; then
    # Get task metadata
    BEAD_JSON=$(cd "$PROJECT_DIR" && bd show "$BEAD_ID" --json 2>/dev/null) || true
    if [ -n "$BEAD_JSON" ]; then
        BEAD_TITLE=$(echo "$BEAD_JSON" | jq -r '.[0].title // empty' 2>/dev/null) || true

        # Walk parent chain to find feature and epic
        source "$SCRIPT_DIR/lib/resolve-parent-chain.sh"
        resolve_parent_chain "$PROJECT_DIR" "$BEAD_JSON"

        # Query feature progress (sibling tasks)
        if [ -n "$FEATURE_ID" ]; then
            SIBLING_JSON=$(cd "$PROJECT_DIR" && bd list --parent="$FEATURE_ID" --all --json 2>/dev/null) || true
            if [ -n "$SIBLING_JSON" ]; then
                TOTAL=$(echo "$SIBLING_JSON" | jq 'length' 2>/dev/null) || TOTAL=0
                CLOSED=$(echo "$SIBLING_JSON" | jq '[.[] | select(.status == "closed")] | length' 2>/dev/null) || CLOSED=0
                FEATURE_PROGRESS="$CLOSED of $TOTAL tasks closed"
            fi
        fi

        # Query epic progress (child features)
        if [ -n "$EPIC_ID" ]; then
            EPIC_CHILDREN_JSON=$(cd "$PROJECT_DIR" && bd list --parent="$EPIC_ID" --all --json 2>/dev/null) || true
            if [ -n "$EPIC_CHILDREN_JSON" ]; then
                EPIC_TOTAL=$(echo "$EPIC_CHILDREN_JSON" | jq 'length' 2>/dev/null) || EPIC_TOTAL=0
                EPIC_CLOSED=$(echo "$EPIC_CHILDREN_JSON" | jq '[.[] | select(.status == "closed")] | length' 2>/dev/null) || EPIC_CLOSED=0
                EPIC_PROGRESS="$EPIC_CLOSED of $EPIC_TOTAL features closed"
            fi
        fi
    fi
fi

# --- Assemble context block ---
CONTEXT="### Pipeline Outcome
- **Bead:** $BEAD_ID${BEAD_TITLE:+ — $BEAD_TITLE}
- **Outcome:** $OUTCOME"

if [ -n "$FAILED_STAGE" ]; then
    CONTEXT="$CONTEXT
- **Failed at stage:** $FAILED_STAGE"
fi

CONTEXT="$CONTEXT
- **Duration:** ${DURATION}s
- **Max retries configured:** $MAX_RETRIES

### Retry History
- test-writer/test-review attempts: $TEST_REVIEW_ATTEMPTS
- execute/execute-review attempts: $EXEC_REVIEW_ATTEMPTS
- sign-off attempts: $SIGNOFF_ATTEMPTS"

if [ -n "$LAST_FEEDBACK" ]; then
    CONTEXT="$CONTEXT

### Last Review Feedback
$LAST_FEEDBACK"
fi

if [ -n "$WORKLOG_CONTENTS" ]; then
    CONTEXT="$CONTEXT

### Worklog Contents
$WORKLOG_CONTENTS"
else
    CONTEXT="$CONTEXT

### Worklog Contents
(no worklog found)"
fi

CONTEXT="$CONTEXT

### Feature & Epic Hierarchy"

if [ -n "$FEATURE_ID" ]; then
    CONTEXT="$CONTEXT
- **Feature:** $FEATURE_ID — $FEATURE_TITLE
- **Feature progress:** ${FEATURE_PROGRESS:-unknown}"
fi

if [ -n "$EPIC_ID" ]; then
    CONTEXT="$CONTEXT
- **Epic:** $EPIC_ID — $EPIC_TITLE
- **Epic progress:** ${EPIC_PROGRESS:-unknown}"
fi

if [ -z "$FEATURE_ID" ] && [ -z "$EPIC_ID" ]; then
    CONTEXT="$CONTEXT
(no feature/epic hierarchy found)"
fi

# --- Build prompt ---
# Use awk for safe substitution — bash ${//} treats & and \ as special in replacement
CONTEXT_FILE=$(mktemp)
trap 'rm -f "$CONTEXT_FILE"' EXIT
printf '%s\n' "$CONTEXT" > "$CONTEXT_FILE"

PROMPT=$(awk -v ctx_file="$CONTEXT_FILE" '
  /\{\{CONTEXT\}\}/ {
    while ((getline line < ctx_file) > 0) print line
    close(ctx_file)
    next
  }
  {print}
' "$PROMPT_FILE")
rm -f "$CONTEXT_FILE"

# --- Invoke Claude ---
SUMMARY_TIMEOUT="${CAPSULE_SUMMARY_TIMEOUT:-600}"
CLAUDE_OUTPUT=""
CLAUDE_EXIT=0
CLAUDE_OUTPUT=$(cd "$PROJECT_DIR" && timeout "$SUMMARY_TIMEOUT" claude -p "$PROMPT" --dangerously-skip-permissions 2>/dev/null) || CLAUDE_EXIT=$?

if [ "$CLAUDE_EXIT" -eq 124 ]; then
    echo "WARNING: Summary generation timed out after ${SUMMARY_TIMEOUT}s" >&2
    exit 1
fi

if [ "$CLAUDE_EXIT" -ne 0 ]; then
    echo "WARNING: Summary generation failed (claude exit $CLAUDE_EXIT)" >&2
    exit 1
fi

if [ -z "$CLAUDE_OUTPUT" ]; then
    echo "WARNING: Summary generation produced no output" >&2
    exit 1
fi

# --- Output to stdout ---
printf '%s\n' "$CLAUDE_OUTPUT"

# --- Save to archive ---
mkdir -p "$ARCHIVE_DIR"
printf '%s\n' "$CLAUDE_OUTPUT" > "$ARCHIVE_DIR/summary.md"

# --- Post as bead comment ---
if $HAS_BD; then
    (cd "$PROJECT_DIR" && bd comments add "$BEAD_ID" "$CLAUDE_OUTPUT" 2>/dev/null) || true
fi
