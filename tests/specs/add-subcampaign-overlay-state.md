# Test Specification: Add subcampaign overlay state

## Bead: cap-9f0.4.3

## Tracer
State management — proves nested task state coexists with parent campaign state.

## Context
- Add `subcampaign *subcampaignState` field to `campaignState`
- Handle `SubCampaignStartMsg` by creating a `subcampaignState` with nested tasks
- Handle `SubCampaignDoneMsg` by clearing the subcampaign overlay
- Route `CampaignTaskStartMsg` and `CampaignTaskDoneMsg` to subcampaign when active

## Test Cases

| Input | Expected Output | Notes |
|-------|-----------------|-------|
| SubCampaignStartMsg received | subcampaignState created with nested tasks | Overlay initialized |
| SubCampaignDoneMsg received | subcampaign set to nil | Overlay cleared |
| CampaignTaskStartMsg while subcampaign active | Routed to subcampaignState | Nested routing |
| CampaignTaskDoneMsg while subcampaign active | Routed to subcampaignState | Nested routing |
| Parent state during active subcampaign | Parent tasks/statuses unchanged | Parent preserved |
| Subcampaign nil when no subcampaign active | No nested state allocated | Clean default |

## Edge Cases
- [ ] PhaseUpdateMsg routes to subcampaign's pipeline when subcampaign active
- [ ] Parent selectedIdx preserved during subcampaign
- [ ] View renders nested tasks indented under running parent task
- [ ] Subcampaign completed/failed counts roll up into parent task status
- [ ] SubCampaignDoneMsg with all tasks failed marks parent task as failed
- [ ] SubCampaignStartMsg when subcampaign already active (replace, not stack)

## Implementation Notes
- Add `subcampaignState` struct (similar shape to campaignState but simpler: tasks, statuses, currentIdx, pipeline)
- Add `subcampaign *subcampaignState` pointer field to `campaignState` — nil means no active subcampaign
- In `campaignState.Update`: check if subcampaign is non-nil for CampaignTaskStartMsg/DoneMsg routing
- SubCampaignStartMsg handler: create subcampaignState, store on campaignState
- SubCampaignDoneMsg handler: set subcampaign = nil
- View rendering: when subcampaign active and parent task is running, render subcampaign tasks indented below
- subcampaignState only needs one level — max recursion depth is 3 (epic -> feature -> task)
