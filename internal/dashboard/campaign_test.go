package dashboard

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func sampleCampaignTasks() []CampaignTaskInfo {
	return []CampaignTaskInfo{
		{BeadID: "cap-001", Title: "First task", Priority: 1},
		{BeadID: "cap-002", Title: "Second task", Priority: 2},
		{BeadID: "cap-003", Title: "Third task", Priority: 3},
	}
}

// --- Constructor tests ---

func TestCampaign_NewState(t *testing.T) {
	// Given: a campaign start message with parent and tasks
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())

	// Then: state is initialized correctly
	if cs.parentID != "cap-feat" {
		t.Errorf("parentID = %q, want %q", cs.parentID, "cap-feat")
	}
	if cs.parentTitle != "Feature Title" {
		t.Errorf("parentTitle = %q, want %q", cs.parentTitle, "Feature Title")
	}
	if len(cs.tasks) != 3 {
		t.Errorf("len(tasks) = %d, want 3", len(cs.tasks))
	}
	if cs.currentIdx != -1 {
		t.Errorf("currentIdx = %d, want -1 (no task running)", cs.currentIdx)
	}
	if cs.completed != 0 {
		t.Errorf("completed = %d, want 0", cs.completed)
	}
	if cs.failed != 0 {
		t.Errorf("failed = %d, want 0", cs.failed)
	}
}

// --- Update routing tests ---

func TestCampaign_TaskStartMsg(t *testing.T) {
	// Given: a campaign state with tasks
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())

	// When: first task starts
	cs, _ = cs.Update(CampaignTaskStartMsg{BeadID: "cap-001", Index: 0, Total: 3})

	// Then: currentIdx is set and pipeline state is fresh
	if cs.currentIdx != 0 {
		t.Errorf("currentIdx = %d, want 0", cs.currentIdx)
	}
	if cs.taskStatuses[0] != CampaignTaskRunning {
		t.Errorf("taskStatuses[0] = %q, want %q", cs.taskStatuses[0], CampaignTaskRunning)
	}
}

func TestCampaign_TaskStartMsg_ResetsPipeline(t *testing.T) {
	// Given: a campaign state with first task running and some pipeline progress
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())
	cs, _ = cs.Update(CampaignTaskStartMsg{BeadID: "cap-001", Index: 0, Total: 3})
	cs, _ = cs.Update(PhaseUpdateMsg{Phase: "plan", Status: PhaseRunning})

	// When: second task starts
	cs, _ = cs.Update(CampaignTaskStartMsg{BeadID: "cap-002", Index: 1, Total: 3})

	// Then: pipeline state is reset (no phases from previous task)
	if cs.currentIdx != 1 {
		t.Errorf("currentIdx = %d, want 1", cs.currentIdx)
	}
	if len(cs.pipeline.phases) != 0 {
		t.Errorf("pipeline should be reset, but has %d phases", len(cs.pipeline.phases))
	}
}

func TestCampaign_TaskDoneMsg_Success(t *testing.T) {
	// Given: a campaign with first task running
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())
	cs, _ = cs.Update(CampaignTaskStartMsg{BeadID: "cap-001", Index: 0, Total: 3})

	// When: first task completes successfully
	cs, _ = cs.Update(CampaignTaskDoneMsg{BeadID: "cap-001", Index: 0, Success: true, Duration: 5 * time.Second})

	// Then: completed counter increments and status is passed
	if cs.completed != 1 {
		t.Errorf("completed = %d, want 1", cs.completed)
	}
	if cs.taskStatuses[0] != CampaignTaskPassed {
		t.Errorf("taskStatuses[0] = %q, want %q", cs.taskStatuses[0], CampaignTaskPassed)
	}
	if cs.taskDurations[0] != 5*time.Second {
		t.Errorf("taskDurations[0] = %v, want 5s", cs.taskDurations[0])
	}
}

