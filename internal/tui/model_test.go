package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
)

func TestNewModel_InitializesPhases(t *testing.T) {
	phases := []string{"test-writer", "test-review", "execute"}
	m := NewModel(phases)

	if got := len(m.phases); got != 3 {
		t.Fatalf("phases count = %d, want 3", got)
	}
	for i, name := range phases {
		if m.phases[i].Name != name {
			t.Errorf("phases[%d].Name = %q, want %q", i, m.phases[i].Name, name)
		}
		if m.phases[i].Status != StatusPending {
			t.Errorf("phases[%d].Status = %q, want %q", i, m.phases[i].Status, StatusPending)
		}
	}
	if m.done {
		t.Error("new model should not be done")
	}
	if m.err != nil {
		t.Errorf("new model should have nil err, got %v", m.err)
	}
}

func TestNewModel_EmptyPhases(t *testing.T) {
	m := NewModel(nil)
	if len(m.phases) != 0 {
		t.Fatalf("phases count = %d, want 0", len(m.phases))
	}
}

func TestModel_Init_ReturnsTickCmd(t *testing.T) {
	m := NewModel([]string{"phase1"})
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() should return a non-nil Cmd for the spinner")
	}
}

func TestModel_Update_StatusUpdateMsg_Running(t *testing.T) {
	m := NewModel([]string{"test-writer", "test-review"})
	msg := StatusUpdateMsg{
		Phase:    "test-writer",
		Status:   StatusRunning,
		Attempt:  1,
		MaxRetry: 3,
	}

	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.phases[0].Status != StatusRunning {
		t.Errorf("phase status = %q, want %q", updated.phases[0].Status, StatusRunning)
	}
	if updated.phases[0].Attempt != 1 {
		t.Errorf("attempt = %d, want 1", updated.phases[0].Attempt)
	}
	if updated.phases[0].MaxRetry != 3 {
		t.Errorf("maxRetry = %d, want 3", updated.phases[0].MaxRetry)
	}
	if updated.currentIdx != 0 {
		t.Errorf("currentIdx = %d, want 0", updated.currentIdx)
	}
}

func TestModel_Update_StatusUpdateMsg_Transitions(t *testing.T) {
	tests := []struct {
		name   string
		status PhaseStatus
	}{
		{name: "passed", status: StatusPassed},
		{name: "failed", status: StatusFailed},
		{name: "error", status: StatusError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel([]string{"test-writer"})
			msg := StatusUpdateMsg{Phase: "test-writer", Status: tt.status}

			newModel, _ := m.Update(msg)
			updated := newModel.(Model)

			if updated.phases[0].Status != tt.status {
				t.Errorf("phase status = %q, want %q", updated.phases[0].Status, tt.status)
			}
		})
	}
}

func TestModel_Update_StatusUpdateMsg_UnknownPhase(t *testing.T) {
	m := NewModel([]string{"test-writer"})

	msg := StatusUpdateMsg{
		Phase:  "unknown-phase",
		Status: StatusRunning,
	}

	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	// Should not crash, phases remain unchanged
	if updated.phases[0].Status != StatusPending {
		t.Errorf("phase status = %q, want %q (unchanged)", updated.phases[0].Status, StatusPending)
	}
}

func TestModel_Update_StatusUpdateMsg_UpdatesCurrentIdx(t *testing.T) {
	m := NewModel([]string{"test-writer", "test-review", "execute"})

	// When second phase starts running, currentIdx should advance
	msg := StatusUpdateMsg{
		Phase:  "test-review",
		Status: StatusRunning,
	}

	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.currentIdx != 1 {
		t.Errorf("currentIdx = %d, want 1", updated.currentIdx)
	}
}

func TestModel_Update_PipelineDoneMsg(t *testing.T) {
	m := NewModel([]string{"test-writer"})

	newModel, cmd := m.Update(PipelineDoneMsg{})
	updated := newModel.(Model)

	if !updated.done {
		t.Error("model should be done after PipelineDoneMsg")
	}
	// Should return a quit command
	if cmd == nil {
		t.Error("PipelineDoneMsg should produce a quit Cmd")
	}
}

