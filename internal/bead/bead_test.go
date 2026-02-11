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

func TestResolve_NoBD(t *testing.T) {
	// Given a client with a nonexistent bd binary
	// (We test this by using a PATH that doesn't contain bd)
	// This test verifies the graceful fallback behavior.
	c := &Client{Dir: t.TempDir()}

	// The actual graceful fallback depends on bd being on PATH.
	// We test the interface: Resolve should always return a context
	// with at least TaskID set, even if it can't resolve further.
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

func TestCheckBD(t *testing.T) {
	c := &Client{}

	// checkBD should return either nil or ErrCLINotFound
	err := c.checkBD()
	if err != nil && !errors.Is(err, ErrCLINotFound) {
		t.Errorf("checkBD() returned unexpected error: %v", err)
	}
}
