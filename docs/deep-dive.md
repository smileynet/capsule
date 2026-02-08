# Capsule Deep Dive: User Experience, Task Graph & Live Demo

## 1. User Experience Walkthrough

### What the user does

One command, one bead ID:

```bash
./scripts/run-pipeline.sh demo-001.1.1 --project-dir=/tmp/capsule-demo
```

Everything else is automated: worktree creation, test writing, implementation, review, sign-off, merge, cleanup, and bead closure.

### What the user sees

The output is a **5-stage progress report** with a narrative summary at the end.

**Happy path (first attempt success):**

```
=== Capsule Pipeline: demo-001.1.1 ===

[1/5] Prep
  Context:
    Epic: demo-001 — TodoWebApp MVP
    Feature: demo-001.1 — User can manage todos
    Acceptance criteria: found
  Worktree created: .capsule/worktrees/demo-001.1.1
  Branch: capsule-demo-001.1.1
  Worklog: .capsule/worktrees/demo-001.1.1/worklog.md

[2/5] Phase pair: test-writer → test-review
  [1/3] Running test-writer...
  [1/3] Running test-review...
  test-review: PASS

[3/5] Phase pair: execute → execute-review
  [1/3] Running execute...
  [1/3] Running execute-review...
  execute-review: PASS

[4/5] Sign-off
  [1/3] Running sign-off...
  sign-off: PASS

[5/5] Merge
  Running merge agent...
  Merge agent: PASS
  Merged capsule-demo-001.1.1 to main
  Archived worklog to .capsule/logs/demo-001.1.1/
  Removed worktree
  Deleted branch capsule-demo-001.1.1
  Closed bead demo-001.1.1

## Pipeline Summary: demo-001.1.1

### What Was Accomplished
Implemented add-todo functionality with 4 unit tests covering item creation,
input clearing, empty input rejection, and localStorage persistence.

### Challenges Encountered
None — pipeline completed on first attempt.

### End State
Bead closed, code merged to main. 4 tests passing.

### Feature & Epic Progress
Feature: demo-001.1 — 1 of 2 tasks closed
Epic: demo-001 — 0 of 1 features closed

=== Pipeline Complete ===
  Bead: demo-001.1.1
  Status: SUCCESS
```

**On retry (NEEDS_WORK from reviewer):**

```
[2/5] Phase pair: test-writer → test-review
  [1/3] Running test-writer...
  [1/3] Running test-review...
  test-review: NEEDS_WORK (attempt 1/3)
  [2/3] Running test-writer...          ← automatic retry with feedback
  [2/3] Running test-review...
  test-review: PASS
```

The reviewer's feedback is extracted from the signal JSON and appended to the writer's prompt as a "Previous Feedback" section on the next attempt.

**On failure (retries exhausted):**

```
Pipeline aborted at test-writer/test-review (exit 1)
Worktree preserved: .capsule/worktrees/demo-001.1.1

## Pipeline Summary: demo-001.1.1

### What Was Accomplished
Test-writer produced 3 of 4 required tests over 3 attempts.

### Challenges Encountered
Test-review returned NEEDS_WORK 3 times: missing test for "empty input rejected"
acceptance criterion. Each retry partially addressed prior feedback but
introduced new compilation issues.

### End State
Worktree preserved at .capsule/worktrees/demo-001.1.1 for manual inspection.

### Next Steps
- Fix test compilation error in src/todo.test.js line 23
- Add missing test for "empty input rejected" acceptance criterion
```

### UX design principles

1. **Numbered stages** (`[1/5]`, `[2/5]`) — position awareness at a glance
2. **Attempt counters** (`[1/3]`, `[2/3]`) — retry progress within each stage
3. **Terse status lines** (`test-review: PASS`) — scannable in seconds
4. **Narrative summary** — human-readable "what happened" with concrete file/test references
5. **Worktree preserved on failure** — user can inspect, fix, and potentially resume
6. **Bead comment posted** — summary persists in the issue tracker for future reference

---

## 2. System Task Graph

### Pipeline overview

```
                    +---------+
                    |  INPUT  |
                    | bead-id |
                    +----+----+
                         |
                    +----v----+
                    |  PREP   |
                    +----+----+
                         |
               +---------v----------+
               | TEST-WRITER/REVIEW |<--+
               |   (retry loop)     |   | NEEDS_WORK + feedback
               +---------+----------+---+
                         | PASS
               +---------v----------+
               | EXECUTE/REVIEW     |<--+
               |   (retry loop)     |   | NEEDS_WORK + feedback
               +---------+----------+---+
                         | PASS
               +---------v----------+
               |    SIGN-OFF        |<--+
               |   (retry loop)     |   | NEEDS_WORK → re-run execute
               +---------+----------+---+
                         | PASS
                    +----v----+
                    |  MERGE  |
                    +----+----+
                         |
                    +----v----+
                    | SUMMARY |
                    +---------+
```

