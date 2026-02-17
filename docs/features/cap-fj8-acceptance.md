# Epic Acceptance: cap-fj8

## Dashboard TUI Enhancements

**Status:** Accepted
**Date:** 2026-02-16

## What This Delivers

Developers using `capsule dashboard` now have a fully-featured TUI with smart dispatch (features/epics run as campaigns), run history inspection, completed task drill-down, and responsive UX polish including elapsed timers, debounced bead resolution, sticky cursor, and post-pipeline status feedback.

## Features Accepted

| # | Feature | Acceptance Report |
|---|---------|-------------------|
| 1 | cap-fj8.1: Campaign mode in dashboard TUI | [report](cap-fj8.1-acceptance.md) |
| 2 | cap-fj8.2: History view with archived pipeline results | Plated by loop (serve timed out) |
| 3 | cap-fj8.3: Campaign completed task inspection | [report](cap-fj8.3-acceptance.md) |
| 4 | cap-fj8.4: Pipeline context and responsiveness improvements | Validated via quality gates |
| 5 | cap-fj8.5: Add null byte check to validateBeadID | Bug fix, tested |
| 6 | cap-fj8.6: Copy readyBeads slice to prevent aliasing | Bug fix, tested |
| 7 | cap-fj8.7: Handle priority badge ANSI codes in mutedText | Bug fix, tested |

## End-to-End Verification

All quality gates pass:

```bash
make lint          # 0 issues
make test-full     # All Go tests pass (13 packages), all shell tests pass (230+ tests)
make smoke         # Smoke tests pass (102.9s)
```

Cross-feature integration validated at unit level:
- Phase reports flow from pipeline through campaign bridge to task inspection (299 dashboard tests)
- History toggle preserves/restores browse state with defensive slice copies
- Campaign dispatch routes feature/epic beads correctly, falls back to pipeline for tasks
- Bead header, elapsed timer, debounced resolution, and sticky cursor work across modes

## Out of Scope

- Parallel task execution within campaigns
- Campaign state persistence across dashboard restarts
- Grouped browse list with section headers
- Full `?` help overlay

## Known Limitations

- No process-level E2E test for dashboard TUI features (unit/integration coverage is thorough; filed as cap-0f0)
- cap-fj8.2.3 (browse history toggle) required 3+ attempts and was completed via alternative approach in cap-fj8.2.4
