---
allowed-tools: Bash, Read, Glob, Grep
argument-hint: "[bead-id]"
description: Clean up capsule worktrees and branches
---

Clean up capsule pipeline artifacts. Two modes based on whether a bead ID is given.

**Bead ID:** `$ARGUMENTS` (optional).

## Single-bead mode (argument given)

If `$ARGUMENTS` is provided, clean up that specific bead:

1. **Check worktree**: Does `.capsule/worktrees/$ARGUMENTS/` exist?
2. **Check branch**: Does `capsule-$ARGUMENTS` branch exist? (`git branch --list "capsule-$ARGUMENTS"`)
3. **Show context**: If the worklog exists, show the last phase status from `.capsule/worktrees/$ARGUMENTS/worklog.md`

If nothing exists for this bead, report that and stop.

**Ask the user** which action to take:
- **Abort** — `capsule abort $ARGUMENTS` (remove worktree, preserve branch for inspection)
- **Clean** — `capsule clean $ARGUMENTS` (remove worktree and delete branch)
- **Cancel** — do nothing

Do NOT proceed without user confirmation.

After executing, verify clean state:
- Confirm worktree directory is gone
- Confirm branch is gone (if clean was chosen)
- Run `git worktree prune` to clean stale metadata

## Bulk mode (no argument)

If `$ARGUMENTS` is empty, clean up all capsule artifacts:

1. **Preview**: Run `scripts/teardown.sh --dry-run` to show what would be cleaned
2. **Orphaned branches**: List any `capsule-*` branches via `git branch --list 'capsule-*'`
3. Show the preview results to the user

**Ask the user** to confirm before proceeding. Do NOT auto-execute.

If confirmed:
1. Run `scripts/teardown.sh`
2. Delete any remaining orphaned `capsule-*` branches (teardown handles active worktree branches, but orphaned branches from prior aborted runs may remain)
3. Verify clean state: no worktrees, no orphaned branches
