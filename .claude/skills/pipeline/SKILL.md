---
name: pipeline
description: >-
  Capsule pipeline architecture, signal contract, and phase orchestration.
  Auto-applied when working on internal/orchestrator/, internal/provider/,
  scripts/, prompts/, or pipeline-related code.
user-invocable: false
---

# Pipeline

Capsule is a deterministic TDD pipeline with 6 phases orchestrated by structured JSON signals. Every phase emits a signal; the orchestrator decides what happens next. After all phases pass, the CLI performs post-pipeline lifecycle: merge to main, cleanup worktree, close bead.

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

1. **Setup**: Load bead context (`internal/bead`), create worktree, instantiate worklog template
2. **Test-writer / Test-review** (retry pair): Write failing tests, review covers acceptance criteria
3. **Execute / Execute-review** (retry pair): Implement code to pass tests, review for quality
4. **Sign-off**: Quality gate — PASS proceeds, NEEDS_WORK re-runs execute pair
5. **Merge**: Commit implementation and test files in worktree
6. **Post-pipeline** (CLI layer): Merge `--no-ff` to main, archive worklog, remove worktree, close bead

## StatusUpdate with Signal Data

`StatusUpdate` carries a `Signal *provider.Signal` field populated on phase completion. The `plainTextCallback` in `cmd/capsule/main.go` uses this to print enriched phase reports:

```
[20:10:20] [1/6] test-writer passed
         files: src/validate_email_test.go
         summary: Wrote 7 failing tests covering all acceptance criteria
```

- `Signal` is nil for `PhaseRunning` updates
- `Signal` is populated for `PhasePassed`, `PhaseFailed`, `PhaseError` updates
- `FilesChanged` and `Summary` are shown on pass; `Feedback` is shown on failure

## Post-Pipeline Lifecycle

After `RunPipeline` succeeds, `RunCmd.runPostPipeline()` performs:

1. `DetectMainBranch()` — find main/master
2. `MergeToMain(id, mainBranch, commitMsg)` — `git merge --no-ff`
3. `Remove(id, true)` + `Prune()` — cleanup worktree and branch
4. `bead.Close(id)` — close the bead via bd CLI

All post-pipeline steps are **best-effort**: pipeline success is the hard requirement. Merge conflict prints a warning with recovery instructions but returns exit 0.

## Bead Context Resolution

`internal/bead.Client.Resolve(id)` calls `bd show <id> --json` and walks the parent chain (task → feature → epic) to populate `worklog.BeadContext`. Graceful fallback: if `bd` is unavailable, returns context with just TaskID set.

## Key Constraints

1. **Deterministic by default**: Same input bead produces same pipeline behavior
2. **Fail fast, preserve state**: ERROR aborts but keeps worktree for inspection
3. **Selective merge**: Only implementation and test files reach main
4. **External prompts**: Prompt templates live in files, not compiled into the binary
5. **Structured signals**: Machine-readable JSON output from every phase
6. **Progressive layers**: Scripts (Epic 1) → CLI (Epic 2) → TUI (Epic 3)

## Reference

- `docs/signal-contract.md` - Signal details and status codes
- `docs/commands.md` - Script and CLI specifications
- `docs/go-conventions.md` §11 - Beads CLI integration patterns
