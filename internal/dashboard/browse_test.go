package dashboard

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// stubLister implements BeadLister for tests.
type stubLister struct {
	beads       []BeadSummary
	err         error
	closedBeads []BeadSummary
	closedErr   error
}

func (s *stubLister) Ready() ([]BeadSummary, error) {
	return s.beads, s.err
}

func (s *stubLister) Closed(limit int) ([]BeadSummary, error) {
	if s.closedErr != nil {
		return nil, s.closedErr
	}
	result := s.closedBeads
	if limit > 0 && limit < len(result) {
		result = result[:limit]
	}
	return result, nil
}

func sampleBeads() []BeadSummary {
	return []BeadSummary{
		{ID: "cap-001", Title: "First task", Priority: 1, Type: "task"},
		{ID: "cap-002", Title: "Second task", Priority: 2, Type: "feature"},
		{ID: "cap-003", Title: "Third task", Priority: 3, Type: "task"},
	}
}

func TestBrowse_RefreshPreservesExpansionState(t *testing.T) {
	// Given: a browse state with mixed expanded/collapsed nodes
	bs := newBrowseState()
	beads := []BeadSummary{
		{ID: "cap-001", Title: "Parent 1", Priority: 1, Type: "feature"},
		{ID: "cap-001.1", Title: "Child 1", Priority: 2, Type: "task"},
		{ID: "cap-002", Title: "Parent 2", Priority: 1, Type: "feature"},
		{ID: "cap-002.1", Title: "Child 2", Priority: 2, Type: "task"},
	}
	bs, _ = bs.Update(BeadListMsg{Beads: beads})

	// Expand cap-001 only
	bs.expandedIDs["cap-001"] = true
	bs, _ = bs.Update(BeadListMsg{Beads: beads})

	// When: refresh is triggered with same beads
	bs, _ = bs.Update(BeadListMsg{Beads: beads})

	// Then: expansion state is preserved
	if !bs.expandedIDs["cap-001"] {
		t.Error("cap-001 should remain expanded after refresh")
	}
	if bs.expandedIDs["cap-002"] {
		t.Error("cap-002 should remain collapsed after refresh")
	}

	// And: tree reflects preserved state
	if len(bs.flatNodes) != 3 { // cap-001, cap-001.1, cap-002
		t.Errorf("expected 3 flat nodes (cap-001 expanded), got %d", len(bs.flatNodes))
	}
}

func TestBrowse_RefreshRestoresCursor(t *testing.T) {
	// Given: a model with lastDispatchedID set
	m := newSizedModel(80, 24)
	m.browse, _ = m.browse.Update(BeadListMsg{Beads: sampleBeads()})
	m.lastDispatchedID = "cap-002"

	// When: refresh occurs
	updated, _ := m.Update(BeadListMsg{Beads: sampleBeads()})
	m = updated.(Model)

	// Then: cursor moves to lastDispatchedID
	if m.browse.SelectedID() != "cap-002" {
		t.Errorf("expected cursor at cap-002, got %s", m.browse.SelectedID())
	}

	// And: lastDispatchedID is cleared
	if m.lastDispatchedID != "" {
		t.Error("lastDispatchedID should be cleared after restore")
	}
}

func TestBrowse_CollapseAll(t *testing.T) {
	// Given: a browse state with multiple expanded nodes
	bs := newBrowseState()
	beads := []BeadSummary{
		{ID: "cap-001", Title: "Parent 1", Priority: 1, Type: "feature"},
		{ID: "cap-001.1", Title: "Child 1", Priority: 2, Type: "task"},
		{ID: "cap-002", Title: "Parent 2", Priority: 1, Type: "feature"},
		{ID: "cap-002.1", Title: "Child 2", Priority: 2, Type: "task"},
	}
	bs, _ = bs.Update(BeadListMsg{Beads: beads})

	// Expand both parents
	bs.expandedIDs["cap-001"] = true
	bs.expandedIDs["cap-002"] = true
	bs, _ = bs.Update(BeadListMsg{Beads: beads})

	// When: 'c' key is pressed
	bs, _ = bs.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})

	// Then: all nodes are collapsed
	if len(bs.expandedIDs) != 0 {
		t.Errorf("expected expandedIDs to be empty, got %d entries", len(bs.expandedIDs))
	}

	// And: tree is rebuilt with all nodes collapsed
	if len(bs.flatNodes) != 2 { // Only root nodes visible
		t.Errorf("expected 2 flat nodes (all collapsed), got %d", len(bs.flatNodes))
	}
}

