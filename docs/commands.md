# Capsule Pipeline: Command Reference & Epic 2 Specification

This document is the specification for the Epic 2 Go rewrite. Every script, prompt invocation, signal contract, retry rule, and lifecycle is documented here.

---

## Directory Structure

```
capsule/
├── scripts/                    # Pipeline scripts (this spec)
│   ├── setup-template.sh       # Create test environment from template
│   ├── prep.sh                 # Create worktree and worklog
│   ├── run-phase.sh            # Invoke a claude phase
│   ├── parse-signal.sh         # Extract and validate signal JSON
│   ├── run-pipeline.sh         # Orchestrate full pipeline
│   ├── merge.sh                # Agent-reviewed merge to main
│   └── teardown.sh             # Clean up worktrees and output
├── prompts/                    # Phase prompt templates
│   ├── test-writer.md          # RED: write failing tests
│   ├── test-review.md          # Review test quality
│   ├── execute.md              # GREEN: implement code
│   ├── execute-review.md       # Review implementation
│   ├── sign-off.md             # Final verification
│   └── merge.md                # Stage and commit in worktree
├── templates/
│   ├── worklog.md.template     # Worklog template (envsubst)
│   └── demo-capsule/           # Template project
│       ├── src/                # Go source code
│       ├── CLAUDE.md           # Project conventions
│       ├── README.md           # Project description
│       ├── issues.jsonl        # Bead fixtures (1 epic, 1 feature, 2 tasks)
│       └── test-fixtures.sh    # Fixture validation tests
├── tests/
│   ├── features/               # BDD feature files (Gherkin)
│   ├── specs/                  # TDD test specifications
│   └── scripts/                # Shell-based test scripts
├── docs/
│   ├── commands.md             # This file
│   └── signal-contract.md      # Signal contract reference
└── .capsule/                   # Runtime state (gitignored)
    ├── worktrees/<bead-id>/    # Active git worktrees
    │   └── .capsule/output/   # Phase output logs (inside each worktree)
    └── logs/<bead-id>/         # Archived worklogs and phase output
```

---

## Scripts

### setup-template.sh

Create a fresh test environment from the demo-capsule template.

**Usage:**

```
setup-template.sh [TARGET_DIR]
```

**Arguments:**

| Argument     | Required | Default    | Description                        |
|-------------|----------|------------|------------------------------------|
| `TARGET_DIR` | no       | `mktemp -d` | Directory to create the project in |

**Prerequisites:** `git`, `bd`

**Behavior:**

1. Validates prerequisites and template directory existence
2. Creates target directory (or temp dir if not specified)
3. Rejects directories that already contain a git repo
4. Initializes a git repo with test user config
5. Copies template source files (`src/`, `CLAUDE.md`, `README.md`)
6. Runs `bd init --prefix=demo` and `bd import` from fixtures
7. Commits everything as "Add template project and bead fixtures"
8. Prints the project directory path to stdout

**Exit codes:**

| Code | Meaning                                      |
|------|----------------------------------------------|
| 0    | Success, project directory printed to stdout |
| 1    | Failure (missing prereqs, existing repo, etc.) |

**Environment variables:** None.

**Stdout:** Absolute path to the created project directory.

**Stderr:** Error messages on failure.

---

### prep.sh

Create a git worktree and instantiate a worklog for a bead.

**Usage:**

```
prep.sh <bead-id> [--project-dir=DIR]
```

**Arguments:**

| Argument        | Required | Default | Description                   |
|----------------|----------|---------|-------------------------------|
| `bead-id`      | yes      | —       | The bead to prepare           |
| `--project-dir` | no       | `.`     | Project root directory        |

**Prerequisites:** `git`, `bd`, `jq`, `envsubst`

**Behavior:**

1. Validates bead exists via `bd show <bead-id> --json`
2. Rejects if worktree already exists at `.capsule/worktrees/<bead-id>/`
3. Extracts bead context: title, description, acceptance criteria
4. Walks parent chain to find feature and epic ancestors
5. Creates git worktree on branch `capsule-<bead-id>` from HEAD
6. Creates `.capsule/logs/` directory if not present
7. Instantiates worklog from `templates/worklog.md.template` via `envsubst`

**Template variables:**

The template file uses `{{VAR}}` delimiters (double curly braces). The shell implementation converts these to `${VAR}` via sed before piping to `envsubst`. For the Go rewrite, use native Go template syntax (`{{.Var}}`).

