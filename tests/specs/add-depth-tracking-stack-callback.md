# Test Specification: Add depth tracking and stack to campaign callback

## Bead: cap-9f0.4.1

## Tracer
Foundation — proves callback distinguishes root vs nested campaigns.

## Context
- Add `depth int` and `stack []parentState` to `dashboardCampaignCallback`
- Route messages differently based on depth: depth 0 sends `CampaignStartMsg`, depth >0 sends `SubCampaignStartMsg`
- Stack tracks parent state (taskIndex, taskTotal) so it can be restored after subcampaign completes

## Test Cases

| Input | Expected Output | Notes |
|-------|-----------------|-------|
| OnCampaignStart at depth 0 | CampaignStartMsg sent via statusFn | Root campaign |
| OnCampaignStart at depth 1 | SubCampaignStartMsg sent via statusFn | Nested campaign |
| OnCampaignStart pushes state | Stack contains parent taskIndex/taskTotal | State preserved |
| OnCampaignComplete pops state | Stack shrinks, parent taskIndex/taskTotal restored | State restored |
| OnTaskStart at depth 0 | CampaignTaskStartMsg with root indices | Correct routing |
| OnTaskStart at depth 1 | CampaignTaskStartMsg with nested indices | Correct routing |
| Depth restores after subcampaign | depth == 0 after subcampaign completes | Clean teardown |

## Edge Cases
- [ ] Stack empty at OnCampaignComplete depth 0 (no pop, no panic)
- [ ] Nested callback with 0 children (OnCampaignStart then immediate OnCampaignComplete)
- [ ] OnTaskFail at depth 1 routes correctly
- [ ] Maximum nesting depth (epic -> feature -> task = depth 2)
- [ ] taskIndex resets to 0 at each new depth level

## Implementation Notes
- Add `depth int` field to `dashboardCampaignCallback`
- Add `stack []callbackFrame` where `callbackFrame` holds `{taskIndex, taskTotal, parentID}`
- `OnCampaignStart`: if depth == 0, send CampaignStartMsg; else push frame, send SubCampaignStartMsg; increment depth
- `OnCampaignComplete`: decrement depth; if depth > 0, pop frame and restore taskIndex/taskTotal; send appropriate done msg
- OnTaskStart/OnTaskComplete/OnTaskFail: use current depth's taskIndex/taskTotal
- This struct is only called from the campaign runner goroutine — no mutex needed