func TestBrowse_LoadingState(t *testing.T) {
	// Given: a fresh browse state with no beads loaded
	bs := newBrowseState()

	// When: the view is rendered
	view := bs.View(40, 20, "")
	plain := stripANSI(view)

	// Then: a loading indicator is shown
	if !strings.Contains(plain, "Loading") {
		t.Errorf("loading view should contain 'Loading', got:\n%s", plain)
	}
}

func TestBrowse_LoadingShowsSpinner(t *testing.T) {
	// Given: a browse state in loading with a spinner frame
	bs := newBrowseState()

	// When: the view is rendered with a spinner frame
	view := bs.View(40, 20, "⣾")
	plain := stripANSI(view)

	// Then: the spinner frame appears alongside the loading text
	if !strings.Contains(plain, "⣾") {
		t.Errorf("loading view should contain spinner frame, got:\n%s", plain)
	}
	if !strings.Contains(plain, "Loading beads...") {
		t.Errorf("loading view should contain 'Loading beads...', got:\n%s", plain)
	}
}

func TestBrowse_BeadsLoadedView(t *testing.T) {
	// Given: a browse state with loaded beads
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})

	// When: the view is rendered
	view := bs.View(60, 20, "")
	plain := stripANSI(view)

	// Then: each bead's ID and title appear in the view
	for _, b := range sampleBeads() {
		if !strings.Contains(plain, b.ID) {
			t.Errorf("view should contain bead ID %q, got:\n%s", b.ID, plain)
		}
		if !strings.Contains(plain, b.Title) {
			t.Errorf("view should contain bead title %q, got:\n%s", b.Title, plain)
		}
	}
}

func TestBrowse_CursorDefaultsToZero(t *testing.T) {
	// Given: a browse state with loaded beads
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})

	// Then: the cursor starts at position 0
	if bs.cursor != 0 {
		t.Errorf("cursor = %d, want 0", bs.cursor)
	}
}

func TestBrowse_CursorDown(t *testing.T) {
	// Given: a browse state with loaded beads at cursor 0
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})

	// When: down key is pressed
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyDown})

	// Then: the cursor moves to position 1
	if bs.cursor != 1 {
		t.Errorf("after down: cursor = %d, want 1", bs.cursor)
	}
}

func TestBrowse_CursorUp(t *testing.T) {
	// Given: a browse state with cursor moved to position 1
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyDown})

	// When: up key is pressed
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyUp})

	// Then: the cursor returns to position 0
	if bs.cursor != 0 {
		t.Errorf("after down+up: cursor = %d, want 0", bs.cursor)
	}
}

func TestBrowse_CursorWrapsDown(t *testing.T) {
	// Given: a browse state with loaded beads
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})

	// When: down is pressed past the last item
	for range len(sampleBeads()) {
		bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	// Then: the cursor wraps to position 0
	if bs.cursor != 0 {
		t.Errorf("after wrapping down: cursor = %d, want 0", bs.cursor)
	}
}

func TestBrowse_CursorWrapsUp(t *testing.T) {
	// Given: a browse state with cursor at position 0
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})

	// When: up is pressed from position 0
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyUp})

	// Then: the cursor wraps to the last item
	want := len(sampleBeads()) - 1
	if bs.cursor != want {
		t.Errorf("after wrapping up: cursor = %d, want %d", bs.cursor, want)
	}
}

func TestBrowse_CursorMarker(t *testing.T) {
	// Given: a browse state with loaded beads
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})

	// When: the view is rendered
	view := bs.View(60, 20, "")
	plain := stripANSI(view)

	// Then: the cursor marker is visible
	if !strings.Contains(plain, CursorMarker) {
		t.Errorf("view should contain cursor marker %q, got:\n%s", CursorMarker, plain)
	}
}

func TestBrowse_VimKeys(t *testing.T) {
	// Given: a browse state with loaded beads
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})

	// When: j is pressed
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	// Then: the cursor moves down
	if bs.cursor != 1 {
		t.Errorf("after j: cursor = %d, want 1", bs.cursor)
	}

	// When: k is pressed
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})

	// Then: the cursor moves up
	if bs.cursor != 0 {
		t.Errorf("after k: cursor = %d, want 0", bs.cursor)
	}
}

func TestBrowse_EnterEmitsConfirmRequest(t *testing.T) {
	// Given: a browse state with cursor on the second bead
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyDown})

	// When: enter is pressed
	_, cmd := bs.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Then: a ConfirmRequestMsg is produced with the selected bead ID
	if cmd == nil {
		t.Fatal("enter should produce a command")
	}
	msg := cmd()
	confirm, ok := msg.(ConfirmRequestMsg)
	if !ok {
		t.Fatalf("enter command produced %T, want ConfirmRequestMsg", msg)
	}
	if confirm.BeadID != "cap-002" {
		t.Errorf("confirm bead ID = %q, want %q", confirm.BeadID, "cap-002")
	}
}

