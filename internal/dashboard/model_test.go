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
	// Given: a newly created model
	m := NewModel()

	// Then: the default mode is browse
	if m.mode != ModeBrowse {
		t.Errorf("mode = %d, want ModeBrowse (%d)", m.mode, ModeBrowse)
	}
}

func TestNewModel_DefaultFocus(t *testing.T) {
	// Given: a newly created model
	m := NewModel()

	// Then: the default focus is the left pane
	if m.focus != PaneLeft {
		t.Errorf("focus = %d, want PaneLeft (%d)", m.focus, PaneLeft)
	}
}

func TestModel_TabTogglesFocus(t *testing.T) {
	// Given: a sized model with left pane focused
	m := newSizedModel(90, 40)

	// When: tab is pressed
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)

	// Then: focus switches to the right pane
	if m.focus != PaneRight {
		t.Errorf("after first Tab: focus = %d, want PaneRight (%d)", m.focus, PaneRight)
	}

	// When: tab is pressed again
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)

	// Then: focus switches back to the left pane
	if m.focus != PaneLeft {
		t.Errorf("after second Tab: focus = %d, want PaneLeft (%d)", m.focus, PaneLeft)
	}
}

func TestModel_QuitInBrowseMode(t *testing.T) {
	// Given: a model in browse mode
	m := newSizedModel(90, 40)

	// When: q is pressed
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// Then: a quit command is returned
	if cmd == nil {
		t.Fatal("q in browse mode should return a quit command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("q command produced %T, want tea.QuitMsg", msg)
	}
}

func TestModel_CtrlCQuits(t *testing.T) {
	// Given: a model in browse mode
	m := newSizedModel(90, 40)

	// When: ctrl+c is pressed
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

	// Then: a quit command is returned
	if cmd == nil {
		t.Fatal("ctrl+c should return a quit command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("ctrl+c command produced %T, want tea.QuitMsg", msg)
	}
}

func TestModel_WindowSizeMsg(t *testing.T) {
	// Given: a model with no dimensions set
	m := NewModel()

	// When: a window size message is received
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 50})
	m = updated.(Model)

	// Then: width and height are stored
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
			// Given: a model in the specified mode
			m := newSizedModel(90, 40)
			m.mode = tt.mode

			// When: the view is rendered
			view := m.View()

			// Then: a non-empty view is produced
			if view == "" {
				t.Error("View() returned empty string")
			}
		})
	}
}

func TestModel_WindowResizeUpdatesLayout(t *testing.T) {
	// Given: a model with initial dimensions 80x30
	m := NewModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	m = updated.(Model)
	if m.width != 80 || m.height != 30 {
		t.Errorf("after first resize: %dx%d, want 80x30", m.width, m.height)
	}

	// When: the window is resized to 120x50
	updated, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 50})
	m = updated.(Model)

	// Then: dimensions are updated
	if m.width != 120 || m.height != 50 {
		t.Errorf("after second resize: %dx%d, want 120x50", m.width, m.height)
	}
}

func TestResolveBeadCmd_ReturnsBeadResolvedMsg(t *testing.T) {
	// Given: a resolver with cap-001 detail
	resolver := &stubResolver{details: map[string]BeadDetail{
		"cap-001": sampleDetail(),
	}}

	// When: resolveBeadCmd is called for cap-001
	cmd := resolveBeadCmd(resolver, "cap-001")

	// Then: a BeadResolvedMsg with the correct detail is produced
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
	// Given: a resolver that always returns an error
	resolver := &stubResolver{err: fmt.Errorf("resolve failed")}

	// When: resolveBeadCmd is called
	cmd := resolveBeadCmd(resolver, "cap-999")

	// Then: a BeadResolvedMsg with an error is produced
	msg := cmd().(BeadResolvedMsg)
	if msg.Err == nil {
		t.Fatal("expected error from resolver")
	}
	if msg.ID != "cap-999" {
		t.Errorf("resolved ID = %q, want %q", msg.ID, "cap-999")
	}
}

func TestFormatBeadDetail_ContainsAllFields(t *testing.T) {
	// Given: a bead detail with all fields populated
	detail := sampleDetail()

	// When: it is formatted as text
	text := formatBeadDetail(detail)

	// Then: all fields appear in the output
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
	// Given: a bead detail with no epic or feature
	detail := BeadDetail{
		ID:          "cap-solo",
		Title:       "Standalone",
		Priority:    2,
		Type:        "bug",
		Description: "Fix the thing.",
	}

	// When: it is formatted as text
	text := formatBeadDetail(detail)

	// Then: Epic and Feature headers are omitted
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
	// Deliver the bead list directly (bypasses Init batch).
	// This sets pendingResolveID via debounce; deliver the debounce tick
	// and resolve result so cap-001 is fully resolved.
	updated, _ = m.Update(BeadListMsg{Beads: sampleBeads()})
	m = updated.(Model)
	updated, _ = m.Update(resolveDebounceMsg{ID: m.pendingResolveID})
	m = updated.(Model)
	return m, resolver
}

func TestModel_BeadListTriggersResolve(t *testing.T) {
	// Given: a model with lister and resolver
	// When: the bead list is loaded (newResolverModel delivers list + debounce)
	m, _ := newResolverModel(90, 40)

	// Then: the first bead is being resolved (debounce tick already delivered)
	if m.resolvingID != "cap-001" {
		t.Errorf("resolvingID = %q, want %q", m.resolvingID, "cap-001")
	}
	if m.detailID != "cap-001" {
		t.Errorf("detailID = %q, want %q", m.detailID, "cap-001")
	}
}

func TestModel_CursorMoveTriggersResolve(t *testing.T) {
	// Given: a model with first bead already resolved
	m, resolver := newResolverModel(90, 40)
	updated, _ := m.Update(BeadResolvedMsg{ID: "cap-001", Detail: sampleDetail()})
	m = updated.(Model)
	resolver.calls = 0

	// When: cursor moves down to cap-002
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)

	// Then: a debounced resolve is initiated for cap-002
	if m.detailID != "cap-002" {
		t.Errorf("detailID = %q, want %q", m.detailID, "cap-002")
	}
	if m.pendingResolveID != "cap-002" {
		t.Errorf("pendingResolveID = %q, want %q", m.pendingResolveID, "cap-002")
	}
	if cmd == nil {
		t.Fatal("cursor move should produce a debounce command")
	}
}

func TestModel_CacheMissTriggersResolve(t *testing.T) {
	// Given: a model with cap-001 resolved but cap-002 not cached
	m, resolver := newResolverModel(90, 40)
	updated, _ := m.Update(BeadResolvedMsg{ID: "cap-001", Detail: sampleDetail()})
	m = updated.(Model)
	resolver.calls = 0

	// When: cursor moves to cap-002 (cache miss) and debounce fires
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	_, cmd := m.Update(resolveDebounceMsg{ID: "cap-002"})

	// Then: the debounce fires the resolver
	if cmd == nil {
		t.Fatal("debounce should produce a resolve command")
	}
	msgs := execBatch(t, cmd)
	var found bool
	for _, msg := range msgs {
		if resolved, ok := msg.(BeadResolvedMsg); ok {
			found = true
			if resolved.ID != "cap-002" {
				t.Errorf("resolved ID = %q, want %q", resolved.ID, "cap-002")
			}
		}
	}
	if !found {
		t.Fatal("expected BeadResolvedMsg in batch")
	}
	if resolver.calls != 1 {
		t.Errorf("resolver.calls = %d, want 1", resolver.calls)
	}
}

func TestModel_CacheHitSkipsResolve(t *testing.T) {
	// Given: a model with both cap-001 and cap-002 cached
	m, resolver := newResolverModel(90, 40)
	updated, _ := m.Update(BeadResolvedMsg{ID: "cap-001", Detail: sampleDetail()})
	m = updated.(Model)
	// Move to cap-002 and complete the debounce+resolve cycle.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, cmd := m.Update(resolveDebounceMsg{ID: "cap-002"})
	m = updated.(Model)
	for _, msg := range execBatch(t, cmd) {
		updated, _ = m.Update(msg)
		m = updated.(Model)
	}
	resolver.calls = 0

	// When: cursor moves back to cap-001 (cache hit)
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)

	// Then: no resolve command is produced and resolver is not called
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
	// Given: a model resolving cap-001
	m, _ := newResolverModel(90, 40)

	// When: the resolve result arrives
	detail := sampleDetail()
	updated, _ := m.Update(BeadResolvedMsg{ID: "cap-001", Detail: detail})
	m = updated.(Model)

	// Then: the detail is cached and resolving state is cleared
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
	// Given: a model resolving cap-001
	m, _ := newResolverModel(90, 40)

	// When: the resolve fails with an error
	updated, _ := m.Update(BeadResolvedMsg{ID: "cap-001", Err: fmt.Errorf("network error")})
	m = updated.(Model)

	// Then: the error is stored and resolving state is cleared
	if m.resolveErr == nil {
		t.Fatal("expected resolveErr to be set")
	}
	if m.resolvingID != "" {
		t.Errorf("resolvingID = %q, want empty after error", m.resolvingID)
	}
}

func TestModel_StaleResolveDoesNotClearLoading(t *testing.T) {
	// Given: a model that resolved cap-001, then moved to cap-002 and cap-003
	// rapidly, and the debounce for cap-003 has fired (so it's resolving cap-003)
	m, _ := newResolverModel(90, 40)
	updated, _ := m.Update(BeadResolvedMsg{ID: "cap-001", Detail: sampleDetail()})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	// Fire the debounce for cap-003 (the latest pending)
	updated, _ = m.Update(resolveDebounceMsg{ID: "cap-003"})
	m = updated.(Model)
	if m.resolvingID != "cap-003" {
		t.Fatalf("resolvingID = %q, want %q", m.resolvingID, "cap-003")
	}

	// When: a stale cap-002 result arrives
	updated, _ = m.Update(BeadResolvedMsg{
		ID:     "cap-002",
		Detail: BeadDetail{ID: "cap-002", Title: "Second task"},
	})
	m = updated.(Model)

	// Then: cap-002 is cached but resolvingID still points to cap-003
	if _, ok := m.cache.Get("cap-002"); !ok {
		t.Fatal("stale resolve should still be cached")
	}
	if m.resolvingID != "cap-003" {
		t.Errorf("resolvingID = %q, want %q after stale resolve", m.resolvingID, "cap-003")
	}

	// When: the current cap-003 result arrives
	updated, _ = m.Update(BeadResolvedMsg{
		ID:     "cap-003",
		Detail: BeadDetail{ID: "cap-003", Title: "Third task"},
	})
	m = updated.(Model)

	// Then: resolvingID is cleared
	if m.resolvingID != "" {
		t.Errorf("resolvingID = %q, want empty after current resolve", m.resolvingID)
	}
}