func TestCampaign_TaskDoneMsg_StoresPhaseReports(t *testing.T) {
	// Given: a campaign with first task running
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())
	cs, _ = cs.Update(CampaignTaskStartMsg{BeadID: "cap-001", Index: 0, Total: 3})

	// When: first task completes with phase reports
	reports := []PhaseReport{
		{PhaseName: "plan", Status: PhasePassed, Summary: "planned", Duration: 2 * time.Second},
		{PhaseName: "code", Status: PhasePassed, Summary: "coded", FilesChanged: []string{"main.go"}, Duration: 3 * time.Second},
	}
	cs, _ = cs.Update(CampaignTaskDoneMsg{
		BeadID:       "cap-001",
		Index:        0,
		Success:      true,
		Duration:     5 * time.Second,
		PhaseReports: reports,
	})

	// Then: phase reports are stored for the task
	stored := cs.taskReports["cap-001"]
	if len(stored) != 2 {
		t.Fatalf("taskReports[cap-001] len = %d, want 2", len(stored))
	}
	if stored[0].PhaseName != "plan" {
		t.Errorf("stored[0].PhaseName = %q, want %q", stored[0].PhaseName, "plan")
	}
	if stored[1].Summary != "coded" {
		t.Errorf("stored[1].Summary = %q, want %q", stored[1].Summary, "coded")
	}
}

func TestCampaign_TaskDoneMsg_EmptyPhaseReports(t *testing.T) {
	// Given: a campaign with first task running
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())
	cs, _ = cs.Update(CampaignTaskStartMsg{BeadID: "cap-001", Index: 0, Total: 3})

	// When: task completes with no phase reports
	cs, _ = cs.Update(CampaignTaskDoneMsg{
		BeadID:   "cap-001",
		Index:    0,
		Success:  true,
		Duration: 1 * time.Second,
	})

	// Then: no panic and no reports stored for task
	stored := cs.taskReports["cap-001"]
	if len(stored) != 0 {
		t.Errorf("taskReports[cap-001] should be empty, got %d", len(stored))
	}
}

func TestCampaign_TaskDoneMsg_MultipleTasksStoreIndependently(t *testing.T) {
	// Given: a campaign with multiple tasks
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())

	// When: first task completes with reports
	cs, _ = cs.Update(CampaignTaskStartMsg{BeadID: "cap-001", Index: 0, Total: 3})
	cs, _ = cs.Update(CampaignTaskDoneMsg{
		BeadID:       "cap-001",
		Index:        0,
		Success:      true,
		Duration:     2 * time.Second,
		PhaseReports: []PhaseReport{{PhaseName: "plan", Status: PhasePassed}},
	})

	// And: second task completes with different reports
	cs, _ = cs.Update(CampaignTaskStartMsg{BeadID: "cap-002", Index: 1, Total: 3})
	cs, _ = cs.Update(CampaignTaskDoneMsg{
		BeadID:       "cap-002",
		Index:        1,
		Success:      true,
		Duration:     3 * time.Second,
		PhaseReports: []PhaseReport{{PhaseName: "code", Status: PhasePassed}, {PhaseName: "test", Status: PhasePassed}},
	})

	// Then: each task's reports are stored independently
	if len(cs.taskReports["cap-001"]) != 1 {
		t.Errorf("taskReports[cap-001] len = %d, want 1", len(cs.taskReports["cap-001"]))
	}
	if len(cs.taskReports["cap-002"]) != 2 {
		t.Errorf("taskReports[cap-002] len = %d, want 2", len(cs.taskReports["cap-002"]))
	}
}

func TestCampaign_TaskDoneMsg_UnknownIndexIgnored(t *testing.T) {
	// Given: a campaign with tasks
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())

	// When: a done message arrives with an out-of-range index
	cs, _ = cs.Update(CampaignTaskDoneMsg{
		BeadID:       "cap-unknown",
		Index:        99,
		Success:      true,
		Duration:     1 * time.Second,
		PhaseReports: []PhaseReport{{PhaseName: "plan", Status: PhasePassed}},
	})

	// Then: no panic, reports still stored by bead ID
	stored := cs.taskReports["cap-unknown"]
	if len(stored) != 1 {
		t.Errorf("taskReports[cap-unknown] len = %d, want 1", len(stored))
	}
}

func TestCampaign_TaskDoneMsg_Failure(t *testing.T) {
	// Given: a campaign with first task running
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())
	cs, _ = cs.Update(CampaignTaskStartMsg{BeadID: "cap-001", Index: 0, Total: 3})

	// When: first task fails
	cs, _ = cs.Update(CampaignTaskDoneMsg{BeadID: "cap-001", Index: 0, Success: false, Duration: 3 * time.Second})

	// Then: failed counter increments and status is failed
	if cs.failed != 1 {
		t.Errorf("failed = %d, want 1", cs.failed)
	}
	if cs.taskStatuses[0] != CampaignTaskFailed {
		t.Errorf("taskStatuses[0] = %q, want %q", cs.taskStatuses[0], CampaignTaskFailed)
	}
}

