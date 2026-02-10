# Changelog

All notable changes to Capsule are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added
- Worktree management: create, remove, list, and check existence of isolated git worktrees per mission (`internal/worktree`)
- Worklog lifecycle: template instantiation, phase entry append, and archive to `.capsule/logs/` (`internal/worklog`)
- Input validation and sentinel errors (`ErrAlreadyExists`, `ErrNotFound`, `ErrInvalidID`) for both packages
- Run a full AI-driven development pipeline on any task with a single command (`run-pipeline.sh <bead-id>`)
- Automatically generate tests from task acceptance criteria, then implement code to pass them
- AI reviews tests and implementation at each stage, retrying with feedback on failure
- Final sign-off validates all tests pass, code is commit-ready, and acceptance criteria are met
- Only implementation and test files land on main; worklogs archived for audit trail
- Worktree isolation: each task runs in its own git worktree, cleaned up after merge