func TestModel_ViewRightShowsLoading(t *testing.T) {
	// Given: a model with a bead being resolved
	m, _ := newResolverModel(90, 40)

	// When: the view is rendered
	view := m.View()

	// Then: a loading indicator is shown in the right pane
	if !containsPlainText(view, "Loading") {
		t.Errorf("right pane should show loading indicator, got:\n%s", stripANSI(view))
	}
}

func TestModel_ViewRightShowsDetail(t *testing.T) {
	// Given: a model with cap-001 resolved
	m, _ := newResolverModel(90, 40)
	updated, _ := m.Update(BeadResolvedMsg{ID: "cap-001", Detail: sampleDetail()})
	m = updated.(Model)

	// When: the view is rendered
	view := m.View()
	plain := stripANSI(view)

	// Then: the detail title and description are shown
	if !strings.Contains(plain, "First task") {
		t.Errorf("right pane should show detail title, got:\n%s", plain)
	}
	if !strings.Contains(plain, "Implement the first feature.") {
		t.Errorf("right pane should show description, got:\n%s", plain)
	}
}

func TestModel_ViewRightShowsError(t *testing.T) {
	// Given: a model where resolve failed
	m, _ := newResolverModel(90, 40)
	updated, _ := m.Update(BeadResolvedMsg{ID: "cap-001", Err: fmt.Errorf("network error")})
	m = updated.(Model)

	// When: the view is rendered
	view := m.View()
	plain := stripANSI(view)

	// Then: the error header and message are shown
	if !strings.Contains(plain, "Could not load bead detail") {
		t.Errorf("right pane should show 'Could not load bead detail', got:\n%s", plain)
	}
	if !strings.Contains(plain, "network error") {
		t.Errorf("right pane should show error message, got:\n%s", plain)
	}
}

func TestModel_RightPaneScrollKeys(t *testing.T) {
	// Given: a model with a large detail loaded and right pane focused
	m, _ := newResolverModel(90, 10)
	bigDetail := sampleDetail()
	bigDetail.Description = strings.Repeat("Line of text\n", 50)
	updated, _ := m.Update(BeadResolvedMsg{ID: "cap-001", Detail: bigDetail})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.focus != PaneRight {
		t.Fatal("expected focus on right pane after tab")
	}

	// When: down arrow is pressed
	initialOffset := m.viewport.YOffset
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)

	// Then: the viewport scrolls down
	if m.viewport.YOffset <= initialOffset {
		t.Errorf("viewport YOffset should increase after down key, got %d (was %d)", m.viewport.YOffset, initialOffset)
	}
}

func TestModel_RefreshInvalidatesCache(t *testing.T) {
	// Given: a model with cap-001 cached
	m, _ := newResolverModel(90, 40)
	updated, _ := m.Update(BeadResolvedMsg{ID: "cap-001", Detail: sampleDetail()})
	m = updated.(Model)
	if _, ok := m.cache.Get("cap-001"); !ok {
		t.Fatal("expected cap-001 in cache before refresh")
	}

	// When: a RefreshBeadsMsg is received
	updated, cmd := m.Update(RefreshBeadsMsg{})
	m = updated.(Model)

	// Then: the cache is empty and a fetch command is returned
	if _, ok := m.cache.Get("cap-001"); ok {
		t.Fatal("expected cache to be empty after refresh")
	}
	if cmd == nil {
		t.Fatal("refresh should produce a fetch command")
	}
}

func TestModel_RefreshClearsPendingResolveID(t *testing.T) {
	// Given: a model with a pendingResolveID from a prior debounce
	m, _ := newResolverModel(90, 40)
	m.pendingResolveID = "stale-bead"
	m.resolvingID = "in-flight"
	m.resolveErr = fmt.Errorf("old error")

	// When: a RefreshBeadsMsg is received
	updated, _ := m.Update(RefreshBeadsMsg{})
	m = updated.(Model)

	// Then: all resolve state is cleared
	if m.pendingResolveID != "" {
		t.Errorf("pendingResolveID = %q, want empty after RefreshBeadsMsg", m.pendingResolveID)
	}
	if m.resolvingID != "" {
		t.Errorf("resolvingID = %q, want empty after RefreshBeadsMsg", m.resolvingID)
	}
	if m.resolveErr != nil {
		t.Errorf("resolveErr = %v, want nil after RefreshBeadsMsg", m.resolveErr)
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
	// Given: a model in pipeline mode with 3 phases
	m := newPipelineModel(90, 40, []string{"plan", "code", "test"})

	// When: the view is rendered
	view := m.View()
	plain := stripANSI(view)

	// Then: all phase names are visible
	for _, name := range []string{"plan", "code", "test"} {
		if !strings.Contains(plain, name) {
			t.Errorf("pipeline view should contain phase %q, got:\n%s", name, plain)
		}
	}
}

func TestModel_PhaseUpdateMsgRoutes(t *testing.T) {
	// Given: a model in pipeline mode
	m := newPipelineModel(90, 40, []string{"plan", "code"})

	// When: a phase update message is received
	updated, _ := m.Update(PhaseUpdateMsg{Phase: "plan", Status: PhaseRunning})
	m = updated.(Model)

	// Then: the phase status is updated
	if m.pipeline.phases[0].Status != PhaseRunning {
		t.Errorf("phase 'plan' status = %q, want running", m.pipeline.phases[0].Status)
	}
}

func TestModel_PipelineKeyRoutesLeft(t *testing.T) {
	// Given: a model in pipeline mode with left pane focused
	m := newPipelineModel(90, 40, []string{"plan", "code", "test"})

	// When: down key is pressed
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)

	// Then: the pipeline cursor moves
	if m.pipeline.cursor != 1 {
		t.Errorf("pipeline cursor = %d, want 1 after down key", m.pipeline.cursor)
	}
}

func TestModel_PipelineQuitDoesNotQuitInPipelineMode(t *testing.T) {
	// Given: a model in pipeline mode
	m := newPipelineModel(90, 40, []string{"plan"})

	// When: q is pressed
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// Then: the program does not quit (no tea.QuitMsg)
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); ok {
			t.Error("q should not quit in pipeline mode")
		}
	}
}

func TestModel_PipelineRightPaneShowsReport(t *testing.T) {
	// Given: a model in pipeline mode with "plan" passed
	m := newPipelineModel(90, 40, []string{"plan", "code"})
	updated, _ := m.Update(PhaseUpdateMsg{
		Phase:    "plan",
		Status:   PhasePassed,
		Duration: 2 * time.Second,
		Summary:  "Planning complete",
	})
	m = updated.(Model)

	// When: the view is rendered
	view := m.View()
	plain := stripANSI(view)

	// Then: the report status and summary are shown in the right pane
	if !strings.Contains(plain, "Passed") {
		t.Errorf("pipeline right pane should show report status, got:\n%s", plain)
	}
	if !strings.Contains(plain, "Planning complete") {
		t.Errorf("pipeline right pane should show report summary, got:\n%s", plain)
	}
}

func TestModel_PipelineRightPaneShowsWaiting(t *testing.T) {
	// Given: a model in pipeline mode with all phases pending
	m := newPipelineModel(90, 40, []string{"plan", "code"})

	// When: the view is rendered
	view := m.View()
	plain := stripANSI(view)

	// Then: "Waiting" is shown for the pending phase
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
			// Given: a model in the specified mode
			m := newSizedModel(90, 40)
			m.mode = tt.mode

			// When: the view is rendered
			view := m.View()

			// Then: mode-appropriate help text is shown
			if !containsPlainText(view, tt.wantText) {
				t.Errorf("View() should contain %q", tt.wantText)
			}
		})
	}
}

// --- listenForEvents unit tests ---

func TestListenForEvents_NilChannel(t *testing.T) {
	// Given: a nil channel
	// When: listenForEvents is called
	cmd := listenForEvents(nil)

	// Then: nil is returned
	if cmd != nil {
		t.Error("listenForEvents(nil) should return nil")
	}
}

func TestListenForEvents_ReceivesEvent(t *testing.T) {
	// Given: a channel with a PhaseUpdateMsg
	ch := make(chan tea.Msg, 1)
	ch <- PhaseUpdateMsg{Phase: "plan", Status: PhaseRunning}

	// When: listenForEvents reads from the channel
	cmd := listenForEvents(ch)
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()

	// Then: the PhaseUpdateMsg is received
	pu, ok := msg.(PhaseUpdateMsg)
	if !ok {
		t.Fatalf("expected PhaseUpdateMsg, got %T", msg)
	}
	if pu.Phase != "plan" {
		t.Errorf("Phase = %q, want %q", pu.Phase, "plan")
	}
}

func TestListenForEvents_ClosedChannel(t *testing.T) {
	// Given: a closed channel
	ch := make(chan tea.Msg)
	close(ch)

	// When: listenForEvents reads from the channel
	cmd := listenForEvents(ch)
	msg := cmd()

	// Then: a channelClosedMsg is returned
	if _, ok := msg.(channelClosedMsg); !ok {
		t.Fatalf("expected channelClosedMsg, got %T", msg)
	}
}

// --- dispatchPipeline unit tests ---

