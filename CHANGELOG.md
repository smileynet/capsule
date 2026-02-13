# Changelog

All notable changes to Capsule are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added
- Go CLI skeleton with Kong: `capsule version`, `capsule run`, `capsule abort`, `capsule clean` (`cmd/capsule`)
- Provider interface with ClaudeProvider subprocess execution, signal parsing, and timeout handling (`internal/provider`)
- Provider registry with factory pattern for name-based provider instantiation
- Configuration loading with layered priority: defaults < user < project < env vars (`internal/config`)
- Prompt loader and composer with template interpolation (`internal/prompt`)
- Worktree management: create, remove, list, and check existence of isolated git worktrees per mission (`internal/worktree`)
- Worklog lifecycle: template instantiation, phase entry append, and archive to `.capsule/logs/` (`internal/worklog`)
- Input validation and sentinel errors (`ErrAlreadyExists`, `ErrNotFound`, `ErrInvalidID`) for worktree and worklog packages
- Orchestrator sequences 6 pipeline phases with retry logic: test-writer → test-review → execute → execute-review → sign-off → merge (`internal/orchestrator`)
- Plain text status callback prints timestamped phase progress with retry indicators
- Structured exit codes: 0=success, 1=pipeline failure, 2=setup error
- Graceful Ctrl+C handling via context cancellation
- Run a full AI-driven development pipeline on any task with a single command (`run-pipeline.sh <bead-id>`)
- Automatically generate tests from task acceptance criteria, then implement code to pass them
- AI reviews tests and implementation at each stage, retrying with feedback on failure
- Final sign-off validates all tests pass, code is commit-ready, and acceptance criteria are met
- Only implementation and test files land on main; worklogs archived for audit trail
- Worktree isolation: each task runs in its own git worktree, cleaned up after merge
- Advanced retry strategies for pipeline phase pairs (cap-6vp.3)
  - Unified retry configuration via ResolveRetryStrategy (phase-level MaxRetries override pipeline defaults)
  - Configurable timeout backoff: BackoffFactor multiplies effective timeout per retry attempt
  - Provider escalation: switch to a more capable provider after N failed attempts via EscalateProvider/EscalateAfter
