# Feature Acceptance: cap-kxw.3

## Polish, shared lifecycle, and edge case handling

**Status:** Accepted
**Date:** 2026-02-14
**Parent Epic:** cap-kxw (Task Dashboard TUI)

## What Was Requested

Users can rely on the dashboard to handle edge cases gracefully (resize, abort, empty lists, missing tools) and share post-pipeline lifecycle with capsule run so that pipelines dispatched from the dashboard complete the same merge/cleanup/close workflow.

## Acceptance Criteria

| # | Criterion | Evidence | Verified |
|---|-----------|----------|----------|
| 1 | Given a successful pipeline, when returning to browse, then post-pipeline lifecycle runs in the background | `TestSummary_ReturnToBrowseFiresPostPipeline`, `TestSummary_ReturnToBrowseWithoutPostPipelineProducesRefreshOnly`, `TestSummary_PostPipelineDoneMsgHandledSilently`, `TestModel_AbortDoesNotRunPostPipeline` | Yes |
| 2 | Given a terminal resize, when the window size changes, then both panes re-layout proportionally | `TestModel_WindowResizeUpdatesLayout`, `TestModel_ResizeRecalculatesViewportDimensions`, `TestModel_ResizeSmallTerminalClampsMinLeft`, `TestPaneWidths_Normal`, `TestPaneWidths_MinLeft`, `TestPaneWidths_VerySmall`, `TestPaneWidths_Zero` | Yes |
| 3 | Given Ctrl+C during pipeline, when abort triggers, then cleanup runs and dashboard returns to browse | `TestModel_PipelineCtrlCCancels`, `TestModel_AbortSetsAbortingFlag`, `TestModel_AbortChannelClosedTransitionsToBrowse`, `TestModel_AbortDoesNotRunPostPipeline`, `TestModel_AbortViewShowsAbortingIndicator` | Yes |
| 4 | Given bd is not installed, when dashboard launches, then a clear error message is shown | `DashboardCmd.Run()` guard via `exec.LookPath("bd")`, `TestSmoke_DashboardTTY` (binary-level error path) | Yes |
| 5 | Given an empty bead list, when dashboard renders, then 'No ready beads' message with refresh hint is shown | `TestBrowse_EmptyList` | Yes |
| 6 | Given a pipeline abort is in progress, when I press Ctrl+C again, then the dashboard exits immediately | `TestModel_DoublePressCtrlCForceQuits`, `TestModel_DoublePressQForceQuits` | Yes |

## How to Verify

```bash
go test -v ./internal/dashboard/...   # All dashboard tests
make smoke                            # Smoke tests including dashboard binary test
```

## Out of Scope

- Pipeline narrative summary agent (separate feature cap-i25)
- Multi-CLI provider support (separate epic cap-10s)

## Known Limitations

- Post-pipeline lifecycle status is not displayed to the user (runs silently in background)
- The `bd not installed` check uses `exec.LookPath` which requires PATH to be correct at runtime
