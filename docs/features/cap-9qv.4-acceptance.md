# Feature Acceptance: cap-9qv.4

## Worktree creation/cleanup and worklog lifecycle

**Status:** Accepted
**Date:** 2026-02-10
**Parent Epic:** cap-9qv (Go CLI Tool)

## What Was Requested

As an orchestrator, worktree and worklog management so that each mission runs in isolation with full context tracking via Go packages.

## Acceptance Criteria

| # | Criterion | Evidence | Verified |
|---|-----------|----------|----------|
| 1 | Given a bead ID, when worktree.Create runs, then `.capsule/worktrees/<id>/` exists on branch `capsule-<id>` | `TestCreate/creates_worktree_and_branch` (worktree_test.go) | Yes |
| 2 | Given a bead context, when worklog.Create runs, then worklog.md in worktree has mission briefing | `TestCreate` (worklog_test.go) — verifies all placeholder substitutions | Yes |
| 3 | Given a completed mission, when worklog.Archive runs, then worklog is in `.capsule/logs/<id>/` | `TestArchive` (worklog_test.go) — verifies content at archiveDir/<beadID>/worklog.md | Yes |
| 4 | Given a worktree, when worktree.Remove runs, then worktree and branch are cleaned up | `TestRemove/removes_worktree_and_branch` (worktree_test.go) | Yes |

## How to Verify

```bash
go test ./internal/worktree/ ./internal/worklog/ -v -count=1
```

## Out of Scope

- CLI-level commands (worktree/worklog are internal packages consumed by the orchestrator)
- Conflict resolution during worktree operations
- Multi-worktree orchestration (single mission at a time)

## Known Limitations

- Archive is a copy, not a move — worktree removal is a separate step handled by the orchestrator
- No file locking or atomic writes — acceptable for single-user CLI tool
