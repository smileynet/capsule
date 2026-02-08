#!/usr/bin/env bash
# run-pipeline.sh — Orchestrate the full capsule pipeline for a bead.
#
# Usage: run-pipeline.sh <bead-id> [--project-dir=DIR] [--max-retries=N]
#   bead-id:       The bead to run the pipeline for
#   --project-dir: Project root directory (default: current directory)
#   --max-retries: Maximum retries per phase pair (default: 3)
#
# Pipeline stages:
#   1. Prep: create worktree and worklog
#   2. Phase pair: test-writer → test-review (max retries)
#   3. Phase pair: execute → execute-review (max retries)
#   4. Sign-off (max retries; NEEDS_WORK retries execute phase)
#   5. Merge: agent-reviewed merge to main
#
# Exit codes:
#   0 — Pipeline completed successfully
#   1 — Pipeline failed (phase returned NEEDS_WORK and retries exhausted)
#   2 — Pipeline errored (phase returned ERROR or prerequisite failure)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PREP_SCRIPT="$SCRIPT_DIR/prep.sh"
RUN_PHASE="$SCRIPT_DIR/run-phase.sh"
MERGE_SCRIPT="$SCRIPT_DIR/merge.sh"

# --- Parse arguments ---
BEAD_ID=""
PROJECT_DIR="."
MAX_RETRIES=3

for arg in "$@"; do
    case "$arg" in
        --project-dir=*)
            PROJECT_DIR="${arg#--project-dir=}"
            ;;
        --max-retries=*)
            MAX_RETRIES="${arg#--max-retries=}"
            ;;
        -*)
            echo "ERROR: Unknown option: $arg" >&2
            echo "Usage: run-pipeline.sh <bead-id> [--project-dir=DIR] [--max-retries=N]" >&2
            exit 2
            ;;
        *)
            if [ -z "$BEAD_ID" ]; then
                BEAD_ID="$arg"
            else
                echo "ERROR: Unexpected argument: $arg" >&2
                echo "Usage: run-pipeline.sh <bead-id> [--project-dir=DIR] [--max-retries=N]" >&2
                exit 2
            fi
            ;;
    esac
done

if [ -z "$BEAD_ID" ]; then
    echo "ERROR: bead-id is required" >&2
    echo "Usage: run-pipeline.sh <bead-id> [--project-dir=DIR] [--max-retries=N]" >&2
    exit 2
fi

if ! [[ "$MAX_RETRIES" =~ ^[1-9][0-9]*$ ]]; then
    echo "ERROR: --max-retries must be a positive integer, got: $MAX_RETRIES" >&2
    exit 2
fi

# Resolve project directory to absolute path
PROJECT_DIR="$(cd "$PROJECT_DIR" && pwd)"

# --- Prerequisite checks ---
if ! command -v jq >/dev/null 2>&1; then
    echo "ERROR: jq is required but not installed" >&2
    exit 2
fi

for script in "$PREP_SCRIPT" "$RUN_PHASE" "$MERGE_SCRIPT"; do
    if [ ! -f "$script" ]; then
        echo "ERROR: $(basename "$script") not found at $script" >&2
        exit 2
    fi
done

# --- Tracking variables for summary ---
PIPELINE_START=$(date +%s)
TEST_REVIEW_ATTEMPTS=0
EXEC_REVIEW_ATTEMPTS=0
SIGNOFF_ATTEMPTS=0

