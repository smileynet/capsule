# Brainstorm: Campaign UX — Nested Campaigns, Worktree Lifecycle, Error Handling

**Date:** 2026-02-21
**Status:** Complete — directions identified, scope defined
**Input:** `docs/handoff-campaign-ux.md`

## Problem Statement

Running a campaign at the **epic level** (epic → features → tasks) is broken
in three distinct ways:

1. **Visual corruption**: When an epic dispatches a feature, the feature's
   `OnCampaignStart` fires `CampaignStartMsg`, which replaces the entire
   dashboard `campaignState`. The epic-level view vanishes. After the feature
   completes, task indices are wrong and the display is confused.

2. **Worktree accumulation**: Campaign tasks leave behind orphaned worktrees
   and unmerged branches. `runPostPipeline` only calls `beads.Close(beadID)` —
   it skips `MergeToMain`, `Remove`, and `Prune`. The next task branches from
   stale main, missing previous changes.

3. **Silent failures**: Six instances of `_ = r.store.Save(state)` in the
   campaign runner. `OnTaskFail` in the dashboard adapter discards the error
   parameter entirely. Failed tasks show a red X with no explanation.

### Who Experiences This

Developers using capsule to run multi-task campaigns (features or epics) via
the dashboard. Single-task runs work correctly. Feature-level runs work in the
dashboard. Only epic-level dispatch is visually broken.

### What Happens If We Don't Solve It

- Epic campaigns are unusable via the dashboard
- Worktrees accumulate, eventually causing git errors
- Users cannot diagnose why tasks failed without checking logs manually
- Multi-feature work requires manual sequential dispatch of each feature

## Technical Exploration

### Current Architecture

**Campaign runner** (`internal/campaign/campaign.go`):
- `Runner` struct holds `PipelineRunner`, `BeadClient`, `StateStore`, `Callback`
- `runRecursive(ctx, parentID, depth, visited)` is the core loop
- Features/epics recurse; tasks dispatch via `pipeline.RunPipeline`
- `maxCampaignDepth = 3` (epic → feature → task)
- `visited` map prevents cycles

**Callback interface** (`campaign.Callback`):
- `OnCampaignStart(parentID, tasks)` fires at every recursion depth
- Dashboard adapter sends `CampaignStartMsg` each time → replaces state
- Adapter resets `taskIndex = 0` on every `OnCampaignStart`

**Dashboard state** (`internal/dashboard/campaign.go`):
- `campaignState` is value-typed (Bubble Tea pattern)
- `CampaignStartMsg` in `model.go` calls `newCampaignState()` → full replacement
- No concept of nested/stacked campaigns or subcampaign overlay

**Worktree lifecycle** (`cmd/capsule/main.go`):
- `postPipeline()` does: DetectMainBranch → MergeToMain → Remove → Prune → Close
- Called for single-task runs and standalone background pipeline completions
- **Not called for campaign tasks** — `campaign.Runner.runPostPipeline` only does `beads.Close`
- `PostPipelineFunc` exists on dashboard model but explicitly skipped for campaigns

**Error handling**:
- 6× `_ = r.store.Save(state)` in campaign.go
- `dashboardCampaignCallback.OnTaskFail(beadID string, _ error)` — error discarded
- `CampaignTaskDoneMsg` has no error text field
- `fileDiscoveries` is the only place errors surface to stderr

### Relevant Code Paths

| File | Lines | Role |
|------|-------|------|
| `internal/campaign/campaign.go` | 157–265 | `runRecursive` — core loop, recursion, state saves |
| `internal/campaign/campaign.go` | 358–361 | `runPostPipeline` — only closes bead |
| `cmd/capsule/main.go` | 314–356 | `postPipeline()` — full merge/cleanup |
| `cmd/capsule/main.go` | 912–1043 | `dashboardCampaignCallback` — all event adapters |
| `internal/dashboard/campaign.go` | 14–30 | `campaignState` struct |
| `internal/dashboard/model.go` | 391–397 | `CampaignStartMsg` handling — full state replacement |
| `internal/dashboard/msg.go` | 116 | `PostPipelineFunc` type |

### Existing Patterns We Can Follow

