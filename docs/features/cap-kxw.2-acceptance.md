# Feature Acceptance: cap-kxw.2

## Pipeline mode with phase list and report browsing

**Status:** Accepted
**Date:** 2026-02-13
**Parent Epic:** cap-kxw (Task Dashboard TUI)

## What Was Requested

Users can dispatch a pipeline from the dashboard and see phase progress in the left pane with detailed phase reports in the right pane, providing clear visibility into what each phase produced.

## Acceptance Criteria

| # | Criterion | Evidence | Verified |
|---|-----------|----------|----------|
| 1 | Given Enter pressed on a bead, when pipeline dispatches, then left pane switches to phase list with status indicators (spinner, checkmark, cross) | `TestBrowse_EnterDispatchesSelectedBead`, `TestModel_DispatchWithRunnerTransitions`, `TestPipeline_ViewPassedPhase`, `TestPipeline_ViewFailedPhase`, `TestPipeline_ViewPendingPhases`, `TestPipeline_SpinnerTick` | Yes |
| 2 | Given a completed phase, when its report is available, then selecting it in the left pane shows summary, files changed, and duration in the right pane | `TestPipeline_ReportStoredOnPass`, `TestPipeline_ReportStoredOnFail`, `TestPipeline_ViewReportPassed`, `TestPipeline_ViewReportFailed`, `TestModel_PipelineRightPaneShowsReport` | Yes |
| 3 | Given a running pipeline, when phases complete, then the cursor auto-follows the running phase | `TestPipeline_AutoFollowTracksRunningPhase`, `TestPipeline_MultiplePhaseProgression` | Yes |
| 4 | Given the user presses up/down during pipeline, then auto-follow disables and the user can browse completed phase reports | `TestPipeline_CursorDisablesAutoFollow`, `TestPipeline_AutoFollowDisabledDoesNotTrack`, `TestPipeline_CursorDown`, `TestPipeline_CursorUp`, `TestPipeline_VimKeys` | Yes |
| 5 | Given pipeline completion, when all phases finish, then the right pane shows overall result summary | `TestSummary_RightPaneShowsPassSummary`, `TestSummary_RightPaneShowsFailSummary`, `TestSummary_LeftPaneShowsFrozenPhases`, `TestModel_ChannelClosedTransitionsToSummary`, `TestModel_PipelineFullFlow` | Yes |
| 6 | Given a running pipeline, when I press q/Ctrl+C, then the pipeline aborts gracefully | `TestModel_PipelineQuitCancels`, `TestModel_PipelineCtrlCCancels`, `TestModel_PipelineQuitDoesNotQuitInPipelineMode` | Yes |

## How to Verify

```bash
go test -v ./internal/dashboard/...   # All dashboard tests (103 tests)
```

## Out of Scope

- Post-pipeline lifecycle (merge, cleanup, close bead) — deferred to cap-kxw.3.1
- Error states (loading spinners, failed loads) — deferred to cap-kxw.3.2
- Edge cases (terminal resize, double abort) — deferred to cap-kxw.3.3

## Known Limitations

- Summary mode does not show post-pipeline status (merge/cleanup) — requires shared lifecycle from cap-kxw.3.1
- No binary-level smoke test for pipeline mode (filed as cap-kxw.3.5)
- Tests lack Given-When-Then structural comments (filed as cap-kxw.3.4)
