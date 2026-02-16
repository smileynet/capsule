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

func TestModel_View_BeadHeader(t *testing.T) {
	m := NewModel([]string{"test-writer"}, WithBeadHeader("cap-042", "Fix login bug"))

	view := m.View()

	lines := strings.Split(view, "\n")
	if len(lines) == 0 {
		t.Fatal("view should have at least one line")
	}
	if !strings.Contains(lines[0], "cap-042") {
		t.Errorf("first line should contain bead ID, got: %q", lines[0])
	}
	if !strings.Contains(lines[0], "Fix login bug") {
		t.Errorf("first line should contain bead title, got: %q", lines[0])
	}
}

func TestModel_View_NoBeadHeader_WhenEmpty(t *testing.T) {
	m := NewModel([]string{"test-writer"})

	view := m.View()

	// Without bead header, first line should be a phase line
	if strings.Contains(view, "cap-") {
		t.Error("view should not contain any bead ID prefix when no header configured")
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

// --- Detail view tests ---

func TestModel_Update_KeyMsg_D_TogglesDetailOn(t *testing.T) {
	m := NewModel([]string{"test-writer"})

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	updated := newModel.(Model)

	if !updated.detailVisible {
		t.Error("pressing d should toggle detail view on")
	}
}

func TestModel_Update_KeyMsg_D_TogglesDetailOff(t *testing.T) {
	m := NewModel([]string{"test-writer"})
	m.detailVisible = true

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	updated := newModel.(Model)

	if updated.detailVisible {
		t.Error("pressing d again should toggle detail view off")
	}
}

func TestModel_Update_KeyMsg_D_IgnoredWhenDone(t *testing.T) {
	m := NewModel([]string{"test-writer"})
	m.done = true

	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	updated := newModel.(Model)

	if updated.detailVisible {
		t.Error("d should be ignored when pipeline is done")
	}
}

func TestModel_Update_OutputMsg_StoresContent(t *testing.T) {
	m := NewModel([]string{"test-writer"})

	newModel, _ := m.Update(OutputMsg{Content: "line 1\nline 2\nline 3"})
	updated := newModel.(Model)

	if updated.detailContent != "line 1\nline 2\nline 3" {
		t.Errorf("detailContent = %q, want %q", updated.detailContent, "line 1\nline 2\nline 3")
	}
}

func TestModel_Update_OutputMsg_UpdatesViewport(t *testing.T) {
	m := NewModel([]string{"test-writer"})
	m.viewport.Width = 80
	m.viewport.Height = 10

	newModel, _ := m.Update(OutputMsg{Content: "line 1\nline 2"})
	updated := newModel.(Model)

	view := updated.viewport.View()
	if !strings.Contains(view, "line 1") {
		t.Errorf("viewport should contain output content, got: %q", view)
	}
}

func TestModel_View_DetailVisible_ShowsViewport(t *testing.T) {
	m := NewModel([]string{"test-writer"})
	m.detailVisible = true
	m.detailContent = "some output"
	m.viewport.Width = 80
	m.viewport.Height = 10
	m.viewport.SetContent("some output")

	view := m.View()

	if !strings.Contains(view, "some output") {
		t.Errorf("view with detail visible should show output content, got:\n%s", view)
	}
}

func TestModel_View_DetailHidden_NoViewportContent(t *testing.T) {
	m := NewModel([]string{"test-writer"})
	m.detailVisible = false
	m.detailContent = "some output"

	view := m.View()

	if strings.Contains(view, "some output") {
		t.Error("view with detail hidden should not show output content")
	}
}

func TestModel_View_DetailVisible_EmptyContent_ShowsPlaceholder(t *testing.T) {
	m := NewModel([]string{"test-writer"})
	m.detailVisible = true
	m.width = 80
	m.height = 24

	view := m.View()

	if !strings.Contains(view, "No output yet") {
		t.Errorf("detail view with no content should show placeholder, got:\n%s", view)
	}
}

func TestModel_Update_OutputMsg_ReplacesContent(t *testing.T) {
	m := NewModel([]string{"test-writer"})

	first, _ := m.Update(OutputMsg{Content: "first"})
	second, _ := first.Update(OutputMsg{Content: "second"})
	updated := second.(Model)

	if updated.detailContent != "second" {
		t.Errorf("detailContent = %q, want %q", updated.detailContent, "second")
	}
}

func TestModel_Update_WindowSizeMsg_ResizesViewport(t *testing.T) {
	m := NewModel([]string{"test-writer"})

	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	updated := newModel.(Model)

	if updated.viewport.Width != 120 {
		t.Errorf("viewport width = %d, want 120", updated.viewport.Width)
	}
	if updated.viewport.Height == 0 {
		t.Error("viewport height should be set after WindowSizeMsg")
	}
}

// --- Elapsed time ticker tests ---

func TestModel_Update_StatusUpdateMsg_Running_SetsPhaseStartedAt(t *testing.T) {
	m := NewModel([]string{"test-writer", "test-review"})
	msg := StatusUpdateMsg{Phase: "test-writer", Status: StatusRunning}

	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.phaseStartedAt.IsZero() {
		t.Error("phaseStartedAt should be set when a phase starts running")
	}
}

func TestModel_Update_StatusUpdateMsg_Running_ResetsPhaseStartedAt(t *testing.T) {
	m := NewModel([]string{"test-writer", "test-review"})
	m.Update(StatusUpdateMsg{Phase: "test-writer", Status: StatusRunning})
	time.Sleep(2 * time.Millisecond)

	newModel, _ := m.Update(StatusUpdateMsg{Phase: "test-review", Status: StatusRunning})
	updated := newModel.(Model)

	// phaseStartedAt should be reset (not zero)
	if updated.phaseStartedAt.IsZero() {
		t.Error("phaseStartedAt should be set for new running phase")
	}
}

func TestModel_View_ElapsedTime_ForRunningPhase(t *testing.T) {
	m := NewModel([]string{"test-writer"})
	m.phases[0].Status = StatusRunning
	m.phaseStartedAt = time.Now().Add(-42 * time.Second)

	view := m.View()

	if !strings.Contains(view, "(42s)") {
		t.Errorf("running phase should show elapsed time '(42s)', got:\n%s", view)
	}
}

func TestModel_View_ElapsedTime_NotShownForPendingPhase(t *testing.T) {
	m := NewModel([]string{"test-writer"})
	// phases are pending by default

	view := m.View()

	if strings.Contains(view, "s)") {
		t.Errorf("pending phase should not show elapsed time, got:\n%s", view)
	}
}

func TestModel_Update_ElapsedTickMsg_ReturnsTickWhenRunning(t *testing.T) {
	m := NewModel([]string{"test-writer"})
	m.phases[0].Status = StatusRunning
	m.phaseStartedAt = time.Now()

	_, cmd := m.Update(elapsedTickMsg{})

	if cmd == nil {
		t.Error("elapsedTickMsg should produce a follow-up tick when a phase is running")
	}
}

func TestModel_Update_ElapsedTickMsg_NoTickWhenNotRunning(t *testing.T) {
	m := NewModel([]string{"test-writer"})

	_, cmd := m.Update(elapsedTickMsg{})

	if cmd != nil {
		t.Error("elapsedTickMsg should not produce a tick when no phase is running")
	}
}

func TestModel_Init_ReturnsElapsedTick(t *testing.T) {
	m := NewModel([]string{"test-writer"})
	cmd := m.Init()

	// Init should return a batch that includes both the spinner tick and elapsed tick.
	if cmd == nil {
		t.Fatal("Init() should return a non-nil Cmd")
	}
}

// TestModel_Teatest_AbortFlow verifies the abort lifecycle through the full Bubble Tea program.
func TestModel_Teatest_AbortFlow(t *testing.T) {
	cancelled := false
	m := NewModel([]string{"test-writer", "test-review"}, WithCancelFunc(func() { cancelled = true }))

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(80, 24))

	// Pipeline starts running.
	tm.Send(StatusUpdateMsg{Phase: "test-writer", Status: StatusRunning, Attempt: 1, MaxRetry: 3})

	// User presses q to abort.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// Pipeline completes after graceful shutdown.
	tm.Send(StatusUpdateMsg{Phase: "test-writer", Status: StatusPassed})
	tm.Send(PipelineDoneMsg{})

	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))

	final := tm.FinalModel(t).(Model)
	if !cancelled {
		t.Error("cancel function should have been called")
	}
	if !final.done {
		t.Error("final model should be done")
	}
	if final.aborting {
		t.Error("aborting should be cleared after PipelineDoneMsg")
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