**Background mode resilience**: When a user presses Esc during a campaign,
`sendToBackground()` preserves `m.backgroundMode`, `m.eventCh`, and
`m.campaign` state. Messages continue routing correctly via `listenForEvents`.
This pattern proves the channel-based message flow works for concurrent state
updates.

**Standalone pipeline phases**: `pipelineState` already displays nested phase
rows within a task. The visual pattern (indented rows with spinners, checkmarks,
timing) is exactly what we need to replicate at the task-within-feature level.

**Merge conflict handling**: `worktree.Manager.MergeToMain` already detects
conflicts, aborts the merge, and returns `ErrMergeConflict`. The `postPipeline`
function already prints resolution instructions. We just need to wire this into
campaign task completion.

## Agent Pair Catalog

All behavior in the pipeline is driven by writer/reviewer agent pairs. When
items are missed during a campaign, the correct response is to invoke the
specific agent pair that addresses the gap — not to patch around it.

### Existing Pairs

| Writer (Worker) | Reviewer | Loop exits on | Purpose |
|-----------------|----------|---------------|---------|
| `test-writer` | `test-review` | PASS | Coverage, failure mode, isolation (DEFAULT) |
| `test-writer` | `test-quality` | PASS | Deeper structural audit (THOROUGH only) |
| `execute` | `execute-review` | PASS | Correctness, scope, quality, no test mods |
| `execute` | `sign-off` | PASS | Final readiness; retries back to `execute` |
| `merge` | (none) | always Worker | Stage + commit; single attempt, no reviewer |

### Pair Selection Heuristic

When something goes wrong mid-campaign, identify which agent pair should fix it:

| Situation | Agent Pair | Rationale |
|-----------|-----------|-----------|
| Tests don't compile | `test-writer` → `test-review` | Coverage/isolation issue |
| Implementation fails tests | `execute` → `execute-review` | Code quality issue |
| Merge conflict after task | `execute` → `sign-off` | Sign-off validates commit-readiness; conflict means prior sign-off missed something |
| Commit includes wrong files | `merge` (re-run) | Merge phase classifies files |
| Feature validation fails | Validation pipeline (configurable) | Feature-level acceptance check |

### Merge Conflict Resolution via Agent Pair

Per user direction: conflicts should be resolved by the assigned agent first,
with human escalation only if the agent cannot resolve.

**Strategy:**
1. `PostTaskFunc` attempts `MergeToMain`. On `ErrMergeConflict`:
2. Invoke the `execute` → `sign-off` pair in the conflicted worktree with
   conflict-resolution context (files in conflict, both sides of the diff).
3. If the pair resolves (sign-off PASS) → retry merge.
4. If the pair fails after max retries → pause campaign, emit non-blocking
   notification to the status bar / toast with resolution instructions.
5. Campaign remains resumable from the paused task.

This follows the research finding that merge conflicts should be scored by
type: additive changes → agent resolves; business logic / API contracts →
escalate. The sign-off reviewer naturally applies this judgment since it
already checks commit-readiness.

## Best Practices Applied (from Research)

Key patterns from `docs/agent-orchestration-patterns.md`:

### Writer/Reviewer Pairs
- **Structured rubric, not free-form critique** — reviewers emit `PASS`/`NEEDS_WORK`
  via signal contract (already implemented correctly)
- **Pass feedback verbatim** into worker retry context (already implemented via
  `prompt.Context.Feedback`)
- **Escalate to stronger provider after N failures** (already implemented via
  `EscalateAfter`)
- **Antipattern**: reviewer that modifies artifacts (capsule reviewers correctly
  only evaluate)

### Campaign Error Handling (Saga Pattern)
- Tasks with side effects (worktrees, merged commits, filed discoveries) need
  **compensating actions** on failure
- Visibility: "1 merged, 1 failed, 3 pending" not just a simple count
- **Antipattern**: `_ = r.store.Save(state)` — replace with logged warnings
- **Antipattern**: error messages without the reviewer's final feedback string

### Nested Orchestration
- Each level owns its own state (keyed by `parentID`)
- Callbacks should carry depth context so the TUI can route without inference
- **Antipattern**: global state replacement (the documented `CampaignStartMsg` bug)
- **Antipattern**: shared mutable state across recursion levels