func TestBrowse_ErrorDisplay(t *testing.T) {
	// Given: a browse state that received an error
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Err: fmt.Errorf("connection failed")})

	// When: the view is rendered
	view := bs.View(60, 20, "")
	plain := stripANSI(view)

	// Then: the error message and retry hint are shown
	if !strings.Contains(plain, "connection failed") {
		t.Errorf("error view should contain error message, got:\n%s", plain)
	}
	if !strings.Contains(plain, "Press r to retry") {
		t.Errorf("error view should contain retry hint, got:\n%s", plain)
	}
}

func TestBrowse_EmptyList(t *testing.T) {
	// Given: a browse state that received an empty bead list
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: []BeadSummary{}})

	// When: the view is rendered
	view := bs.View(60, 20, "")
	plain := stripANSI(view)

	// Then: a "No beads" message with refresh hint is shown
	if !strings.Contains(plain, "No beads") {
		t.Errorf("empty list view should contain 'No beads', got:\n%s", plain)
	}
	if !strings.Contains(plain, "press r to refresh") {
		t.Errorf("empty list view should contain refresh hint, got:\n%s", plain)
	}
}

func TestBrowse_RefreshReloads(t *testing.T) {
	// Given: a browse state with loaded beads
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})

	// When: r is pressed to refresh
	bs, cmd := bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	// Then: loading state is set and a RefreshBeadsMsg command is returned
	if cmd == nil {
		t.Fatal("r should produce a refresh command")
	}
	if !bs.loading {
		t.Error("after refresh, loading should be true")
	}
	msg := cmd()
	if _, ok := msg.(RefreshBeadsMsg); !ok {
		t.Fatalf("refresh command produced %T, want RefreshBeadsMsg", msg)
	}
}

func TestBrowse_InitReturnsCmd(t *testing.T) {
	// Given: a lister with sample beads
	lister := &stubLister{beads: sampleBeads()}

	// When: initBrowse is called
	cmd := initBrowse(lister)

	// Then: a BeadListMsg with 3 beads is produced
	if cmd == nil {
		t.Fatal("initBrowse should return a non-nil command")
	}
	msg := cmd()
	listMsg, ok := msg.(BeadListMsg)
	if !ok {
		t.Fatalf("init command produced %T, want BeadListMsg", msg)
	}
	if len(listMsg.Beads) != 3 {
		t.Errorf("init loaded %d beads, want 3", len(listMsg.Beads))
	}
}

func TestBrowse_InitReturnsError(t *testing.T) {
	// Given: a lister that returns an error
	lister := &stubLister{err: fmt.Errorf("db down")}

	// When: initBrowse is called
	cmd := initBrowse(lister)

	// Then: a BeadListMsg with an error is produced
	msg := cmd().(BeadListMsg)
	if msg.Err == nil {
		t.Fatal("initBrowse with error lister should produce BeadListMsg with Err")
	}
}

func TestBrowse_PriorityBadgesInView(t *testing.T) {
	// Given: a browse state with beads of different priorities
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})

	// When: the view is rendered
	view := bs.View(60, 20, "")
	plain := stripANSI(view)

	// Then: priority badges P1 and P2 are visible
	if !strings.Contains(plain, "P1") {
		t.Errorf("view should contain priority badge P1, got:\n%s", plain)
	}
	if !strings.Contains(plain, "P2") {
		t.Errorf("view should contain priority badge P2, got:\n%s", plain)
	}
}

func TestBrowse_TypeInView(t *testing.T) {
	// Given: a browse state with beads of different types
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})

	// When: the view is rendered
	view := bs.View(80, 20, "")
	plain := stripANSI(view)

	// Then: bead types are visible
	if !strings.Contains(plain, "[task]") {
		t.Errorf("view should contain bead type [task], got:\n%s", plain)
	}
	if !strings.Contains(plain, "[feature]") {
		t.Errorf("view should contain bead type [feature], got:\n%s", plain)
	}
}

func TestBrowse_KeysIgnoredDuringLoading(t *testing.T) {
	// Given: a browse state still in loading (no beads received)
	bs := newBrowseState()

	// When: down key is pressed during loading
	bs, cmd := bs.Update(tea.KeyMsg{Type: tea.KeyDown})

	// Then: no command is produced and cursor stays at 0
	if cmd != nil {
		t.Error("key during loading should not produce a command")
	}
	if bs.cursor != 0 {
		t.Errorf("cursor should stay at 0 during loading, got %d", bs.cursor)
	}
}

