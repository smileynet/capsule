---
name: pipeline
description: >-
  Capsule pipeline architecture, signal contract, and phase orchestration.
  Auto-applied when working on internal/orchestrator/, internal/provider/,
  scripts/, prompts/, or pipeline-related code.
user-invocable: false
---

# Pipeline

Capsule is a deterministic TDD pipeline with 5 stages orchestrated by structured JSON signals. Every phase emits a signal; the orchestrator decides what happens next.

## Signal Contract

```json
{
  "status": "PASS | NEEDS_WORK | ERROR",
  "feedback": "Human-readable explanation",
  "files_changed": ["path/to/file.go"],
  "summary": "One-line description"
}
```

- **PASS**: Phase succeeded, advance to next stage
- **NEEDS_WORK**: Re-run paired phase (test/execute pairs only) with feedback appended
- **ERROR**: Abort pipeline, preserve worktree for inspection
- **Output convention**: Signal is the last JSON object in stdout

## Pipeline Stages

1. **Prep**: Load bead context, create worktree, instantiate worklog template
2. **Test-writer / Test-review** (retry pair): Write failing tests, review covers acceptance criteria
3. **Execute / Execute-review** (retry pair): Implement code to pass tests, review for quality
4. **Sign-off**: Quality gate — PASS proceeds, NEEDS_WORK re-runs execute pair
5. **Merge**: Commit implementation and test files as `<bead-id>: <title>`, merge `--no-ff` to main, archive worklog, remove worktree, close bead

## Key Constraints

1. **Deterministic by default**: Same input bead produces same pipeline behavior
2. **Fail fast, preserve state**: ERROR aborts but keeps worktree for inspection
3. **Selective merge**: Only implementation and test files reach main
4. **External prompts**: Prompt templates live in files, not compiled into the binary
5. **Structured signals**: Machine-readable JSON output from every phase
6. **Progressive layers**: Scripts (Epic 1) → CLI (Epic 2) → TUI (Epic 3)

## Reference

- `docs/signal-contract.md` - Signal details and status codes
- `docs/planning/context-capsule-v2/architecture.md` - Architecture and data flow
- `docs/commands.md` - Script specifications
- `docs/deep-dive.md` - Full pipeline walkthrough
