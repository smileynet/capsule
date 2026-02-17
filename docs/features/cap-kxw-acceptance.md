# Epic Acceptance: cap-kxw

## Task Dashboard TUI

**Status:** Accepted
**Date:** 2026-02-14

## What This Delivers

Users can run `capsule dashboard` to get a rich interactive terminal interface for browsing ready beads, dispatching pipelines, watching phase progress in real time, and viewing results — all in a continuous loop without leaving the terminal. The dashboard provides a two-pane layout with bead list navigation on the left and context-sensitive detail on the right, replacing the workflow of manually running `bd ready` followed by `capsule run`.

## Features Accepted

| # | Feature | Acceptance Report |
|---|---------|-------------------|
| 1 | cap-kxw.1: Dashboard shell with two-pane layout and bead browsing | [report](cap-kxw.1-acceptance.md) |
| 2 | cap-kxw.2: Pipeline mode with phase list and report browsing | [report](cap-kxw.2-acceptance.md) |
| 3 | cap-kxw.3: Polish, shared lifecycle, and edge case handling | [report](cap-kxw.3-acceptance.md) |

## End-to-End Verification

**Journey 1: Browse and inspect beads**
Launch dashboard, see loading spinner, bead list populates with ID/priority/title/type. Navigate with arrow/vim keys, right pane updates with resolved detail (hierarchy, description, acceptance criteria). Tab switches focus between panes. Press 'r' to refresh, 'q' to quit.

**Journey 2: Dispatch pipeline and watch progress**
Press Enter on a bead, left pane switches to phase list with status indicators (spinner/checkmark/cross). Phases auto-follow the running phase. Press up/down to browse completed phase reports in the right pane. On completion, summary mode shows pass/fail result with timing. Press any key to return to browse with cache refresh.

**Journey 3: Abort and recover**
During pipeline execution, press q or Ctrl+C to abort. Dashboard shows aborting indicator, cleanup runs, and returns to browse mode. Post-pipeline lifecycle does not run on abort. Double-press Ctrl+C force quits immediately.

**Journey 4: Edge cases**
Terminal resize re-layouts both panes proportionally. Empty bead list shows "No ready beads" with refresh hint. Missing bd shows clear error message. Non-TTY environment rejects with helpful error.

```bash
make lint                                     # 0 issues
make test-full                                # All packages pass
make smoke                                    # Smoke tests pass (20.7s)
go test -v ./internal/dashboard/...           # 150 dashboard tests
go test -v ./cmd/capsule/ -run Dashboard      # CLI integration tests
```

## Out of Scope

- Pipeline narrative summary agent (separate feature)
- Multi-CLI provider support (separate epic cap-10s)
- Post-pipeline lifecycle status display (runs silently in background)

## Known Limitations

- No positive-path smoke test for the dashboard TUI itself (Bubble Tea's interactive nature requires PTY automation infrastructure; negative-path TTY rejection is tested at binary level, and 150 unit/component tests cover all user journeys programmatically)
- Border color change on focus is verified via state toggle, not visual output assertion (lipgloss rendering varies by terminal)
- `bd not installed` check uses `exec.LookPath` which depends on PATH at runtime
