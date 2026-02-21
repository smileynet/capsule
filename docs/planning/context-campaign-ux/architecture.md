# Campaign UX — Architecture Notes

## Key Constraints

1. **Campaign package doesn't know about worktrees** — uses `PipelineRunner` interface.
   Worktree lifecycle must be injected, not embedded.

2. **Dashboard state is value-typed** — Bubble Tea pattern. `campaignState` is a struct,
   not a pointer. Updates return new copies via `Update()`.

3. **Callback interface is shared** — `campaign.Callback` serves both CLI plain text
   and dashboard TUI adapters. Changes affect both.

4. **Event channel is the bridge** — campaign goroutine → `chan tea.Msg` (buffered 16)
   → dashboard model. All state changes flow through messages.

5. **Background mode** — user can Esc to browse while campaign runs. State survives
   mode transition via `m.backgroundMode` + continued `listenForEvents`.

6. **Max recursion depth is 3** — epic → feature → task. Only one nesting level
   needed for subcampaign overlay.

## Message Flow

```
campaign.Runner goroutine
  → callback.OnXxx()
  → dashboardCampaignCallback sends tea.Msg via statusFn
  → chan tea.Msg (buffered 16)
  → listenForEvents() reads one msg per tick
  → Model.Update(msg) dispatches to campaignState
```

## Files That Will Change

| File | Change |
|------|--------|
| `internal/campaign/campaign.go` | Add `PostTaskFunc` to Config/Runner, call after task success, log save errors |
| `cmd/capsule/main.go` | Wire `PostTaskFunc` with `postPipeline` logic, add error to callback adapter |
| `internal/dashboard/msg.go` | New `SubCampaignStartMsg`, `SubCampaignDoneMsg`, add `Error` to `CampaignTaskDoneMsg` |
| `internal/dashboard/campaign.go` | Subcampaign overlay state, error display in right pane |
| `internal/dashboard/model.go` | Handle new message types, route to campaign state |

## Patterns to Preserve

- `postPipeline()` is best-effort (warnings, not errors)
- Campaign resume from persisted state must still work after PostTaskFunc changes
- CLI plain text callback gets same events (may log subcampaign info differently)
- Value-typed state copies in Bubble Tea Update methods