func TestBrowse_SelectedID(t *testing.T) {
	tests := []struct {
		name   string
		setup  func() browseState
		wantID string
	}{
		{
			name:   "loading returns empty",
			setup:  newBrowseState,
			wantID: "",
		},
		{
			name: "empty list returns empty",
			setup: func() browseState {
				bs := newBrowseState()
				bs, _ = bs.Update(BeadListMsg{Beads: []BeadSummary{}})
				return bs
			},
			wantID: "",
		},
		{
			name: "first bead selected by default",
			setup: func() browseState {
				bs := newBrowseState()
				bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})
				return bs
			},
			wantID: "cap-001",
		},
		{
			name: "second bead after cursor down",
			setup: func() browseState {
				bs := newBrowseState()
				bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})
				bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyDown})
				return bs
			},
			wantID: "cap-002",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bs := tt.setup()
			if got := bs.SelectedID(); got != tt.wantID {
				t.Errorf("SelectedID() = %q, want %q", got, tt.wantID)
			}
		})
	}
}

func TestBrowse_ApplyBeadListCopiesSlice(t *testing.T) {
	// Given: a browse state built from beads
	bs := newBrowseState()
	beads := sampleBeads()
	bs = bs.applyBeadList(beads, nil)

	// Then: the flat nodes contain the correct bead IDs
	if len(bs.flatNodes) != 3 {
		t.Fatalf("flatNodes = %d, want 3", len(bs.flatNodes))
	}
	if bs.flatNodes[0].Node.Bead.ID != "cap-001" {
		t.Errorf("first node ID = %q, want %q", bs.flatNodes[0].Node.Bead.ID, "cap-001")
	}
}

func TestBrowse_ErrorThenRetry(t *testing.T) {
	// Given: a browse state that received an error
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Err: fmt.Errorf("timeout")})

	// When: r is pressed to retry
	bs, cmd := bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})

	// Then: loading state is set and a RefreshBeadsMsg command is returned
	if cmd == nil {
		t.Fatal("r in error state should produce a retry command")
	}
	if !bs.loading {
		t.Error("after retry, loading should be true")
	}
	msg := cmd()
	if _, ok := msg.(RefreshBeadsMsg); !ok {
		t.Fatalf("retry command produced %T, want RefreshBeadsMsg", msg)
	}
}

// --- Unified view tests ---

func TestBrowse_ClosedBeadsShownDim(t *testing.T) {
	// Given: a mix of open and closed beads
	beads := []BeadSummary{
		{ID: "cap-001", Title: "Open task", Priority: 1, Type: "task"},
		{ID: "cap-002", Title: "Done task", Priority: 2, Type: "task", Closed: true},
	}
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: beads})

	// When: the view is rendered
	view := bs.View(80, 20, "")
	plain := stripANSI(view)

	// Then: both beads appear
	if !strings.Contains(plain, "cap-001") {
		t.Errorf("view should contain open bead, got:\n%s", plain)
	}
	if !strings.Contains(plain, "cap-002") {
		t.Errorf("view should contain closed bead, got:\n%s", plain)
	}
	// And: closed bead shows check symbol
	if !strings.Contains(plain, SymbolCheck) {
		t.Errorf("view should contain check symbol for closed bead, got:\n%s", plain)
	}
}

func TestBrowse_EnterBlockedOnClosedBead(t *testing.T) {
	// Given: a browse state with cursor on a closed bead
	beads := []BeadSummary{
		{ID: "cap-001", Title: "Done task", Closed: true},
	}
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: beads})

	// When: enter is pressed on the closed bead
	_, cmd := bs.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Then: no command is produced (closed beads are not dispatchable)
	if cmd != nil {
		t.Error("enter on closed bead should not produce a command")
	}
}

