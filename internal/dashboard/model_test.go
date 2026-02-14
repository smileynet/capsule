package dashboard

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// stubResolver implements BeadResolver for tests.
type stubResolver struct {
	details map[string]BeadDetail
	err     error
	calls   int
}

func (s *stubResolver) Resolve(id string) (BeadDetail, error) {
	s.calls++
	if s.err != nil {
		return BeadDetail{}, s.err
	}
	if d, ok := s.details[id]; ok {
		return d, nil
	}
	return BeadDetail{}, fmt.Errorf("not found: %s", id)
}

func sampleDetail() BeadDetail {
	return BeadDetail{
		ID:           "cap-001",
		Title:        "First task",
		Priority:     1,
		Type:         "task",
		Description:  "Implement the first feature.",
		Acceptance:   "Tests pass.",
		EpicID:       "cap-e01",
		EpicTitle:    "Sample Epic",
		FeatureID:    "cap-f01",
		FeatureTitle: "Sample Feature",
	}
}

func newSizedModel(w, h int) Model {
	m := NewModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return updated.(Model)
}

func TestNewModel_DefaultMode(t *testing.T) {
	m := NewModel()
	if m.mode != ModeBrowse {
		t.Errorf("mode = %d, want ModeBrowse (%d)", m.mode, ModeBrowse)
	}
}

func TestNewModel_DefaultFocus(t *testing.T) {
	m := NewModel()
	if m.focus != PaneLeft {
		t.Errorf("focus = %d, want PaneLeft (%d)", m.focus, PaneLeft)
	}
}

func TestModel_TabTogglesFocus(t *testing.T) {
	m := newSizedModel(90, 40)

	// Tab should switch from left to right.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.focus != PaneRight {
		t.Errorf("after first Tab: focus = %d, want PaneRight (%d)", m.focus, PaneRight)
	}

	// Tab again should switch back to left.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.focus != PaneLeft {
		t.Errorf("after second Tab: focus = %d, want PaneLeft (%d)", m.focus, PaneLeft)
	}
}

func TestModel_QuitInBrowseMode(t *testing.T) {
	m := newSizedModel(90, 40)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("q in browse mode should return a quit command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("q command produced %T, want tea.QuitMsg", msg)
	}
}

func TestModel_CtrlCQuits(t *testing.T) {
	m := newSizedModel(90, 40)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("ctrl+c should return a quit command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("ctrl+c command produced %T, want tea.QuitMsg", msg)
	}
}

func TestModel_WindowSizeMsg(t *testing.T) {
	m := NewModel()

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 50})
	m = updated.(Model)

	if m.width != 120 {
		t.Errorf("width = %d, want 120", m.width)
	}
	if m.height != 50 {
		t.Errorf("height = %d, want 50", m.height)
	}
}

func TestModel_ModeRouting(t *testing.T) {
	tests := []struct {
		name string
		mode Mode
	}{
		{"browse", ModeBrowse},
		{"pipeline", ModePipeline},
		{"summary", ModeSummary},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newSizedModel(90, 40)
			m.mode = tt.mode

			view := m.View()
			if view == "" {
				t.Error("View() returned empty string")
			}
		})
	}
}

func TestModel_WindowResizeUpdatesLayout(t *testing.T) {
	m := NewModel()

	// Set initial size.
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	m = updated.(Model)
	if m.width != 80 || m.height != 30 {
		t.Errorf("after first resize: %dx%d, want 80x30", m.width, m.height)
	}

	// Resize again.
	updated, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 50})
	m = updated.(Model)
	if m.width != 120 || m.height != 50 {
		t.Errorf("after second resize: %dx%d, want 120x50", m.width, m.height)
	}
}

func TestResolveBeadCmd_ReturnsBeadResolvedMsg(t *testing.T) {
	resolver := &stubResolver{details: map[string]BeadDetail{
		"cap-001": sampleDetail(),
	}}
	cmd := resolveBeadCmd(resolver, "cap-001")
	if cmd == nil {
		t.Fatal("resolveBeadCmd should return a non-nil command")
	}
	msg := cmd()
	resolved, ok := msg.(BeadResolvedMsg)
	if !ok {
		t.Fatalf("command produced %T, want BeadResolvedMsg", msg)
	}
	if resolved.ID != "cap-001" {
		t.Errorf("resolved ID = %q, want %q", resolved.ID, "cap-001")
	}
	if resolved.Err != nil {
		t.Errorf("unexpected error: %v", resolved.Err)
	}
	if resolved.Detail.Title != "First task" {
		t.Errorf("resolved title = %q, want %q", resolved.Detail.Title, "First task")
	}
}

