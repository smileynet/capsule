# Feature Acceptance: cap-fj8.3

## Campaign completed task inspection

**Status:** Accepted
**Date:** 2026-02-16
**Parent Epic:** cap-fj8 (Dashboard TUI Enhancements)

## What Was Requested

Developers can now navigate to completed tasks within a campaign and expand them to view their phase reports, enabling review of what happened during each task's pipeline run.

## Acceptance Criteria

| # | Criterion | Evidence | Verified |
|---|-----------|----------|----------|
| 1 | In campaign mode, cursor can navigate to completed tasks | `TestCampaign_SelectedIdx_DownKey`, `TestCampaign_SelectedIdx_UpKey`, `TestCampaign_SelectedIdx_JKey`, `TestCampaign_SelectedIdx_KKey`, `TestCampaign_View_CursorMarkerOnSelected` | Yes |
| 2 | Selecting a completed task expands its phases (progressive disclosure) | `TestCampaign_View_SelectedCompletedTaskShowsPhases`, `TestCampaign_View_UnselectedCompletedTaskNoPhases`, `TestCampaign_View_OnlySelectedTaskExpanded` | Yes |
| 3 | Right pane shows phase reports for the selected completed task | `TestCampaign_ViewReport_SelectedCompletedTask`, `TestModel_CampaignViewRightShowsPhaseReport` | Yes |
| 4 | Phase results stored from CampaignTaskDoneMsg for later inspection | `TestCampaign_TaskDoneMsg_StoresPhaseReports`, `TestCampaign_TaskDoneMsg_MultipleTasksStoreIndependently`, `TestCampaignTaskDoneMsg_PhaseReports` | Yes |

## How to Verify

```bash
# Run all tests (lint + full suite including shell tests)
make lint && make test-full

# Run campaign inspection-specific tests
go test ./internal/dashboard/ -v -run "Campaign.*(SelectedIdx|SelectedCompleted|ViewReport|TaskDoneMsg|OnlySelected)"
go test ./internal/dashboard/ -v -run "CampaignTaskDoneMsg"
```

## Out of Scope

- Filtering or searching completed tasks within a campaign
- Persisting inspection state across dashboard restarts
- Exporting phase reports from the dashboard

## Known Limitations

- No test for ViewReport when cursor is on a completed task with zero stored phase reports (recommended improvement, non-blocking)
- No dedicated smoke test for campaign inspection (covered at TUI model integration level via `TestModel_CampaignFullFlow`)
