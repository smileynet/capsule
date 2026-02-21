# Test Specification: Add PostTaskFunc to campaign.Config

## Bead: cap-9f0.1.1

## Tracer
Foundation — proves the injection point works and is called at the right time.

## Context
- Add `PostTaskFunc func(beadID string) error` to `campaign.Config`
- Call in `runRecursive` after successful pipeline, before state advance
- Replace hardcoded `runPostPipeline` (line 248 of campaign.go)
- Nil PostTaskFunc falls back to bead close (preserves current behavior)

## Test Cases

| Input | Expected Output | Notes |
|-------|-----------------|-------|
| PostTaskFunc set, task succeeds | PostTaskFunc called with correct beadID | Called after success |
| PostTaskFunc set, task fails | PostTaskFunc not called | Only on success path |
| PostTaskFunc set, recursive child (feature/epic) | PostTaskFunc not called for recursive entry | Recursion handles its own lifecycle |
| PostTaskFunc nil, task succeeds | beads.Close(beadID) called (fallback) | Preserves existing behavior |
| PostTaskFunc returns error | Task marked failed, error propagated | Treated as task failure |
| PostTaskFunc returns nil | State advances to next task | Normal progression |

## Edge Cases
- [ ] PostTaskFunc called with correct beadID (not parent ID)
- [ ] PostTaskFunc error does not trigger circuit breaker (or does — decide)
- [ ] PostTaskFunc not called when campaign is aborted mid-task (ctx cancelled)
- [ ] PostTaskFunc not called when pipeline is paused (ErrPipelinePaused)
- [ ] Multiple tasks: PostTaskFunc called once per successful task

## Implementation Notes
- Add `PostTaskFunc func(beadID string) error` field to `Config` struct in campaign.go
- Replace `r.runPostPipeline(task.BeadID)` with: if PostTaskFunc != nil, call it; else fall back to `r.beads.Close(beadID)`
- PostTaskFunc error should set task.Status = TaskFailed and call OnTaskFail
- Test with mock PostTaskFunc that records calls and optionally returns errors
