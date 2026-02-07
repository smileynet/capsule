# Feature Acceptance: cap-8ax.2

## Worktree creation and worklog instantiation from bead template

**Status:** Accepted
**Date:** 2026-02-07
**Parent Epic:** cap-8ax (Tracer Bullet: Scripts & Direct Claude CLI)

## What Was Requested

A prep script that creates an isolated worktree and instantiates a worklog with full bead context.

## Acceptance Criteria

| # | Criterion | Evidence | Verified |
|---|-----------|----------|----------|
| 1 | prep.sh creates a git worktree at `.capsule/worktrees/<bead-id>/` | test-prep-script.sh [1/6] | Yes |
| 2 | Worklog contains Mission Briefing with epic, feature, and task context and acceptance criteria | test-worklog-template.sh [1/5, 2/5], test-prep-script.sh [3/6] | Yes |
| 3 | Worktree branch is named `capsule-<bead-id>` | test-prep-script.sh [2/6] | Yes |
| 4 | Worklog content is bead-specific, not generic | test-prep-script.sh [3/6, E3] | Yes |

## How to Verify

```bash
bash tests/scripts/test-prep-script.sh
bash tests/scripts/test-worklog-template.sh
```

## Out of Scope

- Worktree cleanup (handled by merge phase)
- Worklog format customization
- Interactive bead selection

## Known Limitations

- Worklog uses envsubst-based rendering, limiting placeholders to environment-safe strings
