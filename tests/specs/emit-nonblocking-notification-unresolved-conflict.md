# Test Specification: Emit non-blocking notification for unresolved conflicts

## Bead: cap-9f0.2.3

## Tracer
User notification — proves unresolved conflicts are visible without blocking.

## Context
- Add `CampaignPausedMsg` with reason and conflict details to dashboard msg.go
- Dashboard renders: toast notification, status bar update, right pane with resolution instructions
- CLI adapter prints conflict details to stderr
- Campaign pauses but does not crash or lose state

## Test Cases

| Input | Expected Output | Notes |
|-------|-----------------|-------|
| Unresolved merge conflict | CampaignPausedMsg emitted with reason "merge_conflict" | Message created |
| CampaignPausedMsg received in dashboard | Toast notification rendered | User sees notification |
| CampaignPausedMsg received in dashboard | Status bar shows pause reason | Persistent indicator |
| CampaignPausedMsg with conflict details | Right pane shows resolution instructions | Actionable guidance |
| CampaignPausedMsg in CLI mode | Conflict details printed to stderr | CLI fallback |
| CampaignPausedMsg with beadID | Instructions include branch name (capsule-<beadID>) | Correct branch reference |

## Edge Cases
- [ ] Toast auto-clears after timeout but status bar persists
- [ ] Right pane content updates when paused task is selected
- [ ] Multiple pauses (resume then pause again) — state accumulates correctly
- [ ] CampaignPausedMsg in background mode (user has Esc'd to browse) — toast still shown
- [ ] CampaignPausedMsg fields: Reason, BeadID, MainBranch, Instructions

## Implementation Notes
- Add CampaignPausedMsg struct to internal/dashboard/msg.go with fields: Reason string, BeadID string, Detail string
- Handle CampaignPausedMsg in campaignState.Update — set a paused flag and store detail
- Status bar rendering: check paused flag, show "Paused: <reason>" in status line
- Right pane: when paused task selected, show resolution instructions (git checkout, git merge, capsule clean)
- CLI: campaignPlainTextCallback prints to stderr in OnTaskFail when error wraps ErrMergeConflict
