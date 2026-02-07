# Feature Acceptance: cap-8ax.5

## Sign-off and merge-to-main headless claude prompt pair

**Status:** Accepted
**Date:** 2026-02-07
**Parent Epic:** cap-8ax (Tracer Bullet: Scripts & Direct Claude CLI)

## User Story

As a pipeline orchestrator, I want sign-off and merge prompts so that only relevant implementation files are merged to main and worklogs are archived to .capsule/logs/.

## Acceptance Criteria Verification

| # | Criterion | Verified | Evidence |
|---|-----------|----------|----------|
| 1 | Sign-off validates quality and returns PASS or NEEDS_WORK | Yes | test-phase-pair-merge.sh [1/6], test-prompt-templates.sh sign-off section |
| 2 | Only implementation/test files on main (no worklog.md) | Yes | test-phase-pair-merge.sh [3/6], test-merge-script.sh [2/11] |
| 3 | Worklog archived to .capsule/logs/\<bead-id\>/ | Yes | test-merge-script.sh [3/11, 7/11], test-phase-pair-merge.sh [4/6] |
| 4 | Mission worktree removed after merge | Yes | test-merge-script.sh [4/11, 5/11], test-phase-pair-merge.sh [5/6] |
| 5 | Bead status closed after merge | Yes | test-merge-script.sh [6/11], test-phase-pair-merge.sh [6/6] |

## Quality Checks

### Test Results

- **test-merge-script.sh**: 11/11 passed (unit + error handling)
- **test-phase-pair-merge.sh**: 14/14 passed (E2E chain)

### BDD Review (Maitre)

- All acceptance criteria have dedicated tests across multiple files
- Given-When-Then structure consistently used
- Error scenarios covered: missing worktree, sign-off not PASS, missing worklog, agent NEEDS_WORK
- E2E test exercises full user workflow with real git/bead operations

**Verdict:** APPROVED

## Deliverables

| Deliverable | Path |
|-------------|------|
| Sign-off prompt | prompts/sign-off.md |
| Merge prompt | prompts/merge.md |
| Merge driver script | scripts/merge.sh |
| BDD feature file | tests/features/f-1.5-signoff-merge.feature |
| Merge unit tests | tests/scripts/test-merge-script.sh |
| E2E pair test | tests/scripts/test-phase-pair-merge.sh |

## Tasks Completed

- [x] cap-8ax.5.1: Create sign-off prompt template
- [x] cap-8ax.5.2: Create merge prompt template and thin merge driver script
- [x] cap-8ax.5.3: Validate sign-off/merge pair end-to-end

## Notes

- Worklog.md appears on main via --no-ff merge (branch history), but the merge agent's commit itself excludes it. Archival to .capsule/logs/ is the canonical audit trail.
- The merge prompt instructs Claude to selectively stage only implementation files, providing agent-reviewed merge rather than mechanical filtering.
