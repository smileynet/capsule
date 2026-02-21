# AI Agent Orchestration Patterns

Best practices and antipatterns for multi-agent systems, with specific recommendations
for the capsule TDD pipeline architecture (epic → feature → task, writer/reviewer pairs,
nested campaign orchestration).

Sources drawn from: AWS Prescriptive Guidance agentic patterns, Anthropic engineering
blog, Google ADK developer guide, Smashing Magazine agentic UX, Portkey LLM resilience,
Arion Research conflict playbook, Tweag agentic coding handbook, sedkodes deterministic
SWE agent analysis, and direct inspection of capsule's existing orchestrator and campaign
packages.

---

## 1. Writer/Reviewer Agent Pairs

### What they are in capsule

The orchestrator implements writer/reviewer pairs as linked `PhaseDefinition` entries
connected by `RetryTarget`. When a reviewer returns `NEEDS_WORK`, `runPhasePair` re-runs
the worker with the reviewer's `Feedback` field injected into the prompt context, then
re-runs the reviewer. This loop continues up to `MaxAttempts`.

The current default pipeline has three pairs:
- `test-writer` → `test-review`
- `execute` → `execute-review`
- `execute` → `sign-off` (same worker, second independent reviewer)

### Best practices

**Separate concerns cleanly.** The worker's job is to produce an artifact. The reviewer's
job is to evaluate it against a rubric, not to fix it. Workers that also criticize and
reviewers that also patch create ambiguous signal (which phase owns the output?) and
compound retry state.

**Give reviewers a structured rubric, not free-form critique.** Reviewers should evaluate
against explicit criteria (test coverage, naming conventions, compilation success, no
regressions) rather than producing open-ended editorial feedback. This produces
consistent, actionable `Feedback` strings the worker can act on. AWS prescriptive guidance
calls this the "critique prompt or evaluation rubric" pattern.

**Pass feedback verbatim into the worker's next context.** Capsule already does this
correctly: `workerCtx.Feedback = feedback` threads the reviewer's signal into the next
worker invocation. The entire prior reviewer `Feedback` string should be preserved — do
not summarize or truncate it, since the worker needs the full specificity.

**Use a fixed retry ceiling.** The AWS evaluator loop pattern specifies "loop until
criteria met, approved, or retry limit reached." Open-ended loops create runaway cost and
wall-clock time. The ceiling should be visible in the UI (capsule already exposes
`MaxRetry` in `StatusUpdate`). Three attempts is appropriate for most phases; `merge`
should have exactly one (merges are not a quality-improvement loop — they either work or
they do not).

**Escalate to a stronger provider after N failures, not at the start.** The current
`EscalateProvider` / `EscalateAfter` mechanism in `RetryStrategy` is the right shape.
A fast/cheap model handles the first attempts; the stronger model is reserved for
hard cases. This avoids paying for escalation on trivially fixable issues.

**Treat consecutive reviewer passes as the exit condition, not a simple pass count.**
If `execute-review` passes but `sign-off` returns `NEEDS_WORK`, the pair loop for
`sign-off` should re-run `execute` — not `execute-review`. Each reviewer pair is an
independent quality gate. Capsule's `RetryTarget` field handles this correctly: both
`execute-review` and `sign-off` target `execute`, so each independently retries the
worker.

**Workers that return ERROR are terminal; workers that return NEEDS_WORK are not.**
The orchestrator already treats `StatusError` from a worker as immediate pipeline
failure, and `StatusNeedsWork` as "the reviewer will evaluate." This is correct.
A worker signaling NEEDS_WORK should be treated like PASS for routing purposes — the
artifact exists and the reviewer must judge it.

**Distinguish "quality failure" from "execution failure" in escalation.**
`StatusNeedsWork` after max retries means the agent pair cannot produce acceptable
output — this is a quality failure and should escalate to human review with the final
`Feedback` string preserved. `StatusError` means the provider or tool failed — this
may be retried with backoff, escalated to a different provider, or aborted.