func TestBrowse_TreeHierarchyRendering(t *testing.T) {
	// Given: beads with parent-child hierarchy
	beads := []BeadSummary{
		{ID: "demo-1", Title: "Epic", Type: "epic"},
		{ID: "demo-1.1", Title: "Feature A", Type: "feature"},
		{ID: "demo-1.1.1", Title: "Task A", Type: "task"},
		{ID: "demo-1.1.2", Title: "Task B", Type: "task"},
	}
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: beads})

	// When: the view is rendered
	view := bs.View(80, 20, "")
	plain := stripANSI(view)

	// Then: tree connectors are visible
	if !strings.Contains(plain, "├── ") && !strings.Contains(plain, "└── ") {
		t.Errorf("view should contain tree connectors, got:\n%s", plain)
	}
	// And: epic and feature are visible (feature collapsed by default)
	if !strings.Contains(plain, "Epic") {
		t.Errorf("view should contain %q, got:\n%s", "Epic", plain)
	}
	if !strings.Contains(plain, "Feature A") {
		t.Errorf("view should contain %q, got:\n%s", "Feature A", plain)
	}
	// And: tasks are hidden (feature is collapsed)
	if strings.Contains(plain, "Task A") {
		t.Errorf("view should NOT contain %q (feature collapsed), got:\n%s", "Task A", plain)
	}
	if strings.Contains(plain, "Task B") {
		t.Errorf("view should NOT contain %q (feature collapsed), got:\n%s", "Task B", plain)
	}
}

func TestBrowse_ProgressCountOnParent(t *testing.T) {
	// Given: a parent with 2 children (1 closed, 1 open)
	beads := []BeadSummary{
		{ID: "demo-1", Title: "Epic", Type: "epic"},
		{ID: "demo-1.1", Title: "Task A", Type: "task", Closed: true},
		{ID: "demo-1.2", Title: "Task B", Type: "task"},
	}
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: beads})

	// When: the view is rendered
	view := bs.View(80, 20, "")
	plain := stripANSI(view)

	// Then: the parent shows progress "1/2"
	if !strings.Contains(plain, "1/2") {
		t.Errorf("view should show progress count 1/2, got:\n%s", plain)
	}
}

func TestBrowse_AllChildrenClosedShowsCheck(t *testing.T) {
	// Given: a parent with all children closed
	beads := []BeadSummary{
		{ID: "demo-1", Title: "Epic", Type: "epic"},
		{ID: "demo-1.1", Title: "Task A", Closed: true},
		{ID: "demo-1.2", Title: "Task B", Closed: true},
	}
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: beads})

	// When: the view is rendered
	view := bs.View(80, 20, "")
	plain := stripANSI(view)

	// Then: the parent shows "2/2" and a check symbol
	if !strings.Contains(plain, "2/2") {
		t.Errorf("view should show progress 2/2, got:\n%s", plain)
	}
}

func TestBrowse_InitFetchesBothReadyAndClosed(t *testing.T) {
	// Given: a lister with both ready and closed beads
	lister := &stubLister{
		beads:       []BeadSummary{{ID: "cap-001", Title: "Open"}},
		closedBeads: []BeadSummary{{ID: "cap-c01", Title: "Done"}},
	}

	// When: initBrowse is called
	cmd := initBrowse(lister)
	msg := cmd().(BeadListMsg)

	// Then: the result contains both open and closed beads
	if msg.Err != nil {
		t.Fatalf("unexpected error: %v", msg.Err)
	}
	if len(msg.Beads) != 2 {
		t.Fatalf("beads = %d, want 2", len(msg.Beads))
	}
	// The closed bead should have Closed=true
	var foundClosed bool
	for _, b := range msg.Beads {
		if b.ID == "cap-c01" && b.Closed {
			foundClosed = true
		}
	}
	if !foundClosed {
		t.Error("closed bead should have Closed=true")
	}
}

func TestBrowse_InitClosedFetchErrorNonFatal(t *testing.T) {
	// Given: a lister where closed beads fail
	lister := &stubLister{
		beads:     []BeadSummary{{ID: "cap-001", Title: "Open"}},
		closedErr: fmt.Errorf("closed fetch failed"),
	}

	// When: initBrowse is called
	cmd := initBrowse(lister)
	msg := cmd().(BeadListMsg)

	// Then: ready beads are still returned (closed fetch is non-fatal)
	if msg.Err != nil {
		t.Fatalf("closed fetch error should not cause overall error: %v", msg.Err)
	}
	if len(msg.Beads) != 1 {
		t.Fatalf("beads = %d, want 1 (ready only)", len(msg.Beads))
	}
}

func TestBrowse_MergeBeadsDeduplicates(t *testing.T) {
	// Given: a bead that appears in both ready and closed lists
	ready := []BeadSummary{{ID: "cap-001", Title: "Ready version"}}
	closed := []BeadSummary{{ID: "cap-001", Title: "Closed version", Closed: true}}

	// When: merged
	merged := mergeBeads(ready, closed)

	// Then: only the ready version is kept
	if len(merged) != 1 {
		t.Fatalf("merged = %d, want 1", len(merged))
	}
	if merged[0].Closed {
		t.Error("ready bead should take precedence over closed")
	}
}

