---
allowed-tools: Bash, Read, Glob, Grep
argument-hint: "<bead-id>"
description: Run a capsule TDD pipeline for a bead
---

Run the capsule pipeline for the given bead ID.

**Bead ID:** `$ARGUMENTS` (required — if empty, ask the user which bead to run).

## Pre-flight

1. Verify `capsule` binary is available: `capsule --version`
2. Verify `prompts/` directory exists with required files
3. Verify `templates/worklog.md.template` exists

If any check fails, report what's missing and suggest running `/capsule-setup`.

## Stale worktree check

Check if `.capsule/worktrees/$ARGUMENTS/` already exists.

If it does:
- Show the last 20 lines of the worklog if it exists: `.capsule/worktrees/$ARGUMENTS/worklog.md`
- Show the branch state: `git -C .capsule/worktrees/$ARGUMENTS log --oneline -5`
- **Ask the user** what to do:
  - **Abort** — run `capsule abort $ARGUMENTS` (preserves branch, removes worktree)
  - **Clean** — run `capsule clean $ARGUMENTS` (removes worktree and branch)
  - **Cancel** — stop, don't run anything

Do NOT proceed without user confirmation.

## Run pipeline

Execute: `capsule run $ARGUMENTS`

Use a timeout of 600 seconds (10 minutes) for this command.

## Interpret results

**Exit code 0 (success):**
- Show the archived worklog: `.capsule/logs/$ARGUMENTS/worklog.md`
- Show the archived summary if it exists: `.capsule/logs/$ARGUMENTS/summary.md`
- Report success

**Exit code 1 (pipeline error):**
- Show the error output
- Suggest: "Run `/capsule-inspect $ARGUMENTS` to diagnose the failure"

**Exit code 2 (setup error):**
- Show the error output
- This is a configuration or prerequisite problem, not a pipeline logic failure