### Antipatterns

**Infinite retry without ceiling.** Without `MaxAttempts`, a reviewer that consistently
disagrees with a worker will loop forever, consuming tokens and wall-clock time with
no recovery path.

**Reviewer that modifies artifacts.** If a reviewer patches files and then evaluates
its own patches, the feedback loop collapses into a single agent with confused role
semantics. The reviewer's output should be a structured verdict (`PASS` / `NEEDS_WORK`
with `Feedback`), not a diff.

**Feedback truncation between attempts.** Summarizing or truncating the reviewer's
feedback before passing it to the worker loses specificity. The worker cannot fix what
it cannot read.

**Same provider for worker and reviewer on the same artifact.** Using identical model
weights for both roles produces correlation: the reviewer tends to approve the output
it would itself produce. Use distinct providers or distinct prompts with explicitly
different evaluation stances when independence matters.

**Escalating immediately on first NEEDS_WORK.** Escalation to a more capable (and
expensive) provider should be a last resort after the default provider has exhausted its
retries, not a first response to a quality signal.

### Recommendations for capsule

The `runPhasePair` implementation in `orchestrator.go` is architecturally sound. The
gaps are in configuration and observability:

1. The `EscalateAfter` field in `RetryStrategy` should default to `MaxAttempts - 1`
   so escalation activates only on the final attempt, not mid-sequence.

2. The `StatusUpdate` emitted when `NEEDS_WORK` is received should include the
   `Feedback` string so the TUI can display it in the right pane. Currently `Signal`
   is populated but only on completion — populate it on failure too.

3. Add a `ReviewerRubric` string field to `PhaseDefinition` (or load it from the
   prompt template directory) so the reviewer's evaluation criteria are visible in
   config and can be audited without reading the prompt file.

---

## 2. Merge Conflict Resolution by AI Agents

### The escalation decision

AI agents should attempt auto-resolution only when the conflict is structurally
unambiguous — the classic case is two branches that each add lines to the same file in
non-overlapping regions. Semantic conflicts (where both branches intend to change the
same behavior in incompatible ways) require human judgment.

The most practical heuristic from tools like Syncwright: attach a confidence score to
each proposed resolution. Default auto-apply threshold is 0.8 (configurable). Below
threshold, the agent writes the proposed resolution to a staging file and surfaces it
for human review rather than applying it.

**Confidence scoring factors:**
- Conflict region is purely additive (new lines only, no edits to shared lines): high confidence
- Conflict region involves variable rename, function signature change, or type annotation: medium confidence
- Conflict region is inside generated code or vendor directories: high confidence (take theirs or ours by convention)
- Conflict region is inside a lock file or migration: medium confidence with explicit strategy
- Conflict region involves business logic, API contracts, or test expectations: low confidence, escalate

### Best practices

**Prefer prevention over resolution.** The most effective AI merge strategy is frequent
integration: smaller, more frequent PRs from the AI agent produce fewer conflicts than
large batches. In a TDD pipeline, merging after each task (rather than after each
feature) keeps the conflict surface minimal.

**Apply resolution to a staging branch, not directly to main.** The agent should:
1. Create a merge branch: `merge/capsule-<beadID>`
2. Apply the proposed resolution
3. Run the gate suite (compile, tests) to verify the resolution does not break the build
4. Only fast-forward main if gates pass

This matches capsule's existing `merge` phase semantics but makes the intermediate
branch explicit so it can be inspected if the merge fails.

**Surface the three-way diff, not just the resolution.** When a human must review,
show them what both branches intended (ours, theirs, base) alongside the proposed
resolution. A raw conflict marker dump is harder to evaluate than a structured
side-by-side.

**Log every auto-resolution with its confidence score.** If a silently auto-resolved
conflict later causes a regression, the audit trail must show what was resolved and why.
This is the "action audit log" pattern from Smashing Magazine's agentic UX guide.

**Never force-push.** Even when the merge agent is confident, rewriting shared history
destroys the ability to bisect and attribute changes. Fast-forward merges or merge
commits are the safe alternatives.