### Stage 1: PREP

**Script:** `scripts/prep.sh`

| Aspect | Detail |
|--------|--------|
| **Input** | Bead ID, project directory |
| **Process** | 1. Validate bead exists via `bd show --json` 2. Extract title, description, acceptance criteria 3. Walk parent chain (task → feature → epic) via `lib/resolve-parent-chain.sh` 4. Create git worktree on branch `capsule-<bead-id>` from HEAD 5. Instantiate worklog from `templates/worklog.md.template` via `envsubst` |
| **Output** | Git worktree at `.capsule/worktrees/<bead-id>/`, `worklog.md` with mission briefing |
| **Expects** | Bead exists with a title; worktree does not already exist |
| **Requires** | `git`, `bd`, `jq`, `envsubst` |

**Parent chain resolution** (`lib/resolve-parent-chain.sh`):
1. Extract `.[0].parent` field (primary) or first `parent-child` dependency (fallback)
2. Query parent via `bd show`, check `issue_type`
3. If parent is `feature`: set FEATURE_* vars, then look one level up for epic
4. If parent is `epic`: set EPIC_* vars directly
5. Graceful degradation — returns empty values if any step fails

**Acceptance criteria extraction** (fallback chain):
1. `.[0].acceptance_criteria` field from bd JSON
2. Text between `## Acceptance Criteria` and next `#` heading in description
3. Text between `## Requirements` and next `#` heading in description

### Stage 2: TEST-WRITER → TEST-REVIEW (retry loop)

**Scripts:** `scripts/run-phase.sh` invoking `prompts/test-writer.md` and `prompts/test-review.md`

**Test-Writer (Phase 1):**

| Aspect | Detail |
|--------|--------|
| **Input** | Worktree with `worklog.md` + `AGENTS.md`; optional feedback from prior review |
| **Process** | Claude reads worklog (acceptance criteria) + AGENTS.md (conventions). Writes failing tests — one per AC minimum. Tests must compile but fail due to missing implementation (RED phase TDD). No implementation code allowed. |
| **Output** | Test files in worktree, worklog "Phase 1: test-writer" section updated |
| **Signal** | PASS: all ACs covered, tests compile and fail correctly. NEEDS_WORK: couldn't cover all ACs. ERROR: couldn't read worklog or project structure. |

**Test-Review (Phase 2):**

| Aspect | Detail |
|--------|--------|
| **Input** | Worktree with test files + updated worklog |
| **Process** | Claude reviews tests against ACs. Runs test command from AGENTS.md to verify tests fail for the right reason (missing impl, not compilation errors). Checks isolation, naming, edge cases. |
| **Output** | Worklog "Phase 2: test-review" section updated |
| **Signal** | PASS: tests sufficient and failing correctly. NEEDS_WORK: specific issues with file, test name, problem, fix. ERROR: no test files found or can't run tests. |

**Retry mechanism:**

```
attempt = 0; feedback = ""
while attempt < max_retries:
    attempt++
    run test-writer (with feedback on retry)
    if writer fails → return ERROR (abort immediately)
    run test-review
    if PASS → return success
    if ERROR → return ERROR
    if NEEDS_WORK → extract .feedback from signal JSON, loop
return FAILED (retries exhausted)
```

Feedback is appended to the test-writer prompt:
```markdown
---

## Previous Feedback

The previous review returned NEEDS_WORK with the following feedback. Address these issues:

<feedback text>
```

### Stage 3: EXECUTE → EXECUTE-REVIEW (retry loop)

**Scripts:** `scripts/run-phase.sh` invoking `prompts/execute.md` and `prompts/execute-review.md`

**Execute (Phase 3):**

| Aspect | Detail |
|--------|--------|
| **Input** | Worktree with failing tests + worklog history; optional feedback |
| **Process** | 1. Confirm RED state (run tests, verify they fail) 2. Write minimum implementation to pass all tests (GREEN) 3. Optional refactor (tests must still pass) 4. Verify GREEN state |
| **Output** | Implementation files, worklog "Phase 3: execute" updated with RED→GREEN confirmation |
| **Signal** | PASS: all tests pass. NEEDS_WORK: some tests still failing. ERROR: tests already pass or compilation errors in tests. |
| **Constraint** | Must NOT modify test files. |

