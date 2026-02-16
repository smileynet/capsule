# Feature Acceptance: cap-fj8.1

## Campaign mode in dashboard TUI

**Status:** Accepted
**Date:** 2026-02-16
**Parent Epic:** cap-fj8 (Dashboard TUI Enhancements)

## What Was Requested

Developers can now select a feature or epic bead in the dashboard and have all its ready child tasks run sequentially as a campaign, with live progress and summary.

## Acceptance Criteria

| # | Criterion | Evidence | Verified |
|---|-----------|----------|----------|
| 1 | Selecting a task bead runs a single pipeline (unchanged behavior) | `TestModel_DispatchRoutesToModeByType` (task → ModePipeline) | Yes |
| 2 | Selecting a feature bead discovers its ready child tasks and runs them sequentially | `TestModel_DispatchFeatureGoesToCampaign`, `TestRun_HappyPath`, `TestModel_CampaignFullFlow` | Yes |
| 3 | Selecting an epic bead discovers its ready child tasks and runs them sequentially | `TestModel_DispatchRoutesToModeByType` (epic → ModeCampaign) | Yes |
| 4 | Campaign view shows task queue with inline phase nesting for the running task | `TestCampaign_View_RunningTaskShowsPhases`, `TestModel_CampaignViewLeftShowsCampaignState` | Yes |
| 5 | Completed tasks show as single line with pass/fail indicator and duration | `TestCampaign_View_CompletedTaskCollapsed`, `TestCampaign_View_FailedTaskCollapsed` | Yes |
| 6 | Right pane shows phase report for cursor-selected phase (same as pipeline mode) | `TestCampaign_ViewReport_DelegatesToPipeline`, `TestModel_CampaignViewRightShowsPhaseReport` | Yes |
| 7 | User can abort campaign with q/ctrl+c | `TestModel_CampaignQuitCancels`, `TestModel_CampaignDoublePressQForceQuits` | Yes |
| 8 | Campaign summary shows after all tasks complete | `TestModel_CampaignChannelClosedTransitionsToSummary`, `TestModel_CampaignSummaryViewRightShowsSummary` | Yes |
| 9 | Bead type with empty/unknown value defaults to single pipeline (safe fallback) | `TestModel_DispatchRoutesToModeByType` (empty → ModePipeline) | Yes |

## How to Verify

```bash
# Run all tests (lint + full suite including shell tests)
make lint && make test-full

# Run campaign-specific tests
go test ./internal/campaign/ -v
go test ./internal/dashboard/ -v -run "Campaign|Dispatch"
```

## Out of Scope

- Campaign state persistence across dashboard restarts (deferred to cap-fj8.3)
- History view with archived pipeline results (deferred to cap-fj8.2)
- Inspecting completed task details within a finished campaign (deferred to cap-fj8.3)
- Parallel task execution within a campaign

## Known Limitations

- Context cancellation at the campaign runner layer is tested only at the TUI level, not at the runner unit test level (recommended improvement, non-blocking)
- No dedicated smoke test for campaign mode (existing smoke tests cover dashboard and pipeline modes separately)
