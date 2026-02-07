# Feature Acceptance: cap-8ax.4

## Execute and execute-review headless claude prompt pair

**Status:** Accepted
**Date:** 2026-02-07
**Parent Epic:** cap-8ax (Tracer Bullet: Scripts & Direct Claude CLI)

## What Was Requested

An execute/execute-review prompt pair so the pipeline produces implementation code that passes tests with a review feedback loop.

## Acceptance Criteria

| # | Criterion | Evidence | Verified |
|---|-----------|----------|----------|
| 1 | Execute creates implementation that passes existing tests | test-prompt-templates.sh execute [1/6..6/6], test-phase-pair-execute.sh [1/5] | Yes |
| 2 | Execute returns JSON signal with status (PASS/NEEDS_WORK/ERROR) and files_changed | test-phase-pair-execute.sh [1/5, E5] | Yes |
| 3 | Execute-review verifies test passage and code quality | test-phase-pair-execute.sh [2/5] | Yes |
| 4 | NEEDS_WORK feedback triggers retry with improved implementation | test-phase-pair-execute.sh [3/5] | Yes |

## How to Verify

```bash
bash tests/scripts/test-prompt-templates.sh
bash tests/scripts/test-phase-pair-execute.sh
```

## Out of Scope

- Automatic retry orchestration
- Test modification by execute prompt
- Linting enforcement

## Known Limitations

- Execute-review lacks dedicated unit tests in test-prompt-templates.sh (validated only through E2E)
- Mock-based E2E tests cannot verify real test execution
