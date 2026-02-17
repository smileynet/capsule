package dashboard

import (
	"strings"
	"testing"
)

func TestBuildTree_FlatList(t *testing.T) {
	// Given: beads with no parent-child relationships
	beads := []BeadSummary{
		{ID: "cap-001", Title: "Task A"},
		{ID: "cap-002", Title: "Task B"},
		{ID: "cap-003", Title: "Task C"},
	}

	// When: the tree is built
	roots := buildTree(beads)

	// Then: all beads are roots
	if len(roots) != 3 {
		t.Fatalf("roots = %d, want 3", len(roots))
	}
	for i, root := range roots {
		if len(root.Children) != 0 {
			t.Errorf("root[%d] has %d children, want 0", i, len(root.Children))
		}
	}
}

func TestBuildTree_SingleLevel(t *testing.T) {
	// Given: an epic with 2 feature children
	beads := []BeadSummary{
		{ID: "demo-1", Title: "Epic", Type: "epic"},
		{ID: "demo-1.1", Title: "Feature A", Type: "feature"},
		{ID: "demo-1.2", Title: "Feature B", Type: "feature"},
	}

	// When: the tree is built
	roots := buildTree(beads)

	// Then: there is 1 root with 2 children
	if len(roots) != 1 {
		t.Fatalf("roots = %d, want 1", len(roots))
	}
	root := roots[0]
	if root.Bead.ID != "demo-1" {
		t.Errorf("root ID = %q, want %q", root.Bead.ID, "demo-1")
	}
	if len(root.Children) != 2 {
		t.Fatalf("children = %d, want 2", len(root.Children))
	}
	if root.Children[0].Bead.ID != "demo-1.1" {
		t.Errorf("child[0] ID = %q, want %q", root.Children[0].Bead.ID, "demo-1.1")
	}
	if root.Children[1].Bead.ID != "demo-1.2" {
		t.Errorf("child[1] ID = %q, want %q", root.Children[1].Bead.ID, "demo-1.2")
	}
}

func TestBuildTree_MultiLevel(t *testing.T) {
	// Given: epic → feature → task (3-level hierarchy)
	beads := []BeadSummary{
		{ID: "demo-1", Title: "Epic", Type: "epic"},
		{ID: "demo-1.1", Title: "Feature", Type: "feature"},
		{ID: "demo-1.1.1", Title: "Task A", Type: "task"},
		{ID: "demo-1.1.2", Title: "Task B", Type: "task"},
	}

	// When: the tree is built
	roots := buildTree(beads)

	// Then: structure is demo-1 → demo-1.1 → [demo-1.1.1, demo-1.1.2]
	if len(roots) != 1 {
		t.Fatalf("roots = %d, want 1", len(roots))
	}
	epic := roots[0]
	if len(epic.Children) != 1 {
		t.Fatalf("epic children = %d, want 1", len(epic.Children))
	}
	feature := epic.Children[0]
	if feature.Bead.ID != "demo-1.1" {
		t.Errorf("feature ID = %q, want %q", feature.Bead.ID, "demo-1.1")
	}
	if len(feature.Children) != 2 {
		t.Fatalf("feature children = %d, want 2", len(feature.Children))
	}
	if feature.Children[0].Bead.ID != "demo-1.1.1" {
		t.Errorf("task[0] ID = %q, want %q", feature.Children[0].Bead.ID, "demo-1.1.1")
	}
	if feature.Children[1].Bead.ID != "demo-1.1.2" {
		t.Errorf("task[1] ID = %q, want %q", feature.Children[1].Bead.ID, "demo-1.1.2")
	}
}

func TestBuildTree_OrphanChildren(t *testing.T) {
	// Given: child beads without their parent in the list
	beads := []BeadSummary{
		{ID: "demo-1.1", Title: "Feature A"},
		{ID: "demo-1.2", Title: "Feature B"},
	}

	// When: the tree is built
	roots := buildTree(beads)

	// Then: both are roots (orphans)
	if len(roots) != 2 {
		t.Fatalf("roots = %d, want 2", len(roots))
	}
}

func TestBuildTree_ChildrenSortedByID(t *testing.T) {
	// Given: children in unsorted order
	beads := []BeadSummary{
		{ID: "demo-1", Title: "Epic"},
		{ID: "demo-1.3", Title: "Third"},
		{ID: "demo-1.1", Title: "First"},
		{ID: "demo-1.2", Title: "Second"},
	}

	// When: the tree is built
	roots := buildTree(beads)

	// Then: children are sorted by ID
	if len(roots) != 1 {
		t.Fatalf("roots = %d, want 1", len(roots))
	}
	children := roots[0].Children
	if len(children) != 3 {
		t.Fatalf("children = %d, want 3", len(children))
	}
	for i, want := range []string{"demo-1.1", "demo-1.2", "demo-1.3"} {
		if children[i].Bead.ID != want {
			t.Errorf("child[%d] = %q, want %q", i, children[i].Bead.ID, want)
		}
	}
}