func TestResolveBeadCmd_ReturnsError(t *testing.T) {
	resolver := &stubResolver{err: fmt.Errorf("resolve failed")}
	cmd := resolveBeadCmd(resolver, "cap-999")
	msg := cmd().(BeadResolvedMsg)
	if msg.Err == nil {
		t.Fatal("expected error from resolver")
	}
	if msg.ID != "cap-999" {
		t.Errorf("resolved ID = %q, want %q", msg.ID, "cap-999")
	}
}

func TestFormatBeadDetail_ContainsAllFields(t *testing.T) {
	detail := sampleDetail()
	text := formatBeadDetail(detail)

	for _, want := range []string{
		"cap-001",
		"First task",
		"task",
		"Implement the first feature.",
		"Tests pass.",
		"cap-e01",
		"Sample Epic",
		"cap-f01",
		"Sample Feature",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("formatBeadDetail should contain %q, got:\n%s", want, text)
		}
	}
}

func TestFormatBeadDetail_OmitsEmptyHierarchy(t *testing.T) {
	detail := BeadDetail{
		ID:          "cap-solo",
		Title:       "Standalone",
		Priority:    2,
		Type:        "bug",
		Description: "Fix the thing.",
	}
	text := formatBeadDetail(detail)
	if strings.Contains(text, "Epic:") {
		t.Errorf("should not contain Epic header for empty epic, got:\n%s", text)
	}
	if strings.Contains(text, "Feature:") {
		t.Errorf("should not contain Feature header for empty feature, got:\n%s", text)
	}
}

func newResolverModel(w, h int) (Model, *stubResolver) {
	resolver := &stubResolver{details: map[string]BeadDetail{
		"cap-001": sampleDetail(),
		"cap-002": {ID: "cap-002", Title: "Second task", Priority: 2, Type: "feature", Description: "Second desc."},
		"cap-003": {ID: "cap-003", Title: "Third task", Priority: 3, Type: "task", Description: "Third desc."},
	}}
	lister := &stubLister{beads: sampleBeads()}
	m := NewModel(WithBeadLister(lister), WithBeadResolver(resolver))
	updated, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = updated.(Model)
	// Deliver the bead list.
	cmd := m.Init()
	msg := cmd()
	updated, _ = m.Update(msg)
	m = updated.(Model)
	return m, resolver
}

func TestModel_BeadListTriggersResolve(t *testing.T) {
	m, _ := newResolverModel(90, 40)

	// After bead list loads, a resolve command should be returned.
	// The maybeResolve should have fired for the first bead.
	// Since cursor is at 0 ("cap-001") and cache is empty, resolvingID should be set.
	if m.resolvingID != "cap-001" {
		t.Errorf("resolvingID = %q, want %q", m.resolvingID, "cap-001")
	}
	if m.detailID != "cap-001" {
		t.Errorf("detailID = %q, want %q", m.detailID, "cap-001")
	}
}

func TestModel_CursorMoveTriggersResolve(t *testing.T) {
	m, resolver := newResolverModel(90, 40)

	// Deliver initial resolve result.
	updated, _ := m.Update(BeadResolvedMsg{ID: "cap-001", Detail: sampleDetail()})
	m = updated.(Model)
	resolver.calls = 0

	// Move cursor down → should trigger resolve for cap-002.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)

	if m.detailID != "cap-002" {
		t.Errorf("detailID = %q, want %q", m.detailID, "cap-002")
	}
	if m.resolvingID != "cap-002" {
		t.Errorf("resolvingID = %q, want %q", m.resolvingID, "cap-002")
	}
	if cmd == nil {
		t.Fatal("cursor move should produce a resolve command")
	}
}

func TestModel_CacheMissTriggersResolve(t *testing.T) {
	m, resolver := newResolverModel(90, 40)

	// Deliver initial resolve.
	updated, _ := m.Update(BeadResolvedMsg{ID: "cap-001", Detail: sampleDetail()})
	m = updated.(Model)
	resolver.calls = 0

	// Move to cap-002 (not cached).
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if cmd == nil {
		t.Fatal("cache miss should produce a resolve command")
	}

	// Execute the command to call the resolver.
	msg := cmd()
	resolved := msg.(BeadResolvedMsg)
	if resolved.ID != "cap-002" {
		t.Errorf("resolved ID = %q, want %q", resolved.ID, "cap-002")
	}
	if resolver.calls != 1 {
		t.Errorf("resolver.calls = %d, want 1", resolver.calls)
	}
}

