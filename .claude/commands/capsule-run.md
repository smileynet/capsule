---
allowed-tools: Bash, Read, Glob, Grep
argument-hint: "<bead-id>"
description: Run a capsule TDD pipeline for a bead
---

Run the capsule pipeline for the given bead ID.

**Bead ID:** `$ARGUMENTS` (required — if empty, ask the user which bead to run).

## Pre-flight checks

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

## Output format

### Before execution — show what we're about to do:

~~~
CAPSULE RUN: $ARGUMENTS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Loading bead context...

  Task:    <id> — <title>
  Feature: <feature-id> — <feature-title> (<closed>/<total> tasks done)
  Epic:    <epic-id> — <epic-title>

INTENDED CHANGE:
  <first paragraph of bead description>

  Acceptance criteria:
    - <criterion 1>
    - <criterion 2>

───────────────────────────────────────────
PIPELINE
───────────────────────────────────────────

Starting 6-phase TDD pipeline...
  Phases: test-writer → test-review → execute → execute-review → sign-off → merge
  Provider: claude | Timeout: 300s | Retries: 3/phase
~~~

To show the bead context, run `bd show $ARGUMENTS --json` and walk the parent chain
using `bd show <parent-id> --json` for feature and epic context.

### During execution — run the CLI and narrate phase progress:

Execute `capsule run $ARGUMENTS` with 600s timeout. The CLI prints
enriched phase reports (files, summary, feedback) after each phase.
Let it stream to the user.

After each phase completes, the command can add narrative context
by reading the worklog. The worklog at `.capsule/worktrees/$ARGUMENTS/worklog.md`
is updated after each phase with detailed entries. Between phases, read the
latest entry and summarize what happened:

~~~
[20:10:20] [1/6] test-writer passed
         files: src/validate_email_test.go, src/validate_email.go
         summary: Wrote 7 failing tests covering all acceptance criteria

  RED phase complete. 7 tests written, all failing as expected.
  Tests cover: valid email format, missing @, missing domain, empty string,
  descriptive error messages.

[20:10:56] [2/6] test-review passed
         summary: All criteria covered, correct failure mode

  Tests approved — adequate coverage and correct failure mode.
~~~

Note: The narrative lines (indented plain text) come from the command
reading the worklog between CLI status lines. The `files:` and `summary:`
lines come from the CLI's enriched StatusCallback.

### After execution — validate and narrate results:

**On success (exit 0):**

Validate the CLI performed all post-pipeline steps:
1. Check merge: `git log --oneline -1 main` — should contain the bead commit
2. Check worktree: `.capsule/worktrees/$ARGUMENTS/` should NOT exist
3. Check branch: `git branch --list "capsule-$ARGUMENTS"` should be empty
4. Check bead: `bd show $ARGUMENTS --json` — check status
5. Check worklog: `.capsule/logs/$ARGUMENTS/worklog.md` should exist

Read the archived worklog and extract key details:
- Which phases retried (if any)
- What files were created/modified
- The implementation approach (from execute phase entry)

~~~
───────────────────────────────────────────
RESULT
───────────────────────────────────────────

Pipeline: ✓ all 6 phases passed (first attempt)
Merge:    ✓ capsule-$ARGUMENTS → main
Cleanup:  ✓ worktree removed, branch deleted
Bead:     ✓ $ARGUMENTS closed

WHAT WAS BUILT:
  <summary from sign-off phase entry — what tests were written,
   what implementation approach, what files changed>

Files delivered:
  A src/validate_email.go
  A src/validate_email_test.go

Worklog: .capsule/logs/$ARGUMENTS/worklog.md

NEXT STEP: /capsule-run <next-ready-bead> or /capsule-gate feature
~~~

Report any deviations:
~~~
Pipeline: ✓ all 6 phases passed
Merge:    ⚠ not merged — merge conflict detected
Cleanup:  — skipped (merge incomplete)
Bead:     — still open

  The pipeline succeeded but the merge to main failed.
  The worktree branch has the implementation.

  To fix:
    git checkout main
    git merge --no-ff capsule-$ARGUMENTS
    # resolve conflicts
    capsule clean $ARGUMENTS
    bd close $ARGUMENTS
~~~

**On failure (exit 1):**
~~~
───────────────────────────────────────────
RESULT
───────────────────────────────────────────

Pipeline: ✗ failed at <phase> (attempt <N>/<max>)
Worktree: preserved at .capsule/worktrees/$ARGUMENTS/

WHAT HAPPENED:
  <error message from CLI>

  The <phase> phase returned <status> after <N> attempts.
  <Quote the feedback if NEEDS_WORK>

NEXT STEP: /capsule-inspect $ARGUMENTS (diagnose the failure)
~~~