func TestCampaign_PhaseUpdateMsg_ForwardsToEmbeddedPipeline(t *testing.T) {
	// Given: a campaign with first task running and pipeline initialized
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())
	cs, _ = cs.Update(CampaignTaskStartMsg{BeadID: "cap-001", Index: 0, Total: 3})
	// Simulate phase names being set (via the pipeline state)
	cs.pipeline = newPipelineState(samplePhaseNames())

	// When: a phase update arrives
	cs, _ = cs.Update(PhaseUpdateMsg{Phase: "plan", Status: PhaseRunning})

	// Then: embedded pipeline state is updated
	if !cs.pipeline.running {
		t.Error("embedded pipeline should be running")
	}
	if cs.pipeline.phases[0].Status != PhaseRunning {
		t.Errorf("pipeline phase 0 status = %q, want %q", cs.pipeline.phases[0].Status, PhaseRunning)
	}
}

func TestCampaign_SpinnerTick_ProducesCmd(t *testing.T) {
	// Given: a campaign state with a spinner
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())

	// When: a spinner tick is processed
	tickMsg := cs.pipeline.spinner.Tick()
	_, cmd := cs.Update(tickMsg)

	// Then: a follow-up command is produced
	if cmd == nil {
		t.Error("spinner tick should produce a follow-up command")
	}
}

func TestCampaign_UnknownMsg_NoOp(t *testing.T) {
	// Given: a campaign state
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())

	// When: an unknown message type is processed
	cs2, cmd := cs.Update(BeadListMsg{})

	// Then: state is unchanged and no command is produced
	if cmd != nil {
		t.Error("unknown msg should produce nil command")
	}
	if cs2.parentID != cs.parentID {
		t.Error("state should be unchanged for unknown msg")
	}
}

// --- View tests ---

func TestCampaign_View_AllPending(t *testing.T) {
	// Given: a campaign state with all tasks pending (no task started)
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())

	// When: the view is rendered
	view := cs.View(60, 20)
	plain := stripANSI(view)

	// Then: header shows parent info and all tasks are listed as pending
	if !strings.Contains(plain, "cap-feat") {
		t.Errorf("view should contain parent ID, got:\n%s", plain)
	}
	if !strings.Contains(plain, "Feature Title") {
		t.Errorf("view should contain parent title, got:\n%s", plain)
	}
	if !strings.Contains(plain, "First task") {
		t.Errorf("view should contain task title, got:\n%s", plain)
	}
	if !strings.Contains(plain, "○") {
		t.Errorf("pending tasks should show ○ indicator, got:\n%s", plain)
	}
}

func TestCampaign_View_RunningTaskShowsPhases(t *testing.T) {
	// Given: a campaign with first task running and phases populated
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())
	cs, _ = cs.Update(CampaignTaskStartMsg{BeadID: "cap-001", Index: 0, Total: 3})
	cs.pipeline = newPipelineState([]string{"plan", "code"})
	cs, _ = cs.Update(PhaseUpdateMsg{Phase: "plan", Status: PhaseRunning})

	// When: the view is rendered
	view := cs.View(60, 20)
	plain := stripANSI(view)

	// Then: the running task shows indented phase names
	if !strings.Contains(plain, "plan") {
		t.Errorf("running task should show phase 'plan', got:\n%s", plain)
	}
	if !strings.Contains(plain, "code") {
		t.Errorf("running task should show phase 'code', got:\n%s", plain)
	}
}

func TestCampaign_View_CompletedTaskCollapsed(t *testing.T) {
	// Given: a campaign with first task passed
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())
	cs, _ = cs.Update(CampaignTaskStartMsg{BeadID: "cap-001", Index: 0, Total: 3})
	cs, _ = cs.Update(CampaignTaskDoneMsg{BeadID: "cap-001", Index: 0, Success: true, Duration: 5 * time.Second})

	// When: the view is rendered
	view := cs.View(60, 20)
	plain := stripANSI(view)

	// Then: completed task shows checkmark and is on one line (collapsed)
	if !strings.Contains(plain, "✓") {
		t.Errorf("completed task should show ✓, got:\n%s", plain)
	}
	if !strings.Contains(plain, "5.0s") {
		t.Errorf("completed task should show duration, got:\n%s", plain)
	}
}