func TestDispatchPipeline_SendsEventsAndDone(t *testing.T) {
	// Given: a runner that emits two phase events and succeeds
	runner := &mockRunner{
		events: []PhaseUpdateMsg{
			{Phase: "plan", Status: PhaseRunning},
			{Phase: "plan", Status: PhasePassed, Duration: time.Second},
		},
		output: PipelineOutput{Success: true},
	}
	ch := make(chan tea.Msg, 16)
	ctx := context.Background()

	// When: dispatchPipeline runs to completion
	dispatchPipeline(ctx, runner, PipelineInput{BeadID: "cap-001"}, ch)

	// Then: two PhaseUpdateMsgs are sent
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

	// And: a PipelineDoneMsg with Success=true is sent
	msg := <-ch
	done, ok := msg.(PipelineDoneMsg)
	if !ok {
		t.Fatalf("expected PipelineDoneMsg, got %T", msg)
	}
	if !done.Output.Success {
		t.Error("expected Success = true")
	}

	// And: the channel is closed
	_, ok = <-ch
	if ok {
		t.Error("channel should be closed after dispatchPipeline returns")
	}
}

func TestDispatchPipeline_SendsError(t *testing.T) {
	// Given: a runner that returns an error
	runner := &mockRunner{err: fmt.Errorf("pipeline failed")}
	ch := make(chan tea.Msg, 16)

	// When: dispatchPipeline runs
	dispatchPipeline(context.Background(), runner, PipelineInput{}, ch)

	// Then: a PipelineErrorMsg is sent
	msg := <-ch
	errMsg, ok := msg.(PipelineErrorMsg)
	if !ok {
		t.Fatalf("expected PipelineErrorMsg, got %T", msg)
	}
	if errMsg.Err == nil || errMsg.Err.Error() != "pipeline failed" {
		t.Errorf("unexpected error: %v", errMsg.Err)
	}

	// And: the channel is closed
	_, ok = <-ch
	if ok {
		t.Error("channel should be closed")
	}
}

// --- Model dispatch wiring tests ---

func TestModel_DispatchWithRunnerTransitions(t *testing.T) {
	// Given: a model with a pipeline runner configured
	runner := &mockRunner{output: PipelineOutput{Success: true}}
	m := NewModel(
		WithPipelineRunner(runner),
		WithPhaseNames([]string{"plan"}),
	)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)

	// When: a DispatchMsg is received
	updated, cmd := m.Update(DispatchMsg{BeadID: "cap-001"})
	m = updated.(Model)

	// Then: the model transitions to pipeline mode with cancel and event channel set
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
	// Given: a model with no pipeline runner
	m := newSizedModel(90, 40)

	// When: a DispatchMsg is received
	updated, cmd := m.Update(DispatchMsg{BeadID: "cap-001"})
	m = updated.(Model)

	// Then: the model stays in browse mode
	if m.mode != ModeBrowse {
		t.Errorf("mode = %d, want ModeBrowse (%d)", m.mode, ModeBrowse)
	}
	if cmd != nil {
		t.Error("should return nil command without runner")
	}
}

func TestModel_DispatchPassesBeadTitleToPipeline(t *testing.T) {
	// Given: a model with a pipeline runner
	runner := &mockRunner{output: PipelineOutput{Success: true}}
	m := NewModel(
		WithPipelineRunner(runner),
		WithPhaseNames([]string{"plan"}),
	)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)

	// When: a DispatchMsg with BeadTitle is received
	updated, _ = m.Update(DispatchMsg{BeadID: "cap-042", BeadTitle: "Fix login bug"})
	m = updated.(Model)

	// Then: the pipeline state has the bead header set
	if m.pipeline.beadID != "cap-042" {
		t.Errorf("pipeline.beadID = %q, want %q", m.pipeline.beadID, "cap-042")
	}
	if m.pipeline.beadTitle != "Fix login bug" {
		t.Errorf("pipeline.beadTitle = %q, want %q", m.pipeline.beadTitle, "Fix login bug")
	}
}

func TestModel_DispatchResetsState(t *testing.T) {
	// Given: a model with stale pipeline state from a previous run
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

	// When: a new DispatchMsg is received
	updated, _ = m.Update(DispatchMsg{BeadID: "cap-001"})
	m = updated.(Model)

	// Then: focus, output, and error are reset
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
	// Given: a model in pipeline mode with a cancel function
	m := newSizedModel(90, 40)
	m.mode = ModePipeline
	m.cancelPipeline = func() {}

	// When: channelClosedMsg is received
	updated, _ := m.Update(channelClosedMsg{})
	m = updated.(Model)

	// Then: the model transitions to summary mode with cleanup
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
	// Given: a model in pipeline mode
	m := newSizedModel(90, 40)
	m.mode = ModePipeline

	// When: PipelineDoneMsg is received
	output := PipelineOutput{Success: true}
	updated, _ := m.Update(PipelineDoneMsg{Output: output})
	m = updated.(Model)

	// Then: the pipeline output is stored
	if m.pipelineOutput == nil {
		t.Fatal("pipelineOutput should be set")
	}
	if !m.pipelineOutput.Success {
		t.Error("pipelineOutput.Success should be true")
	}
}

func TestModel_PipelineErrorStoresErr(t *testing.T) {
	// Given: a model in pipeline mode
	m := newSizedModel(90, 40)
	m.mode = ModePipeline

	// When: PipelineErrorMsg is received
	updated, _ := m.Update(PipelineErrorMsg{Err: fmt.Errorf("boom")})
	m = updated.(Model)

	// Then: the pipeline error is stored
	if m.pipelineErr == nil {
		t.Fatal("pipelineErr should be set")
	}
}

// --- Abort tests ---

func TestModel_PipelineQuitCancels(t *testing.T) {
	// Given: a model in pipeline mode with a cancel function
	var cancelled bool
	m := newSizedModel(90, 40)
	m.mode = ModePipeline
	m.cancelPipeline = func() { cancelled = true }

	// When: q is pressed
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// Then: the pipeline cancel function is called
	if !cancelled {
		t.Error("q in pipeline mode should cancel the pipeline")
	}
}

func TestModel_PipelineCtrlCCancels(t *testing.T) {
	// Given: a model in pipeline mode with a cancel function
	var cancelled bool
	m := newSizedModel(90, 40)
	m.mode = ModePipeline
	m.cancelPipeline = func() { cancelled = true }

	// When: ctrl+c is pressed
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

	// Then: the pipeline cancel function is called
	if !cancelled {
		t.Error("ctrl+c in pipeline mode should cancel the pipeline")
	}
}

// --- Integration test: full pipeline flow ---

func TestModel_PipelineFullFlow(t *testing.T) {
	// Given: a model with a runner that runs one phase successfully
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

	// When: pipeline is dispatched and all events are drained
	updated, _ = m.Update(DispatchMsg{BeadID: "cap-001"})
	m = updated.(Model)
	if m.mode != ModePipeline {
		t.Fatal("should be in pipeline mode after dispatch")
	}
	m = drainPipeline(t, m)

	// Then: the model is in summary mode with successful output
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
	// Given: a model with a runner that fails
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

	// When: pipeline is dispatched and all events are drained
	updated, _ = m.Update(DispatchMsg{BeadID: "cap-001"})
	m = updated.(Model)
	m = drainPipeline(t, m)

	// Then: the model is in summary mode with an error
	if m.mode != ModeSummary {
		t.Errorf("mode = %d, want ModeSummary after pipeline error", m.mode)
	}
	if m.pipelineErr == nil {
		t.Fatal("pipelineErr should be set after pipeline error")
	}
}

func TestModel_PhaseUpdateReschedulesListener(t *testing.T) {
	// Given: a model in pipeline mode with an event channel
	m := newSizedModel(90, 40)
	m.mode = ModePipeline
	m.pipeline = newPipelineState([]string{"plan"})
	ch := make(chan tea.Msg, 1)
	ch <- PhaseUpdateMsg{Phase: "plan", Status: PhasePassed}
	m.eventCh = ch

	// When: a PhaseUpdateMsg is processed
	updated, cmd := m.Update(PhaseUpdateMsg{Phase: "plan", Status: PhaseRunning})
	m = updated.(Model)

	// Then: the phase is updated and a listener reschedule command is returned
	if m.pipeline.phases[0].Status != PhaseRunning {
		t.Errorf("phase status = %q, want running", m.pipeline.phases[0].Status)
	}
	if cmd == nil {
		t.Error("PhaseUpdateMsg should return a command to reschedule listener")
	}
}

// --- Browse spinner tests ---

func TestModel_SpinnerTickRoutesBrowseLoading(t *testing.T) {
	// Given: a model in browse mode with bead list loading
	m := NewModel(WithBeadLister(&stubLister{beads: sampleBeads()}))
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)
	// browse is loading by default (newBrowseState sets loading=true)

	// When: a spinner tick is received
	_, cmd := m.Update(m.browseSpinner.Tick())
	// Then: a tick command is returned (spinner keeps animating)
	if cmd == nil {
		t.Error("spinner tick should return a command during browse loading")
	}
}

func TestModel_SpinnerTickRoutesBrowseResolving(t *testing.T) {
	// Given: a model in browse mode resolving a bead (loading done)
	m, _ := newResolverModel(90, 40)
	// After newResolverModel, resolvingID is "cap-001" (loading right pane)

	// When: a spinner tick is received
	_, cmd := m.Update(m.browseSpinner.Tick())

	// Then: a tick command is returned (spinner keeps animating)
	if cmd == nil {
		t.Error("spinner tick should return a command during resolve loading")
	}
}

func TestModel_SpinnerTickIgnoredWhenIdle(t *testing.T) {
	// Given: a model in browse mode with nothing loading
	m, _ := newResolverModel(90, 40)
	updated, _ := m.Update(BeadResolvedMsg{ID: "cap-001", Detail: sampleDetail()})
	m = updated.(Model)

	// When: a spinner tick is received (no loading, no resolving)
	_, cmd := m.Update(m.browseSpinner.Tick())

	// Then: no command is returned (spinner stops)
	if cmd != nil {
		t.Error("spinner tick should not return a command when idle")
	}
}

