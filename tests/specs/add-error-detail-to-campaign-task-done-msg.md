# Test Specification: Add error detail to CampaignTaskDoneMsg

## Bead: cap-9f0.3.2

## Tracer
Error propagation — proves error text reaches the dashboard.

## Context
- Add `Error string` field to `CampaignTaskDoneMsg` in dashboard/msg.go
- Wire `dashboardCampaignCallback.OnTaskFail` to populate the Error field
- Store error per task in `campaignState` for display in the right pane

## Test Cases

| Input | Expected Output | Notes |
|-------|-----------------|-------|
| OnTaskFail called with error | CampaignTaskDoneMsg.Error populated with error text | Error carried through |
| OnTaskComplete called (success) | CampaignTaskDoneMsg.Error is empty string | No error on success |
| campaignState receives task done with error | Error stored by bead ID | Accessible later |
| Error stored, task selected in UI | Right pane shows error text | User sees detail |
| Error with multi-line text | Full text preserved in Error field | No truncation |

## Edge Cases
- [ ] Error text with special characters (newlines, quotes)
- [ ] Error preserved across campaignState updates (value-typed struct copies)
- [ ] Multiple failed tasks each store their own error
- [ ] Error accessible by bead ID (not just by index)
- [ ] Very long error text (pipeline feedback can be verbose)

## Implementation Notes
- Add `Error string` to `CampaignTaskDoneMsg` struct in msg.go
- In `dashboardCampaignCallback.OnTaskFail`: set `Error: err.Error()` on the CampaignTaskDoneMsg
- Add `taskErrors map[string]string` to `campaignState` (keyed by bead ID)
- In `handleTaskDone`: if `msg.Error != ""`, store in `taskErrors[msg.BeadID]`
- In `ViewReport`: for failed tasks, append error text below phase reports
- In `formatTaskReport`: add error section with failure indicator and error message
