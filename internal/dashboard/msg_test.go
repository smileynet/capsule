package dashboard

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCampaignModeConstants(t *testing.T) {
	// Given: the new campaign mode constants
	// Then: they are distinct from existing modes
	modes := []Mode{ModeBrowse, ModePipeline, ModeSummary, ModeCampaign, ModeCampaignSummary}
	seen := make(map[Mode]bool)
	for _, m := range modes {
		if seen[m] {
			t.Errorf("duplicate mode value: %d", m)
		}
		seen[m] = true
	}
}

func TestCampaignTaskStatus_Values(t *testing.T) {
	// Given: defined campaign task statuses
	// Then: each status has the expected string value
	tests := []struct {
		status CampaignTaskStatus
		want   string
	}{
		{CampaignTaskPending, "pending"},
		{CampaignTaskRunning, "running"},
		{CampaignTaskPassed, "passed"},
		{CampaignTaskFailed, "failed"},
		{CampaignTaskSkipped, "skipped"},
	}
	for _, tt := range tests {
		if string(tt.status) != tt.want {
			t.Errorf("CampaignTaskStatus = %q, want %q", tt.status, tt.want)
		}
	}
}

func TestCampaignTaskInfo_Fields(t *testing.T) {
	// Given: a CampaignTaskInfo with all fields set
	info := CampaignTaskInfo{
		BeadID:   "cap-001",
		Title:    "First task",
		Priority: 2,
	}

	// Then: fields are accessible
	if info.BeadID != "cap-001" {
		t.Errorf("BeadID = %q, want %q", info.BeadID, "cap-001")
	}
	if info.Title != "First task" {
		t.Errorf("Title = %q, want %q", info.Title, "First task")
	}
	if info.Priority != 2 {
		t.Errorf("Priority = %d, want 2", info.Priority)
	}
}

func TestCampaignStartMsg_Fields(t *testing.T) {
	// Given: a CampaignStartMsg with parent and tasks
	tasks := []CampaignTaskInfo{
		{BeadID: "cap-001", Title: "Task 1", Priority: 1},
		{BeadID: "cap-002", Title: "Task 2", Priority: 2},
	}
	msg := CampaignStartMsg{
		ParentID: "cap-feat",
		Tasks:    tasks,
	}

	// Then: fields are accessible
	if msg.ParentID != "cap-feat" {
		t.Errorf("ParentID = %q, want %q", msg.ParentID, "cap-feat")
	}
	if len(msg.Tasks) != 2 {
		t.Errorf("Tasks len = %d, want 2", len(msg.Tasks))
	}
}

func TestCampaignTaskStartMsg_Fields(t *testing.T) {
	// Given: a CampaignTaskStartMsg
	msg := CampaignTaskStartMsg{
		BeadID: "cap-001",
		Index:  0,
		Total:  3,
	}

	// Then: fields are accessible
	if msg.BeadID != "cap-001" {
		t.Errorf("BeadID = %q, want %q", msg.BeadID, "cap-001")
	}
	if msg.Index != 0 {
		t.Errorf("Index = %d, want 0", msg.Index)
	}
	if msg.Total != 3 {
		t.Errorf("Total = %d, want 3", msg.Total)
	}
}

func TestCampaignTaskDoneMsg_Fields(t *testing.T) {
	// Given: a CampaignTaskDoneMsg for a successful task
	msg := CampaignTaskDoneMsg{
		BeadID:   "cap-001",
		Index:    0,
		Success:  true,
		Duration: 5 * time.Second,
	}

	// Then: fields are accessible
	if msg.BeadID != "cap-001" {
		t.Errorf("BeadID = %q, want %q", msg.BeadID, "cap-001")
	}
	if !msg.Success {
		t.Error("Success should be true")
	}
	if msg.Duration != 5*time.Second {
		t.Errorf("Duration = %v, want 5s", msg.Duration)
	}
}

func TestCampaignDoneMsg_Fields(t *testing.T) {
	// Given: a CampaignDoneMsg with results including skipped tasks
	msg := CampaignDoneMsg{
		ParentID:   "cap-feat",
		TotalTasks: 5,
		Passed:     2,
		Failed:     1,
		Skipped:    2,
	}

	// Then: fields are accessible
	if msg.ParentID != "cap-feat" {
		t.Errorf("ParentID = %q, want %q", msg.ParentID, "cap-feat")
	}
	if msg.TotalTasks != 5 {
		t.Errorf("TotalTasks = %d, want 5", msg.TotalTasks)
	}
	if msg.Passed != 2 {
		t.Errorf("Passed = %d, want 2", msg.Passed)
	}
	if msg.Failed != 1 {
		t.Errorf("Failed = %d, want 1", msg.Failed)
	}
	if msg.Skipped != 2 {
		t.Errorf("Skipped = %d, want 2", msg.Skipped)
	}
}

func TestDispatchMsg_BeadType(t *testing.T) {
	// Given: a DispatchMsg with BeadType set
	msg := DispatchMsg{
		BeadID:   "cap-001",
		BeadType: "feature",
	}

	// Then: BeadType is accessible
	if msg.BeadType != "feature" {
		t.Errorf("BeadType = %q, want %q", msg.BeadType, "feature")
	}
}

func TestBrowse_EnterDispatchesWithBeadType(t *testing.T) {
	// Given: a browse state with beads that have types
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})

	// When: enter is pressed on the first bead (type="task")
	_, cmd := bs.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Then: DispatchMsg includes BeadType from the selected bead
	if cmd == nil {
		t.Fatal("enter should produce a command")
	}
	msg := cmd()
	dispatch, ok := msg.(DispatchMsg)
	if !ok {
		t.Fatalf("enter command produced %T, want DispatchMsg", msg)
	}
	if dispatch.BeadType != "task" {
		t.Errorf("dispatch BeadType = %q, want %q", dispatch.BeadType, "task")
	}
}

// Compile-time check: stubCampaignRunner satisfies CampaignRunner.
var _ CampaignRunner = (*stubCampaignRunner)(nil)

type stubCampaignRunner struct{}

func (s *stubCampaignRunner) RunCampaign(
	_ context.Context,
	_ string,
	_ func(tea.Msg),
	_ func(context.Context, PipelineInput, func(PhaseUpdateMsg)) (PipelineOutput, error),
) error {
	return nil
}