func TestModel_InitReturnsBatchWithSpinner(t *testing.T) {
	// Given: a model with a BeadLister
	m := NewModel(WithBeadLister(&stubLister{beads: sampleBeads()}))

	// When: Init is called
	cmd := m.Init()

	// Then: a non-nil command is returned (batch of fetch + spinner tick)
	if cmd == nil {
		t.Fatal("Init with lister should return a batch command")
	}
}

// --- Resize edge case tests ---

func TestModel_ResizeRecalculatesViewportDimensions(t *testing.T) {
	// Given: a model sized at 80x30
	m := newSizedModel(80, 30)
	origVPWidth := m.viewport.Width
	origVPHeight := m.viewport.Height

	// When: the window is resized to 120x50
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 50})
	m = updated.(Model)

	// Then: viewport dimensions are recalculated (larger window → larger viewport)
	if m.viewport.Width <= origVPWidth {
		t.Errorf("viewport width should increase: got %d, was %d", m.viewport.Width, origVPWidth)
	}
	if m.viewport.Height <= origVPHeight {
		t.Errorf("viewport height should increase: got %d, was %d", m.viewport.Height, origVPHeight)
	}
}

func TestModel_ResizeSmallTerminalClampsMinLeft(t *testing.T) {
	// Given: a model sized at a very small width
	m := newSizedModel(40, 20)

	// When: PaneWidths is calculated for that width
	left, _ := PaneWidths(m.width)

	// Then: the left pane respects the minimum width
	if left < MinLeftWidth {
		t.Errorf("left pane = %d, want >= %d (MinLeftWidth)", left, MinLeftWidth)
	}
}

// --- Abort tests ---

func TestModel_AbortSetsAbortingFlag(t *testing.T) {
	// Given: a model in pipeline mode with a cancel function
	var cancelled bool
	m := newSizedModel(90, 40)
	m.mode = ModePipeline
	m.cancelPipeline = func() { cancelled = true }

	// When: q is pressed (first press)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(Model)

	// Then: aborting flag is set and cancel function is called
	if !m.aborting {
		t.Error("aborting should be true after first q in pipeline mode")
	}
	if !cancelled {
		t.Error("cancelPipeline should be called on first q")
	}
}

func TestModel_AbortChannelClosedTransitionsToBrowse(t *testing.T) {
	// Given: a model in pipeline mode that is aborting
	m := newSizedModel(90, 40)
	m.mode = ModePipeline
	m.aborting = true
	m.cancelPipeline = func() {}

	// When: channelClosedMsg is received
	updated, _ := m.Update(channelClosedMsg{})
	m = updated.(Model)

	// Then: the model transitions to browse mode (not summary)
	if m.mode != ModeBrowse {
		t.Errorf("mode = %d, want ModeBrowse (%d) after abort", m.mode, ModeBrowse)
	}
	if m.aborting {
		t.Error("aborting should be cleared after transition to browse")
	}
}

func TestModel_AbortDoesNotRunPostPipeline(t *testing.T) {
	// Given: a model in pipeline mode with postPipeline configured and aborting
	var postPipelineCalled bool
	lister := &stubLister{beads: sampleBeads()}
	m := NewModel(
		WithBeadLister(lister),
		WithPostPipelineFunc(func(beadID string) error {
			postPipelineCalled = true
			return nil
		}),
	)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)
	m.mode = ModePipeline
	m.aborting = true
	m.cancelPipeline = func() {}
	m.dispatchedBeadID = "cap-001"

	// When: channelClosedMsg is received
	updated, cmd := m.Update(channelClosedMsg{})
	m = updated.(Model)

	// Then: mode is browse and postPipeline is not triggered
	if m.mode != ModeBrowse {
		t.Errorf("mode = %d, want ModeBrowse", m.mode)
	}
	// Execute any returned commands to verify postPipeline is not called
	if cmd != nil {
		for _, msg := range execBatch(t, cmd) {
			if _, ok := msg.(PostPipelineDoneMsg); ok {
				t.Error("postPipeline should not fire on abort")
			}
		}
	}
	if postPipelineCalled {
		t.Error("PostPipelineFunc should not be called during abort")
	}
}

func TestModel_AbortViewShowsAbortingIndicator(t *testing.T) {
	// Given: a model in pipeline mode with a running phase and aborting
	m := newPipelineModel(90, 40, []string{"plan", "code"})
	updated, _ := m.Update(PhaseUpdateMsg{Phase: "plan", Status: PhaseRunning})
	m = updated.(Model)
	m.aborting = true
	m.pipeline.aborting = true

	// When: the view is rendered
	view := m.View()
	plain := stripANSI(view)

	// Then: "Aborting" is shown in the view
	if !strings.Contains(plain, "Aborting") {
		t.Errorf("view should show 'Aborting' during abort, got:\n%s", plain)
	}
}

// --- Double-press force quit tests ---

func TestModel_DoublePressQForceQuits(t *testing.T) {
	// Given: a model in pipeline mode that is already aborting
	m := newSizedModel(90, 40)
	m.mode = ModePipeline
	m.aborting = true

	// When: q is pressed again (second press)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// Then: a quit command is returned (force quit)
	if cmd == nil {
		t.Fatal("second q during abort should return a quit command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("second q during abort should produce tea.QuitMsg, got %T", msg)
	}
}

func TestModel_DoublePressCtrlCForceQuits(t *testing.T) {
	// Given: a model in pipeline mode that is already aborting
	m := newSizedModel(90, 40)
	m.mode = ModePipeline
	m.aborting = true

	// When: ctrl+c is pressed again (second press)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})

	// Then: a quit command is returned (force quit)
	if cmd == nil {
		t.Fatal("second ctrl+c during abort should return a quit command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("second ctrl+c during abort should produce tea.QuitMsg, got %T", msg)
	}
}

// --- Global refresh key tests ---

func TestModel_RefreshWorksFromRightPane(t *testing.T) {
	// Given: a model in browse mode with right pane focused and a lister
	lister := &stubLister{beads: sampleBeads()}
	m := NewModel(WithBeadLister(lister))
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)
	updated, _ = m.Update(BeadListMsg{Beads: sampleBeads()})
	m = updated.(Model)
	m.focus = PaneRight

	// When: r is pressed from the right pane
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	m = updated.(Model)

	// Then: a refresh is triggered (browse enters loading state)
	if !m.browse.loading {
		t.Error("r from right pane should trigger refresh (loading=true)")
	}
	if cmd == nil {
		t.Fatal("r from right pane should produce a refresh command")
	}
}

// --- Campaign wiring tests ---

// mockCampaignRunner implements CampaignRunner for tests.
type mockCampaignRunner struct {
	events []tea.Msg
	err    error
}

func (r *mockCampaignRunner) RunCampaign(
	_ context.Context,
	_ string,
	statusFn func(tea.Msg),
	_ func(context.Context, PipelineInput, func(PhaseUpdateMsg)) (PipelineOutput, error),
) error {
	for _, e := range r.events {
		statusFn(e)
	}
	return r.err
}

func TestModel_WithCampaignRunner(t *testing.T) {
	// Given: a campaign runner
	cr := &mockCampaignRunner{}

	// When: a model is created with WithCampaignRunner
	m := NewModel(WithCampaignRunner(cr))

	// Then: the campaign runner is stored
	if m.campaignRunner == nil {
		t.Error("campaignRunner should be set")
	}
}

func TestModel_DispatchFeatureGoesToCampaign(t *testing.T) {
	// Given: a model with both pipeline and campaign runners
	pr := &mockRunner{output: PipelineOutput{Success: true}}
	cr := &mockCampaignRunner{}
	m := NewModel(
		WithPipelineRunner(pr),
		WithCampaignRunner(cr),
		WithPhaseNames([]string{"plan"}),
	)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)

	// When: a feature bead is dispatched
	updated, cmd := m.Update(DispatchMsg{BeadID: "cap-feat", BeadType: "feature"})
	m = updated.(Model)

	// Then: the model transitions to campaign mode
	if m.mode != ModeCampaign {
		t.Errorf("mode = %d, want ModeCampaign (%d)", m.mode, ModeCampaign)
	}
	if m.cancelPipeline == nil {
		t.Error("cancelPipeline should be set for campaign")
	}
	if m.eventCh == nil {
		t.Error("eventCh should be set for campaign")
	}
	if cmd == nil {
		t.Error("campaign dispatch should return a command")
	}
	if m.dispatchedBeadID != "cap-feat" {
		t.Errorf("dispatchedBeadID = %q, want %q", m.dispatchedBeadID, "cap-feat")
	}
}

func TestModel_DispatchRoutesToModeByType(t *testing.T) {
	tests := []struct {
		name     string
		beadType string
		wantMode Mode
	}{
		{"epic goes to campaign", "epic", ModeCampaign},
		{"task goes to pipeline", "task", ModePipeline},
		{"empty type goes to pipeline", "", ModePipeline},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: a model with both pipeline and campaign runners
			pr := &mockRunner{output: PipelineOutput{Success: true}}
			cr := &mockCampaignRunner{}
			m := NewModel(
				WithPipelineRunner(pr),
				WithCampaignRunner(cr),
				WithPhaseNames([]string{"plan"}),
			)
			updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
			m = updated.(Model)

			// When: a bead of the given type is dispatched
			updated, _ = m.Update(DispatchMsg{BeadID: "cap-001", BeadType: tt.beadType})
			m = updated.(Model)

			// Then: the model transitions to the expected mode
			if m.mode != tt.wantMode {
				t.Errorf("mode = %d, want %d", m.mode, tt.wantMode)
			}
		})
	}
}

