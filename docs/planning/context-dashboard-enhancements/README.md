# Context: Dashboard TUI Enhancements

**Status:** archived
**Created:** 2026-02-15
**Updated:** 2026-02-15
**Epic:** cap-fj8

## Overview

Dashboard TUI enhancements adding smart dispatch (type-aware routing for
features/epics → campaign mode), run inspection (history toggle, archived
worklogs/summaries), and UX polish (elapsed timers, debounced resolution,
sticky cursor, post-pipeline feedback).

## Epic Structure

| # | Title | Features | Tasks |
|---|-------|----------|-------|
| cap-fj8 | Dashboard TUI Enhancements | 4 | 15 |

**Feature → Bead mapping:**

| Feature | Bead ID | Tasks | Priority |
|---------|---------|-------|----------|
| Campaign mode in dashboard TUI | cap-fj8.1 | 5 | P2 |
| History view with archived pipeline results | cap-fj8.2 | 4 | P2 |
| Campaign completed task inspection | cap-fj8.3 | 2 | P3 |
| Pipeline context and responsiveness improvements | cap-fj8.4 | 4 | P3 |

## Dependency Graph

```
cap-fj8.1 (Campaign mode)
├── blocks → cap-fj8.2 (History view)
└── blocks → cap-fj8.3 (Campaign task inspection)
    └── cap-fj8.2 + cap-fj8.3 block → cap-fj8.4 (UX polish)
```

Sequential: 1 → 2 + 3 (parallel) → 4

## Key Decisions

- Inline phase nesting for campaign view (avoids mode explosion)
- History as toggle (`h` key), not separate mode
- Progressive disclosure: phases only for running/selected task
- 50 closed bead limit for history (performance)
- Abort entire campaign on q/ctrl+c (simpler; user re-runs to continue)
- Debounced resolution with instant cache bypass
- Post-pipeline status line replaces silent failure swallowing

## Test Specifications

- 4 BDD `.feature` files in `tests/features/` (f-7.1 through f-7.4)
- 13 TDD spec `.md` files in `tests/specs/` (t-7.1.1 through t-7.4.4)

## Key Artifacts

- [Menu Plan](../menu-plan-dashboard-enhancements.yaml) — work breakdown
- [Brainstorm](embedded in plan file) — UX research and user walkthrough
- [BDD Specs](../../tests/features/) — f-7.x feature files
- [TDD Specs](../../tests/specs/) — t-7.x.y task specs