func TestBuildTree_IsLastMarking(t *testing.T) {
	// Given: a parent with 2 children
	beads := []BeadSummary{
		{ID: "demo-1", Title: "Epic"},
		{ID: "demo-1.1", Title: "First"},
		{ID: "demo-1.2", Title: "Second"},
	}

	// When: the tree is built
	roots := buildTree(beads)

	// Then: the last child is marked
	children := roots[0].Children
	if children[0].IsLast {
		t.Error("first child should not be IsLast")
	}
	if !children[1].IsLast {
		t.Error("second child should be IsLast")
	}
}

func TestFlattenTree_BoxDrawing(t *testing.T) {
	// Given: a tree with nested children
	beads := []BeadSummary{
		{ID: "demo-1", Title: "Epic"},
		{ID: "demo-1.1", Title: "Feature"},
		{ID: "demo-1.1.1", Title: "Task A"},
		{ID: "demo-1.1.2", Title: "Task B"},
		{ID: "demo-1.2", Title: "Feature B"},
	}

	// When: the tree is flattened
	roots := buildTree(beads)
	flat := flattenTree(roots)

	// Then: prefixes contain box-drawing characters
	if len(flat) != 5 {
		t.Fatalf("flat nodes = %d, want 5", len(flat))
	}

	// Root has no prefix
	if flat[0].Prefix != "" {
		t.Errorf("root prefix = %q, want empty", flat[0].Prefix)
	}
	// demo-1.1 (not last child of demo-1)
	if !strings.Contains(flat[1].Prefix, "├── ") {
		t.Errorf("node 1 prefix = %q, want to contain '├── '", flat[1].Prefix)
	}
	// demo-1.1.1 (not last child of demo-1.1)
	if !strings.Contains(flat[2].Prefix, "├── ") {
		t.Errorf("node 2 prefix = %q, want to contain '├── '", flat[2].Prefix)
	}
	// demo-1.1.2 (last child of demo-1.1)
	if !strings.Contains(flat[3].Prefix, "└── ") {
		t.Errorf("node 3 prefix = %q, want to contain '└── '", flat[3].Prefix)
	}
	// demo-1.2 (last child of demo-1)
	if !strings.Contains(flat[4].Prefix, "└── ") {
		t.Errorf("node 4 prefix = %q, want to contain '└── '", flat[4].Prefix)
	}
}

func TestFlattenTree_Depth(t *testing.T) {
	// Given: a 3-level tree
	beads := []BeadSummary{
		{ID: "demo-1", Title: "Epic"},
		{ID: "demo-1.1", Title: "Feature"},
		{ID: "demo-1.1.1", Title: "Task"},
	}

	roots := buildTree(beads)
	flat := flattenTree(roots)

	// Then: depths are [0, 1, 2]
	wantDepths := []int{0, 1, 2}
	if len(flat) != len(wantDepths) {
		t.Fatalf("flat nodes = %d, want %d", len(flat), len(wantDepths))
	}
	for i, want := range wantDepths {
		if flat[i].Depth != want {
			t.Errorf("flat[%d].Depth = %d, want %d", i, flat[i].Depth, want)
		}
	}
}

func TestFlattenTree_EmptyInput(t *testing.T) {
	// Given: no roots
	flat := flattenTree(nil)

	// Then: result is empty
	if len(flat) != 0 {
		t.Errorf("flat nodes = %d, want 0", len(flat))
	}
}

func TestFlattenTree_ContinuationPrefix(t *testing.T) {
	// Given: a tree where non-last children have descendants
	beads := []BeadSummary{
		{ID: "demo-1", Title: "Epic"},
		{ID: "demo-1.1", Title: "Feature A"},
		{ID: "demo-1.1.1", Title: "Task A"},
		{ID: "demo-1.2", Title: "Feature B"},
	}

	roots := buildTree(beads)
	flat := flattenTree(roots)

	// Then: task under non-last feature has "│   " continuation
	// flat[2] = demo-1.1.1 (child of demo-1.1 which is not last)
	if !strings.Contains(flat[2].Prefix, "│") {
		t.Errorf("continuation prefix should contain '│', got %q", flat[2].Prefix)
	}
}

func TestTreeProgress_NoChildren(t *testing.T) {
	// Given: a leaf node
	node := &treeNode{Bead: BeadSummary{ID: "task-1"}}

	// When: progress is computed
	stats := treeProgress(node)

	// Then: both counts are 0 (leaf has no children to count)
	if stats.Total != 0 || stats.Closed != 0 {
		t.Errorf("leaf progress = %d/%d, want 0/0", stats.Closed, stats.Total)
	}
}