func TestModel_CacheHitSkipsResolve(t *testing.T) {
	m, resolver := newResolverModel(90, 40)

	// Deliver initial resolve and cache it.
	updated, _ := m.Update(BeadResolvedMsg{ID: "cap-001", Detail: sampleDetail()})
	m = updated.(Model)

	// Move to cap-002, resolve it, cache it.
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	msg := cmd()
	updated, _ = m.Update(msg)
	m = updated.(Model)
	resolver.calls = 0

	// Move back to cap-001 (already cached).
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)

	if m.resolvingID != "" {
		t.Errorf("resolvingID = %q, want empty for cached bead", m.resolvingID)
	}
	if cmd != nil {
		t.Error("cache hit should not produce a resolve command")
	}
	if resolver.calls != 0 {
		t.Errorf("resolver.calls = %d, want 0 (cache hit)", resolver.calls)
	}
}

func TestModel_BeadResolvedMsgUpdatesCache(t *testing.T) {
	m, _ := newResolverModel(90, 40)

	detail := sampleDetail()
	updated, _ := m.Update(BeadResolvedMsg{ID: "cap-001", Detail: detail})
	m = updated.(Model)

	cached, ok := m.cache.Get("cap-001")
	if !ok {
		t.Fatal("expected cache hit after BeadResolvedMsg")
	}
	if cached.Title != "First task" {
		t.Errorf("cached title = %q, want %q", cached.Title, "First task")
	}
	if m.resolvingID != "" {
		t.Errorf("resolvingID = %q, want empty after successful resolve", m.resolvingID)
	}
	if m.resolveErr != nil {
		t.Errorf("resolveErr should be nil, got %v", m.resolveErr)
	}
}

func TestModel_BeadResolvedMsgError(t *testing.T) {
	m, _ := newResolverModel(90, 40)

	updated, _ := m.Update(BeadResolvedMsg{ID: "cap-001", Err: fmt.Errorf("network error")})
	m = updated.(Model)

	if m.resolveErr == nil {
		t.Fatal("expected resolveErr to be set")
	}
	if m.resolvingID != "" {
		t.Errorf("resolvingID = %q, want empty after error", m.resolvingID)
	}
}

func TestModel_StaleResolveDoesNotClearLoading(t *testing.T) {
	m, _ := newResolverModel(90, 40)

	// Initial state: resolving cap-001.
	if m.resolvingID != "cap-001" {
		t.Fatalf("resolvingID = %q, want %q", m.resolvingID, "cap-001")
	}

	// Deliver cap-001 resolve, then move to cap-002 (triggers new resolve).
	updated, _ := m.Update(BeadResolvedMsg{ID: "cap-001", Detail: sampleDetail()})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)

	// Now resolving cap-002. Move again to cap-003 (triggers another resolve).
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)

	if m.resolvingID != "cap-003" {
		t.Fatalf("resolvingID = %q, want %q", m.resolvingID, "cap-003")
	}

	// Stale cap-002 result arrives — should NOT clear resolvingID since cap-003 is in flight.
	updated, _ = m.Update(BeadResolvedMsg{
		ID:     "cap-002",
		Detail: BeadDetail{ID: "cap-002", Title: "Second task"},
	})
	m = updated.(Model)

	// cap-002 should be cached.
	if _, ok := m.cache.Get("cap-002"); !ok {
		t.Fatal("stale resolve should still be cached")
	}

	// resolvingID should still be cap-003 (in flight).
	if m.resolvingID != "cap-003" {
		t.Errorf("resolvingID = %q, want %q after stale resolve", m.resolvingID, "cap-003")
	}

	// Now cap-003 arrives — this should clear resolvingID.
	updated, _ = m.Update(BeadResolvedMsg{
		ID:     "cap-003",
		Detail: BeadDetail{ID: "cap-003", Title: "Third task"},
	})
	m = updated.(Model)
	if m.resolvingID != "" {
		t.Errorf("resolvingID = %q, want empty after current resolve", m.resolvingID)
	}
}

