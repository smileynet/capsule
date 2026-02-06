# Architecture: Capsule v2

## Layered Architecture

```
┌─────────────────────────────────────────────────────────┐
│                      USER LAYER                         │
│  TUI (Bubble Tea)  │  Plain Text  │  JSON Output (CI)  │
├─────────────────────────────────────────────────────────┤
│                    CLI LAYER (Kong)                      │
│  run  │  abort  │  clean  │  version                    │
├─────────────────────────────────────────────────────────┤
│                 ORCHESTRATOR LAYER                       │
│  Pipeline sequencing  │  Phase-pair loops  │  Retries   │
├─────────────────────────────────────────────────────────┤
│                   PROMPT LAYER                          │
│  Template loading  │  Context interpolation  │  Compose │
├───────────────────────┬─────────────────────────────────┤
│    PROVIDER LAYER     │      WORKTREE LAYER             │
│  Provider interface   │  Git worktree CRUD              │
│  ClaudeProvider       │  Branch management              │
│  Signal parsing       │  Selective merge                │
├───────────────────────┼─────────────────────────────────┤
│    WORKLOG LAYER      │      CONFIG LAYER               │
│  Template instantiate │  Layered YAML loading           │
│  Phase entry append   │  Env var overrides              │
│  Archive to logs      │  Per-project config             │
├───────────────────────┴─────────────────────────────────┤
│                  EXTERNAL SYSTEMS                        │
│  claude CLI  │  git  │  beads (bd)  │  filesystem       │
└─────────────────────────────────────────────────────────┘
```

## Data Flow: Pipeline Execution

```
User: capsule run <bead-id>
  │
  ├─ Config: load defaults < user < project < env
  ├─ Beads: bd show <bead-id> → extract context
  ├─ Worktree: git worktree add .capsule/worktrees/<id>
  ├─ Worklog: instantiate template → worklog.md in worktree
  │
  ├─ Phase Pair: test-writer → test-review
  │   ├─ Prompt: load template, interpolate context
  │   ├─ Provider: claude -p <prompt> in worktree
  │   ├─ Signal: parse JSON from stdout
  │   ├─ Worklog: append [TEST-WRITER] entry
  │   ├─ Review: same flow with test-review prompt
  │   └─ Retry: if NEEDS_WORK, re-run writer with feedback
  │
  ├─ Phase Pair: execute → execute-review
  │   └─ (same pattern as above)
  │
  ├─ Phase: sign-off
  │   └─ Quality gate: PASS required to proceed
  │
  ├─ Merge:
  │   ├─ Stage only implementation + test files
  │   ├─ Commit: "<bead-id>: <task-title>"
  │   ├─ Merge to main (--no-ff)
  │   ├─ Archive worklog → .capsule/logs/<bead-id>/
  │   ├─ Remove worktree
  │   └─ Close bead
  │
  └─ Status: display result summary
```

## Signal Contract

Every phase produces a structured JSON signal as its final output:

```json
{
  "status": "PASS | NEEDS_WORK | ERROR",
  "feedback": "Human-readable explanation of result",
  "files_changed": ["path/to/file1.go", "path/to/file2_test.go"],
  "summary": "One-line description of what was done"
}
```

- **PASS**: Phase succeeded, proceed to next
- **NEEDS_WORK**: Worker should retry with feedback
- **ERROR**: Unrecoverable, abort pipeline

## Directory Structure

```
project/
├── .capsule/
│   ├── config.yaml          # Project-specific config
│   ├── worktrees/           # Active mission worktrees
│   │   └── <bead-id>/       # One worktree per mission
│   ├── output/              # Phase output logs (transient)
│   └── logs/                # Archived worklogs (permanent)
│       └── <bead-id>/
│           ├── worklog.md
│           └── phase-outputs/
├── .beads/                  # Beads issue tracker
├── prompts/                 # Phase prompt templates
│   ├── test-writer.md
│   ├── test-review.md
│   ├── execute.md
│   ├── execute-review.md
│   └── sign-off.md
└── scripts/                 # Epic 1 scripts (become spec for Epic 2)
    ├── setup-template.sh
    ├── prep.sh
    ├── run-phase.sh
    ├── run-pipeline.sh
    ├── merge.sh
    └── teardown.sh
```

## Key Constraints

1. **Deterministic by default**: Same input bead → same pipeline behavior
2. **Fail fast, preserve state**: ERROR aborts but keeps worktree for inspection
3. **Selective merge**: Only implementation/test files reach main
4. **External prompts**: Prompt templates live in files, not compiled in
5. **Structured signals**: Machine-readable JSON output from every phase
6. **Progressive layers**: Scripts (Epic 1) → CLI (Epic 2) → TUI (Epic 3)
