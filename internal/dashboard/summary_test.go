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

	// Then: a refresh command is produced (batch includes fetch + spinner tick)
	if cmd == nil {
		t.Fatal("returning to browse should produce a refresh command")
	}
	msgs := execBatch(t, cmd)
	var foundRefresh bool
	for _, msg := range msgs {
		if _, ok := msg.(BeadListMsg); ok {
			foundRefresh = true
		}
	}
	if !foundRefresh {
		t.Fatal("batch should contain BeadListMsg for bead list refresh")
	}
}

func TestSummary_ReturnToBrowseResetsShowClosed(t *testing.T) {
	// Given: a model in summary mode where showClosed was true before pipeline
	m := newPassedSummaryModel(90, 40)
	m.browse.showClosed = true
	m.browse.readyBeads = sampleBeads()

	// When: any key is pressed to return to browse
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)

	// Then: showClosed is reset to false
	if m.browse.showClosed {
		t.Error("returnToBrowse should reset showClosed to false")
	}
	// And: readyBeads is cleared
	if m.browse.readyBeads != nil {
		t.Error("returnToBrowse should clear readyBeads")
	}
}

func TestSummary_ReturnToBrowseAfterAbortResetsShowClosed(t *testing.T) {
	// Given: a model in pipeline mode that is aborting with showClosed=true
	lister := &stubLister{beads: sampleBeads()}
	m := NewModel(WithBeadLister(lister))
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)
	m.mode = ModePipeline
	m.aborting = true
	m.cancelPipeline = func() {}
	m.browse.showClosed = true
	m.browse.readyBeads = sampleBeads()

	// When: channelClosedMsg is received (abort completes)
	updated, _ = m.Update(channelClosedMsg{})
	m = updated.(Model)

	// Then: showClosed is reset to false
	if m.browse.showClosed {
		t.Error("returnToBrowseAfterAbort should reset showClosed to false")
	}
	if m.browse.readyBeads != nil {
		t.Error("returnToBrowseAfterAbort should clear readyBeads")
	}
}

func TestSummary_ReturnToBrowseFromCampaignResetsShowClosed(t *testing.T) {
	// Given: a model in campaign summary mode with showClosed=true
	lister := &stubLister{beads: sampleBeads()}
	m := NewModel(WithBeadLister(lister))
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)
	m.mode = ModeCampaignSummary
	m.browse.showClosed = true
	m.browse.readyBeads = sampleBeads()

	// When: any key is pressed to return to browse
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)

	// Then: showClosed is reset to false
	if m.browse.showClosed {
		t.Error("returnToBrowseFromCampaign should reset showClosed to false")
	}
	if m.browse.readyBeads != nil {
		t.Error("returnToBrowseFromCampaign should clear readyBeads")
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

func TestSummary_ReturnToBrowseFiresPostPipeline(t *testing.T) {
	// Given: a model in summary mode with PostPipelineFunc configured
	var calledBeadID string
	ppFunc := func(beadID string) error {
		calledBeadID = beadID
		return nil
	}
	lister := &stubLister{beads: sampleBeads()}
	m := NewModel(
		WithBeadLister(lister),
		WithPostPipelineFunc(ppFunc),
	)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)
	m.mode = ModeSummary
	m.dispatchedBeadID = "cap-001"
	m.pipeline = newPipelineState([]string{"plan"})
	m.pipeline, _ = m.pipeline.Update(PhaseUpdateMsg{Phase: "plan", Status: PhasePassed, Duration: time.Second})
	m.pipelineOutput = &PipelineOutput{Success: true}

	// When: any key is pressed to return to browse
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)

	// Then: the model transitions to browse mode
	if m.mode != ModeBrowse {
		t.Errorf("mode = %d, want ModeBrowse", m.mode)
	}
	// And: dispatchedBeadID is cleared
	if m.dispatchedBeadID != "" {
		t.Errorf("dispatchedBeadID = %q, want empty", m.dispatchedBeadID)
	}
	// And: a batch command is produced containing both postPipeline and refresh
	if cmd == nil {
		t.Fatal("expected batch command")
	}
	batchMsg := cmd()
	batch, ok := batchMsg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected BatchMsg, got %T", batchMsg)
	}
	// Execute all cmds in the batch to trigger PostPipelineFunc
	for _, c := range batch {
		if c != nil {
			c()
		}
	}
	if calledBeadID != "cap-001" {
		t.Errorf("PostPipelineFunc called with %q, want %q", calledBeadID, "cap-001")
	}
}