func TestModel_DispatchFeatureWithoutCampaignRunnerGoesToPipeline(t *testing.T) {
	// Given: a model with only pipeline runner (no campaign runner)
	pr := &mockRunner{output: PipelineOutput{Success: true}}
	m := NewModel(
		WithPipelineRunner(pr),
		WithPhaseNames([]string{"plan"}),
	)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)

	// When: a feature bead is dispatched
	updated, _ = m.Update(DispatchMsg{BeadID: "cap-feat", BeadType: "feature"})
	m = updated.(Model)

	// Then: falls back to pipeline mode (no campaign runner)
	if m.mode != ModePipeline {
		t.Errorf("mode = %d, want ModePipeline (%d) without campaign runner", m.mode, ModePipeline)
	}
}

func TestModel_CampaignDispatchResetsState(t *testing.T) {
	// Given: a model with stale state from a previous run
	cr := &mockCampaignRunner{}
	pr := &mockRunner{output: PipelineOutput{Success: true}}
	m := NewModel(
		WithPipelineRunner(pr),
		WithCampaignRunner(cr),
		WithPhaseNames([]string{"plan"}),
	)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)
	m.focus = PaneRight
	m.pipelineOutput = &PipelineOutput{}
	m.pipelineErr = fmt.Errorf("old error")

	// When: a feature bead is dispatched
	updated, _ = m.Update(DispatchMsg{BeadID: "cap-feat", BeadType: "feature"})
	m = updated.(Model)

	// Then: focus, output, and error are reset
	if m.focus != PaneLeft {
		t.Error("campaign dispatch should reset focus to left pane")
	}
	if m.pipelineOutput != nil {
		t.Error("campaign dispatch should clear pipelineOutput")
	}
	if m.pipelineErr != nil {
		t.Error("campaign dispatch should clear pipelineErr")
	}
	if m.aborting {
		t.Error("campaign dispatch should clear aborting")
	}
}

// --- dispatchCampaign tests ---

func TestDispatchCampaign_SendsEventsAndDone(t *testing.T) {
	// Given: a campaign runner that emits campaign messages and succeeds
	cr := &mockCampaignRunner{
		events: []tea.Msg{
			CampaignStartMsg{ParentID: "cap-feat", Tasks: sampleCampaignTasks()},
			CampaignTaskStartMsg{BeadID: "cap-001", Index: 0, Total: 3},
			CampaignTaskDoneMsg{BeadID: "cap-001", Index: 0, Success: true, Duration: time.Second},
			CampaignDoneMsg{ParentID: "cap-feat", TotalTasks: 3, Passed: 1, Failed: 0},
		},
	}
	pr := &mockRunner{output: PipelineOutput{Success: true}}
	ch := make(chan tea.Msg, 32)
	ctx := context.Background()

	// When: dispatchCampaign runs to completion
	dispatchCampaign(ctx, cr, pr, "cap-feat", ch)

	// Then: the campaign messages are sent through the channel
	var msgs []tea.Msg
	for msg := range ch {
		msgs = append(msgs, msg)
	}

	if len(msgs) != 4 {
		t.Fatalf("got %d messages, want 4", len(msgs))
	}
	// First should be CampaignStartMsg
	if _, ok := msgs[0].(CampaignStartMsg); !ok {
		t.Errorf("first message should be CampaignStartMsg, got %T", msgs[0])
	}
	// Last should be CampaignDoneMsg
	if _, ok := msgs[3].(CampaignDoneMsg); !ok {
		t.Errorf("last message should be CampaignDoneMsg, got %T", msgs[3])
	}
}

func TestDispatchCampaign_SendsErrorMsg(t *testing.T) {
	// Given: a campaign runner that returns an error
	cr := &mockCampaignRunner{err: fmt.Errorf("campaign failed")}
	pr := &mockRunner{output: PipelineOutput{Success: true}}
	ch := make(chan tea.Msg, 32)
	ctx := context.Background()

	// When: dispatchCampaign runs
	dispatchCampaign(ctx, cr, pr, "cap-feat", ch)

	// Then: a CampaignErrorMsg is sent through the channel
	var msgs []tea.Msg
	for msg := range ch {
		msgs = append(msgs, msg)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message (CampaignErrorMsg), got %d", len(msgs))
	}
	errMsg, ok := msgs[0].(CampaignErrorMsg)
	if !ok {
		t.Fatalf("expected CampaignErrorMsg, got %T", msgs[0])
	}
	if errMsg.Err == nil || errMsg.Err.Error() != "campaign failed" {
		t.Errorf("unexpected error: %v", errMsg.Err)
	}
}

// --- Campaign Update routing tests ---

func TestModel_CampaignStartMsgInitsCampaignState(t *testing.T) {
	// Given: a model in campaign mode
	cr := &mockCampaignRunner{}
	pr := &mockRunner{output: PipelineOutput{Success: true}}
	m := NewModel(
		WithPipelineRunner(pr),
		WithCampaignRunner(cr),
		WithPhaseNames([]string{"plan"}),
	)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)
	// Dispatch to enter campaign mode
	updated, _ = m.Update(DispatchMsg{BeadID: "cap-feat", BeadType: "feature"})
	m = updated.(Model)
	ch := make(chan tea.Msg, 1)
	m.eventCh = ch

	// When: a CampaignStartMsg is received
	tasks := sampleCampaignTasks()
	updated, _ = m.Update(CampaignStartMsg{ParentID: "cap-feat", Tasks: tasks})
	m = updated.(Model)

	// Then: the campaign state is initialized with the tasks
	if len(m.campaign.tasks) != len(tasks) {
		t.Errorf("campaign.tasks len = %d, want %d", len(m.campaign.tasks), len(tasks))
	}
	if m.campaign.parentID != "cap-feat" {
		t.Errorf("campaign.parentID = %q, want %q", m.campaign.parentID, "cap-feat")
	}
}

func TestModel_CampaignStartMsgPassesParentTitle(t *testing.T) {
	// Given: a model in campaign mode
	cr := &mockCampaignRunner{}
	pr := &mockRunner{output: PipelineOutput{Success: true}}
	m := NewModel(
		WithPipelineRunner(pr),
		WithCampaignRunner(cr),
		WithPhaseNames([]string{"plan"}),
	)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)
	updated, _ = m.Update(DispatchMsg{BeadID: "cap-feat", BeadType: "feature"})
	m = updated.(Model)
	ch := make(chan tea.Msg, 1)
	m.eventCh = ch

	// When: a CampaignStartMsg with ParentTitle is received
	updated, _ = m.Update(CampaignStartMsg{
		ParentID:    "cap-feat",
		ParentTitle: "Authentication feature",
		Tasks:       sampleCampaignTasks(),
	})
	m = updated.(Model)

	// Then: the campaign state has the parent title
	if m.campaign.parentTitle != "Authentication feature" {
		t.Errorf("campaign.parentTitle = %q, want %q", m.campaign.parentTitle, "Authentication feature")
	}
}

func TestModel_CampaignTaskStartMsgRoutes(t *testing.T) {
	// Given: a model in campaign mode with campaign started
	m := newCampaignModel(90, 40)

	// When: a CampaignTaskStartMsg is received
	updated, _ := m.Update(CampaignTaskStartMsg{BeadID: "cap-001", Index: 0, Total: 3})
	m = updated.(Model)

	// Then: the campaign state shows the task as running
	if m.campaign.currentIdx != 0 {
		t.Errorf("campaign.currentIdx = %d, want 0", m.campaign.currentIdx)
	}
	if m.campaign.taskStatuses[0] != CampaignTaskRunning {
		t.Errorf("taskStatuses[0] = %q, want %q", m.campaign.taskStatuses[0], CampaignTaskRunning)
	}
}

func TestModel_CampaignTaskDoneMsgRoutes(t *testing.T) {
	// Given: a model in campaign mode with a running task
	m := newCampaignModel(90, 40)
	updated, _ := m.Update(CampaignTaskStartMsg{BeadID: "cap-001", Index: 0, Total: 3})
	m = updated.(Model)

	// When: a CampaignTaskDoneMsg is received
	updated, _ = m.Update(CampaignTaskDoneMsg{BeadID: "cap-001", Index: 0, Success: true, Duration: 5 * time.Second})
	m = updated.(Model)

	// Then: the campaign state reflects completion
	if m.campaign.completed != 1 {
		t.Errorf("campaign.completed = %d, want 1", m.campaign.completed)
	}
}

func TestModel_CampaignDoneMsgTransitionsToSummary(t *testing.T) {
	// Given: a model in campaign mode with event channel
	m := newCampaignModel(90, 40)
	ch := make(chan tea.Msg, 1)
	m.eventCh = ch

	// When: a CampaignDoneMsg is received
	updated, _ := m.Update(CampaignDoneMsg{ParentID: "cap-feat", TotalTasks: 3, Passed: 2, Failed: 1})
	m = updated.(Model)

	// Then: event is stored but mode doesn't change yet (waits for channelClosedMsg)
	// channelClosedMsg will transition to summary
	if m.campaignDone == nil {
		t.Error("campaignDone should be set after CampaignDoneMsg")
	}
}

func TestModel_CampaignChannelClosedTransitionsToSummary(t *testing.T) {
	// Given: a model in campaign mode with campaignDone set
	m := newCampaignModel(90, 40)
	m.cancelPipeline = func() {}
	m.campaignDone = &CampaignDoneMsg{ParentID: "cap-feat", TotalTasks: 3, Passed: 3, Failed: 0}

	// When: channelClosedMsg is received
	updated, _ := m.Update(channelClosedMsg{})
	m = updated.(Model)

	// Then: the model transitions to campaign summary mode
	if m.mode != ModeCampaignSummary {
		t.Errorf("mode = %d, want ModeCampaignSummary (%d)", m.mode, ModeCampaignSummary)
	}
}

func TestModel_CampaignChannelClosedWhileAbortingGoesToBrowse(t *testing.T) {
	// Given: a model in campaign mode that is aborting
	m := newCampaignModel(90, 40)
	m.aborting = true
	m.cancelPipeline = func() {}

	// When: channelClosedMsg is received
	updated, _ := m.Update(channelClosedMsg{})
	m = updated.(Model)

	// Then: the model transitions to browse mode (not campaign summary)
	if m.mode != ModeBrowse {
		t.Errorf("mode = %d, want ModeBrowse (%d) after campaign abort", m.mode, ModeBrowse)
	}
}