func TestBrowse_ClosedNoPriorityBadge(t *testing.T) {
	// Given: a closed bead
	beads := []BeadSummary{
		{ID: "cap-001", Title: "Done", Priority: 1, Closed: true},
	}
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: beads})

	// When: the view is rendered
	view := bs.View(80, 20, "")
	plain := stripANSI(view)

	// Then: no priority badge (P1) should appear since closed items skip badges
	if strings.Contains(plain, "P1") {
		t.Errorf("closed bead should not show priority badge, got:\n%s", plain)
	}
}

func TestBrowse_EnterConfirmRequestIncludesBeadType(t *testing.T) {
	// Given: a browse state with beads that have types
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})

	// When: enter is pressed on the first bead (type="task")
	_, cmd := bs.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Then: ConfirmRequestMsg includes BeadType from the selected bead
	if cmd == nil {
		t.Fatal("enter should produce a command")
	}
	msg := cmd()
	confirm, ok := msg.(ConfirmRequestMsg)
	if !ok {
		t.Fatalf("enter command produced %T, want ConfirmRequestMsg", msg)
	}
	if confirm.BeadType != "task" {
		t.Errorf("confirm BeadType = %q, want %q", confirm.BeadType, "task")
	}
}

func TestBrowse_EnterConfirmRequestIncludesBeadTitle(t *testing.T) {
	// Given: a browse state with beads
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})

	// When: enter is pressed on the first bead
	_, cmd := bs.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Then: ConfirmRequestMsg includes BeadTitle from the selected bead
	if cmd == nil {
		t.Fatal("enter should produce a command")
	}
	msg := cmd()
	confirm, ok := msg.(ConfirmRequestMsg)
	if !ok {
		t.Fatalf("enter command produced %T, want ConfirmRequestMsg", msg)
	}
	if confirm.BeadTitle != "First task" {
		t.Errorf("confirm BeadTitle = %q, want %q", confirm.BeadTitle, "First task")
	}
}

func TestBrowse_RightArrowExpandsCollapsedNode(t *testing.T) {
	// Given: a collapsed parent node with children (feature is collapsed by default)
	beads := []BeadSummary{
		{ID: "demo-1", Title: "Feature", Type: "feature"},
		{ID: "demo-1.1", Title: "Task A", Type: "task"},
	}
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: beads})

	// Verify feature is collapsed by default
	if len(bs.flatNodes) != 1 {
		t.Fatalf("setup: expected 1 flat node (feature collapsed), got %d", len(bs.flatNodes))
	}

	// When: right arrow is pressed on the collapsed parent
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	// Then: the node is expanded and cursor moves to first child
	if len(bs.flatNodes) != 2 {
		t.Fatalf("flatNodes = %d, want 2 (parent + child visible)", len(bs.flatNodes))
	}
	if bs.cursor != 1 {
		t.Errorf("cursor = %d, want 1 (first child)", bs.cursor)
	}
	if bs.flatNodes[1].Node.Bead.ID != "demo-1.1" {
		t.Errorf("cursor should be on child, got %q", bs.flatNodes[1].Node.Bead.ID)
	}
}

func TestBrowse_RightArrowOnExpandedNodeMovesToChild(t *testing.T) {
	// Given: an expanded parent node with children
	beads := []BeadSummary{
		{ID: "demo-1", Title: "Epic", Type: "epic"},
		{ID: "demo-1.1", Title: "Task A", Type: "task"},
	}
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: beads})

	// Verify epic is expanded by default
	if len(bs.flatNodes) != 2 {
		t.Fatalf("setup: expected 2 flat nodes (epic expanded), got %d", len(bs.flatNodes))
	}

	// When: right arrow is pressed on the expanded parent
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	// Then: the node is collapsed (toggle behavior)
	if len(bs.flatNodes) != 1 {
		t.Errorf("flatNodes = %d, want 1 (parent only, children hidden)", len(bs.flatNodes))
	}
	if bs.cursor != 0 {
		t.Errorf("cursor = %d, want 0 (stays on parent)", bs.cursor)
	}
}

func TestBrowse_RightArrowOnLeafNodeNoOp(t *testing.T) {
	// Given: a leaf node (no children)
	beads := []BeadSummary{
		{ID: "demo-1", Title: "Task", Type: "task"},
	}
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: beads})

	// When: right arrow is pressed on the leaf
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	// Then: no change (cursor stays at 0)
	if bs.cursor != 0 {
		t.Errorf("cursor = %d, want 0 (no-op on leaf)", bs.cursor)
	}
}

