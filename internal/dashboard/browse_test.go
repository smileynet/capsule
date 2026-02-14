package dashboard

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// stubLister implements BeadLister for tests.
type stubLister struct {
	beads []BeadSummary
	err   error
}

func (s *stubLister) Ready() ([]BeadSummary, error) {
	return s.beads, s.err
}

func sampleBeads() []BeadSummary {
	return []BeadSummary{
		{ID: "cap-001", Title: "First task", Priority: 1, Type: "task"},
		{ID: "cap-002", Title: "Second task", Priority: 2, Type: "feature"},
		{ID: "cap-003", Title: "Third task", Priority: 3, Type: "task"},
	}
}

func TestBrowse_LoadingState(t *testing.T) {
	bs := newBrowseState()
	view := bs.View(40, 20)
	if !containsPlainText(view, "Loading") {
		t.Errorf("loading view should contain 'Loading', got:\n%s", stripANSI(view))
	}
}

func TestBrowse_BeadsLoadedView(t *testing.T) {
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})

	view := bs.View(60, 20)
	plain := stripANSI(view)

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
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})

	if bs.cursor != 0 {
		t.Errorf("cursor = %d, want 0", bs.cursor)
	}
}

func TestBrowse_CursorDown(t *testing.T) {
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})

	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyDown})
	if bs.cursor != 1 {
		t.Errorf("after down: cursor = %d, want 1", bs.cursor)
	}
}

func TestBrowse_CursorUp(t *testing.T) {
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})

	// Move down first, then up.
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyDown})
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyUp})
	if bs.cursor != 0 {
		t.Errorf("after down+up: cursor = %d, want 0", bs.cursor)
	}
}

func TestBrowse_CursorWrapsDown(t *testing.T) {
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})

	// Move past last item.
	for range len(sampleBeads()) {
		bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if bs.cursor != 0 {
		t.Errorf("after wrapping down: cursor = %d, want 0", bs.cursor)
	}
}

func TestBrowse_CursorWrapsUp(t *testing.T) {
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})

	// Move up from zero wraps to last.
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyUp})
	want := len(sampleBeads()) - 1
	if bs.cursor != want {
		t.Errorf("after wrapping up: cursor = %d, want %d", bs.cursor, want)
	}
}

func TestBrowse_CursorMarker(t *testing.T) {
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})

	view := bs.View(60, 20)
	plain := stripANSI(view)
	if !strings.Contains(plain, CursorMarker) {
		t.Errorf("view should contain cursor marker %q, got:\n%s", CursorMarker, plain)
	}
}

func TestBrowse_VimKeys(t *testing.T) {
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})

	// j moves down.
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if bs.cursor != 1 {
		t.Errorf("after j: cursor = %d, want 1", bs.cursor)
	}

	// k moves up.
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if bs.cursor != 0 {
		t.Errorf("after k: cursor = %d, want 0", bs.cursor)
	}
}

func TestBrowse_EnterDispatchesSelectedBead(t *testing.T) {
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})

	// Move to second item and press enter.
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyDown})
	_, cmd := bs.Update(tea.KeyMsg{Type: tea.KeyEnter})

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
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Err: fmt.Errorf("connection failed")})

	view := bs.View(60, 20)
	plain := stripANSI(view)
	if !strings.Contains(plain, "connection failed") {
		t.Errorf("error view should contain error message, got:\n%s", plain)
	}
	if !strings.Contains(plain, "Press r to retry") {
		t.Errorf("error view should contain retry hint, got:\n%s", plain)
	}
}

func TestBrowse_EmptyList(t *testing.T) {
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: []BeadSummary{}})

	view := bs.View(60, 20)
	plain := stripANSI(view)
	if !strings.Contains(plain, "No ready beads") {
		t.Errorf("empty list view should contain 'No ready beads', got:\n%s", plain)
	}
}

func TestBrowse_RefreshReloads(t *testing.T) {
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})

	bs, cmd := bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
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
	lister := &stubLister{beads: sampleBeads()}
	cmd := initBrowse(lister)
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
	lister := &stubLister{err: fmt.Errorf("db down")}
	cmd := initBrowse(lister)
	msg := cmd().(BeadListMsg)
	if msg.Err == nil {
		t.Fatal("initBrowse with error lister should produce BeadListMsg with Err")
	}
}

func TestBrowse_PriorityBadgesInView(t *testing.T) {
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: sampleBeads()})

	view := bs.View(60, 20)
	plain := stripANSI(view)
	if !strings.Contains(plain, "P1") {
		t.Errorf("view should contain priority badge P1, got:\n%s", plain)
	}
	if !strings.Contains(plain, "P2") {
		t.Errorf("view should contain priority badge P2, got:\n%s", plain)
	}
}

func TestBrowse_KeysIgnoredDuringLoading(t *testing.T) {
	bs := newBrowseState()
	// Still in loading state, keys should be no-ops.
	bs, cmd := bs.Update(tea.KeyMsg{Type: tea.KeyDown})
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

func TestBrowse_ErrorThenRetry(t *testing.T) {
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Err: fmt.Errorf("timeout")})

	// Press r to retry.
	bs, cmd := bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
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
