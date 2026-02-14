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
- Task Dashboard TUI: `capsule dashboard` command with two-pane layout for browsing ready beads (`internal/dashboard`)
  - Left pane: navigable bead list with ID, priority badge (P0-P4), title, and type
  - Right pane: resolved bead detail with hierarchy, description, and acceptance criteria
  - Tab focus switching, cursor navigation (arrow/vim keys), refresh ('r'), and quit ('q')
- Pipeline mode in dashboard: dispatch pipelines from bead list and watch phase progress in real time (`internal/dashboard`)
  - Left pane: phase list with status indicators (spinner, checkmark, cross), auto-follow, retry counters, and duration
  - Right pane: per-phase reports with summary, files changed, duration, and feedback (on failure)
  - Channel-based event bridging between pipeline goroutine and Bubble Tea model
  - Summary mode on completion: pass/fail result with phase count and timing, any-key return to browse
  - Graceful abort with q/Ctrl+C during pipeline execution
  - TTY detection with clear error message when run without a terminal
- Dashboard polish and edge case handling (cap-kxw.3)
  - Shared post-pipeline lifecycle: merge, cleanup, and close bead run in background after pipeline completion
  - Terminal resize: both panes re-layout proportionally on window size changes
  - Graceful abort: Ctrl+C during pipeline triggers cleanup and returns to browse; double-press force quits
  - Missing bd detection: clear error message when bd is not installed
  - Empty bead list: "No ready beads" message with refresh hint
  - Given-When-Then structural comments on all dashboard test files
  - Smoke test for dashboard pipeline mode
