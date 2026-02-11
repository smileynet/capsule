---
allowed-tools: Bash, Read, Glob, Grep
argument-hint: "[bead-id]"
description: Inspect capsule state — dashboard or deep-dive
---

Inspect capsule pipeline state. Two modes based on whether a bead ID is given.

**Bead ID:** `$ARGUMENTS` (optional).

## Dashboard mode (no argument)

If `$ARGUMENTS` is empty, show an overview:

~~~
CAPSULE DASHBOARD
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Binary: capsule <version>
Config: <present/not found> at .capsule/config.yaml

───────────────────────────────────────────
ACTIVE PIPELINES
───────────────────────────────────────────

<For each worktree under .capsule/worktrees/, show bead context and last phase:>

  <id> — <title>
    Branch: capsule-<id> (<N> commits ahead of main)
    Phase:  <last phase from worklog> — <status>
    State:  <classification — see below>

<If none:>
  (no active pipelines)

───────────────────────────────────────────
ARCHIVED RUNS
───────────────────────────────────────────

<For each directory under .capsule/logs/:>
  <id> — merged to main <date>

<If none:>
  (no archived runs)

───────────────────────────────────────────
HEALTH
───────────────────────────────────────────

Orphaned branches: <count>
  (Branches matching capsule-* with no matching worktree.
   Clean up with /capsule-cleanup)

Orphaned worktrees: <count>
  (Worktrees without matching branches — unusual.
   Inspect with /capsule-inspect <id>)
~~~

To gather this data:
1. `capsule --version`
2. List directories under `.capsule/worktrees/`
3. `git branch --list 'capsule-*'`
4. List directories under `.capsule/logs/`
5. Check for `.capsule/config.yaml`

## Deep-dive mode (bead-id given)

If `$ARGUMENTS` is provided, investigate that specific bead.

Classify the state first, then narrate accordingly:

~~~
CAPSULE INSPECT: $ARGUMENTS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

<Read worklog and phase outputs. Classify state:>

───────────────────────────────────────────
DIAGNOSIS: <state classification>
───────────────────────────────────────────
~~~

### State classifications

**COMPLETED** — Pipeline finished, merged to main, bead closed.
~~~
DIAGNOSIS: COMPLETED

  Worklog: .capsule/logs/$ARGUMENTS/worklog.md
  No action needed.
~~~

**PASSED BUT UNMERGED** — All phases passed but merge stalled.
~~~
DIAGNOSIS: PASSED BUT UNMERGED

  Last phase: merge — PASS
  Branch: capsule-$ARGUMENTS (<N> commits, not in main)

  The pipeline completed successfully but the code was not
  merged to main. This can happen if the merge step encountered
  a conflict or the post-pipeline cleanup was interrupted.

  ACTION: git merge --no-ff capsule-$ARGUMENTS
          then: /capsule-cleanup $ARGUMENTS
~~~

**FAILED AT <PHASE>** — Pipeline stopped at a phase.
~~~
DIAGNOSIS: FAILED AT <phase>

  Last phase: <phase> — <NEEDS_WORK|ERROR>
  Attempt: <N>/<max>

  Feedback from last review:
    "<quoted feedback from signal>"

  The <reviewer> found issues with the <worker> output.
  After <N> retry attempts, the pipeline gave up.

  ACTION: Inspect the worktree, fix the issue manually, then
          re-run: /capsule-run $ARGUMENTS
~~~

**ORPHANED** — Branch exists but no worktree (or vice versa).
~~~
DIAGNOSIS: ORPHANED

  This usually happens when a run was interrupted.

  ACTION: /capsule-cleanup $ARGUMENTS
~~~

### How to classify

1. Check if `.capsule/logs/$ARGUMENTS/worklog.md` exists → COMPLETED candidate
2. Check if `.capsule/worktrees/$ARGUMENTS/` exists → active pipeline
3. Read the worklog from whichever location exists
4. Parse phase entries to find last phase and its status
5. Check `git branch --list "capsule-$ARGUMENTS"` for branch existence
6. Check `git merge-base --is-ancestor capsule-$ARGUMENTS main` for merge status
7. Check `bd show $ARGUMENTS --json` for bead status

After the state block, show the full worklog for reference.