# --- Helper: run a phase pair with retry ---
# Usage: run_phase_pair <writer-phase> <review-phase> <worktree> <max-retries> [attempts-var]
# Returns: 0 on PASS, 1 on retries exhausted, 2 on ERROR
# If attempts-var is provided, sets it to the number of attempts used.
run_phase_pair() {
    local writer_phase="$1"
    local review_phase="$2"
    local worktree="$3"
    local max_retries="$4"
    local attempts_var="${5:-}"
    local attempt=0
    local feedback=""

    while [ "$attempt" -lt "$max_retries" ]; do
        attempt=$((attempt + 1))
        echo "  [$attempt/$max_retries] Running $writer_phase..."

        # Run writer phase (with feedback on retry)
        local writer_exit=0
        local writer_output
        if [ -n "$feedback" ]; then
            writer_output=$("$RUN_PHASE" "$writer_phase" "$worktree" --feedback="$feedback" 2>&1) || writer_exit=$?
        else
            writer_output=$("$RUN_PHASE" "$writer_phase" "$worktree" 2>&1) || writer_exit=$?
        fi

        if [ "$writer_exit" -ne 0 ]; then
            echo "  ERROR: $writer_phase failed (exit $writer_exit)" >&2
            echo "$writer_output" >&2
            [ -n "$attempts_var" ] && eval "$attempts_var=$attempt"
            return 2
        fi

        # Run review phase
        echo "  [$attempt/$max_retries] Running $review_phase..."
        local review_exit=0
        local review_output
        review_output=$("$RUN_PHASE" "$review_phase" "$worktree" 2>&1) || review_exit=$?

        if [ "$review_exit" -eq 0 ]; then
            echo "  $review_phase: PASS"
            [ -n "$attempts_var" ] && eval "$attempts_var=$attempt"
            return 0
        fi

        if [ "$review_exit" -eq 2 ]; then
            echo "  ERROR: $review_phase failed" >&2
            echo "$review_output" >&2
            [ -n "$attempts_var" ] && eval "$attempts_var=$attempt"
            return 2
        fi

        # NEEDS_WORK — extract feedback for next attempt
        feedback=$(echo "$review_output" | jq -r '.feedback // empty' 2>/dev/null || echo "$review_output")
        echo "  $review_phase: NEEDS_WORK (attempt $attempt/$max_retries)"
    done

    [ -n "$attempts_var" ] && eval "$attempts_var=$attempt"
    echo "  Retries exhausted for $writer_phase → $review_phase ($max_retries attempts)" >&2
    return 1
}

# --- Helper: run sign-off with retry (retries go back to execute on NEEDS_WORK) ---
# Usage: run_signoff <worktree> <max-retries>
# Returns: 0 on PASS, 1 on retries exhausted, 2 on ERROR
run_signoff() {
    local worktree="$1"
    local max_retries="$2"
    local attempt=0

    while [ "$attempt" -lt "$max_retries" ]; do
        attempt=$((attempt + 1))
        echo "  [$attempt/$max_retries] Running sign-off..."

        local signoff_exit=0
        local signoff_output
        signoff_output=$("$RUN_PHASE" sign-off "$worktree" 2>&1) || signoff_exit=$?

        if [ "$signoff_exit" -eq 0 ]; then
            echo "  sign-off: PASS"
            SIGNOFF_ATTEMPTS=$attempt
            return 0
        fi

        if [ "$signoff_exit" -eq 2 ]; then
            echo "  ERROR: sign-off failed" >&2
            echo "$signoff_output" >&2
            SIGNOFF_ATTEMPTS=$attempt
            return 2
        fi

        # NEEDS_WORK — re-run execute phase before retrying sign-off
        local feedback
        feedback=$(echo "$signoff_output" | jq -r '.feedback // empty' 2>/dev/null || echo "$signoff_output")
        echo "  sign-off: NEEDS_WORK — re-running execute (attempt $attempt/$max_retries)"

        local exec_exit=0
        local exec_output
        exec_output=$("$RUN_PHASE" execute "$worktree" --feedback="$feedback" 2>&1) || exec_exit=$?
        if [ "$exec_exit" -ne 0 ]; then
            echo "  ERROR: execute failed during sign-off retry (exit $exec_exit)" >&2
            echo "$exec_output" >&2
            SIGNOFF_ATTEMPTS=$attempt
            return 2
        fi
    done

    SIGNOFF_ATTEMPTS=$attempt
    echo "  Retries exhausted for sign-off ($max_retries attempts)" >&2
    return 1
}

