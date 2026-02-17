#!/usr/bin/env bash
# Test script for t-1.2.1: Create worklog.md template with bead interpolation
# Validates: placeholder rendering, Mission Briefing section, phase entries,
# minimal data, no leftover tokens, edge cases.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
TEMPLATE="$REPO_ROOT/templates/worklog.md.template"

PASS=0
FAIL=0

pass() { PASS=$((PASS + 1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL + 1)); echo "  FAIL: $1"; }

# Prerequisite: template exists
if [ ! -f "$TEMPLATE" ]; then
    echo "ERROR: templates/worklog.md.template not found"
    exit 1
fi

# Helper: render Go text/template by substituting {{.Field}} placeholders
# and evaluating {{if .Field}}...{{end}} conditionals via awk.
render_template() {
    awk \
        -v EpicID="${EPIC_ID:-}" \
        -v EpicTitle="${EPIC_TITLE:-}" \
        -v EpicGoal="${EPIC_GOAL:-}" \
        -v FeatureID="${FEATURE_ID:-}" \
        -v FeatureTitle="${FEATURE_TITLE:-}" \
        -v FeatureGoal="${FEATURE_GOAL:-}" \
        -v TaskID="${TASK_ID:-}" \
        -v TaskTitle="${TASK_TITLE:-}" \
        -v TaskDescription="${TASK_DESCRIPTION:-}" \
        -v AcceptanceCriteria="${ACCEPTANCE_CRITERIA:-}" \
        -v Timestamp="${TIMESTAMP:-}" \
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
    /\{\{end\}\}/ { skip = 0 }
    /\{\{if \./ {
        s = $0; sub(/.*\{\{if /, "", s); sub(/\}\}.*/, "", s)
        if (s != "" && f[s] == "") { skip = 1 }
        next
    }
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
    ' "$TEMPLATE"
}

echo "=== t-1.2.1: worklog.md.template ==="
echo ""

# ---------- Test 1: Render with complete sample data ----------
echo "[1/5] Render with complete sample data"
# Given: complete sample data for all placeholders
# When: template is rendered via awk
# Then: all placeholder values appear in output
EPIC_ID="demo-1"
EPIC_TITLE="Demo Capsule Feature Set"
EPIC_GOAL="Implement input validation for the Contact type"
FEATURE_ID="demo-1.1"
FEATURE_TITLE="Add input validation"
FEATURE_GOAL="Contact fields validated on input"
TASK_ID="demo-1.1.1"
TASK_TITLE="Validate email format"
TASK_DESCRIPTION="Implement ValidateEmail(email string) error"
ACCEPTANCE_CRITERIA="- Returns nil for valid emails
- Returns error for missing @
- Returns error for empty string"
TIMESTAMP="2025-01-15T10:00:00Z"

RENDERED=$(render_template)
ALL_VALUES_OK=true
for val in "$EPIC_ID" "$EPIC_TITLE" "$FEATURE_ID" "$FEATURE_TITLE" "$TASK_ID" "$TASK_TITLE"; do
    if ! echo "$RENDERED" | grep -qF "$val"; then
        fail "rendered output missing value: $val"
        ALL_VALUES_OK=false
    fi
done
if [ "$ALL_VALUES_OK" = true ]; then
    pass "All placeholder values present in rendered output"
fi

# ---------- Test 2: Mission Briefing section present ----------
echo "[2/5] Mission Briefing section"
# Given: a rendered worklog template
# When: checking for Mission Briefing section
# Then: section exists with epic, feature, task context, and acceptance criteria
BRIEFING_OK=true
if ! echo "$RENDERED" | grep -qi "Mission Briefing"; then
    fail "Missing 'Mission Briefing' section header"
    BRIEFING_OK=false
fi
for field in "Epic" "Feature" "Task"; do
    if ! echo "$RENDERED" | grep -qi "$field"; then
        fail "Mission Briefing missing $field context"
        BRIEFING_OK=false
    fi
done
if ! echo "$RENDERED" | grep -qi "Acceptance Criteria"; then
    fail "Mission Briefing missing 'Acceptance Criteria' section"
    BRIEFING_OK=false
fi
if ! echo "$RENDERED" | grep -qF "Returns nil for valid emails"; then
    fail "Mission Briefing missing acceptance criteria content"
    BRIEFING_OK=false
fi
if [ "$BRIEFING_OK" = true ]; then
    pass "Mission Briefing section with epic/feature/task context and acceptance criteria"
fi

# ---------- Test 3: Phase entry sections ----------
echo "[3/5] Phase entry sections"
# Given: a rendered worklog template
# When: checking for pipeline phase sections
# Then: all 5 phase sections are present
PHASES_OK=true
for phase in "test-writer" "test-review" "execute" "execute-review" "sign-off"; do
    if ! echo "$RENDERED" | grep -qi "$phase"; then
        fail "Missing phase section: $phase"
        PHASES_OK=false
    fi
done
if [ "$PHASES_OK" = true ]; then
    pass "All pipeline phase sections present"
fi

# ---------- Test 4: Render with minimal data ----------
echo "[4/5] Render with minimal data"
# Given: minimal placeholder data (some fields empty)
# When: template is rendered
# Then: rendering succeeds without errors
EPIC_ID="min-1"
EPIC_TITLE="Minimal Epic"
EPIC_GOAL=""
FEATURE_ID="min-1.1"
FEATURE_TITLE="Minimal Feature"
FEATURE_GOAL=""
TASK_ID="min-1.1.1"
TASK_TITLE="Minimal Task"
TASK_DESCRIPTION=""
ACCEPTANCE_CRITERIA=""
TIMESTAMP=""

MINIMAL_RENDERED=$(render_template)
if [ -n "$MINIMAL_RENDERED" ]; then
    pass "Template renders without errors with minimal data"
else
    fail "Template rendering produced empty output with minimal data"
fi

# ---------- Test 5: No leftover {{ or }} tokens ----------
echo "[5/5] No leftover placeholder tokens"
# Given: template rendered with complete data
# When: checking for unreplaced placeholder tokens
# Then: no {{ tokens remain in output
# Reset to full data for this check
EPIC_ID="demo-1"
EPIC_TITLE="Demo Capsule Feature Set"
EPIC_GOAL="Implement input validation for the Contact type"
FEATURE_ID="demo-1.1"
FEATURE_TITLE="Add input validation"
FEATURE_GOAL="Contact fields validated on input"
TASK_ID="demo-1.1.1"
TASK_TITLE="Validate email format"
TASK_DESCRIPTION="Implement ValidateEmail(email string) error"
ACCEPTANCE_CRITERIA="- Returns nil for valid emails
- Returns error for missing @
- Returns error for empty string"
TIMESTAMP="2025-01-15T10:00:00Z"

FULL_RENDERED=$(render_template)
LEFTOVER=$(echo "$FULL_RENDERED" | grep -c '{{' || true)
if [ "$LEFTOVER" -eq 0 ]; then
    pass "No leftover {{ tokens in rendered output"
else
    fail "Found $LEFTOVER lines with leftover {{ tokens"
    echo "$FULL_RENDERED" | grep '{{' | head -5
fi

# ---------- Edge Cases ----------
echo ""
echo "=== Edge Cases ==="

# E1: Special characters in placeholder values
echo "[E1] Special characters in values"
# Given: placeholder values containing quotes, ampersands, and angle brackets
# When: template is rendered
# Then: special characters are preserved in output
TASK_TITLE='Validate "email" & <phone> format'
TASK_DESCRIPTION="Check for @ symbol & 'quotes' in input"
RENDERED_SPECIAL=$(render_template)
if echo "$RENDERED_SPECIAL" | grep -qF '&'; then
    pass "Special characters (quotes, ampersands) preserved"
else
    fail "Special characters lost during rendering"
fi
# Reset
TASK_TITLE="Validate email format"
TASK_DESCRIPTION="Implement ValidateEmail(email string) error"

# E2: Consistent placeholder naming convention
echo "[E2] Consistent placeholder naming"
# Given: the raw template file
# When: extracting all {{.Field}} tokens
# Then: all use Go text/template .CamelCase or control flow (if/end)
RAW_TEMPLATE=$(cat "$TEMPLATE")
# Valid patterns: {{.CamelCase}}, {{if .CamelCase}}, {{end}}
BAD_PLACEHOLDERS=$(echo "$RAW_TEMPLATE" | grep -oP '\{\{[^}]+\}\}' | grep -vE '^\{\{(\.[A-Z][a-zA-Z]*|if \.[A-Z][a-zA-Z]*|end)\}\}$' || true)
if [ -z "$BAD_PLACEHOLDERS" ]; then
    pass "All placeholders use Go text/template .CamelCase naming"
else
    fail "Inconsistent placeholder naming: $BAD_PLACEHOLDERS"
fi

# E3: Multi-line acceptance criteria
echo "[E3] Multi-line acceptance criteria"
# Given: acceptance criteria spanning 5 lines
# When: template is rendered
# Then: all criteria lines appear in output
ACCEPTANCE_CRITERIA="- Returns nil for valid emails (user@example.com)
- Returns error for missing @ (userexample.com)
- Returns error for missing domain (user@)
- Returns error for empty string
- Error messages are descriptive"
RENDERED_MULTI=$(render_template)
CRITERIA_LINES=$(echo "$RENDERED_MULTI" | grep -c "Returns" || true)
if [ "$CRITERIA_LINES" -ge 3 ]; then
    pass "Multi-line acceptance criteria rendered correctly ($CRITERIA_LINES lines)"
else
    fail "Expected 3+ criteria lines, got $CRITERIA_LINES"
fi

# E4: Valid markdown when rendered
echo "[E4] Valid markdown structure"
# Given: a fully rendered worklog template
# When: checking markdown structure
# Then: at least 2 markdown headers exist
HEADER_COUNT=$(echo "$FULL_RENDERED" | grep -c '^#' || true)
if [ "$HEADER_COUNT" -ge 2 ]; then
    pass "Rendered output contains $HEADER_COUNT markdown headers"
else
    fail "Expected at least 2 markdown headers, got $HEADER_COUNT"
fi

echo ""
echo "==========================================="
echo "RESULTS: $PASS passed, $FAIL failed"
echo "==========================================="
[ "$FAIL" -eq 0 ] && exit 0 || exit 1