func TestCampaign_View_FailedTaskCollapsed(t *testing.T) {
	// Given: a campaign with first task failed
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())
	cs, _ = cs.Update(CampaignTaskStartMsg{BeadID: "cap-001", Index: 0, Total: 3})
	cs, _ = cs.Update(CampaignTaskDoneMsg{BeadID: "cap-001", Index: 0, Success: false, Duration: 3 * time.Second})

	// When: the view is rendered
	view := cs.View(60, 20)
	plain := stripANSI(view)

	// Then: failed task shows cross indicator
	if !strings.Contains(plain, "✗") {
		t.Errorf("failed task should show ✗, got:\n%s", plain)
	}
}

func TestCampaign_View_ProgressCounter(t *testing.T) {
	// Given: a campaign with one task done, one running
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())
	cs, _ = cs.Update(CampaignTaskStartMsg{BeadID: "cap-001", Index: 0, Total: 3})
	cs, _ = cs.Update(CampaignTaskDoneMsg{BeadID: "cap-001", Index: 0, Success: true, Duration: 2 * time.Second})
	cs, _ = cs.Update(CampaignTaskStartMsg{BeadID: "cap-002", Index: 1, Total: 3})

	// When: the view is rendered
	view := cs.View(60, 20)
	plain := stripANSI(view)

	// Then: progress counter shows 1/3
	if !strings.Contains(plain, "1/3") {
		t.Errorf("view should show progress '1/3', got:\n%s", plain)
	}
}

func TestCampaign_View_EmptyTasks(t *testing.T) {
	// Given: a campaign state with no tasks
	cs := newCampaignState("cap-feat", "Feature Title", nil)

	// When: the view is rendered
	view := cs.View(60, 20)
	plain := stripANSI(view)

	// Then: "No tasks" is shown
	if !strings.Contains(plain, "No tasks") {
		t.Errorf("empty tasks should show 'No tasks', got:\n%s", plain)
	}
}

// --- ViewReport tests ---

func TestCampaign_ViewReport_DelegatesToPipeline(t *testing.T) {
	// Given: a campaign with a running task and phases
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())
	cs, _ = cs.Update(CampaignTaskStartMsg{BeadID: "cap-001", Index: 0, Total: 3})
	cs.pipeline = newPipelineState(samplePhaseNames())
	cs, _ = cs.Update(PhaseUpdateMsg{Phase: "plan", Status: PhaseRunning})

	// When: ViewReport is called
	view := cs.ViewReport(60, 20)
	plain := stripANSI(view)

	// Then: it delegates to pipelineState.ViewReport() and shows running content
	if !strings.Contains(plain, "plan") {
		t.Errorf("ViewReport should delegate to pipeline, got:\n%s", plain)
	}
	if !strings.Contains(plain, "Running") {
		t.Errorf("ViewReport should show running state, got:\n%s", plain)
	}
}

func TestCampaign_ViewReport_NoRunningTask(t *testing.T) {
	// Given: a campaign state with no running task
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())

	// When: ViewReport is called
	view := cs.ViewReport(60, 20)

	// Then: empty string or fallback is returned
	if view != "" {
		t.Errorf("ViewReport with no running task should return empty, got: %q", view)
	}
}

// --- selectedIdx (cursor) tests ---

func TestCampaign_SelectedIdx_DefaultsToZero(t *testing.T) {
	// Given: a fresh campaign state
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())

	// Then: selectedIdx starts at 0
	if cs.selectedIdx != 0 {
		t.Errorf("selectedIdx = %d, want 0", cs.selectedIdx)
	}
}

func TestCampaign_SelectedIdx_DownKey(t *testing.T) {
	// Given: a campaign state with tasks
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())

	// When: down key is pressed
	cs, _ = cs.Update(tea.KeyMsg{Type: tea.KeyDown})

	// Then: selectedIdx moves to 1
	if cs.selectedIdx != 1 {
		t.Errorf("selectedIdx = %d, want 1", cs.selectedIdx)
	}
}

func TestCampaign_SelectedIdx_UpKey(t *testing.T) {
	// Given: a campaign state with selectedIdx at 1
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())
	cs, _ = cs.Update(tea.KeyMsg{Type: tea.KeyDown})

	// When: up key is pressed
	cs, _ = cs.Update(tea.KeyMsg{Type: tea.KeyUp})

	// Then: selectedIdx moves back to 0
	if cs.selectedIdx != 0 {
		t.Errorf("selectedIdx = %d, want 0", cs.selectedIdx)
	}
}