func TestSummary_ReturnToBrowseWithoutPostPipelineProducesRefreshOnly(t *testing.T) {
	// Given: a model in summary mode with dispatchedBeadID but no PostPipelineFunc
	lister := &stubLister{beads: sampleBeads()}
	m := NewModel(WithBeadLister(lister))
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)
	m.mode = ModeSummary
	m.dispatchedBeadID = "cap-001"
	m.pipeline = newPipelineState([]string{"plan"})
	m.pipeline, _ = m.pipeline.Update(PhaseUpdateMsg{Phase: "plan", Status: PhasePassed, Duration: time.Second})
	m.pipelineOutput = &PipelineOutput{Success: true}

	// When: any key is pressed
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	// Then: a refresh command batch is produced (fetch + spinner tick, no postPipeline)
	if cmd == nil {
		t.Fatal("expected refresh command")
	}
	msgs := execBatch(t, cmd)
	var foundRefresh bool
	for _, msg := range msgs {
		if _, ok := msg.(BeadListMsg); ok {
			foundRefresh = true
		}
		if _, ok := msg.(PostPipelineDoneMsg); ok {
			t.Error("should not fire PostPipelineFunc without PostPipelineFunc configured")
		}
	}
	if !foundRefresh {
		t.Fatal("batch should contain BeadListMsg for bead list refresh")
	}
}

func TestSummary_DispatchStoresBeadID(t *testing.T) {
	// Given: a model with a pipeline runner
	runner := &mockRunner{output: PipelineOutput{Success: true}}
	m := NewModel(
		WithPipelineRunner(runner),
		WithPhaseNames([]string{"plan"}),
	)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)

	// When: a DispatchMsg is received
	updated, _ = m.Update(DispatchMsg{BeadID: "cap-042"})
	m = updated.(Model)

	// Then: the dispatched bead ID is stored
	if m.dispatchedBeadID != "cap-042" {
		t.Errorf("dispatchedBeadID = %q, want %q", m.dispatchedBeadID, "cap-042")
	}
}

func TestSummary_PostPipelineDoneMsgSetsStatusAndTimer(t *testing.T) {
	// Given: a model in browse mode
	m := newSizedModel(90, 40)

	// When: a PostPipelineDoneMsg is received
	updated, cmd := m.Update(PostPipelineDoneMsg{BeadID: "cap-001", Err: fmt.Errorf("merge failed")})
	m = updated.(Model)

	// Then: the mode stays in browse
	if m.mode != ModeBrowse {
		t.Errorf("mode = %d, want ModeBrowse", m.mode)
	}
	// And: statusMsg is set
	if m.statusMsg == "" {
		t.Fatal("statusMsg should be set")
	}
	// And: a timer command is produced to clear the status
	if cmd == nil {
		t.Fatal("PostPipelineDoneMsg should produce a status clear timer")
	}
}

// --- Sticky cursor tests ---

func TestSummary_ReturnToBrowse_SetsLastDispatchedID(t *testing.T) {
	// Given: a model in summary mode with dispatchedBeadID="cap-002"
	lister := &stubLister{beads: sampleBeads()}
	m := NewModel(WithBeadLister(lister))
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)
	m.mode = ModeSummary
	m.dispatchedBeadID = "cap-002"
	m.pipeline = newPipelineState([]string{"plan"})
	m.pipeline, _ = m.pipeline.Update(PhaseUpdateMsg{Phase: "plan", Status: PhasePassed, Duration: time.Second})
	m.pipelineOutput = &PipelineOutput{Success: true}

	// When: any key is pressed to return to browse
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)

	// Then: lastDispatchedID is set to the dispatched bead
	if m.lastDispatchedID != "cap-002" {
		t.Errorf("lastDispatchedID = %q, want %q", m.lastDispatchedID, "cap-002")
	}
}

func TestSummary_StickyCursor_BeadListRestoresCursor(t *testing.T) {
	// Given: a model that returned to browse with lastDispatchedID="cap-002"
	lister := &stubLister{beads: sampleBeads()}
	m := NewModel(WithBeadLister(lister))
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)
	m.lastDispatchedID = "cap-002"

	// When: a BeadListMsg arrives with cap-002 in the list
	updated, _ = m.Update(BeadListMsg{Beads: sampleBeads()})
	m = updated.(Model)

	// Then: the cursor is positioned on cap-002 (index 1)
	if m.browse.cursor != 1 {
		t.Errorf("cursor = %d, want 1 (cap-002)", m.browse.cursor)
	}
	// And: lastDispatchedID is cleared
	if m.lastDispatchedID != "" {
		t.Errorf("lastDispatchedID = %q, want empty after restore", m.lastDispatchedID)
	}
}

func TestSummary_StickyCursor_FallsBackToZeroIfNotFound(t *testing.T) {
	// Given: a model that returned to browse with lastDispatchedID="cap-999"
	lister := &stubLister{beads: sampleBeads()}
	m := NewModel(WithBeadLister(lister))
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)
	m.lastDispatchedID = "cap-999"

	// When: a BeadListMsg arrives without cap-999
	updated, _ = m.Update(BeadListMsg{Beads: sampleBeads()})
	m = updated.(Model)

	// Then: the cursor stays at 0 (default)
	if m.browse.cursor != 0 {
		t.Errorf("cursor = %d, want 0 (fallback)", m.browse.cursor)
	}
	// And: lastDispatchedID is cleared
	if m.lastDispatchedID != "" {
		t.Errorf("lastDispatchedID = %q, want empty after fallback", m.lastDispatchedID)
	}
}

