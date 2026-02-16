package dashboard

import (
	"strings"
	"testing"
	"time"
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
