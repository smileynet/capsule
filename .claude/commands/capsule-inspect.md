---
allowed-tools: Bash, Read, Glob, Grep
argument-hint: "[bead-id]"
description: Inspect capsule state — dashboard or deep-dive
---

Inspect capsule pipeline state. Two modes based on whether a bead ID is given.

**Bead ID:** `$ARGUMENTS` (optional).

## Dashboard mode (no argument)

If `$ARGUMENTS` is empty, show an overview:

1. **Binary version**: `capsule --version`
2. **Active worktrees**: List directories under `.capsule/worktrees/`
3. **Capsule branches**: `git branch --list 'capsule-*'`
4. **Orphaned state**: Branches that exist without a matching worktree, or worktrees without a matching branch
5. **Archived logs**: List directories under `.capsule/logs/`
6. **Config files**: Check for `.capsule/config.yaml` and `~/.config/capsule/config.yaml`

Present as a concise status summary.

## Deep-dive mode (bead-id given)

If `$ARGUMENTS` is provided, investigate that specific bead:

### Active worktree (if `.capsule/worktrees/$ARGUMENTS/` exists)

1. **Worktree path** and **branch**: `git -C .capsule/worktrees/$ARGUMENTS branch --show-current`
2. **Recent commits**: `git -C .capsule/worktrees/$ARGUMENTS log --oneline -10`
3. **Worklog**: Read `.capsule/worktrees/$ARGUMENTS/worklog.md` in full
4. **Phase outputs**: If `.capsule/worktrees/$ARGUMENTS/.capsule/output/` exists, list files and read the most recent log to find the last signal. If not, note that phase output is captured in the worklog instead.
5. **Diagnosis**: Identify the last phase that ran, its status, and any NEEDS_WORK feedback

### Archived only (if no worktree but `.capsule/logs/$ARGUMENTS/` exists)

1. Read `.capsule/logs/$ARGUMENTS/worklog.md`
2. Read `.capsule/logs/$ARGUMENTS/summary.md` if it exists
3. Summarize the outcome

### Nothing found

Report that no state exists for this bead ID.

## Summary

End with a diagnosis and suggested next action:
- If stuck on NEEDS_WORK: quote the feedback, suggest fixing and re-running
- If completed: confirm it merged successfully
- If orphaned: suggest `/capsule-cleanup $ARGUMENTS`