| Variable               | Source                          |
|------------------------|---------------------------------|
| `EPIC_ID`              | Grandparent epic ID (or empty)  |
| `EPIC_TITLE`           | Epic title                      |
| `EPIC_GOAL`            | Epic description                |
| `FEATURE_ID`           | Parent feature ID (or empty)    |
| `FEATURE_TITLE`        | Feature title                   |
| `FEATURE_GOAL`         | Feature description             |
| `TASK_ID`              | Bead ID                         |
| `TASK_TITLE`           | Bead title                      |
| `TASK_DESCRIPTION`     | Bead description                |
| `ACCEPTANCE_CRITERIA`  | Bead acceptance criteria or extracted from description |
| `TIMESTAMP`            | UTC ISO 8601 timestamp          |

**Creates:**

- `.capsule/worktrees/<bead-id>/` — git worktree on branch `capsule-<bead-id>`
- `.capsule/worktrees/<bead-id>/worklog.md` — instantiated worklog
- `.capsule/logs/` — log archive directory

**Exit codes:**

| Code | Meaning                                       |
|------|-----------------------------------------------|
| 0    | Success                                       |
| 1    | Failure (missing bead, worktree exists, etc.) |

**Stdout:** Confirmation lines: worktree path, branch name, worklog path.

---

### parse-signal.sh

Extract and validate the last JSON signal block from stdin.

**Usage:**

```
echo "<phase output>" | parse-signal.sh
```

**Arguments:** None (reads from stdin).

**Prerequisites:** `jq`

**Behavior:**

1. Reads all input from stdin
2. Scans lines from bottom up to find the last valid JSON object
3. Validates required fields: `status`, `feedback`, `files_changed`, `summary`
4. Validates `status` is one of `PASS`, `NEEDS_WORK`, or `ERROR`
5. Validates `files_changed` is an array
6. Outputs the validated JSON (compact, single-line) to stdout

**On failure:** Prints a synthetic ERROR signal to stdout:

```json
{"status":"ERROR","feedback":"<reason>","files_changed":[],"summary":"Phase did not produce a signal"}
```

**Exit codes:**

| Code | Meaning                                   |
|------|-------------------------------------------|
| 0    | Valid signal found and printed             |
| 1    | No valid signal found (synthetic ERROR printed) |
| 2    | Missing prerequisites (jq not installed)  |

---

### run-phase.sh

Invoke a capsule pipeline phase via headless Claude.

**Usage:**

```
run-phase.sh <phase-name> <worktree-path> [--feedback=...]
```

**Arguments:**

| Argument          | Required | Default | Description                                    |
|-------------------|----------|---------|------------------------------------------------|
| `phase-name`      | yes      | —       | Prompt template name (e.g., `test-writer`)     |
| `worktree-path`   | yes      | —       | Path to the git worktree                       |
| `--feedback=...`  | no       | —       | Feedback from previous review (appended to prompt) |

**Prerequisites:** `claude`, `jq`

**Behavior:**

1. Loads prompt from `prompts/<phase-name>.md`
2. If `--feedback` is provided, appends a "Previous Feedback" section to the prompt
3. Creates output directory at `<worktree-path>/.capsule/output/`
4. Invokes `claude -p "$PROMPT" --dangerously-skip-permissions` in the worktree directory
5. Captures stdout to `<worktree-path>/.capsule/output/<phase-name>-<timestamp>-<pid>.log`
6. Captures stderr to a separate `.log.stderr` file
7. Pipes stdout through `parse-signal.sh`
8. Maps the signal status to an exit code

**Claude invocation:**

```bash
claude -p "$PROMPT" --dangerously-skip-permissions
```

- Working directory: the worktree path
- The `--dangerously-skip-permissions` flag grants full file/tool access
- Stdout is captured; stderr goes to a separate log file

**Feedback injection (retry mode):**

When `--feedback` is provided, this section is appended to the prompt:

```markdown
---

## Previous Feedback

The previous review returned NEEDS_WORK with the following feedback. Address these issues:

<feedback content>
```

**Exit codes:**

| Code | Meaning                                 |
|------|-----------------------------------------|
| 0    | PASS — phase completed successfully     |
| 1    | NEEDS_WORK — phase found issues         |
| 2    | ERROR — phase failed or could not run   |

**Output files:**

| File                                            | Contents                |
|-------------------------------------------------|-------------------------|
| `.capsule/output/<phase>-<timestamp>-<pid>.log` | Claude stdout           |
| `.capsule/output/<phase>-<timestamp>-<pid>.log.stderr` | Claude stderr  |

