---
allowed-tools: Bash, Read, Glob, Grep
argument-hint: "[bead-id]"
description: Clean up capsule worktrees and branches
---

Clean up capsule pipeline artifacts. Two modes based on whether a bead ID is given.

**Bead ID:** `$ARGUMENTS` (optional).

## Single-bead mode (argument given)

If `$ARGUMENTS` is provided, clean up that specific bead:

~~~
CAPSULE CLEANUP: $ARGUMENTS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Checking state...

  Worktree: <exists/not found>
  Branch:   capsule-$ARGUMENTS <exists/not found>
  Bead:     <status from bd show>
~~~

If nothing exists for this bead, report that and stop.

### Pre-cleanup safety check

Check if the branch has been merged to main:
  `git merge-base --is-ancestor capsule-$ARGUMENTS main`

If NOT merged and branch has commits beyond main:

~~~
───────────────────────────────────────────
⚠ UNMERGED WORK DETECTED
───────────────────────────────────────────

Branch capsule-$ARGUMENTS has <N> commits not in main:
  <commit 1 oneline>
  <commit 2 oneline>

Cleaning will permanently delete this implementation.
Consider merging first:
  git checkout main && git merge --no-ff capsule-$ARGUMENTS
~~~

### Action prompt

Ask the user with descriptions:
- **Abort** — Remove worktree, keep branch for manual inspection
  (you can still: git log capsule-$ARGUMENTS, git diff main..capsule-$ARGUMENTS)
- **Clean** — Remove worktree AND delete branch permanently
  (irreversible — the implementation will be gone)
- **Cancel** — Do nothing

Do NOT proceed without user confirmation.

### Post-cleanup verification

After executing the chosen action, verify and report:

~~~
───────────────────────────────────────────
RESULT
───────────────────────────────────────────

  Worktree: removed ✓
  Branch:   <deleted ✓ | preserved (abort mode)>
  Prune:    ✓

State is clean. <If bead still open:> Bead $ARGUMENTS is still open — close with `bd close $ARGUMENTS` if work is complete.
~~~

## Bulk mode (no argument)

If `$ARGUMENTS` is empty, clean up all capsule artifacts:

~~~
CAPSULE CLEANUP (bulk)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Scanning for capsule artifacts...

Active worktrees: <count>
<For each, show merge status:>
  <id>  <merged to main ✓ | ⚠ NOT merged — <N> unmerged commits>

Orphaned branches: <count>
  capsule-<id> (no matching worktree)

Preview: <count> worktrees + <count> branches to remove.
~~~

Ask confirmation. Warn prominently if any unmerged work exists.

If confirmed:
1. For each worktree: `capsule clean <id>` or `capsule abort <id>` + branch delete
2. Delete orphaned `capsule-*` branches: `git branch -D capsule-<id>`
3. Run `git worktree prune`
4. Verify clean state: no worktrees, no orphaned branches