### Non-Blocking Notifications (Five-Layer Model)
1. **In-place animation** — spinner → checkmark in task row (every phase transition)
2. **Status bar** — persistent one-liner (task complete/fail, elapsed time)
3. **Toast** — corner overlay, auto-dismiss (campaign complete, circuit breaker)
4. **Right-pane detail** — full error + feedback on selection (user navigates)
5. **Event log** — JSONL file for `capsule status` (all lifecycle events)

Campaign should use layers 1–4. Never block user input for non-critical
notifications. Error detail behind a separate command is an antipattern.

## Three Problems, Three Approaches

### Problem 1: Nested Campaign Animation

**Approach A — Callback stack with subcampaign messages (Recommended)**

Add depth tracking to `dashboardCampaignCallback`. Depth 0 = root campaign
(epic), depth 1+ = subcampaign (feature). New message types:
- `SubCampaignStartMsg{ParentBeadID, Tasks}` — overlay on running feature row
- `SubCampaignDoneMsg{ParentBeadID}` — collapse nested rows

Dashboard `campaignState` gains:
- `subcampaignTasks []CampaignTaskInfo` — tasks of the currently running feature
- `subcampaignStatuses []CampaignTaskStatus` — their status
- A "nested pipeline" embedded in each subcampaign task row

The callback adapter maintains a stack: push on `OnCampaignStart`, pop on
`OnCampaignComplete`. Root level sends `CampaignStartMsg` as today. Nested
levels send `SubCampaignStartMsg` instead.

**Pros**: Minimal changes to `campaign.Runner` (none). Contained to callback
adapter + dashboard. Preserves epic-level view.

**Cons**: `campaignState` struct grows. View rendering more complex. Only
supports one level of nesting visually (depth 2 = feature tasks under epic),
which matches `maxCampaignDepth = 3`.

**Approach B — Tree-structured campaign state**

Replace flat `tasks []CampaignTaskInfo` with a tree: each task node can have
children. `CampaignStartMsg` at any depth inserts children under the running
node instead of replacing root.

**Pros**: Naturally handles arbitrary nesting. Cleaner data model.

**Cons**: Bigger rewrite. Bubble Tea value-typed state with trees is awkward.
Cursor navigation becomes complex. Overkill for max-depth-3 constraint.

**Approach C — Separate sub-model for nested campaigns**

When a feature starts, spawn a child `campaignState` model for its tasks.
The parent `campaignState` delegates Update/View to the child when active.

**Pros**: Clean separation. Each level manages its own state.

**Cons**: Composition of Bubble Tea models is verbose. Two concurrent spinners
need coordination. Message routing between parent and child models.

**Decision: Approach A** — subcampaign overlay is the simplest change that
works for the known constraint (max depth 3, one subcampaign active at a time).

### Problem 2: Worktree Lifecycle in Campaigns

**Approach A — Inject PostTaskFunc into campaign.Runner (Recommended)**

Add a `PostTaskFunc func(beadID string) error` field to `campaign.Config` or
`Runner`. Call it after each successful task pipeline (not for recursive
feature/epic entries). The wiring in `main.go` passes the existing
`postPipeline` logic.

The campaign runner already has the injection point — `runPostPipeline` is
a method that can be replaced with a configurable func.

**Approach B — PostTaskFunc as callback method**

Add `OnPostTask(beadID string) error` to `campaign.Callback`. The callback
adapters (CLI and dashboard) handle merge/cleanup.

**Cons**: Callbacks are notification-only (no return values by convention).
Adding an error return to one method breaks the pattern. The CLI adapter
would need access to the `worktree.Manager`.

**Approach C — Run merge/cleanup inside the PipelineRunner**

Make the pipeline runner itself handle worktree merge after each task.

**Cons**: The pipeline runner shouldn't know about campaign-level concerns.
Also, the pipeline creates the worktree before running, but post-pipeline
merge is a campaign-level decision (maybe skip merge on failure).

**Decision: Approach A** — clean injection, matches the architecture
constraint that campaigns don't know about worktrees.

**Sequencing concern**: Each task must merge to main before the next task
starts, so the next task branches from updated main. This is naturally
satisfied — `runRecursive` is sequential, and `PostTaskFunc` runs before
advancing to the next task.