# --- Helper: run summary (never affects pipeline exit) ---
run_summary() {
    local outcome="$1"
    local failed_stage="${2:-}"
    local duration=$(( $(date +%s) - PIPELINE_START ))

    if [ -f "$SCRIPT_DIR/run-summary.sh" ]; then
        echo ""
        local summary_args=(
            "$BEAD_ID"
            "--project-dir=$PROJECT_DIR"
            "--outcome=$outcome"
            "--test-review-attempts=$TEST_REVIEW_ATTEMPTS"
            "--exec-review-attempts=$EXEC_REVIEW_ATTEMPTS"
            "--signoff-attempts=$SIGNOFF_ATTEMPTS"
            "--max-retries=$MAX_RETRIES"
            "--duration=$duration"
        )
        if [ -n "$failed_stage" ]; then
            summary_args+=("--failed-stage=$failed_stage")
        fi
        "$SCRIPT_DIR/run-summary.sh" "${summary_args[@]}" || true
    fi
}

# =============================================================================
# Pipeline execution
# =============================================================================

echo "=== Capsule Pipeline: $BEAD_ID ==="
echo ""

# --- Stage 1: Prep ---
echo "[1/5] Prep"
PREP_EXIT=0
PREP_OUTPUT=$("$PREP_SCRIPT" "$BEAD_ID" --project-dir="$PROJECT_DIR" 2>&1) || PREP_EXIT=$?

if [ "$PREP_EXIT" -ne 0 ]; then
    echo "ERROR: Prep failed" >&2
    echo "$PREP_OUTPUT" >&2
    exit 2
fi

WORKTREE_DIR="$PROJECT_DIR/.capsule/worktrees/$BEAD_ID"
echo "  Worktree: $WORKTREE_DIR"
echo ""

# --- Stage 2: test-writer → test-review ---
echo "[2/5] Phase pair: test-writer → test-review"
PAIR_EXIT=0
run_phase_pair "test-writer" "test-review" "$WORKTREE_DIR" "$MAX_RETRIES" TEST_REVIEW_ATTEMPTS || PAIR_EXIT=$?

if [ "$PAIR_EXIT" -ne 0 ]; then
    echo ""
    echo "Pipeline aborted at test-writer/test-review (exit $PAIR_EXIT)" >&2
    echo "Worktree preserved: $WORKTREE_DIR" >&2
    run_summary "FAILED" "test-writer/test-review"
    exit "$PAIR_EXIT"
fi
echo ""

# --- Stage 3: execute → execute-review ---
echo "[3/5] Phase pair: execute → execute-review"
PAIR_EXIT=0
run_phase_pair "execute" "execute-review" "$WORKTREE_DIR" "$MAX_RETRIES" EXEC_REVIEW_ATTEMPTS || PAIR_EXIT=$?

if [ "$PAIR_EXIT" -ne 0 ]; then
    echo ""
    echo "Pipeline aborted at execute/execute-review (exit $PAIR_EXIT)" >&2
    echo "Worktree preserved: $WORKTREE_DIR" >&2
    run_summary "FAILED" "execute/execute-review"
    exit "$PAIR_EXIT"
fi
echo ""

# --- Stage 4: Sign-off ---
echo "[4/5] Sign-off"
SIGNOFF_EXIT=0
run_signoff "$WORKTREE_DIR" "$MAX_RETRIES" || SIGNOFF_EXIT=$?

if [ "$SIGNOFF_EXIT" -ne 0 ]; then
    echo ""
    echo "Pipeline aborted at sign-off (exit $SIGNOFF_EXIT)" >&2
    echo "Worktree preserved: $WORKTREE_DIR" >&2
    run_summary "FAILED" "sign-off"
    exit "$SIGNOFF_EXIT"
fi
echo ""

# --- Stage 5: Merge ---
echo "[5/5] Merge"
MERGE_EXIT=0
MERGE_OUTPUT=$("$MERGE_SCRIPT" "$BEAD_ID" --project-dir="$PROJECT_DIR" 2>&1) || MERGE_EXIT=$?

if [ "$MERGE_EXIT" -ne 0 ]; then
    echo "ERROR: Merge failed" >&2
    echo "$MERGE_OUTPUT" >&2
    echo "Worktree preserved: $WORKTREE_DIR" >&2
    run_summary "FAILED" "merge"
    exit "$MERGE_EXIT"
fi
echo "  Merge: complete"
echo ""

# --- Summary ---
run_summary "SUCCESS"
echo ""
echo "=== Pipeline Complete ==="
echo "  Bead: $BEAD_ID"
echo "  Status: SUCCESS"
echo ""
