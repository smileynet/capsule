#!/usr/bin/env bash
# prep.sh — Create a git worktree and instantiate a worklog for a bead.
#
# Usage: prep.sh <bead-id> [--project-dir=DIR]
#   bead-id:      The bead to prepare a worktree for
#   --project-dir: Project root directory (default: current directory)
#
# Creates:
#   .capsule/worktrees/<bead-id>/   — git worktree on branch capsule-<bead-id>
#   .capsule/worktrees/<bead-id>/worklog.md — instantiated from template
#   .capsule/logs/                  — log directory (if not exists)
#
# Exits non-zero with an error message on any failure.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TEMPLATE="$REPO_ROOT/templates/worklog.md.template"

# --- Parse arguments ---
BEAD_ID=""
PROJECT_DIR="."

for arg in "$@"; do
    case "$arg" in
        --project-dir=*)
            PROJECT_DIR="${arg#--project-dir=}"
            ;;
        -*)
            echo "ERROR: Unknown option: $arg" >&2
            echo "Usage: prep.sh <bead-id> [--project-dir=DIR]" >&2
            exit 1
            ;;
        *)
            if [ -z "$BEAD_ID" ]; then
                BEAD_ID="$arg"
            else
                echo "ERROR: Unexpected argument: $arg" >&2
                echo "Usage: prep.sh <bead-id> [--project-dir=DIR]" >&2
                exit 1
            fi
            ;;
    esac
done

if [ -z "$BEAD_ID" ]; then
    echo "ERROR: bead-id is required" >&2
    echo "Usage: prep.sh <bead-id> [--project-dir=DIR]" >&2
    exit 1
fi

# Resolve project directory to absolute path
PROJECT_DIR="$(cd "$PROJECT_DIR" && pwd)"

# --- Prerequisite checks ---
for cmd in git bd jq; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
        echo "ERROR: $cmd is required but not installed" >&2
        exit 1
    fi
done

if [ ! -f "$TEMPLATE" ]; then
    echo "ERROR: worklog template not found: $TEMPLATE" >&2
    exit 1
fi

# --- Validate bead exists ---
BEAD_JSON=$(cd "$PROJECT_DIR" && bd show "$BEAD_ID" --json 2>/dev/null) || {
    echo "ERROR: Bead '$BEAD_ID' does not exist or could not be read" >&2
    exit 1
}

TASK_TITLE=$(printf '%s\n' "$BEAD_JSON" | jq -r '.[0].title // empty')
if [ -z "$TASK_TITLE" ]; then
    echo "ERROR: Bead '$BEAD_ID' has no title (malformed or missing)" >&2
    exit 1
fi

# --- Check worktree doesn't already exist ---
WORKTREE_DIR="$PROJECT_DIR/.capsule/worktrees/$BEAD_ID"
if [ -d "$WORKTREE_DIR" ]; then
    echo "ERROR: Worktree already exists at $WORKTREE_DIR" >&2
    # Diagnose what state exists
    if [ -f "$WORKTREE_DIR/worklog.md" ]; then
        echo "  State: worklog.md exists (prep was completed previously)" >&2
    else
        echo "  State: directory exists but no worklog.md (partial prep)" >&2
    fi
    PHASE_FILES=$(find "$WORKTREE_DIR/.capsule/output" -name "${BEAD_ID}*" 2>/dev/null | head -5) || true
    if [ -n "$PHASE_FILES" ]; then
        echo "  Phase outputs found — pipeline was partially run" >&2
    fi
    echo "" >&2
    echo "To fix (choose one):" >&2
    echo "  1. Re-run with --clean:  run-pipeline.sh $BEAD_ID --clean --project-dir=$PROJECT_DIR" >&2
    echo "  2. Remove manually:      rm -rf $WORKTREE_DIR" >&2
    echo "  3. Full teardown (all worktrees): scripts/teardown.sh --project-dir=$PROJECT_DIR" >&2
    exit 1
fi

# --- Extract bead context ---
TASK_DESCRIPTION=$(printf '%s\n' "$BEAD_JSON" | jq -r '.[0].description // empty')
TASK_AC=$(printf '%s\n' "$BEAD_JSON" | jq -r '.[0].acceptance_criteria // empty')

# Walk up the parent chain to find feature and epic
source "$SCRIPT_DIR/lib/resolve-parent-chain.sh"
resolve_parent_chain "$PROJECT_DIR" "$BEAD_JSON"

# --- Create worktree ---
BRANCH_NAME="capsule-$BEAD_ID"
mkdir -p "$(dirname "$WORKTREE_DIR")"

GIT_WT_ERR=""
GIT_WT_ERR=$( (cd "$PROJECT_DIR" && git worktree add -b "$BRANCH_NAME" "$WORKTREE_DIR" HEAD) 2>&1) || {
    echo "ERROR: Failed to create git worktree" >&2
    printf '%s\n' "$GIT_WT_ERR" >&2
    # Clean up partial directory if created
    [ -d "$WORKTREE_DIR" ] && rm -rf "$WORKTREE_DIR"
    if printf '%s\n' "$GIT_WT_ERR" | grep -q "already exists"; then
        echo "" >&2
        echo "To fix: Delete the stale branch and retry:" >&2
        echo "  cd $PROJECT_DIR && git branch -D $BRANCH_NAME" >&2
    fi
    exit 1
}

