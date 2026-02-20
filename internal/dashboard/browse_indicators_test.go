package dashboard

import (
	"strings"
	"testing"
)

func TestBrowse_ExpandedIndicator(t *testing.T) {
	// Given: an epic with children, expanded by default
	beads := []BeadSummary{
		{ID: "cap-001", Title: "Epic", Priority: 2, Type: "epic"},
		{ID: "cap-001.1", Title: "Task", Priority: 2, Type: "task"},
	}
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: beads})

	// When: the view is rendered
	view := bs.View(80, 20, "")
	plain := stripANSI(view)

	// Then: the epic shows expanded indicator ▼
	if !strings.Contains(plain, "▼") {
		t.Errorf("expanded epic should show ▼ indicator, got:\n%s", plain)
	}
}

func TestBrowse_CollapsedIndicator(t *testing.T) {
	// Given: a feature with children, collapsed by default
	beads := []BeadSummary{
		{ID: "cap-001", Title: "Feature", Priority: 2, Type: "feature"},
		{ID: "cap-001.1", Title: "Task", Priority: 2, Type: "task"},
	}
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: beads})

	// When: the view is rendered
	view := bs.View(80, 20, "")
	plain := stripANSI(view)

	// Then: the feature shows collapsed indicator ▶
	if !strings.Contains(plain, "▶") {
		t.Errorf("collapsed feature should show ▶ indicator, got:\n%s", plain)
	}
}

func TestBrowse_LeafIndicator(t *testing.T) {
	// Given: a task with no children
	beads := []BeadSummary{
		{ID: "cap-001", Title: "Task", Priority: 2, Type: "task"},
	}
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: beads})

	// When: the view is rendered
	view := bs.View(80, 20, "")
	plain := stripANSI(view)

	// Then: the task shows leaf indicator •
	if !strings.Contains(plain, "•") {
		t.Errorf("leaf task should show • indicator, got:\n%s", plain)
	}
}

func TestBrowse_ChildCountBadge(t *testing.T) {
	// Given: an epic with 2 open children
	beads := []BeadSummary{
		{ID: "cap-001", Title: "Epic", Priority: 2, Type: "epic"},
		{ID: "cap-001.1", Title: "Task 1", Priority: 2, Type: "task"},
		{ID: "cap-001.2", Title: "Task 2", Priority: 2, Type: "task"},
	}
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: beads})

	// When: the view is rendered
	view := bs.View(80, 20, "")
	plain := stripANSI(view)

	// Then: the epic shows child count badge [2]
	if !strings.Contains(plain, "[2]") {
		t.Errorf("epic should show [2] child count badge, got:\n%s", plain)
	}
}

func TestBrowse_EmptyNodeShowsZero(t *testing.T) {
	// Given: an epic with no open children (all closed)
	beads := []BeadSummary{
		{ID: "cap-001", Title: "Epic", Priority: 2, Type: "epic"},
		{ID: "cap-001.1", Title: "Task 1", Priority: 2, Type: "task", Closed: true},
	}
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: beads})

	// When: the view is rendered
	view := bs.View(80, 20, "")
	plain := stripANSI(view)

	// Then: the epic shows [0] child count badge
	if !strings.Contains(plain, "[0]") {
		t.Errorf("empty epic should show [0] child count badge, got:\n%s", plain)
	}
}
