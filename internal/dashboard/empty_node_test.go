package dashboard

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

// TestEmptyExpandableNode_ShowsNoOpenTasks tests that when an expandable node
// has zero open children, it displays "(no open tasks)" when expanded.
func TestEmptyExpandableNode_ShowsNoOpenTasks(t *testing.T) {
	// Given: an epic with only closed children
	beads := []BeadSummary{
		{ID: "cap-1", Title: "Epic", Type: "epic"},
		{ID: "cap-1.1", Title: "Task A", Type: "task", Closed: true},
		{ID: "cap-1.2", Title: "Task B", Type: "task", Closed: true},
	}

	// When: the tree is built and the epic is expanded
	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: beads})
	// Expand the epic
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	// Then: the epic should show [0] badge
	view := bs.View(80, 20, "")

	if !strings.Contains(view, "[0]") {
		t.Errorf("View should show [0] badge for empty epic, got:\n%s", view)
	}

	// And: when expanded, should show "(no open tasks)" placeholder
	if !strings.Contains(view, "(no open tasks)") {
		t.Errorf("View should show '(no open tasks)' when epic is expanded with no open children, got:\n%s", view)
	}
}

// TestEmptyExpandableNode_RightArrowExpandsButDoesNotMoveCursor tests that
// pressing right arrow on an empty expandable node expands it but keeps cursor in place.
func TestEmptyExpandableNode_RightArrowExpandsButDoesNotMoveCursor(t *testing.T) {
	// Given: an epic with only closed children, collapsed
	beads := []BeadSummary{
		{ID: "cap-1", Title: "Epic", Type: "epic"},
		{ID: "cap-1.1", Title: "Task A", Type: "task", Closed: true},
		{ID: "cap-2", Title: "Another Epic", Type: "epic"},
	}

	bs := newBrowseState()
	bs, _ = bs.Update(BeadListMsg{Beads: beads})

	initialID := bs.SelectedID()

	// When: right arrow is pressed on the empty epic
	bs, _ = bs.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})

	// Then: the node should be expanded
	if !bs.expandedIDs["cap-1"] {
		t.Error("Epic should be expanded after right arrow")
	}

	// And: cursor should remain on the same node (not move to closed child)
	if bs.SelectedID() != initialID {
		t.Errorf("Selected ID should not change, was %s, now %s", initialID, bs.SelectedID())
	}
}

// TestEmptyExpandableNode_ChildCountBadgeShowsZero tests that the child count
// badge shows [0] for expandable nodes with no open children.
func TestEmptyExpandableNode_ChildCountBadgeShowsZero(t *testing.T) {
	// Given: an epic with only closed children
	node := &treeNode{
		Bead: BeadSummary{ID: "cap-1", Title: "Epic", Type: "epic"},
		Children: []*treeNode{
			{Bead: BeadSummary{ID: "cap-1.1", Title: "Task A", Closed: true}},
			{Bead: BeadSummary{ID: "cap-1.2", Title: "Task B", Closed: true}},
		},
	}

	// When: we count open children
	count := openChildCount(node)

	// Then: count should be 0
	if count != 0 {
		t.Errorf("openChildCount = %d, want 0", count)
	}
}
