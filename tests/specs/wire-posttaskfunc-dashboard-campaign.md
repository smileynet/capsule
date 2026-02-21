# Test Specification: Wire PostTaskFunc in dashboard campaign adapter

## Bead: cap-9f0.1.3

## Tracer
Dashboard wiring — proves merge/cleanup runs when dispatched from TUI.

## Context
- In `dashboardCampaignAdapter.RunCampaign` (main.go), pass `PostTaskFunc` through `campaign.Config`
- The dashboard adapter already has access to worktree manager and bead client via its struct fields
- Currently the dashboard adapter's `RunCampaign` creates a `campaign.NewRunner` with `a.campaignCfg` — PostTaskFunc must be set on that config

## Test Cases

| Input | Expected Output | Notes |
|-------|-----------------|-------|
| PostTaskFunc wired in dashboardCampaignAdapter | PostTaskFunc present in Config passed to NewRunner | Wiring correct |
| Dashboard dispatches campaign, task succeeds | Merge/cleanup runs via PostTaskFunc | Same lifecycle as CLI |
| Dashboard dispatches campaign, 2 tasks pass | PostTaskFunc called twice | Per-task cleanup |
| PostTaskFunc returns nil | Campaign advances to next task | Normal flow |
| PostTaskFunc returns error | CampaignTaskDoneMsg with Success=false | Error surfaced to TUI |

## Edge Cases
- [ ] PostTaskFunc wired identically to CLI (same merge/cleanup sequence)
- [ ] PostTaskFunc runs in campaign goroutine, not main goroutine (thread safety)
- [ ] Dashboard ppFunc already exists for single-pipeline post-pipeline — reuse pattern
- [ ] Worktree manager shared between pipeline and campaign adapters

## Implementation Notes
- `dashboardCampaignAdapter` already has `campaignCfg campaign.Config` — set PostTaskFunc on it
- The DashboardCmd already builds a `ppFunc` for single-pipeline dispatch (line 475) — reuse that pattern
- PostTaskFunc closure: `func(beadID string) error { postPipeline(io.Discard, beadID, wtMgr, bdClient); return nil }`
- Ensure PostTaskFunc is set before passing campaignCfg to campaign.NewRunner in RunCampaign