func TestModel_PhaseUpdateRoutesToCampaignInCampaignMode(t *testing.T) {
	// Given: a model in campaign mode with a task started
	m := newCampaignModel(90, 40)
	updated, _ := m.Update(CampaignTaskStartMsg{BeadID: "cap-001", Index: 0, Total: 3})
	m = updated.(Model)
	// Set pipeline phases on the campaign state
	m.campaign.pipeline = newPipelineState([]string{"plan", "code"})
	ch := make(chan tea.Msg, 1)
	m.eventCh = ch

	// When: a PhaseUpdateMsg is received
	updated, _ = m.Update(PhaseUpdateMsg{Phase: "plan", Status: PhaseRunning})
	m = updated.(Model)

	// Then: the campaign's embedded pipeline is updated
	if m.campaign.pipeline.phases[0].Status != PhaseRunning {
		t.Errorf("campaign pipeline phase status = %q, want %q", m.campaign.pipeline.phases[0].Status, PhaseRunning)
	}
}

func TestModel_SpinnerTickRoutesToCampaign(t *testing.T) {
	// Given: a model in campaign mode
	m := newCampaignModel(90, 40)

	// When: a spinner tick from the campaign's embedded pipeline spinner is received
	_, cmd := m.Update(m.campaign.pipeline.spinner.Tick())

	// Then: a follow-up tick is produced (spinner keeps animating)
	if cmd == nil {
		t.Error("spinner tick should produce a follow-up command in campaign mode")
	}
}

// --- Campaign key handling tests ---

func TestModel_CampaignQuitCancels(t *testing.T) {
	// Given: a model in campaign mode with a cancel function
	var cancelled bool
	m := newCampaignModel(90, 40)
	m.cancelPipeline = func() { cancelled = true }

	// When: q is pressed
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m = updated.(Model)

	// Then: the cancel function is called and aborting is set
	if !cancelled {
		t.Error("q in campaign mode should cancel")
	}
	if !m.aborting {
		t.Error("aborting should be set after q in campaign mode")
	}
	if !m.campaign.pipeline.aborting {
		t.Error("campaign pipeline aborting should be set for visual feedback")
	}
}

func TestModel_CampaignDoublePressQForceQuits(t *testing.T) {
	// Given: a model in campaign mode that is already aborting
	m := newCampaignModel(90, 40)
	m.aborting = true

	// When: q is pressed again
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	// Then: a quit command is returned
	if cmd == nil {
		t.Fatal("second q during campaign abort should return quit")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("expected tea.QuitMsg, got %T", msg)
	}
}

func TestModel_CampaignSummaryAnyKeyReturnsToBrowse(t *testing.T) {
	// Given: a model in campaign summary mode
	lister := &stubLister{beads: sampleBeads()}
	m := NewModel(WithBeadLister(lister))
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)
	m.mode = ModeCampaignSummary

	// When: any key is pressed
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	m = updated.(Model)

	// Then: the model transitions to browse mode
	if m.mode != ModeBrowse {
		t.Errorf("mode = %d, want ModeBrowse (%d)", m.mode, ModeBrowse)
	}
}

func TestModel_CampaignSummarySkipsPostPipeline(t *testing.T) {
	// Given: a model in campaign summary mode with PostPipelineFunc configured
	var postPipelineCalled bool
	lister := &stubLister{beads: sampleBeads()}
	m := NewModel(
		WithBeadLister(lister),
		WithPostPipelineFunc(func(beadID string) error {
			postPipelineCalled = true
			return nil
		}),
	)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)
	m.mode = ModeCampaignSummary
	m.dispatchedBeadID = "cap-feat"

	// When: any key is pressed to return to browse
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})

	// Then: postPipeline is NOT called
	if cmd != nil {
		for _, msg := range execBatch(t, cmd) {
			if _, ok := msg.(PostPipelineDoneMsg); ok {
				t.Error("campaign summary should not fire postPipeline")
			}
		}
	}
	if postPipelineCalled {
		t.Error("PostPipelineFunc should not be called for campaign summary")
	}
}

// --- Campaign view tests ---

func TestModel_CampaignViewLeftShowsCampaignState(t *testing.T) {
	// Given: a model in campaign mode with tasks
	m := newCampaignModel(90, 40)
	updated, _ := m.Update(CampaignStartMsg{
		ParentID: "cap-feat",
		Tasks:    sampleCampaignTasks(),
	})
	m = updated.(Model)

	// When: the view is rendered
	view := m.View()
	plain := stripANSI(view)

	// Then: campaign task info is visible
	if !strings.Contains(plain, "First task") {
		t.Errorf("campaign left pane should show task titles, got:\n%s", plain)
	}
}

func TestModel_CampaignViewRightShowsPhaseReport(t *testing.T) {
	// Given: a model in campaign mode with a running task and phases
	m := newCampaignModel(90, 40)
	updated, _ := m.Update(CampaignStartMsg{
		ParentID: "cap-feat",
		Tasks:    sampleCampaignTasks(),
	})
	m = updated.(Model)
	updated, _ = m.Update(CampaignTaskStartMsg{BeadID: "cap-001", Index: 0, Total: 3})
	m = updated.(Model)
	m.campaign.pipeline = newPipelineState([]string{"plan", "code"})
	updated, _ = m.Update(PhaseUpdateMsg{Phase: "plan", Status: PhaseRunning})
	m = updated.(Model)

	// When: the view is rendered
	view := m.View()
	plain := stripANSI(view)

	// Then: phase report content appears in the right pane
	if !strings.Contains(plain, "plan") {
		t.Errorf("campaign right pane should show phase report, got:\n%s", plain)
	}
}

func TestModel_CampaignSummaryViewLeftShowsFrozenState(t *testing.T) {
	// Given: a model in campaign summary mode with completed campaign
	m := newCampaignModel(90, 40)
	m.mode = ModeCampaignSummary
	m.campaign = newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())
	m.campaign, _ = m.campaign.Update(CampaignTaskStartMsg{BeadID: "cap-001", Index: 0, Total: 3})
	m.campaign, _ = m.campaign.Update(CampaignTaskDoneMsg{BeadID: "cap-001", Index: 0, Success: true, Duration: 5 * time.Second})

	// When: the view is rendered
	view := m.View()
	plain := stripANSI(view)

	// Then: frozen campaign state is visible
	if !strings.Contains(plain, "✓") {
		t.Errorf("campaign summary left pane should show completed task checkmarks, got:\n%s", plain)
	}
}

func TestModel_CampaignSummaryViewRightShowsSummary(t *testing.T) {
	// Given: a model in campaign summary mode with results
	m := newCampaignModel(90, 40)
	m.mode = ModeCampaignSummary
	m.campaign = newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())
	m.campaignDone = &CampaignDoneMsg{ParentID: "cap-feat", TotalTasks: 3, Passed: 2, Failed: 1}

	// When: the view is rendered
	view := m.View()
	plain := stripANSI(view)

	// Then: campaign summary with pass/fail counts is shown
	if !strings.Contains(plain, "2/3") {
		t.Errorf("campaign summary right pane should show '2/3' tasks, got:\n%s", plain)
	}
}

func TestModel_CampaignSummaryViewRightShowsSkipped(t *testing.T) {
	// Given: a model in campaign summary mode with skipped tasks
	m := newCampaignModel(90, 40)
	m.mode = ModeCampaignSummary
	m.campaign = newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())
	m.campaignDone = &CampaignDoneMsg{ParentID: "cap-feat", TotalTasks: 5, Passed: 2, Failed: 1, Skipped: 2}

	// When: the view is rendered
	view := m.View()
	plain := stripANSI(view)

	// Then: skipped count is shown in the summary
	if !strings.Contains(plain, "2 skipped") {
		t.Errorf("campaign summary should show '2 skipped', got:\n%s", plain)
	}
}

func TestModel_CampaignSummaryViewRightHidesZeroSkipped(t *testing.T) {
	// Given: a model in campaign summary mode with no skipped tasks
	m := newCampaignModel(90, 40)
	m.mode = ModeCampaignSummary
	m.campaign = newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())
	m.campaignDone = &CampaignDoneMsg{ParentID: "cap-feat", TotalTasks: 3, Passed: 2, Failed: 1}

	// When: the view is rendered
	view := m.View()
	plain := stripANSI(view)

	// Then: "skipped" should not appear when count is zero
	if strings.Contains(plain, "skipped") {
		t.Errorf("campaign summary should not show 'skipped' when count is zero, got:\n%s", plain)
	}
}

func TestModel_CampaignModeHelpShowsCampaignBindings(t *testing.T) {
	// Given: a model in campaign mode
	m := newSizedModel(90, 40)
	m.mode = ModeCampaign

	// When: the view is rendered
	view := m.View()

	// Then: campaign help text (abort) is shown
	if !containsPlainText(view, "abort") {
		t.Errorf("campaign mode help should show 'abort'")
	}
}

func TestModel_CampaignSummaryHelpShowsContinue(t *testing.T) {
	// Given: a model in campaign summary mode
	m := newSizedModel(90, 40)
	m.mode = ModeCampaignSummary

	// When: the view is rendered
	view := m.View()

	// Then: summary help text (continue) is shown
	if !containsPlainText(view, "continue") {
		t.Errorf("campaign summary mode help should show 'continue'")
	}
}

// --- Campaign full flow test ---

