# Feature Acceptance: cap-9qv.5

## Orchestrator sequencing phase pairs with retry logic

**Status:** Accepted
**Date:** 2026-02-10
**Parent Epic:** cap-9qv (Go CLI Tool)

## What Was Requested

As a user running `capsule run <bead-id>`, the orchestrator now executes all phase pairs with retries and status output, showing deterministic progress through the pipeline.

## Acceptance Criteria

| # | Criterion | Evidence | Verified |
|---|-----------|----------|----------|
| 1 | Phase registry defines all 6 pipeline phases with Worker/Reviewer kinds | `TestDefaultPhases_Count`, `_Order`, `_Kinds`, `_RetryTargets` | Yes |
| 2 | StatusUpdate carries progress info (phase, status, progress, attempt, max retry) | `TestStatusUpdate_Fields`, `TestPhaseStatus_Values`, `TestStatusCallback_Invocation` | Yes |
| 3 | Orchestrator sequences all phases, executing them via provider | `TestRunPipeline_AllPhasesPass`, `TestRunPipeline_StatusCallbacks` | Yes |
| 4 | runPhasePair retries on NEEDS_WORK with feedback passing to worker | `TestRunPhasePair_RetryOnNeedsWork`, `_FeedbackPassedToWorker`, `TestRunPipeline_ReviewerRetryFlow`, `_StandaloneReviewerRetry` | Yes |
| 5 | Pipeline aborts on ERROR status | `TestRunPhasePair_WorkerError`, `_ReviewerError`, `TestRunPipeline_PhaseErrorAborts` | Yes |
| 6 | Max retries exceeded produces PipelineError | `TestRunPhasePair_MaxRetriesExceeded` | Yes |
| 7 | CLI wires config → provider → orchestrator → StatusCallback | `TestFeature_OrchestratorWiring` subtests, `TestFeature_AbortCommand`, `TestFeature_CleanCommand` | Yes |
| 8 | Plain text StatusCallback prints timestamped phase lines | `plainTextCallback formats timestamped lines`, `shows attempt on retry` | Yes |
| 9 | Exit codes: 0=success, 1=pipeline failure, 2=setup error | `exitCode returns 0/1/2` subtests + `TestSmoke_OrchestratorWiring` binary-level validation | Yes |
| 10 | Graceful Ctrl+C handling via context cancellation | `TestRunPipeline_ContextCancelled`, `exitCode returns 1 for context cancellation` | Yes |

## How to Verify

```bash
# Run all unit and integration tests
make test-full

# Run smoke tests (builds binary, validates exit codes end-to-end)
make smoke

# Run linter
make lint
```

## Out of Scope

- TUI/Bubble Tea display (deferred to cap-awd)
- Multi-provider support beyond claude (deferred to cap-10s)
- Robust retry with exponential backoff (deferred to cap-6vp)

## Known Limitations

- Full E2E smoke test of `capsule run` with a real provider requires the `claude` CLI to be installed; smoke tests validate error paths and exit codes only