---

### run-pipeline.sh

Orchestrate the full capsule pipeline for a bead.

**Usage:**

```
run-pipeline.sh <bead-id> [--project-dir=DIR] [--max-retries=N]
```

**Arguments:**

| Argument          | Required | Default | Description                          |
|-------------------|----------|---------|--------------------------------------|
| `bead-id`         | yes      | —       | The bead to run the pipeline for     |
| `--project-dir`   | no       | `.`     | Project root directory               |
| `--max-retries`   | no       | `3`     | Maximum retries per phase pair       |

**Prerequisites:** `jq`, `prep.sh`, `run-phase.sh`, `merge.sh`

**Pipeline stages:**

```
[1/5] Prep            → prep.sh <bead-id>
[2/5] test-writer     → test-review     (phase pair, max retries)
[3/5] execute         → execute-review  (phase pair, max retries)
[4/5] Sign-off        → (retries re-run execute on NEEDS_WORK)
[5/5] Merge           → merge.sh <bead-id>
```

**Phase pair retry logic (`run_phase_pair`):**

1. Run the writer phase (with feedback on retry)
2. If writer returns any non-zero exit (including NEEDS_WORK), abort with ERROR — only the review phase triggers retries
3. Run the review phase
4. If review returns PASS (exit 0), advance to next stage
5. If review returns ERROR (exit 2), abort
6. If review returns NEEDS_WORK (exit 1), extract `feedback` from signal JSON
7. Re-run writer with `--feedback="<extracted feedback>"`
8. Repeat up to `max-retries` times
9. If retries exhausted, abort with exit 1

**Sign-off retry logic (`run_signoff`):**

1. Run `sign-off` phase
2. If PASS, advance to merge
3. If ERROR, abort
4. If NEEDS_WORK, extract feedback and re-run `execute` phase with that feedback
5. Then re-run sign-off
6. Repeat up to `max-retries` times

**Exit codes:**

| Code | Meaning                                              |
|------|------------------------------------------------------|
| 0    | Pipeline completed successfully                      |
| 1    | Pipeline failed (retries exhausted)                  |
| 2    | Pipeline errored (phase ERROR or prerequisite failure) |

**On failure:** Worktree is preserved for manual inspection.

---

### merge.sh

Agent-reviewed merge driver for the capsule pipeline.

**Usage:**

```
merge.sh <bead-id> [--project-dir=DIR]
```

**Arguments:**

| Argument        | Required | Default | Description                          |
|----------------|----------|---------|--------------------------------------|
| `bead-id`      | yes      | —       | The bead whose worktree to merge     |
| `--project-dir` | no       | `.`     | Project root directory               |

**Prerequisites:** `git`, `bd`, `jq`, `claude`

**Behavior:**

1. Validates worktree exists at `.capsule/worktrees/<bead-id>/`
2. Validates `worklog.md` contains "Verdict: PASS" (sign-off must have passed)
3. Gets bead title for commit message: `<bead-id>: <title>`
4. Invokes merge agent via `run-phase.sh merge <worktree-path>`
   - Exports `CAPSULE_COMMIT_MSG` environment variable for the agent
5. Determines main branch (`main` or `master`)
6. Merges worktree branch to main with `--no-ff`
7. Archives worklog and phase outputs to `.capsule/logs/<bead-id>/`
8. Removes worktree via `git worktree remove` (fallback: `rm -rf` + prune)
9. Deletes the `capsule-<bead-id>` branch
10. Closes the bead via `bd close`

**Environment variables:**

| Variable             | Set by    | Description                         |
|---------------------|-----------|-------------------------------------|
| `CAPSULE_COMMIT_MSG` | merge.sh  | Commit message for the merge agent  |

**Exit codes:**

| Code | Meaning                                       |
|------|-----------------------------------------------|
| 0    | Merge completed successfully                  |
| 1    | Merge failed (agent NEEDS_WORK, no sign-off, conflict) |
| 2    | Error (missing dependencies, invalid arguments) |

---

### teardown.sh

Clean up capsule worktrees and output from a project.

**Usage:**

```
teardown.sh [--project-dir=DIR]
```

**Arguments:**

| Argument        | Required | Default | Description            |
|----------------|----------|---------|------------------------|
| `--project-dir` | no       | `.`     | Project root directory |

**Prerequisites:** `git` (used implicitly, no formal check)

**Behavior:**

