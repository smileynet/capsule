package dashboard

import "testing"

func TestTreeNode_ExpansionState(t *testing.T) {
	// Given: a tree node
	node := &treeNode{
		Bead: BeadSummary{ID: "epic-1", Type: "epic"},
	}

	// When: expansion state is checked
	// Then: node should have an expanded field
	if node.expanded {
		t.Error("new node should not be expanded by default")
	}
}

func TestBuildTree_DefaultExpansion(t *testing.T) {
	// Given: beads with epics and features
	beads := []BeadSummary{
		{ID: "epic-1", Title: "Epic", Type: "epic"},
		{ID: "epic-1.1", Title: "Feature A", Type: "feature"},
		{ID: "epic-1.2", Title: "Feature B", Type: "feature"},
	}

	// When: tree is built
	roots := buildTree(beads)

	// Then: epics should be expanded by default
	epic := roots[0]
	if !epic.expanded {
		t.Error("epic should be expanded by default")
	}

	// And: features should not be expanded by default
	for i, child := range epic.Children {
		if child.expanded {
			t.Errorf("feature[%d] should not be expanded by default", i)
		}
	}
}

func TestIsExpandable_EpicWithChildren(t *testing.T) {
	// Given: an epic with children
	node := &treeNode{
		Bead: BeadSummary{ID: "epic-1", Type: "epic"},
		Children: []*treeNode{
			{Bead: BeadSummary{ID: "epic-1.1", Type: "feature"}},
		},
	}

	// When: checking if expandable
	// Then: should be expandable
	if !isExpandable(node) {
		t.Error("epic with children should be expandable")
	}
}

func TestIsExpandable_FeatureWithChildren(t *testing.T) {
	// Given: a feature with children
	node := &treeNode{
		Bead: BeadSummary{ID: "feat-1", Type: "feature"},
		Children: []*treeNode{
			{Bead: BeadSummary{ID: "feat-1.1", Type: "task"}},
		},
	}

	// When: checking if expandable
	// Then: should be expandable
	if !isExpandable(node) {
		t.Error("feature with children should be expandable")
	}
}

func TestIsExpandable_TaskWithChildren(t *testing.T) {
	// Given: a task with children (edge case)
	node := &treeNode{
		Bead: BeadSummary{ID: "task-1", Type: "task"},
		Children: []*treeNode{
			{Bead: BeadSummary{ID: "task-1.1", Type: "task"}},
		},
	}

	// When: checking if expandable
	// Then: should be expandable (any node with children)
	if !isExpandable(node) {
		t.Error("task with children should be expandable")
	}
}

func TestIsExpandable_LeafNode(t *testing.T) {
	// Given: a node with no children
	node := &treeNode{
		Bead:     BeadSummary{ID: "task-1", Type: "task"},
		Children: nil,
	}

	// When: checking if expandable
	// Then: should not be expandable
	if isExpandable(node) {
		t.Error("leaf node should not be expandable")
	}
}

func TestBuildTree_NestedDefaultExpansion(t *testing.T) {
	// Given: epic → feature → task hierarchy
	beads := []BeadSummary{
		{ID: "epic-1", Title: "Epic", Type: "epic"},
		{ID: "epic-1.1", Title: "Feature", Type: "feature"},
		{ID: "epic-1.1.1", Title: "Task", Type: "task"},
	}

	// When: tree is built
	roots := buildTree(beads)

	// Then: epic is expanded
	epic := roots[0]
	if !epic.expanded {
		t.Error("epic should be expanded by default")
	}

	// And: feature is not expanded
	feature := epic.Children[0]
	if feature.expanded {
		t.Error("feature should not be expanded by default")
	}

	// And: task has no expansion state (leaf)
	task := feature.Children[0]
	if task.expanded {
		t.Error("leaf task should not be expanded")
	}
}

func TestFlattenTree_CollapsedNodeHidesChildren(t *testing.T) {
	// Given: epic → feature (collapsed) → task hierarchy
	beads := []BeadSummary{
		{ID: "epic-1", Title: "Epic", Type: "epic"},
		{ID: "epic-1.1", Title: "Feature", Type: "feature"},
		{ID: "epic-1.1.1", Title: "Task A", Type: "task"},
		{ID: "epic-1.1.2", Title: "Task B", Type: "task"},
	}

	// When: tree is built and feature is collapsed
	roots := buildTree(beads)
	roots[0].Children[0].expanded = false

	// And: tree is flattened
	flat := flattenTree(roots)

	// Then: only epic and feature should be visible (tasks hidden)
	if len(flat) != 2 {
		t.Errorf("flat nodes = %d, want 2 (epic + feature, tasks hidden)", len(flat))
	}
	if flat[0].Node.Bead.ID != "epic-1" {
		t.Errorf("flat[0] = %q, want %q", flat[0].Node.Bead.ID, "epic-1")
	}
	if flat[1].Node.Bead.ID != "epic-1.1" {
		t.Errorf("flat[1] = %q, want %q", flat[1].Node.Bead.ID, "epic-1.1")
	}
}

func TestFlattenTree_ExpandedNodeShowsChildren(t *testing.T) {
	// Given: epic → feature (expanded) → task hierarchy
	beads := []BeadSummary{
		{ID: "epic-1", Title: "Epic", Type: "epic"},
		{ID: "epic-1.1", Title: "Feature", Type: "feature"},
		{ID: "epic-1.1.1", Title: "Task A", Type: "task"},
		{ID: "epic-1.1.2", Title: "Task B", Type: "task"},
	}

	// When: tree is built and feature is expanded
	roots := buildTree(beads)
	roots[0].Children[0].expanded = true

	// And: tree is flattened
	flat := flattenTree(roots)

	// Then: all nodes should be visible
	if len(flat) != 4 {
		t.Errorf("flat nodes = %d, want 4 (all visible)", len(flat))
	}
	wantIDs := []string{"epic-1", "epic-1.1", "epic-1.1.1", "epic-1.1.2"}
	for i, want := range wantIDs {
		if flat[i].Node.Bead.ID != want {
			t.Errorf("flat[%d] = %q, want %q", i, flat[i].Node.Bead.ID, want)
		}
	}
}

func TestFlattenTree_MixedExpansionStates(t *testing.T) {
	// Given: epic with 2 features, one expanded, one collapsed
	beads := []BeadSummary{
		{ID: "epic-1", Title: "Epic", Type: "epic"},
		{ID: "epic-1.1", Title: "Feature A", Type: "feature"},
		{ID: "epic-1.1.1", Title: "Task A1", Type: "task"},
		{ID: "epic-1.2", Title: "Feature B", Type: "feature"},
		{ID: "epic-1.2.1", Title: "Task B1", Type: "task"},
	}

	// When: tree is built with mixed expansion
	roots := buildTree(beads)
	roots[0].Children[0].expanded = true  // Feature A expanded
	roots[0].Children[1].expanded = false // Feature B collapsed

	// And: tree is flattened
	flat := flattenTree(roots)

	// Then: epic, both features, and only Feature A's task visible
	if len(flat) != 4 {
		t.Errorf("flat nodes = %d, want 4 (epic + 2 features + 1 task)", len(flat))
	}
	wantIDs := []string{"epic-1", "epic-1.1", "epic-1.1.1", "epic-1.2"}
	for i, want := range wantIDs {
		if i >= len(flat) {
			t.Fatalf("flat[%d] missing, want %q", i, want)
		}
		if flat[i].Node.Bead.ID != want {
			t.Errorf("flat[%d] = %q, want %q", i, flat[i].Node.Bead.ID, want)
		}
	}
}