func TestModel_ViewRightShowsLoading(t *testing.T) {
	m, _ := newResolverModel(90, 40)

	// After bead list load, resolving is true for first bead.
	view := m.View()
	if !containsPlainText(view, "Loading") {
		t.Errorf("right pane should show loading indicator, got:\n%s", stripANSI(view))
	}
}

func TestModel_ViewRightShowsDetail(t *testing.T) {
	m, _ := newResolverModel(90, 40)

	// Deliver resolved detail.
	updated, _ := m.Update(BeadResolvedMsg{ID: "cap-001", Detail: sampleDetail()})
	m = updated.(Model)

	view := m.View()
	plain := stripANSI(view)
	if !strings.Contains(plain, "First task") {
		t.Errorf("right pane should show detail title, got:\n%s", plain)
	}
	if !strings.Contains(plain, "Implement the first feature.") {
		t.Errorf("right pane should show description, got:\n%s", plain)
	}
}

func TestModel_ViewRightShowsError(t *testing.T) {
	m, _ := newResolverModel(90, 40)

	// Deliver error.
	updated, _ := m.Update(BeadResolvedMsg{ID: "cap-001", Err: fmt.Errorf("network error")})
	m = updated.(Model)

	view := m.View()
	plain := stripANSI(view)
	if !strings.Contains(plain, "network error") {
		t.Errorf("right pane should show error message, got:\n%s", plain)
	}
}

func TestModel_RightPaneScrollKeys(t *testing.T) {
	m, _ := newResolverModel(90, 10) // Short height to enable scrolling.

	// Deliver a large detail to ensure viewport has scrollable content.
	bigDetail := sampleDetail()
	bigDetail.Description = strings.Repeat("Line of text\n", 50)
	updated, _ := m.Update(BeadResolvedMsg{ID: "cap-001", Detail: bigDetail})
	m = updated.(Model)

	// Switch to right pane.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.focus != PaneRight {
		t.Fatal("expected focus on right pane after tab")
	}

	// Press down arrow — viewport should scroll.
	initialOffset := m.viewport.YOffset
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	if m.viewport.YOffset <= initialOffset {
		t.Errorf("viewport YOffset should increase after down key, got %d (was %d)", m.viewport.YOffset, initialOffset)
	}
}

func TestModel_RefreshInvalidatesCache(t *testing.T) {
	m, _ := newResolverModel(90, 40)

	// Resolve and cache cap-001.
	updated, _ := m.Update(BeadResolvedMsg{ID: "cap-001", Detail: sampleDetail()})
	m = updated.(Model)

	if _, ok := m.cache.Get("cap-001"); !ok {
		t.Fatal("expected cap-001 in cache before refresh")
	}

	// Simulate refresh: browseState emits RefreshBeadsMsg.
	updated, cmd := m.Update(RefreshBeadsMsg{})
	m = updated.(Model)

	if _, ok := m.cache.Get("cap-001"); ok {
		t.Fatal("expected cache to be empty after refresh")
	}
	if cmd == nil {
		t.Fatal("refresh should produce a fetch command")
	}
}

func newPipelineModel(w, h int, phases []string) Model {
	m := NewModel()
	m.mode = ModePipeline
	m.pipeline = newPipelineState(phases)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return updated.(Model)
}

// mockRunner implements PipelineRunner for testing.
type mockRunner struct {
	events []PhaseUpdateMsg
	output PipelineOutput
	err    error
}

func (r *mockRunner) RunPipeline(_ context.Context, _ PipelineInput, statusFn func(PhaseUpdateMsg)) (PipelineOutput, error) {
	for _, e := range r.events {
		statusFn(e)
	}
	return r.output, r.err
}

// drainPipeline pumps all events from m.eventCh through Update until
// channelClosedMsg transitions the model to summary mode.
func drainPipeline(t *testing.T, m Model) Model {
	t.Helper()
	for i := 0; i < 20; i++ { // safety limit
		cmd := listenForEvents(m.eventCh)
		if cmd == nil {
			break
		}
		msg := cmd()
		updated, _ := m.Update(msg)
		m = updated.(Model)
		if _, ok := msg.(channelClosedMsg); ok {
			break
		}
	}
	return m
}

func TestModel_PipelineModeViewShowsPhases(t *testing.T) {
	m := newPipelineModel(90, 40, []string{"plan", "code", "test"})

	view := m.View()
	plain := stripANSI(view)
	for _, name := range []string{"plan", "code", "test"} {
		if !strings.Contains(plain, name) {
			t.Errorf("pipeline view should contain phase %q, got:\n%s", name, plain)
		}
	}
}