1. Removes all worktrees under `.capsule/worktrees/*/`:
   - Attempts `git worktree remove --force`
   - Falls back to `rm -rf`
   - Deletes associated `capsule-<name>` branches
   - Prunes stale worktree metadata
   - (Phase output inside each worktree is removed with the worktree)
2. Cleans `.capsule/output/` at project root (if any files exist)
3. Preserves `.capsule/logs/` (archived worklogs are never deleted)

**Exit codes:**

| Code | Meaning                         |
|------|---------------------------------|
| 0    | Cleanup completed (or nothing to clean) |

---

## Claude Invocations (Prompt Phases)

All phases are invoked identically through `run-phase.sh`:

```bash
run-phase.sh <phase-name> <worktree-path> [--feedback=...]
```

Which executes:

```bash
cd <worktree-path> && claude -p "<prompt-contents>" --dangerously-skip-permissions
```

### Phase: test-writer

**Prompt file:** `prompts/test-writer.md`

**Purpose:** Write failing tests (TDD RED phase) for the task's acceptance criteria.

**Inputs read by agent:**
- `worklog.md` — mission briefing, acceptance criteria
- `CLAUDE.md` — project conventions, test patterns

**Agent rules:**
- At least one test per acceptance criterion
- Tests MUST fail (implementation missing, not syntax errors)
- Tests MUST compile
- No implementation code allowed
- Follow project conventions from `CLAUDE.md`
- Update existing test files on retry rather than creating new ones
- Appends phase entry to `worklog.md` under "Phase 1: test-writer"

**Signal on success:** `status: "PASS"`, `files_changed` lists created test files.

### Phase: test-review

**Prompt file:** `prompts/test-review.md`

**Purpose:** Review test quality before advancing to implementation.

**Inputs read by agent:**
- `worklog.md` — acceptance criteria + test-writer phase entry
- `CLAUDE.md` — project conventions

**Checks performed:**
- Coverage: every acceptance criterion has tests
- Failure mode: tests fail due to missing implementation, not syntax/compilation
- Isolation: each test tests one thing, clear names, no inter-dependencies
- Quality: follows project conventions, specific assertions, edge cases covered

**Agent runs the project's test command** to verify failure mode.

**Signal on success:** `status: "PASS"` — tests are sufficient. `status: "NEEDS_WORK"` — specific issues listed in `feedback`.

### Phase: execute

**Prompt file:** `prompts/execute.md`

**Purpose:** Implement minimal code to make all tests pass (TDD GREEN phase), then optionally refactor.

**Inputs read by agent:**
- `worklog.md` — all prior phase entries
- `CLAUDE.md` — project conventions

**Agent rules:**
- Confirms RED state (tests fail before implementation)
- Writes minimum code to pass tests
- Must NOT modify test files
- Runs tests after implementation to confirm GREEN state
- Optional refactoring (tests must still pass)
- Appends phase entry to `worklog.md` under "Phase 3: execute"

### Phase: execute-review

**Prompt file:** `prompts/execute-review.md`

**Purpose:** Review implementation quality and scope.

**Checks performed:**
- Correctness: implementation satisfies acceptance criteria
- No scope creep: only acceptance-criteria-scoped changes
- Code quality: clean, readable, follows conventions
- No test modifications: execute phase must not have changed test files

**Agent runs tests** to verify all pass.

### Phase: sign-off

**Prompt file:** `prompts/sign-off.md`

**Purpose:** Final verification that the task is complete and commit-ready.

**Checks performed:**
- All four prior phases completed successfully
- All tests pass
- No temporary files, debug code, or test artifacts in source tree
- Every acceptance criterion has: a test, the test passes, the implementation is correct
- Appends phase entry with "Verdict: PASS" or "Verdict: NEEDS_WORK" to `worklog.md`

**Sign-off PASS** is required before merge can proceed.

### Phase: merge

**Prompt file:** `prompts/merge.md`

**Purpose:** Review worktree files, stage implementation/test code, and create a commit.

**File classification:**
- **MERGE:** Source code, test files, project configuration
- **EXCLUDE:** `worklog.md`, `.capsule/`, temporary files, debug artifacts

**Agent rules:**
- Verifies sign-off PASS in worklog
- Quality check: no debug statements, TODOs, commented-out code
- Stages files with explicit `git add <path>` (never `git add -A`)
- Commits with message from `$CAPSULE_COMMIT_MSG` environment variable
- On PASS, signal includes an extra `commit_hash` field (not part of the core signal contract, but passed through by the parser)

