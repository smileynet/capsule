#!/usr/bin/env bash
# resolve-parent-chain.sh — Shared parent chain resolution for pipeline scripts.
#
# Usage:
#   source "path/to/lib/resolve-parent-chain.sh"
#   resolve_parent_chain "$PROJECT_DIR" "$BEAD_JSON"
#
# Sets these global variables:
#   FEATURE_ID     — Feature ancestor ID (empty if none)
#   FEATURE_TITLE  — Feature title
#   FEATURE_GOAL   — Feature description
#   EPIC_ID        — Epic ancestor ID (empty if none)
#   EPIC_TITLE     — Epic title
#   EPIC_GOAL      — Epic description
#
# Requires: jq, bd

# Extract parent ID from bd show JSON, with fallback to dependencies array.
# See docs/bd-schema.md for the contract.
# Args: $1 = JSON from bd show
_extract_parent_id() {
    local json="$1"
    local parent_id
    parent_id=$(printf '%s\n' "$json" | jq -r '.[0].parent // empty' 2>/dev/null) || true
    if [ -z "$parent_id" ]; then
        parent_id=$(printf '%s\n' "$json" | jq -r \
          '[.[0].dependencies[]? | select(.dependency_type == "parent-child")][0].id // empty' \
          2>/dev/null) || true
    fi
    echo "$parent_id"
}

# Resolve parent chain from task up through feature to epic.
# Args: $1 = project directory, $2 = JSON from bd show
# Sets globals: FEATURE_ID, FEATURE_TITLE, FEATURE_GOAL, EPIC_ID, EPIC_TITLE, EPIC_GOAL
resolve_parent_chain() {
    local project_dir="$1"
    local bead_json="$2"

    # Initialize outputs
    FEATURE_ID=""
    FEATURE_TITLE=""
    FEATURE_GOAL=""
    EPIC_ID=""
    EPIC_TITLE=""
    EPIC_GOAL=""

    local parent_id
    parent_id=$(_extract_parent_id "$bead_json")

    if [ -z "$parent_id" ]; then
        return 0
    fi

    local parent_json
    parent_json=$(cd "$project_dir" && bd show "$parent_id" --json 2>/dev/null) || true
    if [ -z "$parent_json" ]; then
        return 0
    fi

    local parent_type
    parent_type=$(printf '%s\n' "$parent_json" | jq -r '.[0].issue_type // empty' 2>/dev/null) || true

    if [ "$parent_type" = "feature" ]; then
        FEATURE_ID="$parent_id"
        FEATURE_TITLE=$(printf '%s\n' "$parent_json" | jq -r '.[0].title // empty' 2>/dev/null) || true
        FEATURE_GOAL=$(printf '%s\n' "$parent_json" | jq -r '.[0].description // empty' 2>/dev/null | awk '/^## /{exit} {print}') || true

        # Look for epic above feature
        local grandparent_id
        grandparent_id=$(_extract_parent_id "$parent_json")
        if [ -n "$grandparent_id" ]; then
            local grandparent_json
            grandparent_json=$(cd "$project_dir" && bd show "$grandparent_id" --json 2>/dev/null) || true
            if [ -n "$grandparent_json" ]; then
                local grandparent_type
                grandparent_type=$(printf '%s\n' "$grandparent_json" | jq -r '.[0].issue_type // empty' 2>/dev/null) || true
                if [ "$grandparent_type" = "epic" ]; then
                    EPIC_ID="$grandparent_id"
                    EPIC_TITLE=$(printf '%s\n' "$grandparent_json" | jq -r '.[0].title // empty' 2>/dev/null) || true
                    EPIC_GOAL=$(printf '%s\n' "$grandparent_json" | jq -r '.[0].description // empty' 2>/dev/null | awk '/^## /{exit} {print}') || true
                fi
            fi
        fi
    elif [ "$parent_type" = "epic" ]; then
        EPIC_ID="$parent_id"
        EPIC_TITLE=$(printf '%s\n' "$parent_json" | jq -r '.[0].title // empty' 2>/dev/null) || true
        EPIC_GOAL=$(printf '%s\n' "$parent_json" | jq -r '.[0].description // empty' 2>/dev/null | awk '/^## /{exit} {print}') || true
    fi
}
