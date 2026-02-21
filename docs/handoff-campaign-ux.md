# Handoff: Campaign UX — What's Broken and What Users Should See

## The User's Journey Today (What's Broken)

### Running a single task: works great
```
capsule run cap-task-1
```
You get a beautiful animated TUI: spinner on the active phase, elapsed time ticking, checkmarks appearing as phases complete, right pane showing summaries. This is the gold standard.

### Running a feature (campaign of tasks): works in dashboard
```
capsule dashboard → select feature → Enter → Enter
```
Left pane shows task queue with status indicators. When a task is running, its pipeline phases appear nested underneath. Spinner animates. Task completes, checkmark appears, next task starts. This also works.

### Running an epic (campaign of features): broken
```
capsule dashboard → select epic → Enter → Enter
```
**What happens**: The dashboard shows the epic's features as a task list. When the first feature starts and discovers its child tasks, a `CampaignStartMsg` fires that **replaces the entire campaign state**. The epic-level view vanishes. You now see the feature's tasks as if you'd dispatched the feature directly. When the feature finishes, the task index is wrong and the display is confused.

**What should happen**: The epic's features stay visible. When a feature is running, its child tasks appear nested below it with their own progress. Pipeline phases animate under the active task. When the feature completes, the nested tasks collapse and the feature row shows a checkmark.

---

## Three Problems, Three Fixes

### 1. Worktree Lifecycle (data integrity)

**Symptom**: Worktree errors in test repo. Orphaned directories in `.capsule/worktrees/`.

**Root cause**: When a campaign runs tasks, each task's pipeline creates a git worktree. After the pipeline completes, the campaign only closes the bead — nobody merges the changes back to main, removes the worktree, or prunes stale metadata. The merge/cleanup code exists (`postPipeline()` in main.go) but is only wired for single-task runs.

