# Test Specification: Add subcampaign message types

## Bead: cap-9f0.4.2

## Tracer
Message types — proves new events flow through the Bubble Tea event loop.

## Context
- Add `SubCampaignStartMsg` and `SubCampaignDoneMsg` to dashboard/msg.go
- Route in `Model.Update` to campaignState
- These messages carry nested campaign lifecycle without replacing the parent campaign state

## Test Cases

| Input | Expected Output | Notes |
|-------|-----------------|-------|
| SubCampaignStartMsg with parentID and tasks | Message routed to campaignState.Update | Event delivered |
| SubCampaignDoneMsg with parentID and counts | Message routed to campaignState.Update | Event delivered |
| SubCampaignStartMsg in ModeCampaign | campaignState receives and processes msg | Active campaign mode |
| SubCampaignStartMsg in background mode | campaignState receives msg (campaign runs in background) | Background compatible |
| SubCampaignStartMsg fields | ParentID, Tasks populated | Struct complete |
| SubCampaignDoneMsg fields | ParentID, Passed, Failed populated | Struct complete |

## Edge Cases
- [ ] SubCampaignStartMsg when no campaign is active (ignored safely)
- [ ] SubCampaignDoneMsg when no subcampaign is active (ignored safely)
- [ ] SubCampaignStartMsg followed by CampaignTaskStartMsg (correct nesting order)
- [ ] Multiple SubCampaignStartMsg in sequence (epic with multiple features)
- [ ] SubCampaignDoneMsg followed by parent CampaignTaskDoneMsg (correct unwinding)

## Implementation Notes
- Add to msg.go:
  - `SubCampaignStartMsg{ParentID string, Tasks []CampaignTaskInfo}`
  - `SubCampaignDoneMsg{ParentID string, Passed int, Failed int, Skipped int}`
- In Model.Update: route SubCampaignStartMsg and SubCampaignDoneMsg to campaignState.Update (same pattern as CampaignTaskStartMsg)
- campaignState.Update: add cases for SubCampaignStartMsg and SubCampaignDoneMsg
- These messages are sent by dashboardCampaignCallback when depth > 0