func TestCampaign_SelectedIdx_WrapsDown(t *testing.T) {
	// Given: a campaign at the last task
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())
	cs, _ = cs.Update(tea.KeyMsg{Type: tea.KeyDown}) // 1
	cs, _ = cs.Update(tea.KeyMsg{Type: tea.KeyDown}) // 2

	// When: down again (past end)
	cs, _ = cs.Update(tea.KeyMsg{Type: tea.KeyDown})

	// Then: wraps to 0
	if cs.selectedIdx != 0 {
		t.Errorf("selectedIdx = %d, want 0 (wrap)", cs.selectedIdx)
	}
}

func TestCampaign_SelectedIdx_WrapsUp(t *testing.T) {
	// Given: a campaign at selectedIdx 0
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())

	// When: up key (past start)
	cs, _ = cs.Update(tea.KeyMsg{Type: tea.KeyUp})

	// Then: wraps to last task
	if cs.selectedIdx != 2 {
		t.Errorf("selectedIdx = %d, want 2 (wrap)", cs.selectedIdx)
	}
}

func TestCampaign_SelectedIdx_JKey(t *testing.T) {
	// Given: a campaign state
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())

	// When: j key is pressed
	cs, _ = cs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	// Then: selectedIdx moves down
	if cs.selectedIdx != 1 {
		t.Errorf("selectedIdx = %d, want 1", cs.selectedIdx)
	}
}

func TestCampaign_SelectedIdx_EmptyTasksNoOp(t *testing.T) {
	// Given: a campaign with no tasks
	cs := newCampaignState("cap-feat", "Feature Title", nil)

	// When: down/up keys are pressed
	cs, _ = cs.Update(tea.KeyMsg{Type: tea.KeyDown})
	cs, _ = cs.Update(tea.KeyMsg{Type: tea.KeyUp})

	// Then: no panic, selectedIdx stays at 0
	if cs.selectedIdx != 0 {
		t.Errorf("selectedIdx = %d, want 0 (empty task list)", cs.selectedIdx)
	}
}

func TestCampaign_SelectedIdx_KKey(t *testing.T) {
	// Given: a campaign at selectedIdx 1
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())
	cs, _ = cs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	// When: k key is pressed
	cs, _ = cs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})

	// Then: selectedIdx moves up
	if cs.selectedIdx != 0 {
		t.Errorf("selectedIdx = %d, want 0", cs.selectedIdx)
	}
}

// --- View: cursor marker on selected task ---

func TestCampaign_View_CursorMarkerOnSelected(t *testing.T) {
	// Given: a campaign with selectedIdx at 1
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())
	cs, _ = cs.Update(tea.KeyMsg{Type: tea.KeyDown})

	// When: the view is rendered
	view := cs.View(60, 20)
	plain := stripANSI(view)

	// Then: cursor marker appears on "Second task" line
	lines := strings.Split(plain, "\n")
	found := false
	for _, line := range lines {
		if strings.Contains(line, CursorMarker) && strings.Contains(line, "Second task") {
			found = true
		}
	}
	if !found {
		t.Errorf("cursor marker should be on 'Second task', got:\n%s", plain)
	}
}

// --- View: completed task expansion ---

func TestCampaign_View_SelectedCompletedTaskShowsPhases(t *testing.T) {
	// Given: task 0 completed with phase reports, selectedIdx on task 0
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())
	cs, _ = cs.Update(CampaignTaskStartMsg{BeadID: "cap-001", Index: 0, Total: 3})
	cs, _ = cs.Update(CampaignTaskDoneMsg{
		BeadID:   "cap-001",
		Index:    0,
		Success:  true,
		Duration: 5 * time.Second,
		PhaseReports: []PhaseReport{
			{PhaseName: "plan", Status: PhasePassed, Duration: 2 * time.Second},
			{PhaseName: "code", Status: PhasePassed, Duration: 3 * time.Second},
		},
	})
	// selectedIdx is 0 (default), which is the completed task

	// When: the view is rendered
	view := cs.View(60, 20)
	plain := stripANSI(view)

	// Then: phase names appear expanded below the selected completed task
	if !strings.Contains(plain, "plan") {
		t.Errorf("selected completed task should expand phases, 'plan' missing:\n%s", plain)
	}
	if !strings.Contains(plain, "code") {
		t.Errorf("selected completed task should expand phases, 'code' missing:\n%s", plain)
	}
}

