package dashboard

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func newPassedSummaryModel(w, h int) Model {
	lister := &stubLister{beads: sampleBeads()}
	m := NewModel(WithBeadLister(lister))
	updated, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = updated.(Model)

	m.mode = ModeSummary
	m.pipeline = newPipelineState([]string{"plan", "code", "test"})
	m.pipeline, _ = m.pipeline.Update(PhaseUpdateMsg{Phase: "plan", Status: PhasePassed, Duration: 2 * time.Second})
	m.pipeline, _ = m.pipeline.Update(PhaseUpdateMsg{Phase: "code", Status: PhasePassed, Duration: 3 * time.Second})
	m.pipeline, _ = m.pipeline.Update(PhaseUpdateMsg{Phase: "test", Status: PhasePassed, Duration: 1 * time.Second})
	m.pipelineOutput = &PipelineOutput{Success: true}

	return m
}

func newFailedSummaryModel(w, h int) Model {
	lister := &stubLister{beads: sampleBeads()}
	m := NewModel(WithBeadLister(lister))
	updated, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = updated.(Model)

	m.mode = ModeSummary
	m.pipeline = newPipelineState([]string{"plan", "code", "test"})
	m.pipeline, _ = m.pipeline.Update(PhaseUpdateMsg{Phase: "plan", Status: PhasePassed, Duration: 2 * time.Second})
	m.pipeline, _ = m.pipeline.Update(PhaseUpdateMsg{Phase: "code", Status: PhaseFailed, Duration: 5 * time.Second})
	m.pipelineErr = fmt.Errorf("build failed")

	return m
}

func TestSummary_LeftPaneShowsFrozenPhases(t *testing.T) {
	m := newPassedSummaryModel(90, 40)
	view := m.View()
	plain := stripANSI(view)

	if !strings.Contains(plain, "✓") {
		t.Errorf("left pane should show ✓ for passed phases, got:\n%s", plain)
	}
	for _, name := range []string{"plan", "code", "test"} {
		if !strings.Contains(plain, name) {
			t.Errorf("left pane should show phase %q, got:\n%s", name, plain)
		}
	}
}

func TestSummary_RightPaneShowsPassSummary(t *testing.T) {
	m := newPassedSummaryModel(90, 40)
	view := m.View()
	plain := stripANSI(view)

	if !strings.Contains(plain, "Pipeline Passed") {
		t.Errorf("right pane should show 'Pipeline Passed', got:\n%s", plain)
	}
	if !strings.Contains(plain, "3/3 phases passed") {
		t.Errorf("right pane should show '3/3 phases passed', got:\n%s", plain)
	}
	if !strings.Contains(plain, "6.0s") {
		t.Errorf("right pane should show total duration, got:\n%s", plain)
	}
}

func TestSummary_RightPaneShowsFailSummary(t *testing.T) {
	m := newFailedSummaryModel(90, 40)
	view := m.View()
	plain := stripANSI(view)

	if !strings.Contains(plain, "Pipeline Failed") {
		t.Errorf("right pane should show 'Pipeline Failed', got:\n%s", plain)
	}
	if !strings.Contains(plain, "build failed") {
		t.Errorf("right pane should show error message, got:\n%s", plain)
	}
	if !strings.Contains(plain, "1/3 phases passed") {
		t.Errorf("right pane should show '1/3 phases passed', got:\n%s", plain)
	}
}

func TestSummary_AnyKeyTransitionsToBrowse(t *testing.T) {
	m := newPassedSummaryModel(90, 40)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)

	if m.mode != ModeBrowse {
		t.Errorf("mode = %d, want ModeBrowse (%d)", m.mode, ModeBrowse)
	}
	if m.focus != PaneLeft {
		t.Errorf("focus = %d, want PaneLeft (%d)", m.focus, PaneLeft)
	}
}

func TestSummary_AnyKeyInvalidatesCache(t *testing.T) {
	m := newPassedSummaryModel(90, 40)
	m.cache.Set("cap-001", &BeadDetail{ID: "cap-001"})

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)

	if _, ok := m.cache.Get("cap-001"); ok {
		t.Error("cache should be invalidated after returning to browse")
	}
}

func TestSummary_AnyKeyTriggersRefresh(t *testing.T) {
	m := newPassedSummaryModel(90, 40)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	if cmd == nil {
		t.Fatal("returning to browse should produce a refresh command")
	}
	msg := cmd()
	if _, ok := msg.(BeadListMsg); !ok {
		t.Fatalf("command produced %T, want BeadListMsg", msg)
	}
}

func TestSummary_QDoesNotQuit(t *testing.T) {
	m := newPassedSummaryModel(90, 40)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(Model)

	if m.mode != ModeBrowse {
		t.Errorf("q in summary mode should transition to browse, got mode = %d", m.mode)
	}
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); ok {
			t.Error("q in summary mode should not quit")
		}
	}
}

func TestSummary_CtrlCDoesNotQuit(t *testing.T) {
	m := newPassedSummaryModel(90, 40)

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)

	if m.mode != ModeBrowse {
		t.Errorf("ctrl+c in summary mode should transition to browse, got mode = %d", m.mode)
	}
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); ok {
			t.Error("ctrl+c in summary mode should not quit")
		}
	}
}

func TestSummary_FullFlowReturnToBrowse(t *testing.T) {
	runner := &mockRunner{
		events: []PhaseUpdateMsg{
			{Phase: "plan", Status: PhaseRunning},
			{Phase: "plan", Status: PhasePassed, Duration: time.Second},
		},
		output: PipelineOutput{Success: true},
	}
	lister := &stubLister{beads: sampleBeads()}
	m := NewModel(
		WithPipelineRunner(runner),
		WithPhaseNames([]string{"plan"}),
		WithBeadLister(lister),
	)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)

	// Dispatch and drain to summary.
	updated, _ = m.Update(DispatchMsg{BeadID: "cap-001"})
	m = updated.(Model)
	m = drainPipeline(t, m)

	if m.mode != ModeSummary {
		t.Fatalf("mode = %d, want ModeSummary", m.mode)
	}

	// Press any key to return to browse.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m = updated.(Model)

	if m.mode != ModeBrowse {
		t.Errorf("mode = %d, want ModeBrowse after keypress", m.mode)
	}
	if cmd == nil {
		t.Fatal("should produce refresh command")
	}
}
