# Context: Capsule v2

**Status:** finalized
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
| 4 | Robust Task Pipeline | Re-scoped | 3 | 9 |
| 5 | Multi-CLI Support | Placeholder | 2 | — |
| 6 | Task Dashboard TUI | Re-scoped | 3 | 12 |

**Total scoped:** 20 features, 60 tasks across 4 epics

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

### Epic 4: Robust Task Pipeline (Re-scoped)
Wire scaffolded PhaseDefinition fields (Condition, Provider, Timeout) into
runtime, intra-pipeline checkpointing for pause/resume, and advanced retry
strategies (backoff, provider escalation). Foundation built in Epic 2.

### Epic 5: Multi-CLI Support (Unscoped)

### Epic 6: Task Dashboard TUI (Re-scoped)
Rich two-pane interactive TUI for browsing ready beads, inspecting bead
details, dispatching pipelines with phase report browsing, and viewing
results in a continuous loop. New internal/dashboard/ package reimplements
phase rendering for cursor-driven pane layout. Tab switches focus between
left (list) and right (detail/report) panes. Zero changes to internal/tui/.

## Finalization Summary

All scoped work has been converted to beads with full hierarchy and dependencies.

| Type | Count | Bead ID Range |
|------|-------|---------------|
| Epics | 5 | cap-8ax, cap-9qv, cap-awd, cap-6vp, cap-10s |
| Features | 19 (14 scoped + 5 placeholder) | cap-{epic}.1 — cap-{epic}.N |
| Tasks | 39 | cap-{epic}.{feature}.{task} |
| Dependencies | 39 edges (task) + 4 edges (epic) | — |

**Epic → Bead mapping:**

| Epic | Bead ID | Features | Tasks |
|------|---------|----------|-------|
| 1 — Tracer Bullet | cap-8ax | 6 | 20 |
| 2 — Go CLI Tool | cap-9qv | 5 | 13 |
| 3 — TUI (Bubble Tea) | cap-awd | 3 | 6 |
| 4 — Robust Pipeline | cap-6vp | 3 (re-scoped) | 9 |
| 5 — Multi-CLI Support | cap-10s | 2 (placeholder) | — |
| 6 — Task Dashboard TUI | cap-kxw | 3 | 12 |

**Test specifications:**
- 14 BDD `.feature` files in `tests/features/` (one per scoped feature)
- 38 TDD spec `.md` files in `tests/specs/` (one per task with `tdd: true`)

## Key Artifacts

- [Brainstorm](../brainstorm-capsule-v2.md) — problem analysis, decisions, risks
- [Architecture](architecture.md) — layered architecture diagram
- [Decisions Log](decisions.log) — chronological decision record
- [Menu Plan](../menu-plan.yaml) — full work breakdown with dependencies
- [BDD Specs](../../tests/features/) — Gherkin feature files
- [TDD Specs](../../tests/specs/) — task-level test specifications

## Testing Strategy

| Level | Scope | Type |
|-------|-------|------|
| Task | Unit of implementation | TDD / unit tests |
| Feature | User-observable behavior | BDD / Given-When-Then |
| Epic | Full pipeline | E2E / smoke tests |