### Problem 3: Error Observability

**Approach — Multi-part fix (Recommended)**

This is straightforward; no alternatives needed:

1. **State save failures**: Replace `_ = r.store.Save(state)` with
   `if err := r.store.Save(state); err != nil { log to stderr }`.
   Campaign should continue — state saves are best-effort.

2. **Error text in messages**: Add `Error string` field to
   `CampaignTaskDoneMsg`. Wire `OnTaskFail(beadID, err)` to populate it.
   The adapter currently discards `err` with `_ error`.

3. **Dashboard error display**: When a failed task is selected in the
   campaign view, show the error text in the right pane. This follows the
   existing pattern where the right pane shows phase reports for completed
   tasks.

4. **Bead close failures**: Replace `_ = r.beads.Close(beadID)` in
   `runPostPipeline` with logged warning.

## Risks & Unknowns

| Risk | Impact | Mitigation |
|------|--------|------------|
| Merge conflict mid-campaign | Blocks remaining tasks | Agent pair resolves first; pause + toast notification if unresolvable |
| Subcampaign state struct complexity | Hard to maintain | Keep it flat — only one nesting level, no tree |
| Background mode + nested messages | Messages arrive for wrong mode | Already proven to work — messages route via `backgroundMode` check |
| Campaign resume after worktree merge | State indexes off | PostTaskFunc runs after pipeline, before state advance — index still correct |
| Two spinners (task + phase) | Animation conflicts | Bubble Tea handles concurrent spinners already in pipeline view |

## Suggested Scope

**Three features, one epic:**

| Feature | Tasks (est) | Description |
|---------|-------------|-------------|
| Worktree lifecycle | 2–3 | PostTaskFunc injection, merge/cleanup per task, conflict handling |
| Nested campaign animation | 3–4 | Callback stack, SubCampaign messages, dashboard overlay, view rendering |
| Error observability | 2–3 | Log state saves, error in messages, dashboard error display |

**Recommended sequencing:**
1. **Worktree lifecycle first** — data integrity before visual polish
2. **Error observability second** — small, independent, immediately useful
3. **Nested animation last** — most complex, but depends on the others being solid

## Observability Improvements (Deferred)

The handoff doc includes an observability wishlist. These are valid but
separate from the three core problems. Capture as a separate epic or backlog:

- Campaign-level worklog aggregation
- `capsule status` command
- Structured JSONL event log
- Dashboard elapsed time + ETA
- Error accumulator / summary view
- Worktree health check at campaign start

## Resolved Questions

1. **Merge conflict policy** (resolved): Agent resolves first. On
   `ErrMergeConflict`, invoke `execute` → `sign-off` pair with conflict
   context. If the agent cannot resolve after max retries, pause campaign
   and emit a non-blocking notification (status bar / toast) with resolution
   instructions. Campaign remains resumable.

2. **Agent pair selection** (resolved): All behavior should be driven by
   the appropriate agent pair. When items are missed, identify which pair
   addresses the gap (see Agent Pair Catalog above) and invoke it rather
   than patching around the issue.

## Open Questions

1. **PostTaskFunc for recursive entries**: When a feature finishes (all its
   tasks completed), should `PostTaskFunc` also run for the feature's own
   bead? Currently `runPostPipeline` runs for it. Probably yes — the feature
   bead itself may need its worktree closed.

2. **Validation and worktrees**: Feature validation runs after all tasks.
   Does validation need its own worktree, or does it run on main after all
   merges?

## References

- Handoff document: `docs/handoff-campaign-ux.md`
- Agent orchestration patterns: `docs/agent-orchestration-patterns.md`
- Campaign runner: `internal/campaign/campaign.go`
- Dashboard campaign state: `internal/dashboard/campaign.go`
- Dashboard model: `internal/dashboard/model.go`
- Dashboard messages: `internal/dashboard/msg.go`
- Worktree manager: `internal/worktree/worktree.go`
- Post-pipeline: `cmd/capsule/main.go:314`
- Phase definitions: `internal/orchestrator/phases.go`
- Prompt templates: `prompts/*.md`
- Signal contract: `docs/signal-contract.md`
- Previous brainstorm: `docs/planning/brainstorm-capsule-v2.md`
