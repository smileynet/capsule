# Feature Acceptance: cap-6vp.1

## Configurable phase definitions

**Status:** Accepted
**Date:** 2026-02-13
**Parent Epic:** cap-6vp (Robust Task Pipeline)

## What Was Requested

Wire the three scaffolded PhaseDefinition fields (Condition, Provider, Timeout) into the RunPipeline execution loop so they take effect at runtime, enabling per-phase customization without code changes.

## Acceptance Criteria

| # | Criterion | Evidence | Verified |
|---|-----------|----------|----------|
| 1 | Given a phase with Condition: "files_match:\<glob\>", RunPipeline evaluates the glob against worktree files and skips the phase if no files match | `TestEvaluateCondition_FilesMatch_Found`, `TestEvaluateCondition_FilesMatch_NotFound`, `TestRunPipeline_ConditionSkipsPhase`, `TestRunPipeline_ConditionRunsPhaseWhenMet`, `TestRunPipeline_ConditionErrorAborts` | Yes |
| 2 | Given a phase with Provider: "\<name\>", executePhase uses the named provider instead of the orchestrator's default | `TestRunPipeline_PhaseProviderOverride` | Yes |
| 3 | Given a phase with Timeout: \<duration\>, executePhase wraps context with context.WithTimeout using the specified duration | `TestExecutePhase_TimeoutSetsDeadline`, `TestExecutePhase_NoTimeoutNoDeadline`, `TestExecutePhase_TimeoutAppliesToGate` | Yes |

## How to Verify

```bash
# Run condition tests
go test ./internal/orchestrator/ -run "TestEvaluateCondition|TestRunPipeline_Condition" -v

# Run provider override test
go test ./internal/orchestrator/ -run "TestRunPipeline_PhaseProviderOverride" -v

# Run timeout tests
go test ./internal/orchestrator/ -run "TestExecutePhase_Timeout|TestExecutePhase_NoTimeout" -v

# Run full test suite
make test-full

# Run linter
make lint
```

## Out of Scope

- Recursive glob patterns (`**`) — `filepath.Glob` matches single-level only
- Multiple conditions per phase (only one Condition string supported)
- Runtime condition types beyond `files_match` (extensible but not yet needed)

## Known Limitations

- Condition evaluation happens before the worktree is fully set up if the phase is early in the pipeline. Worktree must exist for glob matching to work.
- Provider override requires the provider to be pre-registered in the registry; unknown names cause a PipelineError.