**Execute-Review (Phase 4):**

| Aspect | Detail |
|--------|--------|
| **Input** | Worktree with implementation + tests + worklog |
| **Process** | Run tests (all must pass). Review correctness vs ACs. Check for scope creep, debug statements, test modifications. |
| **Output** | Worklog "Phase 4: execute-review" updated |
| **Signal** | PASS: correct, minimal, clean. NEEDS_WORK: specific issues. ERROR: can't run tests. |

Same retry mechanism as Stage 2.

### Stage 4: SIGN-OFF (special retry)

**Script:** `scripts/run-phase.sh` invoking `prompts/sign-off.md`

| Aspect | Detail |
|--------|--------|
| **Input** | Worktree with complete implementation + full worklog history |
| **Process** | 1. Verify all 4 prior phases completed successfully 2. Run tests (all pass) 3. Check commit-ready state (no tmp files, debug code, TODOs) 4. Verify each AC: test exists → test passes → implementation correct |
| **Output** | Worklog "Phase 5: sign-off" updated with "Verdict: PASS" or "NEEDS_WORK" |
| **Signal** | PASS: everything verified end-to-end. NEEDS_WORK: specific issues (including incomplete prior phases or failing tests). ERROR: cannot operate (worklog missing, cannot run tests). |

**Special retry behavior** — differs from phase pairs:

```
attempt = 0
while attempt < max_retries:
    attempt++
    run sign-off
    if PASS → return success
    if ERROR → return ERROR
    if NEEDS_WORK:
        extract feedback
        re-run EXECUTE with feedback    ← goes back to execute, not sign-off
        if execute fails → return ERROR
        loop (re-run sign-off)
return FAILED
```

> **Note:** During sign-off retry, execute runs standalone (without execute-review).
> Any non-PASS result from execute immediately aborts the pipeline — there is
> no retry loop around execute within the sign-off retry.

### Stage 5: MERGE

**Scripts:** `scripts/merge.sh` invoking `prompts/merge.md`

| Aspect | Detail |
|--------|--------|
| **Input** | Worktree with signed-off implementation |
| **Preconditions** | `grep 'Verdict: PASS' worklog.md` must succeed |
| **Process** | 1. Build commit message: `<bead-id>: <title>` 2. Export `CAPSULE_COMMIT_MSG` env var 3. Invoke merge agent which classifies files (MERGE vs EXCLUDE), stages with explicit `git add <path>` (never `git add -A`), commits 4. Switch to main, `git merge --no-ff` 5. Archive worklog + phase outputs to `.capsule/logs/<bead-id>/` 6. Remove worktree, delete branch, close bead |
| **Output** | Merge commit on main, archived logs, closed bead |
| **Excludes** | `worklog.md`, `.capsule/` artifacts, temp files, debug code |

**File classification rules (merge agent):**
- **MERGE**: Source code, test files, config files (go.mod, package.json, etc.)
- **EXCLUDE**: worklog.md, `.capsule/` directory, temp files, debug artifacts

### Stage 6: SUMMARY (non-blocking)

**Script:** `scripts/run-summary.sh` invoking `prompts/summary.md`

| Aspect | Detail |
|--------|--------|
| **Input** | Pipeline metrics (duration, retry counts, outcome), worklog, bead hierarchy |
| **Process** | 1. Locate worklog (archive first, then worktree) 2. Walk parent chain for hierarchy 3. Query progress via `bd list --parent=... --all --json` 4. Assemble context block, inject into prompt template (replaces `{{CONTEXT}}`) 5. Invoke Claude |
| **Output** | Markdown to stdout + `.capsule/logs/<bead-id>/summary.md` + bead comment via `bd comments add` |
| **Wrapped** | Always called with `|| true` — never affects pipeline exit code |

Summary context block includes: outcome, duration, retry counts per stage, last feedback, worklog contents, feature progress (N of M tasks closed), epic progress (N of M features closed).

### Signal Contract

Every phase (except summary) emits a JSON signal as its **last JSON object** in stdout:

```json
{
  "status": "PASS | NEEDS_WORK | ERROR",
  "feedback": "Human-readable explanation",
  "files_changed": ["relative/path/to/file"],
  "summary": "One-line description"
}
```

