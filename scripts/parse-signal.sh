#!/usr/bin/env bash
# parse-signal.sh — Extract and validate the last JSON signal block from stdin.
#
# Usage: echo "<phase output>" | parse-signal.sh
#
# Reads stdin, finds the last JSON object ({...}), validates it conforms to
# the signal contract (see docs/signal-contract.md), and prints it to stdout.
#
# Exit codes:
#   0 — Valid signal found and printed
#   1 — No valid signal found (synthetic ERROR signal printed)
set -euo pipefail

# --- Prerequisite checks ---
if ! command -v jq >/dev/null 2>&1; then
    echo "ERROR: jq is required but not installed" >&2
    exit 2
fi

# --- Read all input ---
INPUT=$(cat)

# --- Synthetic ERROR signal helper ---
error_signal() {
    local msg="$1"
    jq -n -c --arg fb "$msg" '{status:"ERROR",feedback:$fb,files_changed:[],summary:"Phase did not produce a signal"}'
    exit 1
}

# --- Extract the last valid JSON object from input ---
# Strategy: scan lines from bottom up, try each line (and accumulations)
# through jq to find the last valid JSON object.
LAST_JSON=""

# Try each line from the bottom as a potential single-line JSON object
while IFS= read -r line; do
    if [ -n "$line" ] && echo "$line" | jq empty 2>/dev/null; then
        LAST_JSON="$line"
        break
    fi
done <<< "$(echo "$INPUT" | tac)"

# --- No JSON found ---
if [ -z "$LAST_JSON" ]; then
    error_signal "No signal JSON found in phase output"
fi

# --- Validate required fields ---
for field in status feedback files_changed summary; do
    if [ "$(echo "$LAST_JSON" | jq "has(\"$field\")")" != "true" ]; then
        error_signal "Missing required field: $field"
    fi
done

# --- Validate status value ---
STATUS_VAL=$(echo "$LAST_JSON" | jq -r '.status')
case "$STATUS_VAL" in
    PASS|NEEDS_WORK|ERROR) ;;
    *)
        error_signal "Invalid status value: $STATUS_VAL (must be PASS, NEEDS_WORK, or ERROR)"
        ;;
esac

# --- Validate files_changed is array ---
IS_ARRAY=$(echo "$LAST_JSON" | jq '.files_changed | type == "array"')
if [ "$IS_ARRAY" != "true" ]; then
    error_signal "files_changed must be an array"
fi

# --- Output validated signal ---
echo "$LAST_JSON" | jq -c .
exit 0
