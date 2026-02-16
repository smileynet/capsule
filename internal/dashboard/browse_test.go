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

func TestBrowse_EnterDispatchesSelectedBead(t *testing.T) {
	// Given: a browse state with cursor on the second bead
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyDown})

	// When: enter is pressed
	_, cmd := bs.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Then: a DispatchMsg is produced with the selected bead ID
	if cmd == nil {
		t.Fatal("enter should produce a command")
	}
	msg := cmd()
	dispatch, ok := msg.(DispatchMsg)
	if !ok {
		t.Fatalf("enter command produced %T, want DispatchMsg", msg)
	}
	if dispatch.BeadID != "cap-002" {
		t.Errorf("dispatch bead ID = %q, want %q", dispatch.BeadID, "cap-002")
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

	// Then: a "No ready beads" message with refresh hint is shown
	if !strings.Contains(plain, "No ready beads") {
		t.Errorf("empty list view should contain 'No ready beads', got:\n%s", plain)
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
			// Given: a browse state still loading
			// Then: SelectedID returns empty
			name:   "loading returns empty",
			setup:  newBrowseState,
			wantID: "",
		},
		{
			// Given: a browse state with an empty bead list
			// Then: SelectedID returns empty
			name: "empty list returns empty",
			setup: func() browseState {
				bs := newBrowseState()
				bs, _ = bs.Update(BeadListMsg{Beads: []BeadSummary{}})
				return bs
			},
			wantID: "",
		},
		{
			// Given: a browse state with loaded beads and cursor at 0
			// Then: SelectedID returns the first bead
			name: "first bead selected by default",
			setup: func() browseState {
				bs := newBrowseState()
				bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})
				return bs
			},
			wantID: "cap-001",
		},
		{
			// Given: a browse state with cursor moved down once
			// Then: SelectedID returns the second bead
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

func sampleClosedBeads() []BeadSummary {
	return []BeadSummary{
		{ID: "cap-c01", Title: "Done task", Priority: 2, Type: "task"},
		{ID: "cap-c02", Title: "Done feature", Priority: 1, Type: "feature"},
	}
}

func TestBrowse_HToggleSwitchesToClosed(t *testing.T) {
	// Given: a browse state showing ready beads
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})

	// When: h is pressed to toggle history
	bs, cmd := bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})

	// Then: showClosed is true and a ToggleHistoryMsg command is returned
	if !bs.showClosed {
		t.Error("after h, showClosed should be true")
	}
	if cmd == nil {
		t.Fatal("h should produce a command to fetch closed beads")
	}
	msg := cmd()
	if _, ok := msg.(ToggleHistoryMsg); !ok {
		t.Fatalf("h command produced %T, want ToggleHistoryMsg", msg)
	}
}

func TestBrowse_HToggleSwitchesBackToReady(t *testing.T) {
	// Given: a browse state showing closed beads
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	bs, _ = bs.Update(ClosedBeadListMsg{Beads: sampleClosedBeads()})

	// When: h is pressed again to toggle back
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})

	// Then: showClosed is false and ready beads are displayed
	if bs.showClosed {
		t.Error("after second h, showClosed should be false")
	}
	if len(bs.beads) != len(sampleBeads()) {
		t.Errorf("after toggle back, beads count = %d, want %d", len(bs.beads), len(sampleBeads()))
	}
}

func TestBrowse_ClosedBeadsLoadedView(t *testing.T) {
	// Given: a browse state that received closed beads
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	bs, _ = bs.Update(ClosedBeadListMsg{Beads: sampleClosedBeads()})

	// When: the view is rendered
	view := bs.View(80, 20, "")
	plain := stripANSI(view)

	// Then: closed bead IDs appear
	for _, b := range sampleClosedBeads() {
		if !strings.Contains(plain, b.ID) {
			t.Errorf("view should contain closed bead ID %q, got:\n%s", b.ID, plain)
		}
	}
}

func TestBrowse_ClosedBeadsCursorResetsToZero(t *testing.T) {
	// Given: a browse state with cursor at position 1 in ready list
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyDown})

	// When: h is pressed and closed beads are loaded
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	bs, _ = bs.Update(ClosedBeadListMsg{Beads: sampleClosedBeads()})

	// Then: cursor is reset to 0
	if bs.cursor != 0 {
		t.Errorf("cursor after switching to closed = %d, want 0", bs.cursor)
	}
}

func TestBrowse_ClosedBeadsError(t *testing.T) {
	// Given: a browse state that received a closed beads error
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	bs, _ = bs.Update(ClosedBeadListMsg{Err: fmt.Errorf("db error")})

	// When: the view is rendered
	view := bs.View(80, 20, "")
	plain := stripANSI(view)

	// Then: error is shown
	if !strings.Contains(plain, "db error") {
		t.Errorf("view should show error, got:\n%s", plain)
	}
}

func TestBrowse_ClosedBeadsNavigation(t *testing.T) {
	// Given: a browse state showing closed beads
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	bs, _ = bs.Update(ClosedBeadListMsg{Beads: sampleClosedBeads()})

	// When: down key is pressed
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyDown})

	// Then: cursor moves through closed beads
	if bs.cursor != 1 {
		t.Errorf("after down in closed mode: cursor = %d, want 1", bs.cursor)
	}
}

func TestBrowse_HKeyIgnoredDuringLoading(t *testing.T) {
	// Given: a browse state still loading
	bs := newBrowseState()

	// When: h is pressed during loading
	bs, cmd := bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})

	// Then: no command is produced and showClosed stays false
	if cmd != nil {
		t.Error("h during loading should not produce a command")
	}
	if bs.showClosed {
		t.Error("showClosed should stay false during loading")
	}
}

func TestBrowse_EnterIgnoredInClosedView(t *testing.T) {
	// Given: a browse state showing closed beads
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	bs, _ = bs.Update(ClosedBeadListMsg{Beads: sampleClosedBeads()})

	// When: enter is pressed on a closed bead
	_, cmd := bs.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Then: no command is produced (closed beads are not dispatchable)
	if cmd != nil {
		t.Error("enter on closed beads should not produce a command")
	}
}

func TestBrowse_ApplyBeadListCopiesSlice(t *testing.T) {
	// Given: an incoming bead slice
	incoming := sampleBeads()

	// When: applyBeadList stores the beads
	bs := newBrowseState()
	bs = bs.applyBeadList(incoming, nil)

	// Then: mutating the original slice does not affect the stored beads
	incoming[0].Title = "MUTATED"
	if bs.beads[0].Title == "MUTATED" {
		t.Error("applyBeadList should copy the slice; mutation of original affected stored beads")
	}
}

func TestBrowse_ToggleHistoryCopiesReadyBeads(t *testing.T) {
	// Given: a browse state showing ready beads
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})

	// When: h is pressed to toggle to closed (saves readyBeads)
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})

	// Then: mutating bs.beads does not affect readyBeads
	bs.beads[0].Title = "MUTATED"
	if bs.readyBeads[0].Title == "MUTATED" {
		t.Error("toggle should copy beads to readyBeads; mutation of beads affected readyBeads")
	}
}

func TestBrowse_ToggleBackCopiesBeads(t *testing.T) {
	// Given: a browse state that toggled to closed view
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}}) // toggle to closed

	// When: h is pressed again to toggle back (restores readyBeads)
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})

	// Then: mutating the restored beads does not affect readyBeads
	bs.beads[0].Title = "MUTATED"
	if bs.readyBeads[0].Title == "MUTATED" {
		t.Error("toggle-back should copy readyBeads; mutation of beads affected readyBeads")
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
