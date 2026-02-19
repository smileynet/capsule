package dashboard

import (
	"strings"
	"testing"
)

func TestConfirm_ViewPipeline(t *testing.T) {
	// Given: a confirm state for a task
	cs := confirmState{
		beadID:    "cap-001",
		beadType:  "task",
		beadTitle: "Validate email format",
	}

	// When: the view is rendered
	view := cs.View(80, 40)

	// Then: it shows the pipeline confirmation
	if !strings.Contains(view, "Run pipeline for cap-001?") {
		t.Errorf("should show pipeline prompt, got:\n%s", view)
	}
	if !strings.Contains(view, "Validate email format") {
		t.Errorf("should show bead title, got:\n%s", view)
	}
	if !strings.Contains(view, "Create a worktree branch") {
		t.Errorf("should show consequences, got:\n%s", view)
	}
	if !strings.Contains(view, "[Enter] Confirm") {
		t.Errorf("should show confirm hint, got:\n%s", view)
	}
	if !strings.Contains(view, "[Esc] Cancel") {
		t.Errorf("should show cancel hint, got:\n%s", view)
	}
}

func TestConfirm_ViewCampaign(t *testing.T) {
	// Given: a confirm state for a feature with children
	cs := confirmState{
		beadID:    "demo-1",
		beadType:  "feature",
		beadTitle: "Contact Management Library",
		children: []confirmChild{
			{ID: "demo-1.1.1", Title: "Validate email format"},
			{ID: "demo-1.1.2", Title: "Validate phone format"},
		},
	}

	// When: the view is rendered
	view := cs.View(80, 40)

	// Then: it shows the campaign confirmation with task count
	if !strings.Contains(view, "Run campaign for demo-1? (2 tasks)") {
		t.Errorf("should show campaign prompt with task count, got:\n%s", view)
	}
	if !strings.Contains(view, "Contact Management Library") {
		t.Errorf("should show bead title, got:\n%s", view)
	}
	if !strings.Contains(view, "1. demo-1.1.1") {
		t.Errorf("should list first child, got:\n%s", view)
	}
	if !strings.Contains(view, "2. demo-1.1.2") {
		t.Errorf("should list second child, got:\n%s", view)
	}
}

func TestConfirm_ViewCampaignWithValidation(t *testing.T) {
	// Given: a confirm state for a feature with validation configured
	cs := confirmState{
		beadID:        "demo-1",
		beadType:      "feature",
		beadTitle:     "Contact Management Library",
		hasValidation: true,
		children: []confirmChild{
			{ID: "demo-1.1.1", Title: "Validate email format"},
		},
	}

	// When: the view is rendered
	view := cs.View(80, 40)

	// Then: it shows validation text and step numbers
	if !strings.Contains(view, "(1 task + validation)") {
		t.Errorf("should show validation in count, got:\n%s", view)
	}
	if !strings.Contains(view, "Step 1") {
		t.Errorf("should show Step 1, got:\n%s", view)
	}
	if !strings.Contains(view, "Step 2") {
		t.Errorf("should show Step 2, got:\n%s", view)
	}
	if !strings.Contains(view, "Run acceptance pipeline on demo-1") {
		t.Errorf("should describe validation step, got:\n%s", view)
	}
}

func TestConfirm_ViewCampaignNoValidation(t *testing.T) {
	// Given: a campaign confirm without validation
	cs := confirmState{
		beadID:    "demo-1",
		beadType:  "epic",
		beadTitle: "Big Epic",
		children:  []confirmChild{{ID: "demo-1.1", Title: "Task"}},
	}

	// When: the view is rendered
	view := cs.View(80, 40)

	// Then: no "Step" text or validation mention
	if strings.Contains(view, "Step") {
		t.Errorf("should not show Step text without validation, got:\n%s", view)
	}
	if strings.Contains(view, "validation") {
		t.Errorf("should not mention validation, got:\n%s", view)
	}
}

func TestConfirm_PipelineForFeatureWithNoChildren(t *testing.T) {
	// Given: a feature with no children (falls back to pipeline view)
	cs := confirmState{
		beadID:    "demo-1",
		beadType:  "feature",
		beadTitle: "Empty Feature",
	}

	// When: the view is rendered
	view := cs.View(80, 40)

	// Then: it shows pipeline-style confirmation (no children = not a campaign)
	if !strings.Contains(view, "Run pipeline for demo-1?") {
		t.Errorf("should show pipeline prompt for featureless feature, got:\n%s", view)
	}
}

func TestCollectOpenChildren_DirectChildrenOnly(t *testing.T) {
	// Given: a tree with parent, children, and grandchildren
	beads := []BeadSummary{
		{ID: "demo-1", Title: "Epic", Type: "epic"},
		{ID: "demo-1.1", Title: "Feature A", Type: "feature"},
		{ID: "demo-1.1.1", Title: "Task A", Type: "task"},
		{ID: "demo-1.1.2", Title: "Task B", Type: "task"},
		{ID: "demo-1.2", Title: "Feature B", Type: "feature"},
	}
	roots := buildTree(beads)
	flat := flattenTree(roots)

	// When: collecting open children of demo-1
	children := collectOpenChildren(flat, "demo-1")

	// Then: only direct children (features) are returned, not grandchildren (tasks)
	if len(children) != 2 {
		t.Fatalf("children = %d, want 2", len(children))
	}
	if children[0].ID != "demo-1.1" {
		t.Errorf("children[0].ID = %q, want %q", children[0].ID, "demo-1.1")
	}
	if children[1].ID != "demo-1.2" {
		t.Errorf("children[1].ID = %q, want %q", children[1].ID, "demo-1.2")
	}
}

func TestCollectOpenChildren_SkipsClosedChildren(t *testing.T) {
	// Given: a tree with one closed and one open child
	beads := []BeadSummary{
		{ID: "demo-1", Title: "Epic", Type: "epic"},
		{ID: "demo-1.1", Title: "Done", Type: "task", Closed: true},
		{ID: "demo-1.2", Title: "Open", Type: "task"},
	}
	roots := buildTree(beads)
	flat := flattenTree(roots)

	// When: collecting open children
	children := collectOpenChildren(flat, "demo-1")

	// Then: only the open child is returned
	if len(children) != 1 {
		t.Fatalf("children = %d, want 1", len(children))
	}
	if children[0].ID != "demo-1.2" {
		t.Errorf("children[0].ID = %q, want %q", children[0].ID, "demo-1.2")
	}
}

func TestCollectOpenChildren_ParentNotFound(t *testing.T) {
	// Given: a tree that doesn't contain the requested parent
	beads := []BeadSummary{
		{ID: "cap-001", Title: "Task"},
	}
	roots := buildTree(beads)
	flat := flattenTree(roots)

	// When: collecting children of a non-existent parent
	children := collectOpenChildren(flat, "nonexistent")

	// Then: no children returned
	if len(children) != 0 {
		t.Errorf("children = %d, want 0", len(children))
	}
}

func TestCollectOpenChildren_EmptyTree(t *testing.T) {
	// When: collecting from an empty tree
	children := collectOpenChildren(nil, "demo-1")

	// Then: no children returned
	if len(children) != 0 {
		t.Errorf("children = %d, want 0", len(children))
	}
}
