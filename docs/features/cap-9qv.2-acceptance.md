# Feature Acceptance: cap-9qv.2

## Provider interface wrapping headless claude subprocess

**Status:** Accepted
**Date:** 2026-02-10
**Parent Epic:** cap-9qv (Go CLI Tool)

## What Was Requested

A Provider interface with Execute() so the orchestrator can call AI agents as subprocess calls with structured output parsing.

## Acceptance Criteria

| # | Criterion | Evidence | Verified |
|---|-----------|----------|----------|
| 1 | Given a Provider, when Execute(ctx, prompt, workDir) is called, then it returns Result{Output, ExitCode, Duration, Signal} | `TestClaudeProvider_Execute/successful_execution_with_signal`, `TestResultParseSignal` | Yes |
| 2 | Given ClaudeProvider, when Execute runs, then it invokes claude -p with correct flags in workDir | `TestClaudeProvider_Execute` (re-exec pattern validates subprocess contract) | Yes |
| 3 | Given a timeout context, when the subprocess exceeds it, then the process is killed and a TimeoutError is returned | `TestClaudeProvider_Execute/timeout_kills_process` | Yes |
| 4 | Given claude output with JSON signal, when Result is returned, then Signal field contains parsed PhaseResult | `TestParseSignal` (12 subtests covering valid/invalid/edge cases) | Yes |

## How to Verify

```bash
go test ./internal/provider/ -v -count=1
```

## Out of Scope

- Multiple provider backends (only ClaudeProvider implemented)
- Streaming output (batch subprocess only)
- Provider health checks or availability detection

## Known Limitations

- Tests use re-exec pattern (`TestHelperProcess`) rather than the actual `claude` binary — validates the contract without requiring external dependencies
- Given-When-Then comments not yet added to provider tests (filed as P4 bead)