func TestModel_PhaseUpdateMsgRoutes(t *testing.T) {
	m := newPipelineModel(90, 40, []string{"plan", "code"})

	updated, _ := m.Update(PhaseUpdateMsg{Phase: "plan", Status: PhaseRunning})
	m = updated.(Model)

	if m.pipeline.phases[0].Status != PhaseRunning {
		t.Errorf("phase 'plan' status = %q, want running", m.pipeline.phases[0].Status)
	}
}

func TestModel_PipelineKeyRoutesLeft(t *testing.T) {
	m := newPipelineModel(90, 40, []string{"plan", "code", "test"})

	// Left pane focused: down key should move pipeline cursor.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)

	if m.pipeline.cursor != 1 {
		t.Errorf("pipeline cursor = %d, want 1 after down key", m.pipeline.cursor)
	}
}

func TestModel_PipelineQuitDoesNotQuitInPipelineMode(t *testing.T) {
	m := newPipelineModel(90, 40, []string{"plan"})

	// q should cancel the pipeline, not quit the program.
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); ok {
			t.Error("q should not quit in pipeline mode")
		}
	}
}

func TestModel_PipelineRightPaneShowsReport(t *testing.T) {
	m := newPipelineModel(90, 40, []string{"plan", "code"})

	// Mark plan as passed with a summary.
	updated, _ := m.Update(PhaseUpdateMsg{
		Phase:    "plan",
		Status:   PhasePassed,
		Duration: 2 * time.Second,
		Summary:  "Planning complete",
	})
	m = updated.(Model)

	view := m.View()
	plain := stripANSI(view)
	if !strings.Contains(plain, "Passed") {
		t.Errorf("pipeline right pane should show report status, got:\n%s", plain)
	}
	if !strings.Contains(plain, "Planning complete") {
		t.Errorf("pipeline right pane should show report summary, got:\n%s", plain)
	}
}

func TestModel_PipelineRightPaneShowsWaiting(t *testing.T) {
	m := newPipelineModel(90, 40, []string{"plan", "code"})

	view := m.View()
	plain := stripANSI(view)
	if !strings.Contains(plain, "Waiting") {
		t.Errorf("pipeline right pane should show 'Waiting' for pending phase, got:\n%s", plain)
	}
}

func TestModel_HelpBarReflectsMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     Mode
		wantText string
	}{
		{"browse", ModeBrowse, "run pipeline"},
		{"pipeline", ModePipeline, "abort"},
		{"summary", ModeSummary, "continue"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newSizedModel(90, 40)
			m.mode = tt.mode

			view := m.View()
			if !containsPlainText(view, tt.wantText) {
				t.Errorf("View() should contain %q", tt.wantText)
			}
		})
	}
}

// --- listenForEvents unit tests ---

func TestListenForEvents_NilChannel(t *testing.T) {
	cmd := listenForEvents(nil)
	if cmd != nil {
		t.Error("listenForEvents(nil) should return nil")
	}
}

func TestListenForEvents_ReceivesEvent(t *testing.T) {
	ch := make(chan tea.Msg, 1)
	ch <- PhaseUpdateMsg{Phase: "plan", Status: PhaseRunning}

	cmd := listenForEvents(ch)
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	pu, ok := msg.(PhaseUpdateMsg)
	if !ok {
		t.Fatalf("expected PhaseUpdateMsg, got %T", msg)
	}
	if pu.Phase != "plan" {
		t.Errorf("Phase = %q, want %q", pu.Phase, "plan")
	}
}

func TestListenForEvents_ClosedChannel(t *testing.T) {
	ch := make(chan tea.Msg)
	close(ch)

	cmd := listenForEvents(ch)
	msg := cmd()
	if _, ok := msg.(channelClosedMsg); !ok {
		t.Fatalf("expected channelClosedMsg, got %T", msg)
	}
}

// --- dispatchPipeline unit tests ---

