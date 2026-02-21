# Test Specification: Wire PostTaskFunc in CLI campaign command

## Bead: cap-9f0.1.2

## Tracer
CLI wiring — proves existing merge/cleanup runs per campaign task.

## Context
- In `CampaignCmd.Run` (main.go), construct a `PostTaskFunc` closure calling `postPipeline`
- Pass via `campaign.Config.PostTaskFunc`
- Currently `CampaignCmd.Run` does not call `postPipeline` at all — merge/cleanup only happens for `RunCmd`
- The closure needs access to `worktree.Manager` and `beadResolver` from the enclosing scope

## Test Cases

| Input | Expected Output | Notes |
|-------|-----------------|-------|
| PostTaskFunc closure constructed | Closure calls postPipeline with correct beadID | Wiring correct |
| Campaign runs 3 tasks, all pass | postPipeline called 3 times (once per task) | Per-task lifecycle |
| Task 1 succeeds, merge completes | Worktree removed before task 2 starts | Clean slate for next task |
| Task 1 succeeds, merge conflicts | ErrMergeConflict returned from PostTaskFunc | Error propagates to campaign |
| Campaign with 0 tasks | PostTaskFunc never called | No tasks = no lifecycle |

## Edge Cases
- [ ] PostTaskFunc closure captures correct worktree manager (not stale reference)
- [ ] Worktree cleanup runs even if bead close fails (best-effort)
- [ ] PostTaskFunc error includes enough context for campaign runner to log
- [ ] postPipeline writes warnings to io.Discard (CLI campaign is not interactive)

## Implementation Notes
- Build PostTaskFunc closure in CampaignCmd.Run after creating wtMgr and bdClient
- Closure signature: `func(beadID string) error`
- Closure body calls `postPipeline(io.Discard, beadID, wtMgr, bdClient)` and returns nil (postPipeline is best-effort today)
- To return merge conflict errors, postPipeline needs refactoring to return error — or PostTaskFunc calls merge/cleanup directly
- Set `campaignCfg.PostTaskFunc = ppFunc` before passing to `campaign.NewRunner`
