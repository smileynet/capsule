# Context: Capsule v2

**Status:** scoped
**Created:** 2026-02-06
**Updated:** 2026-02-06

## Overview

Capsule v2 is a deterministic AI agent orchestrator that runs headless
Claude CLI calls through phase-pair pipelines with worktree isolation,
worklog tracking, and structured signal contracts.

## Epics

| # | Title | Scope | Features | Tasks |
|---|-------|-------|----------|-------|
| 1 | Tracer Bullet: Scripts & Direct Claude CLI | Fully scoped | 6 | 20 |
| 2 | Go CLI Tool (Kong) | Fully scoped | 5 | 13 |
| 3 | TUI (Bubble Tea) | Fully scoped | 3 | 6 |
| 4 | Robust Task Pipeline | Placeholder | 3 | — |
| 5 | Multi-CLI Support | Placeholder | 2 | — |

**Total scoped:** 14 features, 39 tasks across 3 epics

## Epic Summaries

### Epic 1: Tracer Bullet (Scripts & Direct Claude CLI)
Prove the headless claude workflow end-to-end using shell scripts, template
beads, and direct `claude -p` invocations. No Go code. Output: working
scripts, captured commands, proven prompt pairs.

### Epic 2: Go CLI Tool
Transform Epic 1's validated scripts into a Go CLI using Kong. Provider
interface wraps claude subprocess. Orchestrator sequences phase pairs.
Plain text status output. All behavior matches Epic 1's proven workflow.

### Epic 3: TUI (Bubble Tea)
Add Bubble Tea TUI for live phase status display. Dual-mode: TUI when TTY
detected, plain text when piped. Progressive enhancement from Epic 2's
plain text output.

### Epic 4: Robust Task Pipeline (Unscoped)
More granular pipeline steps. Configurable phase definitions, pluggable
quality gates, pipeline pause/resume.

### Epic 5: Multi-CLI Support (Unscoped)
Support multiple AI CLI tools beyond Claude Code. Provider interface
extensions for OpenCode, Kiro, and other headless CLI tools.

## Key Artifacts

- [Brainstorm](../brainstorm-capsule-v2.md) — problem analysis, decisions, risks
- [Architecture](architecture.md) — layered architecture diagram
- [Decisions Log](decisions.log) — chronological decision record
- [Menu Plan](../menu-plan.yaml) — full work breakdown with dependencies

## Testing Strategy

| Level | Scope | Type |
|-------|-------|------|
| Task | Unit of implementation | TDD / unit tests |
| Feature | User-observable behavior | BDD / Given-When-Then |
| Epic | Full pipeline | E2E / smoke tests |