func TestCampaign_View_UnselectedCompletedTaskNoPhases(t *testing.T) {
	// Given: task 0 completed with phases, but selectedIdx on task 1
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())
	cs, _ = cs.Update(CampaignTaskStartMsg{BeadID: "cap-001", Index: 0, Total: 3})
	cs, _ = cs.Update(CampaignTaskDoneMsg{
		BeadID:   "cap-001",
		Index:    0,
		Success:  true,
		Duration: 5 * time.Second,
		PhaseReports: []PhaseReport{
			{PhaseName: "plan", Status: PhasePassed, Duration: 2 * time.Second},
		},
	})
	// Move selectedIdx to task 1
	cs, _ = cs.Update(tea.KeyMsg{Type: tea.KeyDown})

	// When: the view is rendered
	view := cs.View(60, 20)
	plain := stripANSI(view)

	// Then: no phase expansion (plan should not appear in output
	// since the only task with reports is task 0, and it's not selected)
	lines := strings.Split(plain, "\n")
	for _, line := range lines {
		// Phase lines are deeply indented; task lines are not
		if strings.Contains(line, "      ") && strings.Contains(line, "plan") {
			t.Errorf("unselected completed task should NOT expand phases, got:\n%s", plain)
		}
	}
}

func TestCampaign_View_OnlySelectedTaskExpanded(t *testing.T) {
	// Given: tasks 0 and 1 both completed with phases
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())

	cs, _ = cs.Update(CampaignTaskStartMsg{BeadID: "cap-001", Index: 0, Total: 3})
	cs, _ = cs.Update(CampaignTaskDoneMsg{
		BeadID: "cap-001", Index: 0, Success: true, Duration: 2 * time.Second,
		PhaseReports: []PhaseReport{{PhaseName: "plan", Status: PhasePassed}},
	})
	cs, _ = cs.Update(CampaignTaskStartMsg{BeadID: "cap-002", Index: 1, Total: 3})
	cs, _ = cs.Update(CampaignTaskDoneMsg{
		BeadID: "cap-002", Index: 1, Success: true, Duration: 3 * time.Second,
		PhaseReports: []PhaseReport{{PhaseName: "test", Status: PhasePassed}},
	})

	// selectedIdx on task 1
	cs, _ = cs.Update(tea.KeyMsg{Type: tea.KeyDown})

	// When: the view is rendered
	view := cs.View(60, 20)
	plain := stripANSI(view)

	// Then: only task 1's phases are expanded (test), not task 0's (plan)
	// Count indented phase lines
	lines := strings.Split(plain, "\n")
	expandedPhases := 0
	for _, line := range lines {
		trimmed := strings.TrimLeft(line, " ")
		if len(line)-len(trimmed) >= 6 { // deeply indented = phase line
			expandedPhases++
		}
	}
	if expandedPhases != 1 {
		t.Errorf("only one task should be expanded, got %d phase lines:\n%s", expandedPhases, plain)
	}
}

// --- ViewReport: delegate to stored phase reports for completed task ---

func TestCampaign_ViewReport_SelectedCompletedTask(t *testing.T) {
	// Given: task 0 completed with phase reports, selectedIdx on task 0
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())
	cs, _ = cs.Update(CampaignTaskStartMsg{BeadID: "cap-001", Index: 0, Total: 3})
	cs, _ = cs.Update(CampaignTaskDoneMsg{
		BeadID: "cap-001", Index: 0, Success: true, Duration: 5 * time.Second,
		PhaseReports: []PhaseReport{
			{PhaseName: "plan", Status: PhasePassed, Summary: "All planned"},
			{PhaseName: "code", Status: PhasePassed, Summary: "Code written"},
		},
	})

	// When: ViewReport is called (selectedIdx 0 = completed task with reports)
	view := cs.ViewReport(60, 20)
	plain := stripANSI(view)

	// Then: shows a summary of the completed task's phases
	if !strings.Contains(plain, "plan") {
		t.Errorf("ViewReport should show phase 'plan' for completed task, got:\n%s", plain)
	}
	if !strings.Contains(plain, "code") {
		t.Errorf("ViewReport should show phase 'code' for completed task, got:\n%s", plain)
	}
}