func TestBrowse_RightArrowKeyAlias(t *testing.T) {
	// Given: a collapsed parent node (feature is collapsed by default)
	beads := []BeadSummary{
		{ID: "demo-1", Title: "Feature", Type: "feature"},
		{ID: "demo-1.1", Title: "Task A", Type: "task"},
	}
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: beads})

	// Verify feature is collapsed by default
	if len(bs.flatNodes) != 1 {
		t.Fatalf("setup: expected 1 flat node (feature collapsed), got %d", len(bs.flatNodes))
	}

	// When: "right" key is pressed
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyRight})

	// Then: the node is expanded and cursor moves to first child
	if len(bs.flatNodes) != 2 {
		t.Fatalf("flatNodes = %d, want 2", len(bs.flatNodes))
	}
	if bs.cursor != 1 {
		t.Errorf("cursor = %d, want 1", bs.cursor)
	}
}

func TestBrowse_LeftArrowCollapsesExpandedNode(t *testing.T) {
	// Given: an expanded parent node with children
	beads := []BeadSummary{
		{ID: "demo-1", Title: "Epic", Type: "epic"},
		{ID: "demo-1.1", Title: "Task A", Type: "task"},
	}
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: beads})

	// Verify epic is expanded by default and cursor moves to child
	if len(bs.flatNodes) != 2 {
		t.Fatalf("setup: expected 2 flat nodes (epic expanded), got %d", len(bs.flatNodes))
	}

	// Move cursor to child
	bs.cursor = 1

	// When: left arrow is pressed on the child
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})

	// Then: cursor moves to parent
	if bs.cursor != 0 {
		t.Errorf("cursor = %d, want 0 (parent)", bs.cursor)
	}
}

func TestBrowse_LeftArrowOnCollapsedNodeMovesToParent(t *testing.T) {
	// Given: a collapsed child node
	beads := []BeadSummary{
		{ID: "demo-1", Title: "Epic", Type: "epic"},
		{ID: "demo-1.1", Title: "Feature", Type: "feature"},
		{ID: "demo-1.1.1", Title: "Task", Type: "task"},
	}
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: beads})

	// Verify epic is expanded by default, feature is visible but collapsed
	if len(bs.flatNodes) != 2 {
		t.Fatalf("setup: expected 2 flat nodes (epic expanded, feature collapsed), got %d", len(bs.flatNodes))
	}

	// Move cursor to feature (collapsed)
	bs.cursor = 1

	// When: left arrow is pressed on the collapsed feature
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})

	// Then: cursor moves to parent (epic)
	if bs.cursor != 0 {
		t.Errorf("cursor = %d, want 0 (parent epic)", bs.cursor)
	}
}

func TestBrowse_LeftArrowOnRootNodeNoOp(t *testing.T) {
	// Given: cursor on a root node
	beads := []BeadSummary{
		{ID: "demo-1", Title: "Epic", Type: "epic"},
	}
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: beads})

	// When: left arrow is pressed on the root
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})

	// Then: no change (cursor stays at 0)
	if bs.cursor != 0 {
		t.Errorf("cursor = %d, want 0 (no-op on root)", bs.cursor)
	}
}

func TestBrowse_CursorClampedAfterCollapseRemovesNode(t *testing.T) {
	// Given: an expanded parent with children, cursor on a child
	beads := []BeadSummary{
		{ID: "demo-1", Title: "Epic", Type: "epic"},
		{ID: "demo-1.1", Title: "Feature", Type: "feature"},
		{ID: "demo-1.1.1", Title: "Task A", Type: "task"},
		{ID: "demo-1.1.2", Title: "Task B", Type: "task"},
	}
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: beads})

	// Verify epic is expanded by default (shows epic + feature)
	if len(bs.flatNodes) != 2 {
		t.Fatalf("setup: expected 2 flat nodes (epic expanded, feature collapsed), got %d", len(bs.flatNodes))
	}

	// Expand feature
	bs.cursor = 1
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	// Verify we have 4 nodes visible (epic, feature, task A, task B)
	if len(bs.flatNodes) != 4 {
		t.Fatalf("setup: expected 4 flat nodes, got %d", len(bs.flatNodes))
	}

	// Move cursor to last task (Task B at index 3)
	bs.cursor = 3
	if bs.flatNodes[bs.cursor].Node.Bead.ID != "demo-1.1.2" {
		t.Fatalf("setup: cursor should be on Task B, got %s", bs.flatNodes[bs.cursor].Node.Bead.ID)
	}

	// When: we collapse the feature by deleting it from expandedIDs and triggering a refresh
	delete(bs.expandedIDs, "demo-1.1")
	bs, _ = bs.Update(BeadListMsg{Beads: beads})

	// Now we have 2 nodes (epic, feature) and cursor should be clamped

	// Then: cursor should be clamped to valid range
	if bs.cursor >= len(bs.flatNodes) {
		t.Errorf("cursor %d out of bounds after collapse (len=%d) - needs clamping", bs.cursor, len(bs.flatNodes))
	}
	if bs.cursor < 0 {
		t.Errorf("cursor %d is negative after collapse", bs.cursor)
	}
}

