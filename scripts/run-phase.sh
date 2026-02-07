#!/usr/bin/env bash
# run-phase.sh — Invoke a capsule pipeline phase via headless claude.
#
# Usage: run-phase.sh <phase-name> <worktree-path> [--feedback=...]
#   phase-name:    Name of the prompt template (e.g., test-writer, test-review)
#   worktree-path: Path to the git worktree to run the phase in
#   --feedback:    Optional feedback from a previous review (appended to prompt)
#
# Loads the prompt from prompts/<phase-name>.md, invokes claude -p in the
# worktree directory, captures stdout, parses the signal via parse-signal.sh,
# and prints the parsed signal to stdout.
#
# Exit codes:
#   0 — PASS (phase completed successfully)
#   1 — NEEDS_WORK (phase found issues, feedback available)
#   2 — ERROR (phase failed or could not run)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
PARSE_SIGNAL="$SCRIPT_DIR/parse-signal.sh"

# --- Parse arguments ---
PHASE_NAME=""
WORKTREE_PATH=""
FEEDBACK=""

for arg in "$@"; do
    case "$arg" in
        --feedback=*)
            FEEDBACK="${arg#--feedback=}"
            ;;
        -*)
            echo "ERROR: Unknown option: $arg" >&2
            echo "Usage: run-phase.sh <phase-name> <worktree-path> [--feedback=...]" >&2
            exit 2
            ;;
        *)
            if [ -z "$PHASE_NAME" ]; then
                PHASE_NAME="$arg"
            elif [ -z "$WORKTREE_PATH" ]; then
                WORKTREE_PATH="$arg"
            else
                echo "ERROR: Unexpected argument: $arg" >&2
                echo "Usage: run-phase.sh <phase-name> <worktree-path> [--feedback=...]" >&2
                exit 2
            fi
            ;;
    esac
done

if [ -z "$PHASE_NAME" ]; then
    echo "ERROR: phase-name is required" >&2
    echo "Usage: run-phase.sh <phase-name> <worktree-path> [--feedback=...]" >&2
    exit 2
fi

if [ -z "$WORKTREE_PATH" ]; then
    echo "ERROR: worktree-path is required" >&2
    echo "Usage: run-phase.sh <phase-name> <worktree-path> [--feedback=...]" >&2
    exit 2
fi

# --- Prerequisite checks ---
for cmd in claude jq; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "ERROR: $cmd is required but not installed" >&2
        exit 2
    fi
done

if [ ! -f "$PARSE_SIGNAL" ]; then
    echo "ERROR: parse-signal.sh not found at $PARSE_SIGNAL" >&2
    exit 2
fi

# --- Validate worktree path ---
if [ ! -d "$WORKTREE_PATH" ]; then
    echo "ERROR: Worktree path does not exist: $WORKTREE_PATH" >&2
    exit 2
fi

# --- Validate prompt template ---
PROMPT_FILE="$REPO_ROOT/prompts/$PHASE_NAME.md"
if [ ! -f "$PROMPT_FILE" ]; then
    echo "ERROR: Prompt template not found for phase '$PHASE_NAME': $PROMPT_FILE" >&2
    exit 2
fi

# --- Build prompt ---
PROMPT=$(cat "$PROMPT_FILE")

if [ -n "$FEEDBACK" ]; then
    PROMPT="$PROMPT

---

## Previous Feedback

The previous review returned NEEDS_WORK with the following feedback. Address these issues:

$FEEDBACK"
fi

# --- Create output directory ---
OUTPUT_DIR="$WORKTREE_PATH/.capsule/output"
mkdir -p "$OUTPUT_DIR"

TIMESTAMP=$(date -u +"%Y%m%d-%H%M%S")-$$
LOG_FILE="$OUTPUT_DIR/$PHASE_NAME-$TIMESTAMP.log"

# --- Invoke claude ---
CLAUDE_EXIT=0
CLAUDE_OUTPUT=$(cd "$WORKTREE_PATH" && claude -p "$PROMPT" --dangerously-skip-permissions 2>"$LOG_FILE.stderr") || CLAUDE_EXIT=$?

# --- Capture output to log (stdout only; stderr in separate .stderr file) ---
printf '%s\n' "$CLAUDE_OUTPUT" > "$LOG_FILE"

# --- Handle claude failure ---
if [ "$CLAUDE_EXIT" -ne 0 ]; then
    echo "ERROR: claude exited with code $CLAUDE_EXIT" >&2
    # Try to parse signal anyway in case claude produced output before failing
    SIGNAL=$(printf '%s\n' "$CLAUDE_OUTPUT" | "$PARSE_SIGNAL" 2>/dev/null) || true
    if [ -n "$SIGNAL" ]; then
        printf '%s\n' "$SIGNAL"
    fi
    exit 2
fi

# --- Parse signal ---
PARSE_EXIT=0
SIGNAL=$(printf '%s\n' "$CLAUDE_OUTPUT" | "$PARSE_SIGNAL" 2>/dev/null) || PARSE_EXIT=$?

if [ "$PARSE_EXIT" -ne 0 ]; then
    # parse-signal.sh returns 1 for no valid signal, prints synthetic ERROR
    printf '%s\n' "$SIGNAL"
    exit 2
fi

# --- Map status to exit code ---
STATUS=$(printf '%s\n' "$SIGNAL" | jq -r '.status')
printf '%s\n' "$SIGNAL"

case "$STATUS" in
    PASS)
        exit 0
        ;;
    NEEDS_WORK)
        exit 1
        ;;
    ERROR)
        exit 2
        ;;
    *)
        exit 2
        ;;
esac