# --- Cleanup trap: remove worktree if subsequent steps fail ---
cleanup_worktree() {
    (cd "$PROJECT_DIR" && git worktree remove --force "$WORKTREE_DIR" 2>/dev/null) || rm -rf "$WORKTREE_DIR"
    (cd "$PROJECT_DIR" && git branch -D "$BRANCH_NAME" 2>/dev/null) || true
}
trap cleanup_worktree ERR

# --- Create .capsule/logs directory ---
mkdir -p "$PROJECT_DIR/.capsule/logs"

# --- Instantiate worklog from template ---
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Use acceptance_criteria field if present, otherwise extract from description
ACCEPTANCE_CRITERIA="$TASK_AC"
if [ -z "$ACCEPTANCE_CRITERIA" ]; then
    ACCEPTANCE_CRITERIA=$(printf '%s\n' "$TASK_DESCRIPTION" | \
      sed -n '/^#*[[:space:]]*[Aa]cceptance [Cc]riteria/,/^#/{/^#*[[:space:]]*[Aa]cceptance [Cc]riteria/d;/^#/d;p;}')
fi
if [ -z "$ACCEPTANCE_CRITERIA" ]; then
    ACCEPTANCE_CRITERIA=$(printf '%s\n' "$TASK_DESCRIPTION" | \
      sed -n '/^#*[[:space:]]*[Rr]equirements/,/^#/{/^#*[[:space:]]*[Rr]equirements/d;/^#/d;p;}')
fi

# Strip the extracted AC/Requirements section from description to avoid duplication in worklog
if [ -n "$ACCEPTANCE_CRITERIA" ] && [ -z "$TASK_AC" ]; then
    TASK_DESCRIPTION=$(printf '%s' "$TASK_DESCRIPTION" | \
      awk '/^#+ *(Acceptance Criteria|Requirements)/{skip=1; next} /^#/{skip=0} !skip')
fi

# Render Go text/template to worklog.md using POSIX awk.
# Handles {{.Field}} substitution and {{if .Field}}...{{end}} conditionals.
# Field values are passed via -v to avoid shell-injection from bead content.
# Note: awk -v interprets C escape sequences (\n, \t, \\) in values.
awk \
    -v EpicID="$EPIC_ID" \
    -v EpicTitle="$EPIC_TITLE" \
    -v EpicGoal="$EPIC_GOAL" \
    -v FeatureID="$FEATURE_ID" \
    -v FeatureTitle="$FEATURE_TITLE" \
    -v FeatureGoal="$FEATURE_GOAL" \
    -v TaskID="$BEAD_ID" \
    -v TaskTitle="$TASK_TITLE" \
    -v TaskDescription="$TASK_DESCRIPTION" \
    -v AcceptanceCriteria="$ACCEPTANCE_CRITERIA" \
    -v Timestamp="$TIMESTAMP" \
'
BEGIN {
    f[".EpicID"]             = EpicID
    f[".EpicTitle"]          = EpicTitle
    f[".EpicGoal"]           = EpicGoal
    f[".FeatureID"]          = FeatureID
    f[".FeatureTitle"]       = FeatureTitle
    f[".FeatureGoal"]        = FeatureGoal
    f[".TaskID"]             = TaskID
    f[".TaskTitle"]          = TaskTitle
    f[".TaskDescription"]    = TaskDescription
    f[".AcceptanceCriteria"] = AcceptanceCriteria
    f[".Timestamp"]          = Timestamp
    skip = 0
}
# Reset skip on {{end}} first (handles {{end}}{{if .X}} on same line)
/\{\{end\}\}/ { skip = 0 }
# {{if .Field}} — start conditional block
/\{\{if \./ {
    s = $0; sub(/.*\{\{if /, "", s); sub(/\}\}.*/, "", s)
    if (s != "" && f[s] == "") { skip = 1 }
    next
}
# Pure {{end}} lines (skip already reset above)
/\{\{end\}\}/ { next }
skip { next }
{
    line = $0
    while (match(line, /\{\{\.[-_a-zA-Z0-9]+\}\}/)) {
        token = substr(line, RSTART, RLENGTH)
        key = substr(token, 3, length(token) - 4)
        val = (key in f) ? f[key] : ""
        line = substr(line, 1, RSTART - 1) val substr(line, RSTART + RLENGTH)
    }
    print line
}
' "$TEMPLATE" > "$WORKTREE_DIR/worklog.md"

# --- Clear cleanup trap: worktree setup succeeded ---
trap - ERR

# --- Context quality report ---
echo "Context:"
if [ -n "$EPIC_ID" ]; then
    echo "  Epic: $EPIC_ID — $EPIC_TITLE"
else
    echo "  Epic: (none)" >&2
fi
if [ -n "$FEATURE_ID" ]; then
    echo "  Feature: $FEATURE_ID — $FEATURE_TITLE"
else
    echo "  Feature: (none)" >&2
fi
if [ -n "$ACCEPTANCE_CRITERIA" ]; then
    echo "  Acceptance criteria: found"
else
    echo "  Acceptance criteria: (none)" >&2
fi

echo "Worktree created: $WORKTREE_DIR"
echo "Branch: $BRANCH_NAME"
echo "Worklog: $WORKTREE_DIR/worklog.md"
