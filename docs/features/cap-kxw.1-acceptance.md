# Feature Acceptance: cap-kxw.1

## Dashboard shell with two-pane layout and bead browsing

**Status:** Accepted
**Date:** 2026-02-13
**Parent Epic:** cap-kxw (Task Dashboard TUI)

## What Was Requested

A `capsule dashboard` command that presents a two-pane interactive TUI for browsing ready beads with full context, including bead list navigation on the left and resolved detail on the right.

## Acceptance Criteria

| # | Criterion | Evidence | Verified |
|---|-----------|----------|----------|
| 1 | Given a TTY, when `capsule dashboard` runs, then a loading spinner appears followed by a two-pane layout with rounded borders | `TestModel_DefaultMode`, `TestBrowse_LoadingState`, `TestFocusedBorder_DoesNotPanic`, `TestUnfocusedBorder_DoesNotPanic` | Yes |
| 2 | Given the two-pane layout, when beads load, then the left pane shows ready beads with ID, title, priority badge (P0-P4), and type | `TestBrowse_BeadsLoadedView`, `TestBrowse_PriorityBadgesInView`, `TestBrowse_TypeInView` | Yes |
| 3 | Given a selected bead, when the cursor moves, then the right pane updates with resolved bead detail (hierarchy, description, acceptance criteria) | `TestModel_CursorMoveTriggersResolve`, `TestModel_BeadResolvedMsgUpdatesCache`, `TestModel_ViewRightShowsDetail`, `TestFormatBeadDetail_ContainsAllFields` | Yes |
| 4 | Given the layout, when I press Tab, then focus switches between panes (border color changes) and arrow keys control the focused pane | `TestModel_TabTogglesFocus`, `TestModel_RightPaneScrollKeys` | Yes |
| 5 | Given the bead list, when I press 'r', then the list and detail cache refresh from bd ready | `TestBrowse_RefreshReloads`, `TestModel_RefreshInvalidatesCache` | Yes |
| 6 | Given the bead list, when I press 'q', then the dashboard exits cleanly | `TestModel_QuitInBrowseMode`, `TestModel_CtrlCQuits` | Yes |
| 7 | Given no TTY, when `capsule dashboard` runs, then an error prints saying dashboard requires a terminal | `TestFeature_DashboardCommand/run_returns_error_when_not_a_TTY` | Yes |

## How to Verify

```bash
go test -v ./internal/dashboard/...   # Unit tests (69 tests)
go test -v ./cmd/capsule/ -run Dashboard  # CLI integration tests (4 tests)
```

## Out of Scope

- Pipeline mode with phase list and report browsing (cap-kxw.2)
- Polish, shared lifecycle, and edge case handling (cap-kxw.3)

## Known Limitations

- Border color change on focus is verified via state toggle, not visual output assertion (lipgloss rendering varies by terminal)
- No binary-level smoke test for the dashboard command (non-TTY path verified at unit level)