func TestCampaign_ViewReport_SelectedRunningTask(t *testing.T) {
	// Given: task 0 completed, task 1 running with phases, selectedIdx on task 1
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())
	cs, _ = cs.Update(CampaignTaskStartMsg{BeadID: "cap-001", Index: 0, Total: 3})
	cs, _ = cs.Update(CampaignTaskDoneMsg{
		BeadID: "cap-001", Index: 0, Success: true, Duration: 2 * time.Second,
	})
	cs, _ = cs.Update(CampaignTaskStartMsg{BeadID: "cap-002", Index: 1, Total: 3})
	cs.pipeline = newPipelineState([]string{"plan", "code"})
	cs, _ = cs.Update(PhaseUpdateMsg{Phase: "plan", Status: PhaseRunning})

	// Move selectedIdx to task 1 (the running task)
	cs, _ = cs.Update(tea.KeyMsg{Type: tea.KeyDown})

	// When: ViewReport is called
	view := cs.ViewReport(60, 20)
	plain := stripANSI(view)

	// Then: delegates to live pipeline ViewReport (shows running state)
	if !strings.Contains(plain, "Running") {
		t.Errorf("ViewReport for running task should show live pipeline, got:\n%s", plain)
	}
}

func TestCampaign_ViewReport_SelectedPendingTask(t *testing.T) {
	// Given: all tasks pending, selectedIdx on task 0
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())

	// When: ViewReport is called
	view := cs.ViewReport(60, 20)

	// Then: empty string (pending task has no report)
	if view != "" {
		t.Errorf("ViewReport for pending task should be empty, got: %q", view)
	}
}

// --- Validation state tests ---

func TestCampaign_ValidationStart_SetsValidating(t *testing.T) {
	// Given: a campaign state with all tasks done
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())

	// When: validating flag is set (simulating CampaignValidationStartMsg handling)
	cs.validating = true

	// Then: the view shows validation row
	view := cs.View(60, 20)
	plain := stripANSI(view)
	if !strings.Contains(plain, "Feature validation") {
		t.Errorf("view should show validation row, got:\n%s", plain)
	}
}

func TestCampaign_ValidationDone_Success(t *testing.T) {
	// Given: a campaign state with validation completed successfully
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())
	cs.validationResult = &CampaignValidationDoneMsg{
		Success:  true,
		Duration: 3 * time.Second,
	}

	// When: the view is rendered
	view := cs.View(60, 20)
	plain := stripANSI(view)

	// Then: the validation row shows a check and duration
	if !strings.Contains(plain, "Feature validation") {
		t.Errorf("view should show validation result, got:\n%s", plain)
	}
	if !strings.Contains(plain, SymbolCheck) {
		t.Errorf("passed validation should show check, got:\n%s", plain)
	}
}

func TestCampaign_ValidationDone_Failure(t *testing.T) {
	// Given: a campaign state with validation failed
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())
	cs.validationResult = &CampaignValidationDoneMsg{
		Success:  false,
		Duration: 2 * time.Second,
	}

	// When: the view is rendered
	view := cs.View(60, 20)
	plain := stripANSI(view)

	// Then: the validation row shows a cross
	if !strings.Contains(plain, "Feature validation") {
		t.Errorf("view should show validation result, got:\n%s", plain)
	}
	if !strings.Contains(plain, SymbolCross) {
		t.Errorf("failed validation should show cross, got:\n%s", plain)
	}
}

func TestCampaign_ViewHeader_WithProvider(t *testing.T) {
	// Given: a campaign state with provider set
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())
	cs.provider = "kiro"

	// When: the view is rendered
	view := cs.View(80, 30)
	plain := stripANSI(view)

	// Then: the header shows the provider badge
	lines := strings.Split(plain, "\n")
	if len(lines) == 0 {
		t.Fatal("view should have at least one line")
	}
	if !strings.Contains(lines[0], "[kiro]") {
		t.Errorf("header should contain provider badge [kiro], got: %q", lines[0])
	}
}

func TestCampaign_ViewHeader_NoProvider(t *testing.T) {
	// Given: a campaign state without provider
	cs := newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())

	// When: the view is rendered
	view := cs.View(80, 30)
	plain := stripANSI(view)

	// Then: no provider badge in the header
	lines := strings.Split(plain, "\n")
	if strings.Contains(lines[0], "[") {
		t.Errorf("header should not contain bracket badge, got: %q", lines[0])
	}
}