---

## Signal Contract

Every phase must produce a JSON signal as the last JSON object in its stdout.

### Schema

```json
{
  "status": "PASS | NEEDS_WORK | ERROR",
  "feedback": "Human-readable explanation",
  "files_changed": ["path/to/file"],
  "summary": "One-line description"
}
```

### Fields

| Field           | Type     | Required | Description                                    |
|-----------------|----------|----------|------------------------------------------------|
| `status`        | string   | yes      | `PASS`, `NEEDS_WORK`, or `ERROR`               |
| `feedback`      | string   | yes      | What happened or what to fix                   |
| `files_changed` | string[] | yes      | Files created/modified (relative to worktree)  |
| `summary`       | string   | yes      | One-line description                           |

### Status semantics

| Status       | Orchestrator action                                      |
|-------------|----------------------------------------------------------|
| `PASS`      | Advance to the next phase                                |
| `NEEDS_WORK`| Re-run paired phase with `feedback` appended to prompt   |
| `ERROR`     | Stop the pipeline and report the error                   |

### Exit code mapping

`run-phase.sh` maps signal status to exit codes:

| Status       | Exit code |
|-------------|-----------|
| `PASS`      | 0         |
| `NEEDS_WORK`| 1         |
| `ERROR`     | 2         |

### Validation rules (parse-signal.sh)

1. Input must contain at least one valid JSON object
2. All four fields (`status`, `feedback`, `files_changed`, `summary`) must be present
3. `status` must be one of: `PASS`, `NEEDS_WORK`, `ERROR`
4. `files_changed` must be an array

On validation failure, a synthetic ERROR signal is returned.

---

## Retry Logic

### Phase pair retries

Phase pairs (test-writer/test-review and execute/execute-review) share a retry loop:

1. Run the writer phase
2. Run the review phase
3. If review returns NEEDS_WORK, extract `feedback` from the signal
4. Re-run writer with `--feedback="<feedback>"` appended to the prompt
5. Repeat up to `--max-retries` times (default: 3)
6. If retries exhausted, pipeline aborts

### Sign-off retries

Sign-off has a different retry pattern:

1. Run sign-off
2. If NEEDS_WORK, extract feedback
3. Re-run `execute` phase with the feedback (not sign-off itself)
4. Re-run sign-off
5. Repeat up to `--max-retries` times

### Abort conditions

- Any phase returns ERROR (exit 2)
- Writer phase fails (non-zero exit before review runs)
- Retries exhausted for any phase pair or sign-off

On abort, the worktree is preserved for manual inspection.

---

## Worklog Lifecycle

### Creation

1. `prep.sh` reads the bead's metadata via `bd show <bead-id> --json`
2. Walks the parent chain to find feature and epic ancestors
3. Instantiates `templates/worklog.md.template` via `envsubst` with bead context
4. Places the worklog at `.capsule/worktrees/<bead-id>/worklog.md`

### Population

Each phase appends its entry to the worklog under the corresponding section:

| Section                     | Written by       |
|---------------------------- |-----------------|
| Phase 1: test-writer        | test-writer agent |
| Phase 2: test-review        | test-review agent |
| Phase 3: execute            | execute agent    |
| Phase 4: execute-review     | execute-review agent |
| Phase 5: sign-off           | sign-off agent   |

Each entry includes status, verdict (for review phases), and phase-specific details.

### Archive

After a successful merge, `merge.sh` copies the worklog to `.capsule/logs/<bead-id>/worklog.md`. Phase output logs from `.capsule/output/` in the worktree are also archived.

### Exclusion from merge

The merge agent excludes `worklog.md` from the commit. It is a pipeline artifact, not part of the deliverable code.

---

## Worktree Lifecycle

### Creation (prep.sh)

```
.capsule/worktrees/<bead-id>/     ← git worktree (branch: capsule-<bead-id>)
.capsule/worktrees/<bead-id>/worklog.md
.capsule/worktrees/<bead-id>/.capsule/output/   ← created by run-phase.sh
```

The worktree is branched from HEAD of the project's current branch.

### During pipeline execution

Phases run inside the worktree directory. Each phase:
- Reads `worklog.md` and `CLAUDE.md`
- Creates/modifies source and test files
- Appends to `worklog.md`
- Output logs written to `.capsule/output/<phase>-<timestamp>-<pid>.log`

### Archive and cleanup (merge.sh)