func TestModel_Update_PipelineErrorMsg(t *testing.T) {
	m := NewModel([]string{"test-writer"})
	testErr := errors.New("provider failed")

	newModel, cmd := m.Update(PipelineErrorMsg{Err: testErr})
	updated := newModel.(Model)

	if !updated.done {
		t.Error("model should be done after PipelineErrorMsg")
	}
	if updated.err == nil || updated.err.Error() != "provider failed" {
		t.Errorf("err = %v, want 'provider failed'", updated.err)
	}
	if cmd == nil {
		t.Error("PipelineErrorMsg should produce a quit Cmd")
	}
}

func TestModel_Update_KeyMsg_Q(t *testing.T) {
	m := NewModel([]string{"test-writer"})

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	updated := newModel.(Model)

	if !updated.done {
		t.Error("pressing q should set done")
	}
	if cmd == nil {
		t.Error("pressing q should produce a quit Cmd")
	}
}

func TestModel_Update_KeyMsg_CtrlC(t *testing.T) {
	m := NewModel([]string{"test-writer"})

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	updated := newModel.(Model)

	if !updated.done {
		t.Error("ctrl+c should set done")
	}
	if cmd == nil {
		t.Error("ctrl+c should produce a quit Cmd")
	}
}

func TestModel_Update_StatusUpdateMsg_TracksDuration(t *testing.T) {
	m := NewModel([]string{"test-writer"})

	dur := 2 * time.Second
	msg := StatusUpdateMsg{
		Phase:    "test-writer",
		Status:   StatusPassed,
		Duration: dur,
	}

	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.phases[0].Duration != dur {
		t.Errorf("duration = %v, want %v", updated.phases[0].Duration, dur)
	}
}

func TestModel_View_StatusIndicators(t *testing.T) {
	tests := []struct {
		name      string
		status    PhaseStatus
		wantIn    string
		wantNotIn string
	}{
		{name: "pending", status: StatusPending, wantIn: "○"},
		{name: "running", status: StatusRunning, wantNotIn: "○"},
		{name: "passed", status: StatusPassed, wantIn: "✓"},
		{name: "failed", status: StatusFailed, wantIn: "✗"},
		{name: "error", status: StatusError, wantIn: "✗"},
		{name: "skipped", status: StatusSkipped, wantIn: "–"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewModel([]string{"test-writer"})
			m.phases[0].Status = tt.status

			view := m.View()

			if !strings.Contains(view, "test-writer") {
				t.Error("view should contain phase name")
			}
			if tt.wantIn != "" && !strings.Contains(view, tt.wantIn) {
				t.Errorf("view should contain %q", tt.wantIn)
			}
			if tt.wantNotIn != "" && strings.Contains(view, tt.wantNotIn) {
				t.Errorf("view should not contain %q", tt.wantNotIn)
			}
		})
	}
}

func TestModel_View_WithRetryInfo(t *testing.T) {
	m := NewModel([]string{"test-writer"})
	m.phases[0].Status = StatusRunning
	m.phases[0].Attempt = 2
	m.phases[0].MaxRetry = 3

	view := m.View()

	if !strings.Contains(view, "2/3") {
		t.Error("view should show retry info (2/3)")
	}
}

func TestModel_View_MultiplePhases(t *testing.T) {
	m := NewModel([]string{"test-writer", "test-review", "execute"})
	m.phases[0].Status = StatusPassed
	m.phases[1].Status = StatusRunning
	m.phases[2].Status = StatusPending

	view := m.View()

	if !strings.Contains(view, "test-writer") {
		t.Error("view should contain first phase name")
	}
	if !strings.Contains(view, "test-review") {
		t.Error("view should contain second phase name")
	}
	if !strings.Contains(view, "execute") {
		t.Error("view should contain third phase name")
	}
	if !strings.Contains(view, "✓") {
		t.Error("view should contain passed indicator for first phase")
	}
	if !strings.Contains(view, "○") {
		t.Error("view should contain pending indicator for third phase")
	}
}

func TestModel_View_DoneWithError(t *testing.T) {
	m := NewModel([]string{"test-writer"})
	m.done = true
	m.err = errors.New("pipeline failed")

	view := m.View()

	if !strings.Contains(view, "pipeline failed") {
		t.Error("view should show error message when done with error")
	}
}

