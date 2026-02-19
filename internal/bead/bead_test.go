package bead

import (
	"errors"
	"testing"

	"github.com/smileynet/capsule/internal/worklog"
)

func TestExtractParentID(t *testing.T) {
	c := &Client{}
	tests := []struct {
		name string
		iss  issue
		want string
	}{
		{
			name: "direct parent field",
			iss:  issue{ID: "task-1", Parent: "feature-1"},
			want: "feature-1",
		},
		{
			name: "parent from dependencies",
			iss: issue{
				ID: "task-1",
				Dependencies: []dependency{
					{IssueID: "task-1", DependsOnID: "feature-1", Type: "parent-child"},
				},
			},
			want: "feature-1",
		},
		{
			name: "no parent",
			iss:  issue{ID: "task-1"},
			want: "",
		},
		{
			name: "skips self-referencing dependency",
			iss: issue{
				ID: "task-1",
				Dependencies: []dependency{
					{IssueID: "task-1", DependsOnID: "task-1", Type: "parent-child"},
				},
			},
			want: "",
		},
		{
			name: "skips non-parent-child dependency",
			iss: issue{
				ID: "task-1",
				Dependencies: []dependency{
					{IssueID: "task-1", DependsOnID: "other-1", Type: "blocks"},
				},
			},
			want: "",
		},
		{
			name: "parent field takes precedence over dependencies",
			iss: issue{
				ID:     "task-1",
				Parent: "feature-1",
				Dependencies: []dependency{
					{IssueID: "task-1", DependsOnID: "feature-2", Type: "parent-child"},
				},
			},
			want: "feature-1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.extractParentID(tt.iss)
			if got != tt.want {
				t.Errorf("extractParentID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolve_BDAvailable_InvalidBead(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping bd CLI test in short mode")
	}

	// Given bd is on PATH but the bead doesn't exist
	c := &Client{Dir: t.TempDir()}

	ctx, err := c.Resolve("nonexistent-bead")

	// Then an error is returned (bd available but bead not found)
	if err == nil {
		t.Fatal("expected error for nonexistent bead, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("error = %v, want ErrNotFound", err)
	}
	// And context still has TaskID as fallback
	if ctx.TaskID != "nonexistent-bead" {
		t.Errorf("TaskID = %q, want %q", ctx.TaskID, "nonexistent-bead")
	}
}

func TestResolve_NoBD(t *testing.T) {
	// Given bd is not on PATH — graceful fallback returns context with just TaskID
	c := &Client{Dir: t.TempDir()}

	// If bd is actually on PATH, skip — this test is for missing-bd fallback
	if err := c.checkBD(); err == nil {
		t.Skip("bd is on PATH; cannot test missing-bd fallback in this environment")
	}

	ctx, err := c.Resolve("test-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ctx.TaskID != "test-123" {
		t.Errorf("TaskID = %q, want %q", ctx.TaskID, "test-123")
	}
}

func TestClose_NoBD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping bd CLI test in short mode")
	}

	// Given bd is not available (or bead doesn't exist)
	c := &Client{Dir: t.TempDir()}

	// Close should return an error
	err := c.Close("nonexistent-id")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestBuildContext_FullChain(t *testing.T) {
	// Test that a fully populated context has all fields
	ctx := worklog.BeadContext{
		EpicID:             "epic-1",
		EpicTitle:          "Test Epic",
		EpicGoal:           "Build something",
		FeatureID:          "feature-1",
		FeatureTitle:       "Test Feature",
		FeatureGoal:        "Add validation",
		TaskID:             "task-1",
		TaskTitle:          "Validate email",
		TaskDescription:    "Email validation function",
		AcceptanceCriteria: "- Valid emails pass\n- Invalid emails fail",
	}

	if ctx.EpicID != "epic-1" {
		t.Errorf("EpicID = %q, want %q", ctx.EpicID, "epic-1")
	}
	if ctx.FeatureID != "feature-1" {
		t.Errorf("FeatureID = %q, want %q", ctx.FeatureID, "feature-1")
	}
	if ctx.TaskID != "task-1" {
		t.Errorf("TaskID = %q, want %q", ctx.TaskID, "task-1")
	}
}

func TestClosed_BDAvailable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping bd CLI test in short mode")
	}

	c := &Client{Dir: "."}

	// bd is on PATH; closed list should succeed (even if empty).
	summaries, err := c.Closed(5)
	if err != nil {
		t.Fatalf("Closed() unexpected error: %v", err)
	}

	// Each summary should have non-empty ID.
	for i, s := range summaries {
		if s.ID == "" {
			t.Errorf("summaries[%d].ID is empty", i)
		}
	}

	// Limit should be respected.
	if len(summaries) > 5 {
		t.Errorf("Closed(5) returned %d results, want <= 5", len(summaries))
	}
}

func TestClosed_NoBD(t *testing.T) {
	c := &Client{Dir: t.TempDir()}

	// If bd is actually on PATH, skip — this test is for missing-bd fallback.
	if err := c.checkBD(); err == nil {
		t.Skip("bd is on PATH; cannot test missing-bd fallback")
	}

	_, err := c.Closed(10)
	if err == nil {
		t.Fatal("expected ErrCLINotFound, got nil")
	}
	if !errors.Is(err, ErrCLINotFound) {
		t.Errorf("error = %v, want ErrCLINotFound", err)
	}
}

func TestListChildren_NoBD(t *testing.T) {
	c := &Client{Dir: t.TempDir()}

	// If bd is actually on PATH, skip — this test is for missing-bd fallback.
	if err := c.checkBD(); err == nil {
		t.Skip("bd is on PATH; cannot test missing-bd fallback")
	}

	_, err := c.ListChildren("some-parent")
	if err == nil {
		t.Fatal("expected ErrCLINotFound, got nil")
	}
	if !errors.Is(err, ErrCLINotFound) {
		t.Errorf("error = %v, want ErrCLINotFound", err)
	}
}

func TestListChildren_BDAvailable(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping bd CLI test in short mode")
	}

	c := &Client{Dir: "."}

	// bd is on PATH; listing children should succeed (even if empty).
	summaries, err := c.ListChildren("nonexistent-parent")
	if err != nil {
		t.Fatalf("ListChildren() unexpected error: %v", err)
	}

	// A nonexistent parent should return an empty list, not an error.
	if len(summaries) != 0 {
		t.Errorf("ListChildren(nonexistent-parent) returned %d results, want 0", len(summaries))
	}
}

func TestCheckBD(t *testing.T) {
	c := &Client{}

	// checkBD should return either nil or ErrCLINotFound
	err := c.checkBD()
	if err != nil && !errors.Is(err, ErrCLINotFound) {
		t.Errorf("checkBD() returned unexpected error: %v", err)
	}
}
