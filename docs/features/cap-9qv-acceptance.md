# Epic Acceptance: cap-9qv

## Go CLI Tool

**Status:** Accepted
**Date:** 2026-02-10

## What This Delivers

Capsule is now a Go CLI tool that can run a full AI-driven development pipeline from a single command. Users run `capsule run <bead-id>` to execute a 6-phase pipeline (test-writer, test-review, execute, execute-review, sign-off, merge) with automatic retry on reviewer feedback, plain text status output, and structured exit codes.

## Features Accepted

| # | Feature | Acceptance Report |
|---|---------|-------------------|
| 1 | cap-9qv.1: Go project skeleton with Kong CLI and test harness | [report](cap-9qv.1-acceptance.md) |
| 2 | cap-9qv.2: Provider interface wrapping headless claude subprocess | [report](cap-9qv.2-acceptance.md) |
| 3 | cap-9qv.3: Configuration loading and external prompt management | [report](cap-9qv.3-acceptance.md) |
| 4 | cap-9qv.4: Worktree creation/cleanup and worklog lifecycle | [report](cap-9qv.4-acceptance.md) |
| 5 | cap-9qv.5: Orchestrator sequencing phase pairs with retry logic | [report](cap-9qv.5-acceptance.md) |

## End-to-End Verification

The following user journeys are validated at the binary level via smoke tests:

- **Build and version**: Binary builds with embedded version metadata, `capsule --version` prints it
- **No-args usage**: `capsule` with no command exits non-zero with usage guidance
- **Run with unknown provider**: `capsule run bead --provider nonexistent` exits with code 2 and error message
- **Run without bead-id**: `capsule run` exits non-zero with argument error
- **Abort nonexistent**: `capsule abort nonexistent` exits with code 2 and "no worktree found"
- **Clean nonexistent**: `capsule clean nonexistent` exits with code 2 and "no worktree found"

Integration-level validation (orchestrator_test.go):
- Full pipeline happy path: all 6 phases execute, worktree created, worklog created and archived
- Retry flow: reviewer NEEDS_WORK triggers worker retry with feedback
- Standalone reviewer retry: sign-off retries execute phase
- Error abort: pipeline stops on ERROR status
- Context cancellation: Ctrl+C propagates through PipelineError

```bash
# Run all verification
make lint && make test-full && make smoke
```

## Out of Scope

- TUI/Bubble Tea display (epic: cap-awd)
- Robust task pipeline with exponential backoff (epic: cap-6vp)
- Multi-CLI provider support (epic: cap-10s)

## Known Limitations

- Full E2E smoke test of `capsule run` with a live provider requires the `claude` CLI; smoke tests validate error paths and exit codes only
- The `run` command does not yet populate `PipelineInput.Title` or `PipelineInput.Description` from bead metadata (wiring exists but CLI doesn't fetch bead info)
