# Feature Acceptance: cap-9f0.1

## Chef's Selection

**Feature:** Worktree merge/cleanup per campaign task

**User Story:**
As a developer running a multi-task campaign, I want each task's changes merged to main before the next task starts so that tasks build on each other's work and worktrees don't accumulate.

## Tasting Notes

| Acceptance Criterion | Verification Evidence |
|---------------------|----------------------|
| Each successful task's worktree is merged to main, removed, and pruned before the next task starts | `TestRun_PostTaskFuncCalledAfterSuccess` verifies PostTaskFunc called for each successful task. Manual verification: campaign runs merge and cleanup between tasks. |
| Campaign tasks branch from updated main (containing all prior task merges) | Integration behavior: PostTaskFunc merges to main before next task starts, ensuring sequential task execution builds on prior work. |
| PostTaskFunc is injectable — campaign package does not import worktree package | `campaign.Config.PostTaskFunc` field added. No import of worktree package in campaign code. Dependency injection verified. |
| CLI campaign command (capsule campaign) wires PostTaskFunc with existing merge/cleanup logic | `TestFeature_CampaignPostTaskFunc` verifies CampaignCmd constructs PostTaskFunc closure calling postPipeline with wtMgr and bdClient. |
| Dashboard campaign dispatch wires PostTaskFunc identically | Dashboard command wires PostTaskFunc using same postPipeline closure pattern as CLI. |

## Quality Checks

### Test Results
- All unit tests passing: `go test ./...` ✓
- BDD tests present:
  - `TestFeature_CampaignPostTaskFunc` (CLI wiring)
  - `TestRun_PostTaskFuncCalledAfterSuccess` (campaign execution)
  - `TestRun_PostTaskFuncNotCalledOnFailure` (error handling)

### BDD Test Quality
- Given-When-Then structure: ✓
- Tests map to acceptance criteria: ✓
- User perspective documented: ✓
- Error scenarios included: ✓ (PostTaskFunc not called on failure)
- Real system operations: ✓ (tests exercise actual merge/cleanup flow)

## Kitchen Staff Sign-Off

- **Sous-chef:** Code review complete (all tasks closed)
- **Line cook:** Feature validation complete
- **Maitre:** BDD test quality verified

## Guest Experience

### How to Use

When running a multi-task campaign:

```bash
capsule campaign cap-feature
```

Each task now automatically:
1. Executes the pipeline
2. Merges changes to main
3. Cleans up the worktree
4. Prunes stale branches

The next task branches from the updated main, building on prior work.

### Implementation Details

The feature uses dependency injection:
- `campaign.Config.PostTaskFunc` accepts a callback
- CLI and dashboard wire the callback to `postPipeline()`
- `postPipeline()` calls merge and cleanup operations
- Campaign runner invokes PostTaskFunc after each successful task

## Kitchen Notes

### Limitations
- PostTaskFunc errors are currently logged but don't fail the campaign
- Manual cleanup still required if campaign is interrupted

### Future Enhancements
- Consider making PostTaskFunc errors configurable (fail vs. warn)
- Add retry logic for transient merge failures

### Deployment Notes
- No breaking changes
- Backward compatible (PostTaskFunc is optional)
- No configuration changes required

## Related Orders

### Completed Tasks
- cap-9f0.1.1: Add PostTaskFunc to campaign.Config and call after successful task
- cap-9f0.1.2: Wire PostTaskFunc in CLI campaign command
- cap-9f0.1.3: Wire PostTaskFunc in dashboard campaign dispatch
- cap-9f0.1.5: Fix error swallowing in DashboardCmd PostTaskFunc closure

### Related Features
- Parent epic: cap-9f0 (Campaign mode)