**Validation** (`scripts/parse-signal.sh`):
1. Read all stdin, scan lines bottom-up via `tac`
2. Try each line (individually) through `jq` to find last valid single-line JSON object
3. Validate 4 required fields: `status`, `feedback`, `files_changed`, `summary`
4. Validate `status` ∈ {PASS, NEEDS_WORK, ERROR}
5. Validate `files_changed` is an array
6. Output compact JSON; on failure emit synthetic ERROR signal

Phases may include additional fields beyond the four required ones (e.g., the merge phase includes `commit_hash` in its PASS signal).

**Exit code mapping** (`run-phase.sh`):
| Status | Exit | Meaning |
|--------|------|---------|
| PASS | 0 | Advance to next stage |
| NEEDS_WORK | 1 | Re-run paired writer with feedback |
| ERROR | 2 | Abort pipeline |

### Worklog lifecycle

```
Template                    →  Instantiated              →  Populated by agents
templates/worklog.md.template  worktree/worklog.md          Phase sections filled
                                                            "Status: pending" → details
                                                         →  Archived
                                                            .capsule/logs/<id>/worklog.md
                                                         →  Excluded from merge
                                                            (never committed to main)
```

### Directory structure

```
project/
├── .capsule/
│   ├── worktrees/
│   │   └── <bead-id>/              ← git worktree (branch: capsule-<bead-id>)
│   │       ├── worklog.md          ← instantiated from template
│   │       ├── <source files>      ← written by agents
│   │       └── .capsule/output/    ← phase logs
│   │           └── <phase>-<timestamp>-<pid>.log
│   └── logs/
│       └── <bead-id>/              ← post-merge archive
│           ├── worklog.md
│           ├── <phase logs>
│           └── summary.md
```

---

## 3. Live Demo

### Target: `demo-001.1.1` — "Implement add todo item functionality"

This bead is a task under feature `demo-001.1` ("User can manage todos"), under epic `demo-001` ("TodoWebApp MVP"). It specifies 4 unit tests:

- `test_add_todo_creates_item_with_correct_structure`
- `test_add_todo_clears_input_field`
- `test_add_todo_empty_input_rejected`
- `test_todos_persist_to_localStorage`

### Setup

```bash
# Create fresh demo project from template (includes beads, dependencies, AGENTS.md)
~/code/capsule/scripts/setup-template.sh --template=demo-greenfield /tmp/capsule-demo
cd /tmp/capsule-demo

# Verify
bd ready   # → demo-001.1.1 (only ready task)
```

### Run

```bash
~/code/capsule/scripts/run-pipeline.sh demo-001.1.1 \
  --project-dir=/tmp/capsule-demo \
  --max-retries=3
```

### Expected behavior per stage

1. **Prep** — Creates worktree at `.capsule/worktrees/demo-001.1.1/`. Worklog has full hierarchy: epic "TodoWebApp MVP" → feature "User can manage todos" → task "Implement add todo item functionality". Acceptance criteria extracted from the `## Requirements` section of the bead description.

2. **Test-writer** — Creates `src/todo.test.js` with 4 tests matching the bead's test specifications. Tests use the simple test framework convention from AGENTS.md (no Jest/Mocha). Tests should compile and fail because `src/todo.js` doesn't exist yet.

3. **Test-review** — Verifies all acceptance criteria have test coverage (the bead specifies 4 unit tests across 5 requirements). Runs `node src/todo.test.js` to confirm tests fail for the right reason (missing module, not syntax errors).

4. **Execute** — Creates `src/todo.js` (TodoApp class with `{id, text, completed, createdAt}` structure, `Date.now().toString(36)` IDs) and `index.html`. Runs tests — all 4 pass. Does not modify test files.

5. **Execute-review** — Confirms all tests pass. Checks code is minimal (no extra features beyond what tests require). No debug statements, no scope creep.

6. **Sign-off** — Final end-to-end verification. Each requirement has a test, each test passes, implementation is clean and commit-ready. Writes "Verdict: PASS" to worklog.

7. **Merge** — Agent stages `src/todo.js`, `src/todo.test.js`, `index.html` (excludes `worklog.md` and `.capsule/`). Commits as `demo-001.1.1: Implement add todo item functionality`. Merges to main with `--no-ff`. Archives worklog. Removes worktree. Deletes branch. Closes bead.

8. **Summary** — Reports what was accomplished with concrete references. Notes that feature `demo-001.1` has 1 of 2 tasks closed, epic `demo-001` has 0 of 1 features closed.

### Post-demo verification