func TestTreeProgress_AllOpen(t *testing.T) {
	// Given: a parent with 3 open children
	node := &treeNode{
		Bead: BeadSummary{ID: "feat-1"},
		Children: []*treeNode{
			{Bead: BeadSummary{ID: "feat-1.1"}},
			{Bead: BeadSummary{ID: "feat-1.2"}},
			{Bead: BeadSummary{ID: "feat-1.3"}},
		},
	}

	stats := treeProgress(node)

	if stats.Total != 3 {
		t.Errorf("total = %d, want 3", stats.Total)
	}
	if stats.Closed != 0 {
		t.Errorf("closed = %d, want 0", stats.Closed)
	}
}

func TestTreeProgress_Mixed(t *testing.T) {
	// Given: a parent with 2 closed and 1 open child
	node := &treeNode{
		Bead: BeadSummary{ID: "feat-1"},
		Children: []*treeNode{
			{Bead: BeadSummary{ID: "feat-1.1", Closed: true}},
			{Bead: BeadSummary{ID: "feat-1.2"}},
			{Bead: BeadSummary{ID: "feat-1.3", Closed: true}},
		},
	}

	stats := treeProgress(node)

	if stats.Total != 3 {
		t.Errorf("total = %d, want 3", stats.Total)
	}
	if stats.Closed != 2 {
		t.Errorf("closed = %d, want 2", stats.Closed)
	}
}

func TestTreeProgress_Recursive(t *testing.T) {
	// Given: epic → feature (closed) with 2 closed tasks + feature (open) with 1 open task
	node := &treeNode{
		Bead: BeadSummary{ID: "epic-1"},
		Children: []*treeNode{
			{
				Bead: BeadSummary{ID: "epic-1.1", Closed: true},
				Children: []*treeNode{
					{Bead: BeadSummary{ID: "epic-1.1.1", Closed: true}},
					{Bead: BeadSummary{ID: "epic-1.1.2", Closed: true}},
				},
			},
			{
				Bead: BeadSummary{ID: "epic-1.2"},
				Children: []*treeNode{
					{Bead: BeadSummary{ID: "epic-1.2.1"}},
				},
			},
		},
	}

	stats := treeProgress(node)

	// Total: 2 features + 3 tasks = 5
	// Closed: 1 feature + 2 tasks = 3
	if stats.Total != 5 {
		t.Errorf("total = %d, want 5", stats.Total)
	}
	if stats.Closed != 3 {
		t.Errorf("closed = %d, want 3", stats.Closed)
	}
}

func TestIsChildOf_DirectChild(t *testing.T) {
	if !isChildOf("demo-1.1", "demo-1") {
		t.Error("demo-1.1 should be a child of demo-1")
	}
}

func TestIsChildOf_NotGrandchild(t *testing.T) {
	if isChildOf("demo-1.1.1", "demo-1") {
		t.Error("demo-1.1.1 should NOT be a direct child of demo-1")
	}
}

func TestIsChildOf_Unrelated(t *testing.T) {
	if isChildOf("demo-2", "demo-1") {
		t.Error("demo-2 should not be a child of demo-1")
	}
}

func TestIsChildOf_SameID(t *testing.T) {
	if isChildOf("demo-1", "demo-1") {
		t.Error("demo-1 should not be its own child")
	}
}

func TestIsChildOf_PrefixOverlap(t *testing.T) {
	// "demo-10" starts with "demo-1" but is not a child (no dot separator)
	if isChildOf("demo-10", "demo-1") {
		t.Error("demo-10 should not be a child of demo-1 (prefix overlap without dot)")
	}
}

func TestBuildTree_MultipleRoots(t *testing.T) {
	// Given: two independent trees
	beads := []BeadSummary{
		{ID: "alpha-1", Title: "Alpha Epic"},
		{ID: "alpha-1.1", Title: "Alpha Task"},
		{ID: "beta-1", Title: "Beta Epic"},
		{ID: "beta-1.1", Title: "Beta Task"},
	}

	roots := buildTree(beads)

	if len(roots) != 2 {
		t.Fatalf("roots = %d, want 2", len(roots))
	}
	if roots[0].Bead.ID != "alpha-1" {
		t.Errorf("root[0] = %q, want %q", roots[0].Bead.ID, "alpha-1")
	}
	if roots[1].Bead.ID != "beta-1" {
		t.Errorf("root[1] = %q, want %q", roots[1].Bead.ID, "beta-1")
	}
}

func TestFlattenTree_MultipleRoots(t *testing.T) {
	// Given: two root nodes
	beads := []BeadSummary{
		{ID: "a", Title: "Root A"},
		{ID: "b", Title: "Root B"},
	}

	roots := buildTree(beads)
	flat := flattenTree(roots)

	if len(flat) != 2 {
		t.Fatalf("flat = %d, want 2", len(flat))
	}
	// Both are depth 0 with no prefix
	for i, f := range flat {
		if f.Depth != 0 {
			t.Errorf("flat[%d].Depth = %d, want 0", i, f.Depth)
		}
		if f.Prefix != "" {
			t.Errorf("flat[%d].Prefix = %q, want empty", i, f.Prefix)
		}
	}
}
