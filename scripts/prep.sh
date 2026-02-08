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
for cmd in git bd jq envsubst; do
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

TASK_TITLE=$(echo "$BEAD_JSON" | jq -r '.[0].title // empty')
if [ -z "$TASK_TITLE" ]; then
    echo "ERROR: Bead '$BEAD_ID' not found" >&2
    exit 1
fi

# --- Check worktree doesn't already exist ---
WORKTREE_DIR="$PROJECT_DIR/.capsule/worktrees/$BEAD_ID"
if [ -d "$WORKTREE_DIR" ]; then
    echo "ERROR: Worktree already exists at $WORKTREE_DIR" >&2
    exit 1
fi

# --- Extract bead context ---
TASK_DESCRIPTION=$(echo "$BEAD_JSON" | jq -r '.[0].description // empty')
TASK_AC=$(echo "$BEAD_JSON" | jq -r '.[0].acceptance_criteria // empty')

# Walk up the parent chain to find feature and epic
PARENT_ID=$(echo "$BEAD_JSON" | jq -r '.[0].parent // empty')
if [ -z "$PARENT_ID" ]; then
    PARENT_ID=$(echo "$BEAD_JSON" | jq -r '[.[0].dependencies[]? | select(.dependency_type == "parent-child")][0].id // empty')
fi

FEATURE_ID=""
FEATURE_TITLE=""
FEATURE_GOAL=""
EPIC_ID=""
EPIC_TITLE=""
EPIC_GOAL=""

if [ -n "$PARENT_ID" ]; then
    PARENT_JSON=$(cd "$PROJECT_DIR" && bd show "$PARENT_ID" --json 2>/dev/null) || true
    if [ -n "$PARENT_JSON" ]; then
        PARENT_TYPE=$(echo "$PARENT_JSON" | jq -r '.[0].issue_type // empty')
        if [ "$PARENT_TYPE" = "feature" ]; then
            FEATURE_ID="$PARENT_ID"
            FEATURE_TITLE=$(echo "$PARENT_JSON" | jq -r '.[0].title // empty')
            FEATURE_GOAL=$(echo "$PARENT_JSON" | jq -r '.[0].description // empty')
            # Look for epic above feature
            GRANDPARENT_ID=$(echo "$PARENT_JSON" | jq -r '.[0].parent // empty')
            if [ -z "$GRANDPARENT_ID" ]; then
                GRANDPARENT_ID=$(echo "$PARENT_JSON" | jq -r '[.[0].dependencies[]? | select(.dependency_type == "parent-child")][0].id // empty')
            fi
            if [ -n "$GRANDPARENT_ID" ]; then
                GRANDPARENT_JSON=$(cd "$PROJECT_DIR" && bd show "$GRANDPARENT_ID" --json 2>/dev/null) || true
                if [ -n "$GRANDPARENT_JSON" ]; then
                    GRANDPARENT_TYPE=$(echo "$GRANDPARENT_JSON" | jq -r '.[0].issue_type // empty')
                    if [ "$GRANDPARENT_TYPE" = "epic" ]; then
                        EPIC_ID="$GRANDPARENT_ID"
                        EPIC_TITLE=$(echo "$GRANDPARENT_JSON" | jq -r '.[0].title // empty')
                        EPIC_GOAL=$(echo "$GRANDPARENT_JSON" | jq -r '.[0].description // empty')
                    fi
                fi
            fi
        elif [ "$PARENT_TYPE" = "epic" ]; then
            EPIC_ID="$PARENT_ID"
            EPIC_TITLE=$(echo "$PARENT_JSON" | jq -r '.[0].title // empty')
            EPIC_GOAL=$(echo "$PARENT_JSON" | jq -r '.[0].description // empty')
        fi
    fi
fi

# --- Create worktree ---
BRANCH_NAME="capsule-$BEAD_ID"
mkdir -p "$(dirname "$WORKTREE_DIR")"

(cd "$PROJECT_DIR" && git worktree add -b "$BRANCH_NAME" "$WORKTREE_DIR" HEAD) || {
    echo "ERROR: Failed to create git worktree" >&2
    exit 1
}

# --- Create .capsule/logs directory ---
mkdir -p "$PROJECT_DIR/.capsule/logs"

# --- Instantiate worklog from template ---
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Use acceptance_criteria field if present, otherwise extract from description
ACCEPTANCE_CRITERIA="$TASK_AC"
if [ -z "$ACCEPTANCE_CRITERIA" ]; then
    ACCEPTANCE_CRITERIA=$(echo "$TASK_DESCRIPTION" | \
      sed -n '/^#*[[:space:]]*[Aa]cceptance [Cc]riteria/,/^#/{/^#*[[:space:]]*[Aa]cceptance [Cc]riteria/d;/^#/d;p;}')
fi
if [ -z "$ACCEPTANCE_CRITERIA" ]; then
    ACCEPTANCE_CRITERIA=$(echo "$TASK_DESCRIPTION" | \
      sed -n '/^#*[[:space:]]*[Rr]equirements/,/^#/{/^#*[[:space:]]*[Rr]equirements/d;/^#/d;p;}')
fi

# Convert {{ }} to ${ } for envsubst
sed 's/{{/\${/g; s/}}/}/g' "$TEMPLATE" | \
    EPIC_ID="$EPIC_ID" \
    EPIC_TITLE="$EPIC_TITLE" \
    EPIC_GOAL="$EPIC_GOAL" \
    FEATURE_ID="$FEATURE_ID" \
    FEATURE_TITLE="$FEATURE_TITLE" \
    FEATURE_GOAL="$FEATURE_GOAL" \
    TASK_ID="$BEAD_ID" \
    TASK_TITLE="$TASK_TITLE" \
    TASK_DESCRIPTION="$TASK_DESCRIPTION" \
    ACCEPTANCE_CRITERIA="$ACCEPTANCE_CRITERIA" \
    TIMESTAMP="$TIMESTAMP" \
    envsubst '${EPIC_ID} ${EPIC_TITLE} ${EPIC_GOAL} ${FEATURE_ID} ${FEATURE_TITLE} ${FEATURE_GOAL} ${TASK_ID} ${TASK_TITLE} ${TASK_DESCRIPTION} ${ACCEPTANCE_CRITERIA} ${TIMESTAMP}' \
    > "$WORKTREE_DIR/worklog.md"

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
