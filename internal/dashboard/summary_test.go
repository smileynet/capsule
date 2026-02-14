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
	// Given: a model in summary mode with all phases passed
	m := newPassedSummaryModel(90, 40)

	// When: the view is rendered
	view := m.View()
	plain := stripANSI(view)

	// Then: passed phases with checkmarks and names are shown
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
	// Given: a model in summary mode with a successful pipeline
	m := newPassedSummaryModel(90, 40)

	// When: the view is rendered
	view := m.View()
	plain := stripANSI(view)

	// Then: "Pipeline Passed" with phase count and total duration are shown
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
	// Given: a model in summary mode with a failed pipeline
	m := newFailedSummaryModel(90, 40)

	// When: the view is rendered
	view := m.View()
	plain := stripANSI(view)

	// Then: "Pipeline Failed" with error and partial phase count are shown
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
	// Given: a model in summary mode
	m := newPassedSummaryModel(90, 40)

	// When: any key is pressed
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)

	// Then: the model transitions to browse mode with left pane focused
	if m.mode != ModeBrowse {
		t.Errorf("mode = %d, want ModeBrowse (%d)", m.mode, ModeBrowse)
	}
	if m.focus != PaneLeft {
		t.Errorf("focus = %d, want PaneLeft (%d)", m.focus, PaneLeft)
	}
}

func TestSummary_AnyKeyInvalidatesCache(t *testing.T) {
	// Given: a model in summary mode with a cached bead
	m := newPassedSummaryModel(90, 40)
	m.cache.Set("cap-001", &BeadDetail{ID: "cap-001"})

	// When: any key is pressed
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)

	// Then: the cache is invalidated
	if _, ok := m.cache.Get("cap-001"); ok {
		t.Error("cache should be invalidated after returning to browse")
	}
}

func TestSummary_AnyKeyTriggersRefresh(t *testing.T) {
	// Given: a model in summary mode with a lister
	m := newPassedSummaryModel(90, 40)

	// When: any key is pressed
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	// Then: a BeadListMsg refresh command is produced
	if cmd == nil {
		t.Fatal("returning to browse should produce a refresh command")
	}
	msg := cmd()
	if _, ok := msg.(BeadListMsg); !ok {
		t.Fatalf("command produced %T, want BeadListMsg", msg)
	}
}

func TestSummary_QDoesNotQuit(t *testing.T) {
	// Given: a model in summary mode
	m := newPassedSummaryModel(90, 40)

	// When: q is pressed
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(Model)

	// Then: the model transitions to browse (not quit)
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
	// Given: a model in summary mode
	m := newPassedSummaryModel(90, 40)

	// When: ctrl+c is pressed
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	m = updated.(Model)

	// Then: the model transitions to browse (not quit)
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
	// Given: a model that completed a pipeline and is in summary mode
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
	updated, _ = m.Update(DispatchMsg{BeadID: "cap-001"})
	m = updated.(Model)
	m = drainPipeline(t, m)
	if m.mode != ModeSummary {
		t.Fatalf("mode = %d, want ModeSummary", m.mode)
	}

	// When: any key is pressed to return to browse
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	m = updated.(Model)

	// Then: the model transitions to browse mode with a refresh command
	if m.mode != ModeBrowse {
		t.Errorf("mode = %d, want ModeBrowse after keypress", m.mode)
	}
	if cmd == nil {
		t.Fatal("should produce refresh command")
	}
}