**Consequence**: Worktrees accumulate. Next task branches from stale main (missing previous task's changes). Eventually git errors about existing branches/worktrees.

**Fix idea**: Inject a `PostTaskFunc` into the campaign runner that calls the existing merge/cleanup logic after each successful task.

### 2. Nested Campaign Animation (visual correctness)

**Symptom**: Feature animation doesn't work when run from epic.

**Root cause**: The `campaign.Callback` interface fires `OnCampaignStart` at every recursion depth. The dashboard callback adapter sends `CampaignStartMsg` each time, which replaces the dashboard's flat `campaignState`. The callback also resets its internal task counter.

**Fix idea**: Add a stack to the callback adapter. Depth 1 = root campaign (full state replacement, as today). Depth >1 = subcampaign (overlay on the parent's running feature row). New message types (`SubCampaignStartMsg`/`SubCampaignDoneMsg`) let the dashboard add/remove nested task rows without losing the parent view.

### 3. Silent Error Handling (observability)

**Symptom**: Things fail and nobody knows. The campaign has 6 instances of `_ = r.store.Save(state)`. Failed tasks show no error detail in the dashboard — just a red X with no explanation.

**Fix idea**: Log state save failures to stderr. Add error text to `CampaignTaskDoneMsg`. Show error in the dashboard's right pane when a failed task is selected.

---

## UX Vision: What Each Level Should Look Like

### Epic Level (dispatched from dashboard)

**Pending state** — before dispatch:
```
Browse mode, right pane shows epic detail with acceptance criteria.
Help bar: "Enter: run campaign (2 features, 5 tasks)"
```

**Confirmation screen**:
```
Run campaign for cap-epic?
  Feature 1 — Implement auth (3 tasks)
  Feature 2 — Add dashboard (2 tasks)

  Provider: claude
  Validation: enabled

  [Enter] Run    [Esc] Cancel
```

**In progress** — Feature 1 running, Feature 2 pending:
```
cap-epic  My Epic  0/2  [Feature 1: 1/3]  [claude]
  ⋮ Feature 1
      ✓ Task 1a — Write auth middleware         4.1s
      ⋮ Task 1b — Add login endpoint
          ✓ test-writer                         1.2s
          ⋮ execute                             (3s)
          ○ execute-review
          ○ sign-off
          ○ merge
      ○ Task 1c — Add logout endpoint
  ○ Feature 2
```

**Feature 1 done, Feature 2 running**:
```
cap-epic  My Epic  1/2  [Feature 2: 0/2]  [claude]
  ✓ Feature 1                              28.4s
  ⋮ Feature 2
      ⋮ Task 2a — Create dashboard layout
          ⋮ test-writer                    (1s)
          ○ execute
          ...
      ○ Task 2b — Add data widgets
```

**Completed** — campaign summary:
```
cap-epic  My Epic  2/2  [claude]
  ✓ Feature 1                              28.4s
  ✓ Feature 2                              19.7s

  ✓ Feature validation                      5.2s
```

**Failed** — task failure visible:
```
cap-epic  My Epic  0/2  [Feature 1: 1/3]  [claude]
  ⋮ Feature 1
      ✓ Task 1a — Write auth middleware     4.1s
      ✗ Task 1b — Add login endpoint       12.3s   ← select this
      ○ Task 1c — Add logout endpoint
  ○ Feature 2

Right pane shows:
  Task 1b — Add login endpoint

  ✗  Failed

  pipeline: phase 'execute-review' attempt 3: status NEEDS_WORK

  Feedback:
  The login endpoint returns 500 when the database
  connection pool is exhausted. Add retry logic...
```

### Feature Level (standalone or within epic)

Same as today's campaign view, but with error detail in right pane for failed tasks.

### Task Level (standalone)

Unchanged — this already works correctly with the TUI display.

---

## State Transitions to Support

| Level | State | Visual |
|-------|-------|--------|
| Epic/Feature row | Pending | `○` dimmed text |
| Epic/Feature row | Running | `⋮` spinner, nested children visible |
| Epic/Feature row | Completed | `✓` green, duration shown |
| Epic/Feature row | Failed | `✗` red, selectable for error detail |
| Task row | Pending | `○` dimmed, indented under feature |
| Task row | Running | `⋮` spinner, pipeline phases visible below |
| Task row | Completed | `✓` green, duration, phases collapse |
| Task row | Failed | `✗` red, error in right pane on select |
| Phase row | Pending | `○` dimmed, deep indent |
| Phase row | Running | `⋮` spinner + elapsed time ticking |
| Phase row | Passed | `✓` green + duration |
| Phase row | Failed | `✗` red + duration + retry count |
| Phase row | Skipped | `⊘` dimmed |

---

## Error Handling: What Should Happen When Things Go Wrong

| Scenario | Current Behavior | Desired Behavior |
|----------|-----------------|-----------------|
| Worktree creation fails | Pipeline error, worktree dir may be partial | Clear error: "worktree for cap-xxx already exists. Run: capsule clean cap-xxx" |
| Merge conflict after task | Never happens (merge not called) | Warning in status line, manual resolution instructions, campaign continues to next task |
| Worktree cleanup fails | Never happens | Warning logged, campaign continues (best-effort cleanup) |
| State save fails | Silently swallowed (`_ =`) | Warning to stderr, campaign continues |
| Stale worktree from previous run | `ErrAlreadyExists` blocks new run | Detect + offer cleanup, or auto-prune if safe |
| Task pipeline fails | Red X, no detail | Red X + error message in right pane with phase feedback |
| Circuit breaker trips | Campaign stops | Campaign stops + shows which N tasks failed consecutively |
| Feature validation fails | Red X | Validation error detail visible, feature marked failed |

---

## Observability Wishlist

**Today** (what exists):
- Plain text timestamps for `capsule campaign` CLI
- Phase reports stored per-task in `campaignState.taskReports`
- Campaign state persisted to `.capsule/campaigns/<id>.json`
- Worklog archived to `.capsule/logs/<beadID>/worklog.md`

**Gaps**:
- No campaign-level worklog (only per-task)
- No way to see worktree state during a campaign
- Error messages swallowed, not displayed
- No structured logging (just fmt.Fprintf to stderr)
- Background campaign completion message doesn't show task breakdown
- No elapsed time for the overall campaign

**Possible improvements** (for brainstorming, not all necessarily worth doing):
- Campaign worklog: aggregate all task worklogs into a campaign-level summary
- `capsule status` command: show active worktrees, running campaigns, last error
- Structured event log: JSONL file of all campaign lifecycle events with timestamps
- Dashboard status bar: show campaign elapsed time + ETA based on completed task durations
- Error accumulator: collect all warnings/errors during campaign, show in summary view
- Worktree health check at campaign start: prune stale, warn about existing

---

## Architecture Constraints

Things to keep in mind during brainstorming:

1. **Campaign package doesn't know about worktrees** — it uses `PipelineRunner` interface. Worktree lifecycle is injected via `PostTaskFunc`.
2. **Dashboard state is value-typed** (Bubble Tea pattern) — `campaignState` is a struct, not a pointer. Updates return new copies.
3. **Callback interface is shared** between CLI plain text and dashboard TUI adapters. Changes to `campaign.Callback` affect both.
4. **Event channel is the bridge** — campaign goroutine → `chan tea.Msg` → dashboard model. All state changes must flow through messages.
5. **Background mode** — user can Esc from campaign view to browse while it runs. State must survive the mode transition.
6. **Max recursion depth is 3** — epic → feature → task. The subcampaign overlay only needs one level of nesting for the initial implementation.