func TestSummary_StickyCursor_AbortSkipsLastDispatchedID(t *testing.T) {
	// Given: a model that is aborting a pipeline
	lister := &stubLister{beads: sampleBeads()}
	m := NewModel(WithBeadLister(lister))
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)
	m.mode = ModePipeline
	m.aborting = true
	m.cancelPipeline = func() {}
	m.dispatchedBeadID = "cap-002"

	// When: channelClosedMsg is received (abort completes)
	updated, _ = m.Update(channelClosedMsg{})
	m = updated.(Model)

	// Then: lastDispatchedID is not set (abort should not restore cursor)
	if m.lastDispatchedID != "" {
		t.Errorf("lastDispatchedID = %q, want empty after abort", m.lastDispatchedID)
	}
}

func TestSummary_StickyCursor_CampaignReturnSetsLastDispatchedID(t *testing.T) {
	// Given: a model in campaign summary mode
	lister := &stubLister{beads: sampleBeads()}
	m := NewModel(WithBeadLister(lister))
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)
	m.mode = ModeCampaignSummary
	m.dispatchedBeadID = "cap-002"

	// When: any key is pressed to return to browse
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)

	// Then: lastDispatchedID is set for campaign return too
	if m.lastDispatchedID != "cap-002" {
		t.Errorf("lastDispatchedID = %q, want %q", m.lastDispatchedID, "cap-002")
	}
}

// --- Status line tests ---

func TestSummary_PostPipelineDoneMsg_SetsStatusLine(t *testing.T) {
	// Given: a model in browse mode after returning from a pipeline
	m := newSizedModel(90, 40)

	// When: a PostPipelineDoneMsg arrives with no error
	updated, cmd := m.Update(PostPipelineDoneMsg{BeadID: "cap-001"})
	m = updated.(Model)

	// Then: statusMsg is set with a success message
	if m.statusMsg == "" {
		t.Fatal("statusMsg should be set after PostPipelineDoneMsg")
	}
	if !strings.Contains(m.statusMsg, "cap-001") {
		t.Errorf("statusMsg = %q, should contain bead ID", m.statusMsg)
	}
	// And: a clear command is returned (5s timer)
	if cmd == nil {
		t.Fatal("PostPipelineDoneMsg should produce a status clear timer")
	}
}

func TestSummary_PostPipelineDoneMsg_SetsStatusLineOnError(t *testing.T) {
	// Given: a model in browse mode
	m := newSizedModel(90, 40)

	// When: a PostPipelineDoneMsg arrives with an error
	updated, cmd := m.Update(PostPipelineDoneMsg{BeadID: "cap-001", Err: fmt.Errorf("merge failed")})
	m = updated.(Model)

	// Then: statusMsg is set with a failure message
	if m.statusMsg == "" {
		t.Fatal("statusMsg should be set after PostPipelineDoneMsg with error")
	}
	if !strings.Contains(m.statusMsg, "failed") {
		t.Errorf("statusMsg = %q, should contain 'failed'", m.statusMsg)
	}
	// And: a clear command is returned
	if cmd == nil {
		t.Fatal("PostPipelineDoneMsg with error should produce a status clear timer")
	}
}

func TestSummary_StatusClearMsg_ClearsStatus(t *testing.T) {
	// Given: a model with an active status message
	m := newSizedModel(90, 40)
	m.statusMsg = "cap-001: post-pipeline complete"

	// When: a statusClearMsg is received
	updated, cmd := m.Update(statusClearMsg{})
	m = updated.(Model)

	// Then: the status message is cleared
	if m.statusMsg != "" {
		t.Errorf("statusMsg = %q, want empty after statusClearMsg", m.statusMsg)
	}
	if cmd != nil {
		t.Error("statusClearMsg should not produce a command")
	}
}

func TestSummary_StatusLine_RenderedInBrowseView(t *testing.T) {
	// Given: a model in browse mode with a status message
	lister := &stubLister{beads: sampleBeads()}
	m := NewModel(WithBeadLister(lister))
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)
	updated, _ = m.Update(BeadListMsg{Beads: sampleBeads()})
	m = updated.(Model)
	m.statusMsg = "cap-001: post-pipeline complete"

	// When: the view is rendered
	view := m.View()
	plain := stripANSI(view)

	// Then: the status message appears in the view
	if !strings.Contains(plain, "post-pipeline complete") {
		t.Errorf("browse view should show status message, got:\n%s", plain)
	}
}

func TestSummary_StatusLine_NotRenderedWhenEmpty(t *testing.T) {
	// Given: a model in browse mode with no status message
	lister := &stubLister{beads: sampleBeads()}
	m := NewModel(WithBeadLister(lister))
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)
	updated, _ = m.Update(BeadListMsg{Beads: sampleBeads()})
	m = updated.(Model)

	// When: the view is rendered
	view := m.View()
	plain := stripANSI(view)

	// Then: no status line appears between panes and help
	if strings.Contains(plain, "post-pipeline") {
		t.Errorf("browse view should not show status when empty, got:\n%s", plain)
	}
}