func TestModel_View_DoneSuccess(t *testing.T) {
	m := NewModel([]string{"test-writer"})
	m.done = true
	m.phases[0].Status = StatusPassed

	view := m.View()

	if !strings.Contains(view, "✓") {
		t.Error("view should show passed indicator when done successfully")
	}
}

func TestModel_View_WithDuration(t *testing.T) {
	m := NewModel([]string{"test-writer"})
	m.phases[0].Status = StatusPassed
	m.phases[0].Duration = 5 * time.Second

	view := m.View()

	if !strings.Contains(view, "5.0s") {
		t.Error("view should show duration for completed phases")
	}
}

func TestModel_View_SummaryFooter_AllPassed(t *testing.T) {
	m := NewModel([]string{"test-writer", "test-review"})
	m.phases[0].Status = StatusPassed
	m.phases[0].Duration = 2 * time.Second
	m.phases[1].Status = StatusPassed
	m.phases[1].Duration = 3 * time.Second
	m.done = true

	view := m.View()

	if !strings.Contains(view, "2/2 passed") {
		t.Errorf("summary should show pass count, got:\n%s", view)
	}
	if !strings.Contains(view, "in 5.0s") {
		t.Errorf("summary should show total duration, got:\n%s", view)
	}
	if strings.Contains(view, "Error") {
		t.Error("all-passed summary should not contain error text")
	}
}

func TestModel_View_SummaryFooter_WithError(t *testing.T) {
	m := NewModel([]string{"test-writer", "test-review"})
	m.phases[0].Status = StatusPassed
	m.phases[1].Status = StatusFailed
	m.done = true
	m.err = errors.New("test-review failed")

	view := m.View()

	if !strings.Contains(view, "1/2 passed") {
		t.Errorf("summary should show pass count, got:\n%s", view)
	}
	if !strings.Contains(view, "test-review failed") {
		t.Errorf("summary should show error message, got:\n%s", view)
	}
}

func TestModel_View_SummaryFooter_NotShownWhenRunning(t *testing.T) {
	m := NewModel([]string{"test-writer"})
	m.phases[0].Status = StatusRunning

	view := m.View()

	if strings.Contains(view, "passed") {
		t.Error("summary footer should not appear while pipeline is running")
	}
}

func TestModel_View_SummaryFooter_TotalDuration(t *testing.T) {
	m := NewModel([]string{"phase1", "phase2", "phase3"})
	m.phases[0].Status = StatusPassed
	m.phases[0].Duration = 1500 * time.Millisecond
	m.phases[1].Status = StatusPassed
	m.phases[1].Duration = 2500 * time.Millisecond
	m.phases[2].Status = StatusPassed
	m.phases[2].Duration = 500 * time.Millisecond
	m.done = true

	view := m.View()

	// Total: 4.5s - unique to footer (phase lines show 1.5s, 2.5s, 0.5s)
	if !strings.Contains(view, "in 4.5s") {
		t.Errorf("footer should show total duration 'in 4.5s', got:\n%s", view)
	}
}

// --- Abort tests ---

func TestModel_Update_KeyMsg_Q_WithCancel_SetsAborting(t *testing.T) {
	cancelled := false
	m := NewModel([]string{"test-writer"}, WithCancelFunc(func() { cancelled = true }))

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	updated := newModel.(Model)

	if !updated.aborting {
		t.Error("first q with cancelFunc should set aborting")
	}
	if updated.done {
		t.Error("first q with cancelFunc should not set done")
	}
	if !cancelled {
		t.Error("first q should call cancelFunc")
	}
	if cmd != nil {
		t.Error("first q should not produce quit Cmd")
	}
}

func TestModel_Update_KeyMsg_CtrlC_WithCancel_SetsAborting(t *testing.T) {
	cancelled := false
	m := NewModel([]string{"test-writer"}, WithCancelFunc(func() { cancelled = true }))

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	updated := newModel.(Model)

	if !updated.aborting {
		t.Error("first ctrl+c with cancelFunc should set aborting")
	}
	if !cancelled {
		t.Error("first ctrl+c should call cancelFunc")
	}
	if cmd != nil {
		t.Error("first ctrl+c should not produce quit Cmd")
	}
}

