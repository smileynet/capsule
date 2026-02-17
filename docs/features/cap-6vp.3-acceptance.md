# Feature Acceptance: cap-6vp.3

## Advanced retry strategies

**Status:** Accepted
**Date:** 2026-02-13
**Parent Epic:** cap-6vp (Robust Task Pipeline)

## What Was Requested

Pipeline retry loops now support configurable backoff (increasing timeouts per attempt) and provider escalation (switching to a more capable provider after N failures), all unified through a single ResolveRetryStrategy configuration path.

## Acceptance Criteria

| # | Criterion | Evidence | Verified |
|---|-----------|----------|----------|
| 1 | runPhasePair uses ResolveRetryStrategy instead of reading MaxRetries directly | `TestRunPhasePair_UsesResolveRetryStrategy`, `TestRunPhasePair_PhaseOverrideTakesPrecedence` | Yes |
| 2 | Given BackoffFactor > 1.0, each retry multiplies effective timeout by BackoffFactor^(attempt-1) | `TestRunPhasePair_BackoffMultipliesTimeout` (30s→60s with factor 2.0), `TestRunPhasePair_BackoffNoEffectWhenNoTimeout` | Yes |
| 3 | Given EscalateProvider and EscalateAfter, after N attempts the retry loop switches to the escalation provider | `TestRunPhasePair_EscalateProviderSwitchesAfterN`, `TestRunPhasePair_EscalateProviderNoEffectWhenEmpty`, `TestRunPhasePair_EscalateProviderUnknownReturnsError` | Yes |

## How to Verify

```bash
# Run all retry strategy tests
go test ./internal/orchestrator/ -run "TestRunPhasePair_(UsesResolve|PhaseOverride|Backoff|Escalate)" -v

# Run full test suite
make test-full

# Run linter
make lint
```

## Out of Scope

- Config-to-orchestrator wiring of retry defaults in main.go (WithRetryDefaults not yet called at CLI level)
- Per-phase retry strategy overrides beyond MaxRetries (only pipeline-level BackoffFactor and EscalateProvider)

## Known Limitations

- EscalateAfter=0 with a non-empty EscalateProvider causes immediate escalation from attempt 1 (no attempts with default provider). This is a valid but potentially surprising configuration.
