---
allowed-tools: Bash, Read, Glob, Grep
argument-hint: "<feature|epic>"
description: Run quality gates before closing a feature or epic
---

Run the quality gates required before closing a feature or epic in beads.

**Gate level:** `$ARGUMENTS` (required — must be `feature` or `epic`).

If `$ARGUMENTS` is empty or not one of `feature`/`epic`, ask the user which gate level to run.

## Gate definitions

**Feature gate:** `make lint` then `make test-full`

**Epic gate:** `make lint` then `make test-full` then `make smoke`

## Execution

Run each gate command sequentially. Stop on first failure.

For each gate, report:
- **PASS** if exit code is 0
- **FAIL** if exit code is non-zero — show the relevant error output

Use a 300-second timeout for each gate command.

## Results

**All gates pass:**
Report all green and confirm: "Quality gates passed. Clear to close with `bd close <id>`."

**Any gate fails:**
- Show which gate failed and the error output
- Do not run subsequent gates
- Suggest fixing the issues and re-running `/capsule-gate $ARGUMENTS`