func TestModel_Update_KeyMsg_DoublePress_ForcesQuit(t *testing.T) {
	m := NewModel([]string{"test-writer"}, WithCancelFunc(func() {}))
	m.aborting = true

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	updated := newModel.(Model)

	if !updated.done {
		t.Error("double-press should set done")
	}
	if cmd == nil {
		t.Error("double-press should produce quit Cmd")
	}
}

func TestModel_Update_KeyMsg_CtrlC_DoublePress_ForcesQuit(t *testing.T) {
	m := NewModel([]string{"test-writer"}, WithCancelFunc(func() {}))
	m.aborting = true

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	updated := newModel.(Model)

	if !updated.done {
		t.Error("double-press ctrl+c should set done")
	}
	if cmd == nil {
		t.Error("double-press ctrl+c should produce quit Cmd")
	}
}

func TestModel_Update_KeyMsg_WhenDone_Ignored(t *testing.T) {
	m := NewModel([]string{"test-writer"}, WithCancelFunc(func() {}))
	m.done = true

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	updated := newModel.(Model)

	if updated.aborting {
		t.Error("pressing q when done should not set aborting")
	}
	if cmd != nil {
		t.Error("pressing q when done should not produce cmd")
	}
}

func TestModel_View_AbortingState(t *testing.T) {
	m := NewModel([]string{"test-writer"})
	m.aborting = true
	m.phases[0].Status = StatusRunning

	view := m.View()

	if !strings.Contains(view, "Aborting") {
		t.Errorf("view should show 'Aborting' when aborting, got:\n%s", view)
	}
}

func TestModel_Update_KeyMsg_Q_WithoutCancel_ImmediateQuit(t *testing.T) {
	// Without a cancelFunc, q should still do immediate quit (backward compat).
	m := NewModel([]string{"test-writer"})

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	updated := newModel.(Model)

	if !updated.done {
		t.Error("q without cancelFunc should set done")
	}
	if cmd == nil {
		t.Error("q without cancelFunc should produce quit Cmd")
	}
}

func TestModel_Update_PipelineDoneMsg_ClearsAborting(t *testing.T) {
	m := NewModel([]string{"test-writer"}, WithCancelFunc(func() {}))
	m.aborting = true

	newModel, cmd := m.Update(PipelineDoneMsg{})
	updated := newModel.(Model)

	if !updated.done {
		t.Error("PipelineDoneMsg should set done even when aborting")
	}
	if updated.aborting {
		t.Error("PipelineDoneMsg should clear aborting")
	}
	if cmd == nil {
		t.Error("PipelineDoneMsg should produce quit Cmd")
	}
	view := updated.View()
	if strings.Contains(view, "Aborting") {
		t.Error("View should not show Aborting when done")
	}
}

func TestModel_Update_PipelineErrorMsg_ClearsAborting(t *testing.T) {
	m := NewModel([]string{"test-writer"}, WithCancelFunc(func() {}))
	m.aborting = true

	newModel, cmd := m.Update(PipelineErrorMsg{Err: context.Canceled})
	updated := newModel.(Model)

	if !updated.done {
		t.Error("PipelineErrorMsg should set done even when aborting")
	}
	if cmd == nil {
		t.Error("PipelineErrorMsg should produce quit Cmd")
	}
}

func TestModel_Update_WindowSizeMsg(t *testing.T) {
	m := NewModel([]string{"test-writer"})

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	updated := newModel.(Model)

	if updated.width != 120 {
		t.Errorf("width = %d, want 120", updated.width)
	}
}

// TestModel_Teatest_FullPipeline verifies the model processes messages in sequence via teatest.
func TestModel_Teatest_FullPipeline(t *testing.T) {
	phases := []string{"test-writer", "test-review", "execute"}
	m := NewModel(phases)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	for _, phase := range phases {
		tm.Send(StatusUpdateMsg{Phase: phase, Status: StatusRunning, Attempt: 1, MaxRetry: 3})
		tm.Send(StatusUpdateMsg{Phase: phase, Status: StatusPassed})
	}
	tm.Send(PipelineDoneMsg{})

	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	// Get final model and verify all phases passed
	final := tm.FinalModel(t).(Model)
	for i, name := range phases {
		if final.phases[i].Status != StatusPassed {
			t.Errorf("phase %q status = %q, want %q", name, final.phases[i].Status, StatusPassed)
		}
	}
	if !final.done {
		t.Error("final model should be done")
	}
}
