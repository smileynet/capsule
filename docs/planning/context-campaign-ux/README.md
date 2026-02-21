# Campaign UX — Planning Context

**Status:** finalized
**Created:** 2026-02-21
**Brainstorm:** `docs/planning/brainstorm-campaign-ux.md`
**Input:** `docs/handoff-campaign-ux.md`

## Problem

Running epic-level campaigns in the dashboard is broken: nested `CampaignStartMsg`
replaces the parent view, worktrees accumulate without merge/cleanup, and errors
are silently discarded.

## Approach

Three independent features addressing data integrity, visual correctness, and
observability:

1. **Worktree lifecycle** — Inject `PostTaskFunc` into campaign runner to run
   merge/cleanup after each task. Minimal API change, matches existing
   architecture constraint (campaigns don't know about worktrees).

2. **Nested campaign animation** — Add depth-aware callback stack in the
   dashboard adapter. New `SubCampaignStartMsg`/`SubCampaignDoneMsg` messages
   overlay feature tasks under the running epic row without replacing parent state.

3. **Error observability** — Replace silent `_ =` discards with logged warnings.
   Add error text to `CampaignTaskDoneMsg`. Show error detail in dashboard
   right pane for failed tasks.

## Key Decisions

| ID | Decision | Rationale |
|----|----------|-----------|
| D1 | Subcampaign overlay, not tree state | Max depth is 3; flat overlay is simpler than tree for one nesting level |
| D2 | PostTaskFunc injection, not callback method | Callbacks are notification-only; PostTaskFunc returns error for flow control |
| D3 | Worktree lifecycle first | Data integrity before visual polish |
| D4 | Observability wishlist deferred | Separate epic; core fixes are independent |
| D5 | Agent pair resolves merge conflicts first | Invoke execute → sign-off pair with conflict context; pause + toast if unresolvable |
| D6 | All behavior driven by agent pairs | When items are missed, invoke the contextually appropriate pair, don't patch around |
| D7 | Non-blocking notifications for campaign issues | Status bar / toast, never modal; user can navigate to detail in right pane |

## Scope

**Menu plan:** `docs/planning/menu-plan-campaign-ux.yaml`

- 2 phases, 4 features, 14 tasks
- Phase 1: Campaign Data Integrity (3 features, 9 tasks)
  - Feature 1.1: Worktree merge/cleanup per campaign task (3 tasks)
  - Feature 1.2: Agent-driven merge conflict resolution (3 tasks)
  - Feature 1.3: Campaign error observability (3 tasks)
- Phase 2: Nested Campaign Animation (1 feature, 5 tasks)
  - Feature 2.1: Depth-aware campaign callback adapter (5 tasks)

**Epic bead:** cap-9f0
**BDD specs:** 4 feature files in tests/features/
**TDD specs:** 10 spec files in tests/specs/
**Ready tasks:** cap-9f0.1.1, cap-9f0.3.1, cap-9f0.4.1
