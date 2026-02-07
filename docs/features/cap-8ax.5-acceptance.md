# Feature Acceptance: cap-8ax.5

## Sign-off and merge-to-main headless claude prompt pair

**Status:** Accepted
**Date:** 2026-02-07
**Parent Epic:** cap-8ax (Tracer Bullet: Scripts & Direct Claude CLI)

## What Was Requested

A sign-off/merge prompt pair so that only relevant implementation files are merged to main and worklogs are archived to `.capsule/logs/`.

## Acceptance Criteria

| # | Criterion | Evidence | Verified |
|---|-----------|----------|----------|
| 1 | Sign-off validates quality and returns PASS or NEEDS_WORK | test-phase-pair-merge.sh [1/6], test-prompt-templates.sh sign-off section | Yes |
| 2 | Only implementation/test files appear on main after merge | test-phase-pair-merge.sh [3/6], test-merge-script.sh [2/11] | Yes |
| 3 | Worklog archived to `.capsule/logs/<bead-id>/` | test-merge-script.sh [3/11, 7/11], test-phase-pair-merge.sh [4/6] | Yes |
| 4 | Mission worktree removed after merge | test-merge-script.sh [4/11, 5/11], test-phase-pair-merge.sh [5/6] | Yes |
| 5 | Bead status closed after merge | test-merge-script.sh [6/11], test-phase-pair-merge.sh [6/6] | Yes |

## How to Verify

```bash
bash tests/scripts/test-merge-script.sh
bash tests/scripts/test-phase-pair-merge.sh
```

## Out of Scope

- Automatic conflict resolution during merge
- GUI or interactive merge workflow
- Merge of non-capsule branches

## Known Limitations

- Worklog.md appears on main via `--no-ff` merge (branch history), but the merge commit itself excludes it. Archival to `.capsule/logs/` is the canonical audit trail.
- The merge prompt instructs Claude to selectively stage only implementation files, providing agent-reviewed merge rather than mechanical filtering.