### Antipatterns

**Blind force-push after conflict resolution.** This destroys the history needed to
diagnose regressions and makes it impossible for collaborators to reconcile their
local state.

**Silent conflict dropping (choosing "ours" or "theirs" without recording the decision).**
If an agent silently discards one side of a conflict without recording which side it
chose and why, the losing changes vanish without trace. All auto-resolutions must be
logged with the strategy applied.

**Applying resolution without running tests.** A syntactically valid merge can still
break the build. Always run the compile and test gate before declaring success.

**Treating lock files the same as source files.** Lock files (go.sum, package-lock.json,
Cargo.lock) have well-defined merge strategies (regenerate from source, or take theirs
if both updated the same dependency). Applying LLM reasoning to a lock file diff wastes
tokens and produces worse results than a rule-based strategy.

**Running the merge agent while the worktree is dirty.** The merge phase must start
from a clean working tree on the target branch. Running merge with uncommitted changes
from a prior phase contaminates the merge commit.

### Recommendations for capsule

The existing `merge` phase is a single `Worker` with `MaxRetries: 1`. This is correct —
a merge is not a quality-improvement loop. The gaps:

1. The merge agent should create an intermediate `merge/<beadID>` branch, run a
   compile+test gate, and only fast-forward main if the gate passes. The gate failure
   should surface as a `StatusError`, not a merge conflict.

2. The `runPostPipeline` in `campaign.go` currently calls `r.beads.Close(beadID)` with
   `_ =` (swallowing the error). Merge failures that happen after bead close create an
   inconsistent state (bead closed, code not merged). Merge should run before close, and
   close should be conditional on merge success.

3. Consider adding a `merge-review` phase after `merge` in `ThoroughPhases()` that runs
   a git log check and a test suite on main (not the worktree) to verify the merged
   result. This catches cases where the merge introduced regressions not caught in the
   worktree.

---

## 3. Campaign/Pipeline Error Handling

### Failure taxonomy

Not all errors are equal. The correct response depends on the failure class:

| Failure class | Example | Recommended response |
|---|---|---|
| Transient provider failure | API timeout, 503 | Retry with exponential backoff + jitter |
| Quality failure (NEEDS_WORK) | Reviewer rejects output | Retry pair loop up to MaxAttempts |
| Tool failure (gate) | Compile error, test failure | Retry if non-deterministic, otherwise fail task |
| Infrastructure failure | Worktree creation fails | Fail task with actionable error, do not retry |
| Persistent quality failure | Max retries exceeded | Fail task, preserve feedback, continue campaign or abort per FailureMode |
| Context cancellation | User pressed Esc | Pause (save checkpoint), resume later |
| Circuit breaker threshold | N consecutive task failures | Halt campaign with explanation |

### Pipeline-level: pause vs abort vs skip vs retry

**Pause** is the correct response to user-initiated interruption (`ctx.Err()` or
`ErrPipelinePaused`). Capsule already checkpoints phase results before each phase
boundary and saves campaign state on pause. The checkpoint must be saved at the
*phase boundary*, not at the start of a phase, so a paused pipeline resumes from the
last completed phase rather than re-running it.