```bash
cd /tmp/capsule-demo

# Code is on main
git log --oneline
# → abc1234 Merge demo-001.1.1: Implement add todo item functionality
# → def5678 demo-001.1.1: Implement add todo item functionality
# → 0000000 Initial demo setup

# Bead is closed
bd show demo-001.1.1    # status: closed

# Dependency unblocked
bd ready                # → demo-001.1.2 (now ready!)
bd blocked              # → (empty)

# Tests pass
node src/todo.test.js   # All 4 tests pass

# Archives exist
cat .capsule/logs/demo-001.1.1/summary.md    # Narrative
cat .capsule/logs/demo-001.1.1/worklog.md    # Full phase history
ls .capsule/logs/demo-001.1.1/               # Phase logs

# No pipeline artifacts on main
test -f worklog.md && echo "FAIL" || echo "OK: no worklog on main"
test -d .capsule/worktrees/demo-001.1.1 && echo "FAIL" || echo "OK: worktree removed"
```

---

## 4. Minimal AGENTS.md Template for Capsule Projects

### What capsule phases read from AGENTS.md

| Phase | Uses |
|-------|------|
| test-writer | Project conventions, file locations, naming patterns |
| test-review | Test command (to run tests), conventions |
| execute | Project conventions, file locations |
| execute-review | Test command, conventions |
| sign-off | Test command, conventions (commit-ready checks) |
| merge | Conventions (file classification) |

The test command is critical — 4 of 6 phases run it.

### Minimum viable template

````markdown
# [Project Name]

[One-line description]

## Stack

[Language, framework, version — enough for agents to understand the runtime]

## Structure

```
[dir/file pattern — where source and tests go]
```

## Conventions

- Test files: [naming pattern and location]
- [Key code patterns — naming, structure, dependencies]

## Test Command

```bash
[exact command to run all tests]
```
````

### Reference: demo-greenfield AGENTS.md (~30 lines)

````markdown
# TodoWebApp

Vanilla JS todo app for Line Cook demo.

## Stack

- ES6+ JavaScript, no frameworks
- localStorage for persistence
- No build tools

## Structure

```
index.html      - UI with form and list
src/todo.js     - TodoApp class
src/todo.test.js - Tests (run: node src/todo.test.js)
```

## Conventions

- Simple test framework (no Jest/Mocha)
- Each todo: `{id, text, completed, createdAt}`
- IDs generated with `Date.now().toString(36)`

## Test Command

```bash
node src/todo.test.js
```

## Build

No build step required. Open `index.html` in browser.
````

This provides everything capsule needs: stack (so agents know the language/runtime), structure (so agents know where to put files), conventions (so agents follow project patterns), and test command (so agents can verify their work).

### What NOT to include

- **Setup/install instructions** — agents don't set up the project, they work in it
- **Architecture deep dives** — save for README or ADRs
- **Git workflow** — capsule manages git, not the agents
- **Environment variables** — unless the test command needs them
- **Design rationale** — agents need "what", not "why"

### Scaling guidance

For larger projects, the AGENTS.md should grow proportionally but stay **convention-focused**:

```markdown
## Conventions

- Test files: `*_test.go` in same package
- Validation functions: `Validate<Field>(input string) error`
- HTTP handlers: `internal/api/<resource>.go`
- Database queries: `internal/db/queries/<resource>.sql.go` (sqlc generated)
- Error handling: wrap with `fmt.Errorf("<context>: %w", err)`
```

The key insight: agents need to know **where things go** and **what patterns to follow**, not how the system works end-to-end.

### Template directory specification

Each template directory under `templates/` follows this file manifest:

**Required files:**

| File | Purpose |
|------|---------|
| `AGENTS.md` | Project conventions for pipeline agents (stack, structure, conventions, test command) |
| `issues.jsonl` | Bead fixtures defining the epic/feature/task hierarchy |
| `test-fixtures.sh` | Validation script for the fixture data (JSONL parsing, bd import, hierarchy) |

**Optional files:**

| File | Purpose |
|------|---------|
| `README.md` | Human-readable project description |
| `src/` | Existing source code (brownfield templates only — greenfield templates have no source) |

**Not included (and why):**

- **`.claude/` directory** — Pipeline agents run headless (`claude -p`); auto-loading doesn't apply
- **`worklog.md`** — Generated at runtime by `prep.sh`, not stored in template

**Why AGENTS.md over CLAUDE.md:** AGENTS.md is an open standard for cross-tool compatibility. Pipeline agents receive context through explicit prompt instructions regardless of the conventions filename.