func TestModel_CampaignFullFlow(t *testing.T) {
	// Given: a model with campaign runner that runs two tasks
	cr := &mockCampaignRunner{
		events: []tea.Msg{
			CampaignStartMsg{ParentID: "cap-feat", Tasks: []CampaignTaskInfo{
				{BeadID: "cap-001", Title: "Task 1", Priority: 1},
				{BeadID: "cap-002", Title: "Task 2", Priority: 2},
			}},
			CampaignTaskStartMsg{BeadID: "cap-001", Index: 0, Total: 2},
			CampaignTaskDoneMsg{BeadID: "cap-001", Index: 0, Success: true, Duration: time.Second},
			CampaignTaskStartMsg{BeadID: "cap-002", Index: 1, Total: 2},
			CampaignTaskDoneMsg{BeadID: "cap-002", Index: 1, Success: true, Duration: 2 * time.Second},
			CampaignDoneMsg{ParentID: "cap-feat", TotalTasks: 2, Passed: 2, Failed: 0},
		},
	}
	pr := &mockRunner{output: PipelineOutput{Success: true}}
	lister := &stubLister{beads: sampleBeads()}
	m := NewModel(
		WithPipelineRunner(pr),
		WithCampaignRunner(cr),
		WithPhaseNames([]string{"plan"}),
		WithBeadLister(lister),
	)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)

	// When: a feature bead is dispatched
	updated, _ = m.Update(DispatchMsg{BeadID: "cap-feat", BeadType: "feature"})
	m = updated.(Model)
	if m.mode != ModeCampaign {
		t.Fatalf("mode = %d, want ModeCampaign", m.mode)
	}

	// Drain campaign events
	m = drainPipeline(t, m) // works for campaign too since it uses same channel pattern

	// Then: the model is in campaign summary mode
	if m.mode != ModeCampaignSummary {
		t.Errorf("mode = %d, want ModeCampaignSummary after campaign completes", m.mode)
	}
}

// newCampaignModel creates a model set up in campaign mode for testing.
func newCampaignModel(w, h int) Model {
	cr := &mockCampaignRunner{}
	pr := &mockRunner{output: PipelineOutput{Success: true}}
	m := NewModel(
		WithPipelineRunner(pr),
		WithCampaignRunner(cr),
		WithPhaseNames([]string{"plan"}),
	)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = updated.(Model)
	// Manually enter campaign mode
	m.mode = ModeCampaign
	m.campaign = newCampaignState("cap-feat", "Feature Title", sampleCampaignTasks())
	return m
}

func TestModel_CampaignErrorMsgStoresError(t *testing.T) {
	// Given: a model in campaign mode
	m := newCampaignModel(90, 40)
	// eventCh is set so listenForEvents returns a non-nil re-listen command.
	m.eventCh = make(chan tea.Msg, 1)

	// When: a CampaignErrorMsg is received
	updated, _ := m.Update(CampaignErrorMsg{Err: fmt.Errorf("discovery failed")})
	m = updated.(Model)

	// Then: the campaign error is stored on the model
	if m.campaignErr == nil {
		t.Fatal("campaignErr should be set after CampaignErrorMsg")
	}
	if m.campaignErr.Error() != "discovery failed" {
		t.Errorf("campaignErr = %q, want %q", m.campaignErr, "discovery failed")
	}
}

func TestModel_CampaignErrorShownInSummary(t *testing.T) {
	// Given: a model in campaign summary mode with an error
	m := newCampaignModel(90, 40)
	m.mode = ModeCampaignSummary
	m.campaignErr = fmt.Errorf("discovery failed")
	m.campaignDone = &CampaignDoneMsg{ParentID: "cap-feat", TotalTasks: 0}

	// When: the view is rendered
	view := m.View()
	plain := stripANSI(view)

	// Then: the error message is shown in the summary
	if !strings.Contains(plain, "discovery failed") {
		t.Errorf("campaign summary should show error message, got:\n%s", plain)
	}
}

func TestModel_CampaignChannelClosedWithoutDoneMsgGoesToCampaignSummary(t *testing.T) {
	// Given: a model in campaign mode where the runner errored without sending CampaignDoneMsg
	m := newCampaignModel(90, 40)
	m.cancelPipeline = func() {}
	// campaignDone is nil — the runner errored before sending CampaignDoneMsg

	// When: channelClosedMsg is received
	updated, _ := m.Update(channelClosedMsg{})
	m = updated.(Model)

	// Then: the model transitions to campaign summary (not pipeline summary)
	if m.mode != ModeCampaignSummary {
		t.Errorf("mode = %d, want ModeCampaignSummary (%d)", m.mode, ModeCampaignSummary)
	}
	// And: a synthetic campaignDone is created so the summary view can render
	if m.campaignDone == nil {
		t.Fatal("campaignDone should be synthesized when missing")
	}
	if m.campaignDone.ParentID != "cap-feat" {
		t.Errorf("campaignDone.ParentID = %q, want %q", m.campaignDone.ParentID, "cap-feat")
	}
}

// --- Archive detail tests ---

// stubArchiveReader implements ArchiveReader for tests.
type stubArchiveReader struct {
	summaries map[string]string
	worklogs  map[string]string
}

func (s *stubArchiveReader) ReadSummary(beadID string) (string, error) {
	if v, ok := s.summaries[beadID]; ok {
		return v, nil
	}
	return "", fmt.Errorf("not found: %s", beadID)
}

func (s *stubArchiveReader) ReadWorklog(beadID string) (string, error) {
	if v, ok := s.worklogs[beadID]; ok {
		return v, nil
	}
	return "", fmt.Errorf("not found: %s", beadID)
}

func TestWithArchiveReader(t *testing.T) {
	// Given: a stub archive reader
	ar := &stubArchiveReader{}

	// When: a model is created with WithArchiveReader
	m := NewModel(WithArchiveReader(ar))

	// Then: the archive reader is stored
	if m.archive == nil {
		t.Error("archive reader should be set")
	}
}

func TestFormatClosedBeadDetail_WithSummaryAndWorklog(t *testing.T) {
	// Given: a bead detail with both summary and worklog
	detail := sampleDetail()
	summary := "## Summary\n\nAll phases passed."
	worklog := "# Worklog\n\nPhase 1: passed\nPhase 2: passed"

	// When: formatClosedBeadDetail is called
	text := formatClosedBeadDetail(detail, summary, worklog)

	// Then: the standard detail is present
	if !strings.Contains(text, "First task") {
		t.Errorf("should contain title, got:\n%s", text)
	}

	// And: the archive separator is present
	if !strings.Contains(text, archiveSeparator) {
		t.Errorf("should contain archive separator, got:\n%s", text)
	}

	// And: summary section is present
	if !strings.Contains(text, "All phases passed.") {
		t.Errorf("should contain summary, got:\n%s", text)
	}

	// And: worklog section is present
	if !strings.Contains(text, "Phase 1: passed") {
		t.Errorf("should contain worklog, got:\n%s", text)
	}
}

func TestFormatClosedBeadDetail_SummaryOnly(t *testing.T) {
	// Given: a bead detail with summary but no worklog
	detail := BeadDetail{ID: "cap-001", Title: "Test", Priority: 2, Type: "task"}
	summary := "All passed."

	// When: formatClosedBeadDetail is called with empty worklog
	text := formatClosedBeadDetail(detail, summary, "")

	// Then: summary is present but no worklog header
	if !strings.Contains(text, "All passed.") {
		t.Errorf("should contain summary, got:\n%s", text)
	}
	if strings.Contains(text, "Worklog") {
		t.Errorf("should not contain worklog header when empty, got:\n%s", text)
	}
}

func TestFormatClosedBeadDetail_NoArchive(t *testing.T) {
	// Given: a bead detail with no archive data
	detail := sampleDetail()

	// When: formatClosedBeadDetail is called with empty strings
	text := formatClosedBeadDetail(detail, "", "")

	// Then: it should be equivalent to formatBeadDetail (no separator, no archive sections)
	if strings.Contains(text, archiveSeparator) {
		t.Errorf("should not contain archive separator with no archive data, got:\n%s", text)
	}
}

func newClosedResolverModel(t *testing.T, w, h int) (Model, *stubResolver) {
	t.Helper()
	ar := &stubArchiveReader{
		summaries: map[string]string{
			"cap-c01": "## Summary\n\nTask completed.",
		},
		worklogs: map[string]string{
			"cap-c01": "# Worklog\n\nAll phases passed.",
		},
	}
	// Unified view: mix of open and closed beads.
	allBeads := []BeadSummary{
		{ID: "cap-c01", Title: "Done task", Priority: 2, Type: "task", Closed: true},
		{ID: "cap-c02", Title: "Done feature", Priority: 1, Type: "feature", Closed: true},
	}
	resolver := &stubResolver{details: map[string]BeadDetail{
		"cap-c01": {ID: "cap-c01", Title: "Done task", Priority: 2, Type: "task", Description: "Was completed."},
		"cap-c02": {ID: "cap-c02", Title: "Done feature", Priority: 1, Type: "feature", Description: "Also done."},
	}}
	lister := &stubLister{beads: allBeads}
	m := NewModel(
		WithBeadLister(lister),
		WithBeadResolver(resolver),
		WithArchiveReader(ar),
	)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m = updated.(Model)
	// Deliver bead list (includes closed beads in unified view).
	updated, _ = m.Update(BeadListMsg{Beads: allBeads})
	m = updated.(Model)
	// Complete debounce cycle for cap-c01 (first bead).
	updated, _ = m.Update(resolveDebounceMsg{ID: m.pendingResolveID})
	m = updated.(Model)
	return m, resolver
}

func TestModel_ClosedBeadDetailShowsArchive(t *testing.T) {
	// Given: a model in closed view with archive reader, resolved bead cap-c01
	m, _ := newClosedResolverModel(t, 90, 40)
	// Deliver the resolve result for cap-c01.
	updated, _ := m.Update(BeadResolvedMsg{
		ID:     "cap-c01",
		Detail: BeadDetail{ID: "cap-c01", Title: "Done task", Priority: 2, Type: "task", Description: "Was completed."},
	})
	m = updated.(Model)

	// When: the view is rendered
	view := m.View()
	plain := stripANSI(view)

	// Then: the standard detail is shown
	if !strings.Contains(plain, "Done task") {
		t.Errorf("should contain bead title, got:\n%s", plain)
	}
	// And: archive summary is shown
	if !strings.Contains(plain, "Task completed.") {
		t.Errorf("should contain archive summary, got:\n%s", plain)
	}
	// And: archive worklog is shown
	if !strings.Contains(plain, "All phases passed.") {
		t.Errorf("should contain archive worklog, got:\n%s", plain)
	}
}

