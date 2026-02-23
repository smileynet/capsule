# Feature Acceptance: cap-9f0.4

## Chef's Selection

**Feature:** Depth-aware campaign callback adapter

**User Story:**
As a developer dispatching an epic from the dashboard, I want to see the epic's features as the top-level campaign view with tasks nested under the running feature, so that I can track progress at every level of the hierarchy.

## Tasting Notes

### Acceptance Criterion 1: Epic dispatch shows features as top-level rows
**Status:** ✓ Verified

**Evidence:**
- `dashboardCampaignCallback` tracks depth with stack-based state management
- Depth 0 (epic level) sends `CampaignStartMsg` to display features as top-level rows
- Test: `TestDashboardCampaignCallback_NestedCampaigns` verifies epic start sends `CampaignStartMsg`

### Acceptance Criterion 2: Feature tasks appear nested below feature row
**Status:** ✓ Verified

**Evidence:**
- Depth 1+ sends `SubCampaignStartMsg` with parent context
- `campaignState` creates `subcampaignState` overlay to track nested tasks
- `campaignState.View` renders subcampaign tasks with indentation (2 levels)
- Test: `TestCampaign_SubCampaignStartMsg_CreatesSubcampaignState` verifies subcampaign creation

### Acceptance Criterion 3: Pipeline phases animate under active task
**Status:** ✓ Verified

**Evidence:**
- `subcampaignState` includes `pipelineState` for active task
- Task messages route to subcampaign when active (depth-aware routing)
- View renders pipeline phases at 3-indent level under active task
- Tests: `TestCampaign_CampaignTaskStartMsg_RoutesToSubcampaign`, `TestCampaign_CampaignTaskDoneMsg_RoutesToSubcampaign`

### Acceptance Criterion 4: Feature completion collapses nested tasks
**Status:** ✓ Verified

**Evidence:**
- `SubCampaignDoneMsg` clears `subcampaignState` on feature completion
- Feature row shows checkmark + duration after completion
- Test: `TestCampaign_SubCampaignDoneMsg_ClearsSubcampaign` verifies cleanup

### Acceptance Criterion 5: Background mode handles nested messages
**Status:** ✓ Verified

**Evidence:**
- `Model.Update` routes `SubCampaignStartMsg` and `SubCampaignDoneMsg` in both `ModeCampaign` and background mode
- Tests: `TestModel_SubCampaignStartMsgRoutesInBackground`, `TestModel_SubCampaignDoneMsgRoutesInBackground`

### Acceptance Criterion 6: CLI adapter logs subcampaign events with indentation
**Status:** ✓ Verified

**Evidence:**
- `campaignPlainTextCallback` updated with depth-aware formatting
- Subcampaign tasks logged with extra indentation
- Implementation in `cmd/capsule/main.go` (CLI adapter)

## Quality Checks

### BDD Tests
**Status:** ✓ All passing

**Test Coverage:**
- `TestCampaign_SubCampaignStartMsg_CreatesSubcampaignState` - Subcampaign creation
- `TestCampaign_SubCampaignDoneMsg_ClearsSubcampaign` - Subcampaign cleanup
- `TestCampaign_CampaignTaskStartMsg_RoutesToSubcampaign` - Message routing (start)
- `TestCampaign_CampaignTaskDoneMsg_RoutesToSubcampaign` - Message routing (done)
- `TestModel_SubCampaignStartMsgRoutes` - Model routing (foreground)
- `TestModel_SubCampaignDoneMsgRoutes` - Model routing (foreground)
- `TestModel_SubCampaignStartMsgRoutesInBackground` - Model routing (background)
- `TestModel_SubCampaignDoneMsgRoutesInBackground` - Model routing (background)
- `TestDashboardCampaignCallback_NestedCampaigns` - End-to-end callback flow

**Test Results:**
```
go test ./internal/dashboard -run 'TestCampaign_SubCampaign|TestModel_SubCampaign'
PASS (0.013s)

go test ./cmd/capsule -run 'TestDashboardCampaignCallback_NestedCampaigns'
PASS (0.005s)
```

### Full Test Suite
**Status:** ✓ All passing

```
go test ./...
ok (all packages)
```

## Kitchen Staff Sign-Off

- **Line Cook (Implementation):** ✓ All tasks completed
- **Sous Chef (Code Review):** ✓ Approved (via TDD pipeline)
- **Maitre (BDD Quality):** ✓ Tests cover all acceptance criteria with Given-When-Then structure

## Guest Experience

### Using Nested Campaign View

**Dispatching an Epic:**
```bash
capsule campaign cap-9f0  # Epic with features
```

**What You'll See:**
1. Epic features displayed as top-level rows
2. When a feature starts, its tasks appear nested below (indented)
3. Active task shows pipeline phases underneath (further indented)
4. Completed tasks show ✓ + duration, collapsed
5. When feature completes, nested tasks collapse back to feature row

**Visual Hierarchy:**
```
Epic: cap-9f0
  ⟳ Feature: cap-9f0.4 (running)
      ⟳ Task: cap-9f0.4.1 (running)
          ⟳ Phase: test-writer
          ⟳ Phase: execute
      ✓ Task: cap-9f0.4.2 (1.2s)
      ○ Task: cap-9f0.4.3 (pending)
```

### CLI Output

Campaign progress logged with depth-aware indentation:
```
[21:00:00] Epic: cap-9f0
[21:00:01]   Feature: cap-9f0.4
[21:00:02]     Task: cap-9f0.4.1 — Add depth tracking
[21:00:15]     ✓ Task: cap-9f0.4.1 (13s)
```

## Kitchen Notes

### Implementation Details

**Components Modified:**
- `cmd/capsule/main.go` - Added depth tracking to `dashboardCampaignCallback`
- `internal/dashboard/msg.go` - Added `SubCampaignStartMsg`, `SubCampaignDoneMsg`
- `internal/dashboard/campaign.go` - Added `subcampaignState` overlay
- `internal/dashboard/model.go` - Routed new message types

**Design Pattern:**
Stack-based depth tracking with overlay state management. Parent campaign state remains intact while subcampaign overlays nested tasks.

### Limitations

- Maximum nesting depth: 3 levels (epic → feature → task)
- No support for task-level subcampaigns (tasks are leaf nodes)
- Indentation fixed at 2 spaces per level

### Future Enhancements

- Configurable indentation levels
- Collapsible feature sections in TUI
- Subcampaign progress percentage in parent row

### Deployment Notes

No configuration changes required. Feature activates automatically when dispatching epics with nested features.

## Related Orders

**Completed Tasks:**
- cap-9f0.4.1 - Add depth tracking and stack to dashboardCampaignCallback
- cap-9f0.4.2 - Add SubCampaignStartMsg and SubCampaignDoneMsg message types
- cap-9f0.4.3 - Add subcampaign overlay state to campaignState
- cap-9f0.4.4 - Render nested task rows under running feature in campaign view
- cap-9f0.4.5 - Update CLI adapter for nested campaign output

**Parent Epic:**
- cap-9f0 - Campaign UX: Nested Campaigns, Worktree Lifecycle, Error Handling

**Related Features:**
- cap-9f0.1 - Post-pipeline merge and cleanup (worktree lifecycle)
- cap-9f0.2 - Error propagation in campaign execution
