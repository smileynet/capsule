# Feature Acceptance: cap-8ax.3

## Test-writer and test-review headless claude prompt pair

**Status:** Accepted
**Date:** 2026-02-07
**Parent Epic:** cap-8ax (Tracer Bullet: Scripts & Direct Claude CLI)

## What Was Requested

A test-writer/test-review prompt pair so the pipeline can write failing tests with a structured feedback loop.

## Acceptance Criteria

| # | Criterion | Evidence | Verified |
|---|-----------|----------|----------|
| 1 | test-writer creates test files and returns structured JSON signal | test-phase-pair-test-writer.sh [1/5], test-prompt-templates.sh test-writer [1/6..6/6] | Yes |
| 2 | JSON signal contains status, feedback, and files_changed | test-parse-signal.sh [1/8, 2/8, 5/8..7/8] | Yes |
| 3 | test-review returns PASS or NEEDS_WORK with actionable feedback | test-phase-pair-test-writer.sh [2/5], test-prompt-templates.sh test-review [3/6, 4/6] | Yes |
| 4 | NEEDS_WORK feedback injected into retry and test-writer improves | test-phase-pair-test-writer.sh [3/5], test-run-phase.sh [5/9] | Yes |

## How to Verify

```bash
bash tests/scripts/test-parse-signal.sh
bash tests/scripts/test-run-phase.sh
bash tests/scripts/test-prompt-templates.sh
bash tests/scripts/test-phase-pair-test-writer.sh
```

## Out of Scope

- Automatic retry orchestration (managed by caller)
- Maximum retry enforcement
- Test execution by test-review

## Known Limitations

- Mock-based E2E tests cannot verify real RED state or worklog chronological ordering — validated only with real Claude runs