func TestDispatchPipeline_SendsEventsAndDone(t *testing.T) {
	runner := &mockRunner{
		events: []PhaseUpdateMsg{
			{Phase: "plan", Status: PhaseRunning},
			{Phase: "plan", Status: PhasePassed, Duration: time.Second},
		},
		output: PipelineOutput{Success: true},
	}
	ch := make(chan tea.Msg, 16)
	ctx := context.Background()

	dispatchPipeline(ctx, runner, PipelineInput{BeadID: "cap-001"}, ch)

	// First two messages: PhaseUpdateMsgs.
	for i, want := range []PhaseStatus{PhaseRunning, PhasePassed} {
		msg := <-ch
		pu, ok := msg.(PhaseUpdateMsg)
		if !ok {
			t.Fatalf("event %d: expected PhaseUpdateMsg, got %T", i, msg)
		}
		if pu.Status != want {
			t.Errorf("event %d: status = %q, want %q", i, pu.Status, want)
		}
	}

	// Third message: PipelineDoneMsg.
	msg := <-ch
	done, ok := msg.(PipelineDoneMsg)
	if !ok {
		t.Fatalf("expected PipelineDoneMsg, got %T", msg)
	}
	if !done.Output.Success {
		t.Error("expected Success = true")
	}

	// Channel should be closed.
	_, ok = <-ch
	if ok {
		t.Error("channel should be closed after dispatchPipeline returns")
	}
}

func TestDispatchPipeline_SendsError(t *testing.T) {
	runner := &mockRunner{err: fmt.Errorf("pipeline failed")}
	ch := make(chan tea.Msg, 16)

	dispatchPipeline(context.Background(), runner, PipelineInput{}, ch)

	msg := <-ch
	errMsg, ok := msg.(PipelineErrorMsg)
	if !ok {
		t.Fatalf("expected PipelineErrorMsg, got %T", msg)
	}
	if errMsg.Err == nil || errMsg.Err.Error() != "pipeline failed" {
		t.Errorf("unexpected error: %v", errMsg.Err)
	}

	_, ok = <-ch
	if ok {
		t.Error("channel should be closed")
	}
}

// --- Model dispatch wiring tests ---

func TestModel_DispatchWithRunnerTransitions(t *testing.T) {
	runner := &mockRunner{output: PipelineOutput{Success: true}}
	m := NewModel(
		WithPipelineRunner(runner),
		WithPhaseNames([]string{"plan"}),
	)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)

	updated, cmd := m.Update(DispatchMsg{BeadID: "cap-001"})
	m = updated.(Model)

	if m.mode != ModePipeline {
		t.Errorf("mode = %d, want ModePipeline (%d)", m.mode, ModePipeline)
	}
	if m.cancelPipeline == nil {
		t.Error("cancelPipeline should be set")
	}
	if m.eventCh == nil {
		t.Error("eventCh should be set")
	}
	if cmd == nil {
		t.Error("dispatch should return a command")
	}
}

func TestModel_DispatchWithoutRunnerIgnored(t *testing.T) {
	m := newSizedModel(90, 40)

	updated, cmd := m.Update(DispatchMsg{BeadID: "cap-001"})
	m = updated.(Model)

	if m.mode != ModeBrowse {
		t.Errorf("mode = %d, want ModeBrowse (%d)", m.mode, ModeBrowse)
	}
	if cmd != nil {
		t.Error("should return nil command without runner")
	}
}

func TestModel_DispatchResetsState(t *testing.T) {
	runner := &mockRunner{output: PipelineOutput{Success: true}}
	m := NewModel(
		WithPipelineRunner(runner),
		WithPhaseNames([]string{"plan"}),
	)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)
	m.focus = PaneRight
	m.pipelineOutput = &PipelineOutput{}
	m.pipelineErr = fmt.Errorf("old error")

	updated, _ = m.Update(DispatchMsg{BeadID: "cap-001"})
	m = updated.(Model)

	if m.focus != PaneLeft {
		t.Error("dispatch should reset focus to left pane")
	}
	if m.pipelineOutput != nil {
		t.Error("dispatch should clear pipelineOutput")
	}
	if m.pipelineErr != nil {
		t.Error("dispatch should clear pipelineErr")
	}
}

// --- Channel message handler tests ---

func TestModel_ChannelClosedTransitionsToSummary(t *testing.T) {
	m := newSizedModel(90, 40)
	m.mode = ModePipeline
	m.cancelPipeline = func() {}

	updated, _ := m.Update(channelClosedMsg{})
	m = updated.(Model)

	if m.mode != ModeSummary {
		t.Errorf("mode = %d, want ModeSummary (%d)", m.mode, ModeSummary)
	}
	if m.cancelPipeline != nil {
		t.Error("cancelPipeline should be nil after channel closed")
	}
	if m.eventCh != nil {
		t.Error("eventCh should be nil after channel closed")
	}
}

