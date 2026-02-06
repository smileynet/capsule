# Brainstorm: Capsule v2

**Date:** 2026-02-06
**Status:** Complete — decisions captured, scope defined, menu plan created

## Problem Statement

Legacy capsule uses tmux to orchestrate AI agent TDD workflows. While the
concept proved valuable (deterministic AI agents working through structured
phase pipelines), tmux is fundamentally unreliable for programmatic use:

- **Timing fragility**: send-keys → sleep → capture-pane has no reliable
  synchronization; output may be partial, prompts may not be ready
- **Shell state opacity**: no reliable way to detect command completion,
  error state, or shell readiness
- **Mock complexity**: testing requires elaborate mock hierarchies
  (mockTmuxRunner, mockTmuxClient) that don't mirror real behavior
- **Platform dependence**: tmux availability and behavior varies across
  environments

### What Worked in Legacy Capsule

- **Phase pipeline concept**: structured phases (test-write, test-review,
  execute, execute-review, sign-off) with reviewer feedback loops
- **Beads integration**: using issue tracker to define work, track progress,
  manage dependencies
- **Worklog pattern**: persistent log tracking each phase's actions and
  decisions throughout a mission
- **Worktree isolation**: git worktrees for parallel mission execution
- **Orchestrator pattern**: central coordinator sequencing phases with
  status callbacks

### What Didn't Work

- **Tmux as subprocess interface**: unreliable, untestable, platform-specific
- **Monolithic Go application upfront**: too much infrastructure before
  proving the workflow actually works end-to-end
- **Tight coupling**: provider interface tied to tmux internals, making it
  impossible to swap execution backends
- **Observability through capture**: polling tmux panes for output is
  fundamentally wrong — the subprocess should emit structured signals

## Core Insight

**Prove the workflow before building the tool.** The headless `claude -p`
CLI can be called directly from shell scripts. If we can run the full
pipeline (prep → test-write → test-review → execute → execute-review →
sign-off → merge) with scripts, we know exactly what the Go CLI needs
to implement.

## Approach: Tracer Bullet Strategy

Five progressive epics, each building on proven foundations:

| Epic | Title | Approach | Output |
|------|-------|----------|--------|
| 1 | Tracer Bullet | Shell scripts + direct claude CLI | Working scripts, captured commands, proven prompts |
| 2 | Go CLI | Transform scripts → Go with Kong | `capsule run <bead-id>` CLI tool |
| 3 | TUI | Layer Bubble Tea over CLI | Live phase status display |
| 4 | Robust Pipeline | More phases, pause/resume | Configurable pipeline |
| 5 | Multi-CLI | Support OpenCode, Kiro, etc. | Provider plugins |

## Key Technical Decisions

### D1: Headless Claude CLI as Execution Backend
**Decision:** Use `claude -p <prompt> --dangerously-skip-permissions` instead of tmux.
**Rationale:** Direct subprocess call with stdout capture. No timing issues,
no shell state management, no mock complexity. Structured JSON signal
in output for deterministic parsing.

### D2: Tracer Bullet First (Scripts Before Go)
**Decision:** Epic 1 is entirely shell scripts — no Go code.
**Rationale:** Prove the workflow works before writing application code.
Scripts become the specification for Epic 2. If a prompt pair doesn't
work, fix it in bash before codifying in Go.

### D3: Kong for CLI Framework
**Decision:** Use Kong (github.com/alecthomas/kong) for Go CLI.
**Rationale:** Struct-tag based, type-safe, minimal boilerplate.
Aligns with legacy capsule's existing patterns.

### D4: Bubble Tea for TUI
**Decision:** Use Bubble Tea (github.com/charmbracelet/bubbletea) for TUI.
**Scoring matrix:**

| Criterion | Bubble Tea | Tview | Termbox | Raw ANSI |
|-----------|-----------|-------|---------|----------|
| Testability | 5 | 2 | 2 | 1 |
| Progressive enhancement | 5 | 3 | 2 | 4 |
| Community/ecosystem | 5 | 4 | 2 | 1 |
| Learning curve | 3 | 4 | 3 | 2 |
| Elm architecture fit | 5 | 1 | 1 | 3 |

**Rationale:** Elm architecture (Model-Update-View) enables pure-function
testing with teatest. Lipgloss for styling. Progressive enhancement:
TUI when TTY, plain text when piped.

### D5: External Prompt Templates
**Decision:** Prompts live in `prompts/*.md` files, not embedded in Go.
**Rationale:** Iterate on prompts without recompiling. Template interpolation
for bead context. Reviewable as standalone documents.

### D6: Structured JSON Signal Contract
**Decision:** Every phase must emit a JSON signal as its final output:
```json
{"status": "PASS|NEEDS_WORK|ERROR",
 "feedback": "explanation",
 "files_changed": ["path/to/file.go"],
 "summary": "one-line description"}
```
**Rationale:** Deterministic parsing. Machine-readable phase results.
Enables automated retry logic.

### D7: Worktree Isolation with Worklog
**Decision:** Each mission runs in a git worktree with an instantiated
worklog.md that tracks all phase activity.
**Rationale:** Proven pattern from legacy capsule. Worktree provides
git isolation. Worklog provides context continuity across phases.
Worklog is NOT merged — archived to `.capsule/logs/<bead-id>/`.

### D8: Selective Merge (Implementation Only)
**Decision:** Only implementation and test files merge to main. Worklogs,
temp files, and capsule artifacts are excluded.
**Rationale:** Main branch should reflect only the work product.
Process artifacts preserved in `.capsule/logs/` for auditability.

### D9: Three Testing Layers
**Decision:** Enforce testing at every level:
- **Tasks**: TDD / unit tests
- **Features**: BDD / integration tests (Given-When-Then)
- **Epics**: E2E / smoke tests (full pipeline validation)
**Rationale:** Each layer catches different failure modes. TDD ensures
correctness. BDD ensures acceptance criteria. E2E ensures the workflow.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Claude CLI changes flags/behavior | All phases break | Pin CLI version, abstract behind provider interface |
| JSON signal not reliably in output | Can't parse results | Strict signal contract, fallback parsing (last JSON block) |
| Prompt quality insufficient | Phases produce poor results | Iterate in Epic 1 scripts, easy to modify .md files |
| Worktree conflicts on merge | Lost work | Sign-off phase validates commit-ready state before merge |
| Template project too simple | Doesn't exercise real scenarios | Design template with realistic feature gaps, multiple beads |

## Open Questions (Resolved During Scoping)

- ~~How many retries per phase pair?~~ → Configurable, default 2-3
- ~~Should worklog template be per-project or universal?~~ → Universal template with bead-specific interpolation
- ~~How to handle multi-file changes in signal?~~ → `files_changed` array in signal JSON
- ~~Teardown/cleanup strategy?~~ → Dedicated teardown.sh script + abort command in CLI

## References

- Legacy capsule: `internal/capsule/orchestrator.go`, `internal/provider/`
- Line-cook templates: `templates/` directory pattern for reusable fixtures
- Beads: git-backed issue tracking with dependencies and hierarchy
