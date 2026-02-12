# Feature Acceptance: cap-9qv.1

## Go project skeleton with Kong CLI and test harness

**Status:** Accepted
**Date:** 2026-02-10
**Parent Epic:** cap-9qv (Go CLI Tool)

## What Was Requested

A Go project with Kong CLI, Makefile, and test harness as the foundation for the capsule CLI tool.

## Acceptance Criteria

| # | Criterion | Evidence | Verified |
|---|-----------|----------|----------|
| 1 | Given the project, when `go build ./...` runs, then a capsule binary is produced | `TestSmoke_GoProjectSkeleton/go_build_produces_a_capsule_binary` | Yes |
| 2 | Given the binary, when `capsule version` runs, then version/commit/date is printed | `TestFeature_GoProjectSkeleton/version_flag_prints_version_commit_and_date`, `TestSmoke_GoProjectSkeleton/capsule_version_prints_version_commit_and_date` | Yes |
| 3 | Given the binary, when `capsule run` without args runs, then usage is printed and exit code is non-zero | `TestFeature_GoProjectSkeleton/no_args_shows_usage_and_errors`, `TestSmoke_GoProjectSkeleton/capsule_without_args_exits_non_zero_with_usage` | Yes |
| 4 | Given the Makefile, when `make test` runs, then all tests pass | `TestSmoke_GoProjectSkeleton/make_test-full_passes` | Yes |

## How to Verify

```bash
go test ./cmd/capsule/ -v -count=1        # Unit tests
go test -tags smoke ./cmd/capsule/ -v     # Smoke tests (builds binary)
```

## Out of Scope

- Pipeline orchestration (cap-9qv.5)
- Provider integration (cap-9qv.2)
- Configuration loading (cap-9qv.3)

## Known Limitations

- `RunCmd.Run()`, `AbortCmd.Run()`, `CleanCmd.Run()` return "not implemented" — wiring happens in cap-9qv.5.3