func TestModel_PipelineDoneStoresOutput(t *testing.T) {
	m := newSizedModel(90, 40)
	m.mode = ModePipeline

	output := PipelineOutput{Success: true}
	updated, _ := m.Update(PipelineDoneMsg{Output: output})
	m = updated.(Model)

	if m.pipelineOutput == nil {
		t.Fatal("pipelineOutput should be set")
	}
	if !m.pipelineOutput.Success {
		t.Error("pipelineOutput.Success should be true")
	}
}

func TestModel_PipelineErrorStoresErr(t *testing.T) {
	m := newSizedModel(90, 40)
	m.mode = ModePipeline

	updated, _ := m.Update(PipelineErrorMsg{Err: fmt.Errorf("boom")})
	m = updated.(Model)

	if m.pipelineErr == nil {
		t.Fatal("pipelineErr should be set")
	}
}

// --- Abort tests ---

func TestModel_PipelineQuitCancels(t *testing.T) {
	var cancelled bool
	m := newSizedModel(90, 40)
	m.mode = ModePipeline
	m.cancelPipeline = func() { cancelled = true }

	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	if !cancelled {
		t.Error("q in pipeline mode should cancel the pipeline")
	}
}

func TestModel_PipelineCtrlCCancels(t *testing.T) {
	var cancelled bool
	m := newSizedModel(90, 40)
	m.mode = ModePipeline
	m.cancelPipeline = func() { cancelled = true }

	m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

	if !cancelled {
		t.Error("ctrl+c in pipeline mode should cancel the pipeline")
	}
}

// --- Integration test: full pipeline flow ---

func TestModel_PipelineFullFlow(t *testing.T) {
	runner := &mockRunner{
		events: []PhaseUpdateMsg{
			{Phase: "plan", Status: PhaseRunning},
			{Phase: "plan", Status: PhasePassed, Duration: time.Second},
		},
		output: PipelineOutput{Success: true},
	}
	m := NewModel(
		WithPipelineRunner(runner),
		WithPhaseNames([]string{"plan"}),
	)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)

	// Dispatch starts the pipeline.
	updated, _ = m.Update(DispatchMsg{BeadID: "cap-001"})
	m = updated.(Model)

	if m.mode != ModePipeline {
		t.Fatal("should be in pipeline mode after dispatch")
	}

	// Drain all events through the channel pump.
	m = drainPipeline(t, m)

	if m.mode != ModeSummary {
		t.Errorf("mode = %d, want ModeSummary after pipeline completes", m.mode)
	}
	if m.pipelineOutput == nil {
		t.Fatal("pipelineOutput should be set after successful pipeline")
	}
	if !m.pipelineOutput.Success {
		t.Error("pipelineOutput.Success should be true")
	}
}

func TestModel_PipelineFullFlowError(t *testing.T) {
	runner := &mockRunner{
		events: []PhaseUpdateMsg{
			{Phase: "plan", Status: PhaseRunning},
		},
		err: fmt.Errorf("build failed"),
	}
	m := NewModel(
		WithPipelineRunner(runner),
		WithPhaseNames([]string{"plan"}),
	)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)

	updated, _ = m.Update(DispatchMsg{BeadID: "cap-001"})
	m = updated.(Model)

	m = drainPipeline(t, m)

	if m.mode != ModeSummary {
		t.Errorf("mode = %d, want ModeSummary after pipeline error", m.mode)
	}
	if m.pipelineErr == nil {
		t.Fatal("pipelineErr should be set after pipeline error")
	}
}

func TestModel_PhaseUpdateReschedulesListener(t *testing.T) {
	m := newSizedModel(90, 40)
	m.mode = ModePipeline
	m.pipeline = newPipelineState([]string{"plan"})

	// Set up a channel to verify rescheduling.
	ch := make(chan tea.Msg, 1)
	ch <- PhaseUpdateMsg{Phase: "plan", Status: PhasePassed}
	m.eventCh = ch

	updated, cmd := m.Update(PhaseUpdateMsg{Phase: "plan", Status: PhaseRunning})
	m = updated.(Model)

	if m.pipeline.phases[0].Status != PhaseRunning {
		t.Errorf("phase status = %q, want running", m.pipeline.phases[0].Status)
	}
	// cmd should be non-nil (batch of pipeline update + listener reschedule).
	if cmd == nil {
		t.Error("PhaseUpdateMsg should return a command to reschedule listener")
	}
}