func TestModel_ClosedBeadCacheHitShowsArchive(t *testing.T) {
	// Given: a model in closed view with cap-c01 already resolved (cached)
	m, _ := newClosedResolverModel(t, 90, 40)
	updated, _ := m.Update(BeadResolvedMsg{
		ID:     "cap-c01",
		Detail: BeadDetail{ID: "cap-c01", Title: "Done task", Priority: 2, Type: "task", Description: "Was completed."},
	})
	m = updated.(Model)
	// Move to cap-c02 then back to cap-c01 (cache hit).
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)

	// When: the view is rendered
	view := m.View()
	plain := stripANSI(view)

	// Then: archive data is shown even on cache hit
	if !strings.Contains(plain, "Task completed.") {
		t.Errorf("cache hit should show archive summary, got:\n%s", plain)
	}
}

func TestModel_ReadyBeadDoesNotShowArchive(t *testing.T) {
	// Given: a model with archive reader but showing ready beads
	ar := &stubArchiveReader{
		summaries: map[string]string{"cap-001": "Should not appear"},
	}
	resolver := &stubResolver{details: map[string]BeadDetail{
		"cap-001": sampleDetail(),
	}}
	lister := &stubLister{beads: sampleBeads()}
	m := NewModel(
		WithBeadLister(lister),
		WithBeadResolver(resolver),
		WithArchiveReader(ar),
	)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)
	updated, _ = m.Update(BeadListMsg{Beads: sampleBeads()})
	m = updated.(Model)
	updated, _ = m.Update(resolveDebounceMsg{ID: m.pendingResolveID})
	m = updated.(Model)
	updated, _ = m.Update(BeadResolvedMsg{ID: "cap-001", Detail: sampleDetail()})
	m = updated.(Model)

	// When: the view is rendered
	view := m.View()
	plain := stripANSI(view)

	// Then: archive data is not shown for ready beads
	if strings.Contains(plain, "Should not appear") {
		t.Errorf("ready beads should not show archive data, got:\n%s", plain)
	}
}

func TestModel_ResolveErrorNavigable(t *testing.T) {
	// Given: a model with cap-001 resolve failed
	m, _ := newResolverModel(90, 40)
	updated, _ := m.Update(BeadResolvedMsg{ID: "cap-001", Err: fmt.Errorf("network error")})
	m = updated.(Model)

	// When: cursor moves down (navigating bead list despite error)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)

	// Then: navigation works and a new resolve is triggered for cap-002
	if m.browse.cursor != 1 {
		t.Errorf("cursor = %d, want 1 after down key", m.browse.cursor)
	}
	if m.detailID != "cap-002" {
		t.Errorf("detailID = %q, want %q after navigating", m.detailID, "cap-002")
	}
	if cmd == nil {
		t.Error("navigating after error should trigger resolve for new bead")
	}
}

// --- Debounced resolution tests ---

func TestModel_CacheMissSetsPendingResolveID(t *testing.T) {
	// Given: a model with cap-001 resolved (cached) and cap-002 not cached
	m, _ := newResolverModel(90, 40)
	updated, _ := m.Update(BeadResolvedMsg{ID: "cap-001", Detail: sampleDetail()})
	m = updated.(Model)

	// When: cursor moves to cap-002 (cache miss)
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)

	// Then: pendingResolveID is set to cap-002
	if m.pendingResolveID != "cap-002" {
		t.Errorf("pendingResolveID = %q, want %q", m.pendingResolveID, "cap-002")
	}
	// And: resolver is NOT called yet (debounce delays the call)
	if m.resolvingID != "" {
		t.Errorf("resolvingID = %q, want empty (debounce should delay resolve)", m.resolvingID)
	}
	// And: a debounce tick command is returned (not a direct resolve)
	if cmd == nil {
		t.Fatal("cache miss should return a debounce tick command")
	}
}

func TestModel_DebounceMsgFiresResolve(t *testing.T) {
	// Given: a model with pendingResolveID set to cap-002
	m, resolver := newResolverModel(90, 40)
	updated, _ := m.Update(BeadResolvedMsg{ID: "cap-001", Detail: sampleDetail()})
	m = updated.(Model)
	// Move cursor to cap-002 (sets pendingResolveID via debounce)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	resolver.calls = 0

	// When: the debounce tick fires with matching ID
	updated, cmd := m.Update(resolveDebounceMsg{ID: "cap-002"})
	m = updated.(Model)

	// Then: resolvingID is set and a resolve command is returned
	if m.resolvingID != "cap-002" {
		t.Errorf("resolvingID = %q, want %q", m.resolvingID, "cap-002")
	}
	if m.pendingResolveID != "" {
		t.Errorf("pendingResolveID = %q, want empty after debounce fires", m.pendingResolveID)
	}
	if cmd == nil {
		t.Fatal("debounce tick should produce a resolve command")
	}
	// Execute the command and check it calls the resolver
	msgs := execBatch(t, cmd)
	var found bool
	for _, msg := range msgs {
		if resolved, ok := msg.(BeadResolvedMsg); ok {
			found = true
			if resolved.ID != "cap-002" {
				t.Errorf("resolved ID = %q, want %q", resolved.ID, "cap-002")
			}
		}
	}
	if !found {
		t.Fatal("expected BeadResolvedMsg from debounce resolve")
	}
}

func TestModel_StaleDebounceMsgIgnored(t *testing.T) {
	// Given: a model where cursor moved from cap-002 to cap-003 rapidly
	m, resolver := newResolverModel(90, 40)
	updated, _ := m.Update(BeadResolvedMsg{ID: "cap-001", Detail: sampleDetail()})
	m = updated.(Model)
	// Move to cap-002
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	// Immediately move to cap-003 (before debounce fires)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	resolver.calls = 0

	// Precondition: pendingResolveID is cap-003 (the latest cursor target)
	if m.pendingResolveID != "cap-003" {
		t.Fatalf("pendingResolveID = %q, want %q", m.pendingResolveID, "cap-003")
	}

	// When: the stale debounce tick for cap-002 arrives
	updated, cmd := m.Update(resolveDebounceMsg{ID: "cap-002"})
	m = updated.(Model)

	// Then: the stale debounce is ignored (no resolve fired)
	if m.resolvingID != "" {
		t.Errorf("resolvingID = %q, want empty (stale debounce should be ignored)", m.resolvingID)
	}
	if cmd != nil {
		t.Error("stale debounce should not produce a command")
	}
	// And: pendingResolveID still points to cap-003
	if m.pendingResolveID != "cap-003" {
		t.Errorf("pendingResolveID = %q, want %q", m.pendingResolveID, "cap-003")
	}
}

func TestModel_CacheHitBypassesDebounce(t *testing.T) {
	// Given: a model with both cap-001 and cap-002 cached
	m, resolver := newResolverModel(90, 40)
	updated, _ := m.Update(BeadResolvedMsg{ID: "cap-001", Detail: sampleDetail()})
	m = updated.(Model)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	m = updated.(Model)
	// Deliver debounce tick and resolve for cap-002
	updated, cmd := m.Update(resolveDebounceMsg{ID: "cap-002"})
	m = updated.(Model)
	for _, msg := range execBatch(t, cmd) {
		updated, _ = m.Update(msg)
		m = updated.(Model)
	}
	resolver.calls = 0

	// When: cursor moves back to cap-001 (cache hit)
	updated, cmd = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	m = updated.(Model)

	// Then: cache hit shows instantly with no debounce
	if m.pendingResolveID != "" {
		t.Errorf("pendingResolveID = %q, want empty for cache hit", m.pendingResolveID)
	}
	if m.resolvingID != "" {
		t.Errorf("resolvingID = %q, want empty for cache hit", m.resolvingID)
	}
	if cmd != nil {
		t.Error("cache hit should not produce any command")
	}
	if resolver.calls != 0 {
		t.Errorf("resolver.calls = %d, want 0 for cache hit", resolver.calls)
	}
}

func TestModel_DebounceCmdIsTickBased(t *testing.T) {
	// Given: a model with cap-001 resolved
	m, _ := newResolverModel(90, 40)
	updated, _ := m.Update(BeadResolvedMsg{ID: "cap-001", Detail: sampleDetail()})
	m = updated.(Model)

	// When: cursor moves to cap-002 (cache miss)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyDown})

	// Then: the batch contains a tick-based command (not a direct BeadResolvedMsg)
	if cmd == nil {
		t.Fatal("expected a command from cache miss")
	}
	msgs := execBatch(t, cmd)
	for _, msg := range msgs {
		if _, ok := msg.(BeadResolvedMsg); ok {
			t.Fatal("cache miss should NOT produce immediate BeadResolvedMsg; expected debounce tick")
		}
	}
}

func TestModel_BeadListInitialLoadDebounces(t *testing.T) {
	// Given: a model that just received the bead list (first load)
	resolver := &stubResolver{details: map[string]BeadDetail{
		"cap-001": sampleDetail(),
	}}
	lister := &stubLister{beads: sampleBeads()}
	m := NewModel(WithBeadLister(lister), WithBeadResolver(resolver))
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 90, Height: 40})
	m = updated.(Model)

	// When: the bead list arrives (triggers maybeResolve for first bead)
	updated, _ = m.Update(BeadListMsg{Beads: sampleBeads()})
	m = updated.(Model)

	// Then: pendingResolveID is set (debounce, not immediate resolve)
	if m.pendingResolveID != "cap-001" {
		t.Errorf("pendingResolveID = %q, want %q", m.pendingResolveID, "cap-001")
	}
	// And: resolver is not called yet
	if resolver.calls != 0 {
		t.Errorf("resolver.calls = %d, want 0 (should debounce)", resolver.calls)
	}
}