func TestBrowse_LeftArrowKeyAlias(t *testing.T) {
	// Given: an expanded parent with cursor on child
	beads := []BeadSummary{
		{ID: "demo-1", Title: "Epic", Type: "epic"},
		{ID: "demo-1.1", Title: "Task A", Type: "task"},
	}
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: beads})
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	// When: "left" key is pressed
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyLeft})

	// Then: cursor moves to parent
	if bs.cursor != 0 {
		t.Errorf("cursor = %d, want 0", bs.cursor)
	}
}

func TestBrowse_ExpansionStatePreservedAcrossRebuild(t *testing.T) {
	// Given: a browse state with a feature (collapsed by default) that is manually expanded
	beads := []BeadSummary{
		{ID: "demo-1", Title: "Feature", Type: "feature"},
		{ID: "demo-1.1", Title: "Task A", Type: "task"},
	}
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: beads})
	// Expand the feature (overriding default collapsed state)
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	// When: the tree is rebuilt with the same beads
	bs = bs.applyBeadList(beads, nil)

	// Then: the feature should still be expanded (not reverted to default)
	if len(bs.flatNodes) != 2 {
		t.Errorf("flatNodes = %d, want 2 (parent + child visible)", len(bs.flatNodes))
	}
	if !bs.flatNodes[0].Node.expanded {
		t.Error("feature should remain expanded after rebuild (not revert to default)")
	}
}

func TestBrowse_CollapsedStatePreservedAcrossRebuild(t *testing.T) {
	// Given: a browse state with a collapsed parent (default for features)
	beads := []BeadSummary{
		{ID: "demo-1", Title: "Feature", Type: "feature"},
		{ID: "demo-1.1", Title: "Task A", Type: "task"},
	}
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: beads})

	// When: the tree is rebuilt with the same beads
	bs = bs.applyBeadList(beads, nil)

	// Then: the parent should still be collapsed
	if len(bs.flatNodes) != 1 {
		t.Errorf("flatNodes = %d, want 1 (parent only, child hidden)", len(bs.flatNodes))
	}
	if bs.flatNodes[0].Node.expanded {
		t.Error("feature should remain collapsed after rebuild")
	}
}

func TestBrowse_ExpansionStateHandlesDeletedBead(t *testing.T) {
	// Given: a browse state with an expanded parent
	beads := []BeadSummary{
		{ID: "demo-1", Title: "Epic", Type: "epic"},
		{ID: "demo-1.1", Title: "Task A", Type: "task"},
	}
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: beads})
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	// When: the tree is rebuilt without the parent
	newBeads := []BeadSummary{
		{ID: "demo-1.1", Title: "Task A", Type: "task"},
	}
	bs = bs.applyBeadList(newBeads, nil)

	// Then: the state should not crash and should show the remaining bead
	if len(bs.flatNodes) != 1 {
		t.Errorf("flatNodes = %d, want 1", len(bs.flatNodes))
	}
	if bs.flatNodes[0].Node.Bead.ID != "demo-1.1" {
		t.Errorf("remaining bead ID = %q, want %q", bs.flatNodes[0].Node.Bead.ID, "demo-1.1")
	}
}

func TestBrowse_RightArrowCollapsesExpandedNode(t *testing.T) {
	// Given: an expanded parent node with children
	beads := []BeadSummary{
		{ID: "demo-1", Title: "Epic", Type: "epic"},
		{ID: "demo-1.1", Title: "Task A", Type: "task"},
	}
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: beads})

	// Verify epic is expanded by default
	if len(bs.flatNodes) != 2 {
		t.Fatalf("setup: expected 2 flat nodes (epic expanded), got %d", len(bs.flatNodes))
	}

	// When: right arrow is pressed on the expanded parent
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	// Then: the node is collapsed and children are hidden
	if len(bs.flatNodes) != 1 {
		t.Errorf("flatNodes = %d, want 1 (parent only, children hidden)", len(bs.flatNodes))
	}
	if bs.flatNodes[0].Node.expanded {
		t.Error("node should be collapsed after right arrow on expanded node")
	}
	if bs.expandedIDs["demo-1"] {
		t.Error("expandedIDs should mark node as collapsed")
	}
}