1. Worklog and phase outputs copied to `.capsule/logs/<bead-id>/`
2. Worktree removed via `git worktree remove --force`
3. Branch `capsule-<bead-id>` deleted
4. Bead closed via `bd close`

### Manual cleanup (teardown.sh)

Removes all worktrees and output files. Preserves archived logs.

---

## Template Project

The `templates/demo-capsule/` directory provides a deterministic starting state for pipeline testing.

### Contents

| File              | Purpose                                      |
|-------------------|----------------------------------------------|
| `src/go.mod`      | Go module definition (`example.com/demo-capsule`, Go 1.22) |
| `src/main.go`     | Entry point with `Contact` struct and feature gaps |
| `CLAUDE.md`       | Project conventions for pipeline agents      |
| `README.md`       | Project description                          |
| `issues.jsonl`    | Bead fixtures: 1 epic, 1 feature, 2 tasks   |
| `test-fixtures.sh`| Validation tests for the fixtures            |

### Bead fixtures (issues.jsonl)

| ID          | Type    | Title                       |
|-------------|---------|------------------------------|
| `demo-1`    | epic    | Demo Capsule Feature Set     |
| `demo-1.1`  | feature | Add input validation         |
| `demo-1.1.1`| task    | Validate email format        |
| `demo-1.1.2`| task    | Validate phone format        |

Hierarchy: `demo-1` → `demo-1.1` → `demo-1.1.1`, `demo-1.1.2`

### Feature gaps

The template deliberately omits `ValidateEmail` and `ValidatePhone` functions. These are the tasks that bead fixtures define, providing concrete work for the pipeline to execute.

---

## Error Handling and Recovery

### Script-level error handling

All scripts use `set -euo pipefail`:
- `-e`: Exit on any command failure
- `-u`: Treat unset variables as errors
- `-o pipefail`: Pipe failure propagates

### Prerequisite checks

Every script validates its dependencies before executing:

| Script           | Required commands              |
|------------------|-------------------------------|
| setup-template.sh| git, bd                       |
| prep.sh          | git, bd, jq, envsubst        |
| parse-signal.sh  | jq                           |
| run-phase.sh     | claude, jq                   |
| run-pipeline.sh  | jq (+ subscripts)            |
| merge.sh         | git, bd, jq, claude (via run-phase.sh) |
| teardown.sh      | git                          |

### Recovery procedures

| Failure                    | State preserved          | Recovery                                |
|----------------------------|--------------------------|-----------------------------------------|
| Phase pair retries exhausted | Worktree intact         | Inspect worktree, manually fix, re-run pipeline |
| Sign-off retries exhausted  | Worktree intact         | Inspect worklog, fix issues, re-run     |
| Merge conflict             | Merge aborted, worktree intact | Resolve conflict manually, re-run merge |
| Claude invocation failure   | Output logs preserved    | Check `.capsule/output/` logs, re-run phase |
| Prerequisite missing        | No state created         | Install missing tool, re-run            |

---

## Environment Variables

| Variable             | Used by    | Description                               | Default |
|---------------------|------------|-------------------------------------------|---------|
| `CAPSULE_COMMIT_MSG` | merge.sh → merge agent | Commit message for the merge commit | (required, set by merge.sh) |

---

## Epic 2 Interface Boundaries

This section identifies the key interfaces that the Go rewrite should implement.

### Provider interface

The `claude -p` invocation abstracted as a provider:

```
Input:  prompt string, working directory path
Output: stdout string, stderr string, exit code
```

### Signal parser

The `parse-signal.sh` logic as a function:

```
Input:  raw phase output (string)
Output: Signal struct {Status, Feedback, FilesChanged, Summary}, error
```

### Worktree manager

The `prep.sh` + worktree lifecycle as a manager:

```
Create(beadID, projectDir) → worktree path, error
Archive(beadID, projectDir) → error
Remove(beadID, projectDir) → error
```

### Worklog creator

The template instantiation logic:

```
Create(beadID, worktreePath, beadContext) → error
```

### Phase runner

The `run-phase.sh` logic:

```
Run(phaseName, worktreePath, feedback) → Signal, error
```

### Orchestrator

The `run-pipeline.sh` logic:

```
Run(beadID, projectDir, maxRetries) → error
```

Internal helpers:
- `runPhasePair(writer, reviewer, worktree, maxRetries) → error`
- `runSignoff(worktree, maxRetries) → error`

### Merge driver

The `merge.sh` logic:

```
Merge(beadID, projectDir) → error
```
