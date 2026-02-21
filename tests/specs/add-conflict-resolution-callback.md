# Test Specification: Add conflict resolution callback

## Bead: cap-9f0.2.1

## Tracer
Conflict detection — proves PostTaskFunc can detect and route merge conflicts.

## Context
- Add `ConflictResolver func(beadID string, err error) error` to `campaign.Config`
- In PostTaskFunc wiring, detect `worktree.ErrMergeConflict` and call ConflictResolver
- If ConflictResolver succeeds, retry the merge
- If ConflictResolver fails or is nil, propagate the error normally

## Test Cases

| Input | Expected Output | Notes |
|-------|-----------------|-------|
| Merge returns ErrMergeConflict, ConflictResolver set | ConflictResolver called with beadID and error | Conflict detected and routed |
| ConflictResolver returns nil | Merge retried after resolution | Successful resolution |
| ConflictResolver returns error | Campaign pauses with ErrCampaignPaused | Failed resolution |
| ConflictResolver nil, ErrMergeConflict | Error propagated normally (no panic) | Nil-safe |
| Merge succeeds (no conflict) | ConflictResolver not called | Only on conflict |
| Non-merge error from PostTaskFunc | ConflictResolver not called | Only for ErrMergeConflict |

## Edge Cases
- [ ] ConflictResolver called with correct beadID (not parent)
- [ ] Retry after resolution uses same merge parameters
- [ ] ConflictResolver timeout (long-running manual resolution)
- [ ] Multiple consecutive conflicts across tasks
- [ ] ConflictResolver called from campaign goroutine (thread safety)

## Implementation Notes
- ConflictResolver lives on Config, not on Runner — injected at construction time
- Detection: `errors.Is(err, worktree.ErrMergeConflict)` in PostTaskFunc closure
- On successful resolution (ConflictResolver returns nil), retry `wt.MergeToMain` once
- On failed resolution, return wrapped error so campaign runner can pause
- CLI adapter: ConflictResolver could print instructions and return error (non-interactive)
- Dashboard adapter: ConflictResolver could emit a CampaignPausedMsg