**Abort** (FailureMode = "abort") is appropriate when any task failure invalidates
subsequent tasks — typically when tasks have hard dependencies (task B cannot run if
task A's output is missing). This should be the default for features with a strict
implementation sequence.

**Continue** (FailureMode = "continue") is appropriate when tasks are largely
independent and a failure in one does not block the others. The circuit breaker
provides a safety valve: if N consecutive tasks fail, the campaign stops rather than
exhausting all remaining tasks against a systemic problem.

**Skip** is appropriate only for explicitly optional phases (gate phases with
`Optional: true`). Task-level skipping based on dynamic conditions is an antipattern —
it creates invisible gaps in the campaign that are hard to audit.

**Retry** at the campaign level (retrying a failed task from scratch) is not currently
supported by capsule and is generally not recommended for AI-generated code tasks. The
task failure likely reflects a quality problem with the specification or a systemic
provider issue, neither of which self-resolves on retry. Instead: fail the task, preserve
the error and reviewer feedback, and let the user fix the spec before re-running.

### Circuit breaker calibration

The circuit breaker threshold should reflect what fraction of task failures indicates
a systemic problem versus a local one. For a 5-task feature:
- Threshold 1: Any failure stops the campaign (appropriate when tasks are tightly coupled)
- Threshold 2: Two consecutive failures stop the campaign (default for most features)
- Threshold N (disabled): Campaign runs all tasks regardless of failures (appropriate only for independent, low-stakes tasks)

Capsule's `ConsecFailures` counter (reset on each success) is the right counter to use.
A campaign that alternates success/failure indefinitely suggests the tasks themselves
have inconsistent quality signals, which is a prompt engineering problem rather than
a circuit breaker problem.

### Saga-style compensation

When a campaign runs tasks that produce side effects (worktrees, merged commits, filed
discoveries), failures mid-campaign leave partial state. The saga pattern prescribes
compensating transactions: if task 3 fails after tasks 1 and 2 succeeded, run
compensating actions (close worktrees created by tasks 1 and 2, remove partially filed
discoveries) to restore a clean state before surfacing the error.

In practice for capsule, compensation is currently best-effort: `runPostPipeline` closes
the bead, and `worktreeMgr.Remove` cleans up the worktree. The gap is that if the
campaign is running in "continue" mode and task 2 fails after task 1 succeeded and
merged, there is no compensation — main now has task 1's changes but not task 2's.
This is the correct behavior (partial progress is preserved), but it must be visible:
the dashboard should show "1 merged, 1 failed, 3 pending" rather than a simple count.

### Best practices

**Classify failures before deciding on response.** The Portkey layered approach:
retries handle transient issues, fallbacks handle provider unavailability, circuit
breakers handle systemic degradation. Treat these as distinct layers, not alternatives.

**Make failure visible at the level it occurred.** A task failure should be visible
at the task row, the feature row, and the epic row — each level showing the appropriate
summary (task: phase + feedback, feature: N failed, epic: M features failed). Capsule's
current handoff doc identifies this as the "silent error handling" gap.

**Preserve reviewer feedback on failure.** When a task fails because `execute-review`
returned `NEEDS_WORK` after exhausting retries, the final `Feedback` string is the most
valuable diagnostic artifact. It should be stored in `TaskResult.Error` (or a separate
field) and displayed in the dashboard's right pane when the failed task is selected.

**Do not retry tasks automatically on campaign restart.** If a campaign is resumed after
a failure, tasks marked `TaskFailed` should remain failed unless the user explicitly
requests a retry. Silent re-execution of previously failed tasks can duplicate side
effects and creates audit confusion.

### Antipatterns

**Swallowing errors silently.** The `_ = r.store.Save(state)` pattern throughout
`campaign.go` discards save failures. At minimum, save failures should be written to
stderr. Better: accumulate warnings in campaign state and display them in the summary.

**Aborting on the first transient error without retry.** Network blips, API rate limits,
and cold starts are not task failures. The pipeline should distinguish provider errors
(transient, retry with backoff) from quality signals (NEEDS_WORK, use the pair loop).

**Blocking the entire campaign on a non-blocking failure.** If discovery filing fails
(`BeadCreate` returns an error), the task that produced the discovery should still be
marked complete. The discovery filing failure is a warning, not a task failure.

**Unhelpful error messages.** "worktree for cap-xxx already exists" is actionable.
"pipeline: phase execute-review attempt 3: status NEEDS_WORK" with no feedback is not.
Error messages must include the reviewer's final Feedback string.

### Recommendations for capsule

1. Replace `_ = r.store.Save(state)` with `if err := r.store.Save(state); err != nil {
   fmt.Fprintf(os.Stderr, "campaign: warning: state save: %v\n", err) }` throughout
   `campaign.go`.

2. Add an `ErrorDetail` field to `TaskResult` (separate from `Error`) that stores the
   final reviewer `Feedback` string. Thread this through `OnTaskFail` callback and
   display it in the dashboard right pane.

3. The `FailureMode` config currently supports "abort" | "continue". Add "pause-on-fail"
   as a third mode: fail the task, surface the error, and pause the campaign waiting for
   user confirmation before continuing. This gives users a human-in-the-loop option
   without requiring a full abort.

4. Circuit breaker trips should record which tasks failed consecutively so the error
   message can say "stopped after 3 consecutive failures: cap-1.1, cap-1.2, cap-1.3"
   rather than just "circuit breaker tripped."

---

## 4. Nested Orchestration (Epic → Feature → Task)

### The core state management problem

Capsule's `runRecursive` in `campaign.go` creates independent `State` objects keyed
by `parentID` at each depth level. The epic's state tracks features; each feature's
state tracks tasks. This is architecturally correct — each level owns its own state
and the levels are isolated from each other.

The current bug (identified in `handoff-campaign-ux.md`) is not in the state model
itself but in the dashboard callback: `OnCampaignStart` fires at every depth, and the
dashboard callback replaces the root `campaignState` instead of overlaying a child
state. The fix is in the callback adapter (depth-aware overlay), not in the campaign
runner.

### Best practices

**Each orchestration level owns its own state.** The epic campaign should not hold
references into feature campaign state objects. State flows upward through callbacks
(OnTaskComplete, OnCampaignComplete), never downward through shared pointers. This is
the "output_key to shared session.state" pattern from Google's ADK — state is threaded
by message, not by pointer.

**Callbacks carry depth context.** The `Callback` interface should pass the recursion
depth (or a parent ID path) with every event so the dashboard adapter can route the
event to the correct level of the UI tree. The current interface omits depth, forcing
the adapter to infer it from counter state — which breaks when features have different
numbers of tasks.

**Resume from the failed level, not from the root.** If task cap-1.1.2 fails and the
campaign is paused, resuming should restart from cap-1.1.2 (the task level), not from
cap-1.1 (the feature level) or cap-1 (the epic level). Each `State` object's
`CurrentTaskIdx` and `ConsecFailures` must survive pause/resume independently.

**Enforce the depth limit at entry, not exit.** `maxCampaignDepth = 3` is checked at
the start of `runRecursive`. This is correct. Checking at exit (after the recursion has
already run) would only detect the violation too late.

**Visited-set cycle detection is mandatory.** The `visited map[string]bool` prevents
a bead from being its own grandparent. This is correct and must be retained. The map is
passed by reference through all recursion levels, so any cycle at any depth is caught.

**State isolation between parallel sub-campaigns.** If features within an epic are ever
run in parallel (not the current sequential model), each feature's state must be in a
separate store key. Sharing a store key between concurrent features creates race
conditions. The current `parentID` keying is correct for sequential execution; parallel
execution would require a unique key per invocation (e.g., `parentID + "/" + timestamp`).

### Antipatterns

**Global state replacement.** The dashboard bug in `handoff-campaign-ux.md` is the
canonical example: `CampaignStartMsg` replaces the entire `campaignState` instead of
overlaying it. The fix is explicit: introduce `SubCampaignStartMsg` for depth > 1 and
add a stack or parent-reference to the campaign state in the TUI model.

**Callback hell.** Nesting callbacks inside callbacks to propagate state upward creates
code that is difficult to trace and test. The Bubble Tea message-passing model is the
correct alternative: each event is a discrete `tea.Msg` sent through a channel, and
the model's `Update` function handles routing. All state changes happen in a single
goroutine (the Bubble Tea event loop), eliminating concurrency bugs.

**Shared mutable state across recursion levels.** Passing a pointer to the epic's
`State` into the feature's `runRecursive` so the feature can update epic-level counters
directly creates hidden coupling. Use callbacks for upward communication.

**Excessive depth.** Three levels (epic → feature → task) is the practical limit for
human comprehension. Deeper hierarchies (epic → feature → sub-feature → task) create
visual nesting that is hard to follow in a TUI and hard to reason about in error cases.
The current `maxCampaignDepth = 3` enforces this correctly.

**Recursive retry that restarts from root.** If the epic campaign catches an error from
a feature campaign and retries the feature, all of the feature's completed tasks will
be re-run. The retry must be aware of the feature's `CurrentTaskIdx` and resume, not
restart.

### Recommendations for capsule

1. Add a `Depth int` and `ParentPath []string` to every `Callback` method signature, or
   wrap events in an envelope: `type CampaignEvent struct { Depth int; ParentID string;
   Payload interface{} }`. The dashboard adapter can then route events without inference.

2. Implement the subcampaign overlay in the dashboard by maintaining a campaign stack:
   ```go
   type campaignStack struct {
       levels []campaignLevel  // index 0 = epic, 1 = feature, 2 = task
   }
   ```
   `OnCampaignStart` at depth 0 replaces the root level. At depth > 0 it pushes a new
   level and marks the parent level's currently-running item as "has children."

3. `runPostPipeline` should be injected as a `PostTaskFunc` (as the handoff doc
   suggests) rather than a hard-coded method call. This allows the worktree merge/cleanup
   logic to vary between single-task runs and campaign runs without modifying the
   campaign runner.

4. The visited-set (`map[string]bool`) should be deep-copied before each recursive call
   if parallel sub-campaigns are ever added. Currently it is safe because execution is
   sequential.

---

## 5. Non-Blocking Notifications

### The fundamental tension

Long-running campaigns create a tension between two user needs: (a) flow — the ability
to use the dashboard for other work while the campaign runs, and (b) awareness — knowing
when something important happens without having to actively poll.

The resolution is a layered notification model: ambient status for routine progress,
escalating notifications for events that require a decision.

### Layered notification model

**Layer 1: In-place status animation (ambient)**

The spinner on the active task row in the left pane provides continuous progress
feedback without occupying any additional screen space. No user action required.
This is what capsule's campaign view already provides. It is non-blocking and
non-interruptive.

**Layer 2: Status bar (persistent, non-modal)**

A status bar at the bottom of the dashboard shows campaign-level progress: elapsed time,
current task count, last event. This is visible even when the user is browsing other
beads in the browse view. The Carbon Design System calls this a "banner notification" —
it is in the user's field of view but does not block interaction.

Implementation note: the status bar should update on every `OnTaskComplete` and
`OnTaskFail` event even when the campaign view is not the active mode. This requires
the dashboard model to track campaign progress independently of which mode is displayed.

**Layer 3: Toast notification (ephemeral, attention-seeking)**

When the campaign completes or a circuit breaker trips, a toast appears in a corner of
the terminal and auto-dismisses after 5 seconds. The toast shows: campaign ID, final
status (✓ complete / ✗ failed), task count, elapsed time. If the status requires user
action (e.g., "feature validation failed — select to review"), include a keybinding hint
and keep the toast visible until dismissed.

In a terminal TUI, "toast" is implemented as an overlay rendered by the Bubble Tea model
at the top of the view stack. The `tea.Tick` command drives auto-dismiss.

**Layer 4: Right-pane detail (on-demand, full context)**

When the user selects a failed task in the left pane, the right pane shows the full
error detail: phase name, attempt count, final reviewer feedback. This is the deepest
layer — maximum information, requires explicit navigation. This is where the error
messages from `TaskResult.ErrorDetail` (see Section 3) should surface.

**Layer 5: Event log (forensic, persistent)**

A JSONL file at `.capsule/campaigns/<id>.events.jsonl` records every lifecycle event
with timestamps: campaign start, task start, task complete, task fail, phase transitions,
circuit breaker trips, state save failures. This is the "log notification" pattern from
the Astro UX design system — not for real-time awareness but for post-hoc diagnosis.

### Best practices

**Never block user input to deliver a notification.** Modal dialogs that require
acknowledgment before the campaign can continue are inappropriate for non-critical events.
The only appropriate use of a blocking modal in this context is a confirmation dialog
before a destructive action (starting a campaign, not receiving a notification from one).

**Match notification urgency to event severity.**
- Task completed: ambient (left-pane spinner → checkmark)
- Campaign completed: toast (ephemeral, auto-dismiss)
- Task failed: status bar update + right-pane detail on selection
- Circuit breaker tripped: toast (persistent, requires dismiss) + campaign halt
- State save failure: status bar warning (non-critical, not a toast)

**Provide a way to re-read dismissed notifications.** The event log (`capsule status`)
serves this purpose. Users who dismissed a toast should be able to recover the
information.

**Background mode must not lose events.** When the user presses Esc to enter browse
mode while a campaign runs, the campaign goroutine continues sending messages through
the `chan tea.Msg`. The dashboard model must handle `CampaignTaskDoneMsg` and
`CampaignTaskFailMsg` in browse mode (updating the status bar) even when the campaign
view is not rendered.

**Show elapsed time at every level.** Users calibrate their attention based on how
long something is taking. Displaying elapsed time on the active task row (already done
in capsule's pipeline TUI) and on the campaign status bar gives users enough signal to
decide whether to watch or return to other work.

### Antipatterns

**Blocking modal for every event.** A dialog that says "Task cap-1.1 completed. Press
Enter to continue." on every task completion is the worst case. It serializes human
attention with automated progress, turning a background process into a foreground one.

**No notification at all.** Completing a 20-minute campaign silently, with the user
needing to refresh the dashboard to see the result, is the other failure mode. At
minimum, a toast on completion.

**Spammy status updates.** Sending a status update for every phase of every task in a
campaign floods the status bar and trains the user to ignore it. Rate-limit or aggregate:
show "Feature 1: 3/5 phases complete" rather than streaming every phase name.

**Notifications that require the campaign view to be active.** If the status bar only
updates when the campaign view is rendered, users who switch to browse mode see stale
progress. All notification layers must work independently of which view is active.

**Hiding error detail behind a separate command.** If a failed task requires running
`capsule logs cap-1.1` to see the error, most users will not do it. The error must be
visible in the dashboard on selection with no additional command required.

### Recommendations for capsule

1. Add a `statusBar` field to the dashboard model that holds the last N events (e.g.,
   last 3). The status bar renders this list as a single line at the bottom of the
   terminal using the `WindowSizeMsg`-driven layout. Events are appended by all campaign
   callbacks regardless of active mode.

2. Implement a `ToastMsg` message type that carries text, a severity level, and an
   auto-dismiss duration. The dashboard model renders it as an overlay in the top-right
   corner. `tea.Tick(5 * time.Second, ...)` drives auto-dismiss. Persistent toasts
   (for circuit breaker events) require a dismiss keybinding (e.g., `d`).

3. Persist events to `.capsule/campaigns/<id>.events.jsonl` in the campaign runner
   (in the `Callback` implementations) so `capsule status` can report the last N events
   from a completed campaign.

4. The background mode transition (user presses Esc during campaign) must preserve
   campaign message routing. The `campaignCh chan tea.Msg` should remain active and
   the `Msg` handler in `Update()` should handle campaign messages in all modes, not
   just `ModeCampaign`.

---

## 6. Cross-Cutting Recommendations

### Observability first

The Anthropic multi-agent research team found that "observable decision patterns" were
essential for diagnosing root causes. Every phase transition, every retry, every
escalation, and every state save failure should be logged with a timestamp. Structured
logging (JSONL) is preferable to `fmt.Fprintf(os.Stderr, ...)` because it is
machine-readable and can be queried by `capsule status`.

### Test the failure paths explicitly

Capsule's `make test-full` suite should include tests for:
- Circuit breaker trips (N consecutive failures → `ErrCircuitBroken`)
- Pause/resume across all three recursion depths
- State save failures (mock store that returns error)
- Reviewer feedback threading (verify feedback from attempt 1 appears in attempt 2's worker prompt)
- Escalation provider switching (verify `EscalateProvider` is used after `EscalateAfter` attempts)

The Anthropic team found that human testing caught gaps that automated testing missed
(agents preferring SEO-optimized content over authoritative sources). For capsule, the
equivalent is running the full campaign smoke tests against a real provider to verify
that retry feedback actually improves output — something that mocks cannot verify.

### Explicit > implicit

Every decision in the pipeline — which provider runs a phase, what the retry ceiling is,
whether failures abort or continue — should be visible in the phase definition or config
file, not buried in defaults. The `PhaseDefinition` struct is already structured for this.
Extend the phase YAML schema to surface `max_retries`, `escalate_provider`,
`escalate_after`, and `failure_mode` as explicit fields rather than relying on defaults.

### Idempotency is non-negotiable for retried operations

Worktree creation, worklog creation, and bead closing must be idempotent. If a task is
retried after a partial failure, re-running these setup operations must not produce
duplicate worktrees or duplicate log entries. Capsule's `ErrAlreadyExists` handling in
the worktree package addresses worktree creation idempotency; verify that worklog
creation and bead closing are similarly protected.

---

## Sources

- [AWS Prescriptive Guidance — Evaluator Reflect-Refine Loop Patterns](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/evaluator-reflect-refine-loop-patterns.html)
- [AWS Prescriptive Guidance — Saga Orchestration Patterns](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/saga-orchestration-patterns.html)
- [AWS Prescriptive Guidance — Prompt Chaining Saga Patterns](https://docs.aws.amazon.com/prescriptive-guidance/latest/agentic-ai-patterns/prompt-chaining-saga-patterns.html)
- [Anthropic Engineering — How We Built Our Multi-Agent Research System](https://www.anthropic.com/engineering/multi-agent-research-system)
- [Google Developers — Developer's Guide to Multi-Agent Patterns in ADK](https://developers.googleblog.com/developers-guide-to-multi-agent-patterns-in-adk/)
- [Smashing Magazine — Designing for Agentic AI: Practical UX Patterns](https://www.smashingmagazine.com/2026/02/designing-agentic-ai-practical-ux-patterns/)
- [Portkey — Retries, Fallbacks, and Circuit Breakers in LLM Apps](https://portkey.ai/blog/retries-fallbacks-and-circuit-breakers-in-llm-apps/)
- [Arion Research — Conflict Resolution Playbook: Agentic AI Systems](https://www.arionresearch.com/blog/conflict-resolution-playbook-how-agentic-ai-systems-detect-negotiate-and-resolve-disputes-at-scale)
- [Tweag — Agentic Coding Handbook: TDD Workflow](https://tweag.github.io/agentic-coding-handbook/WORKFLOW_TDD/)
- [sedkodes — Building Competent AI SWE Agents Through Determinism](https://www.sedkodes.com/blog/2025-11-08-ai-workflows)
- [Graphite — The Role of AI in Merge Conflict Resolution](https://graphite.com/guides/ai-code-merge-conflict-resolution)
- [NeuBlink/Syncwright — AI-Powered Git Merge Conflict Resolution](https://github.com/NeuBlink/syncwright)
- [Redis — AI Agent Orchestration for Production Systems](https://redis.io/blog/ai-agent-orchestration/)
- [Towards Data Science — Why Your Multi-Agent System is Failing](https://towardsdatascience.com/why-your-multi-agent-system-is-failing-escaping-the-17x-error-trap-of-the-bag-of-agents/)
- [Latent Space — AI Agents Meet Test-Driven Development](https://www.latent.space/p/anita-tdd)
- [Carbon Design System — Notification Patterns](https://carbondesignsystem.com/patterns/notification-pattern/)
- [Astro UX DS — Notification Patterns](https://www.astrouxds.com/patterns/notifications/)
