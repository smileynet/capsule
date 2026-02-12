package state

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/smileynet/capsule/internal/campaign"
)

func TestFileStore_SaveAndLoad(t *testing.T) {
	// Given a state to persist
	dir := t.TempDir()
	store := NewFileStore(filepath.Join(dir, "campaigns"))

	state := campaign.State{
		ID:           "cap-feature",
		ParentBeadID: "cap-feature",
		Tasks: []campaign.TaskResult{
			{BeadID: "cap-1", Status: campaign.TaskCompleted},
			{BeadID: "cap-2", Status: campaign.TaskPending},
		},
		CurrentTaskIdx: 1,
		StartedAt:      time.Now().Truncate(time.Second),
		Status:         campaign.CampaignRunning,
	}

	// When Save is called
	if err := store.Save(state); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Then Load returns the same state
	loaded, found, err := store.Load("cap-feature")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !found {
		t.Fatal("Load() found = false, want true")
	}
	if loaded.ID != state.ID {
		t.Errorf("ID = %q, want %q", loaded.ID, state.ID)
	}
	if loaded.CurrentTaskIdx != 1 {
		t.Errorf("CurrentTaskIdx = %d, want 1", loaded.CurrentTaskIdx)
	}
	if len(loaded.Tasks) != 2 {
		t.Errorf("Tasks len = %d, want 2", len(loaded.Tasks))
	}
	if loaded.Status != campaign.CampaignRunning {
		t.Errorf("Status = %q, want %q", loaded.Status, campaign.CampaignRunning)
	}
}

func TestFileStore_LoadNotFound(t *testing.T) {
	// Given an empty store
	store := NewFileStore(t.TempDir())

	// When Load is called for a nonexistent ID
	_, found, err := store.Load("nonexistent")

	// Then it returns not found
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if found {
		t.Error("Load() found = true, want false")
	}
}

func TestFileStore_Remove(t *testing.T) {
	// Given a saved state
	dir := t.TempDir()
	store := NewFileStore(dir)
	state := campaign.State{ID: "cap-x", ParentBeadID: "cap-x", Status: campaign.CampaignRunning}
	if err := store.Save(state); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// When Remove is called
	if err := store.Remove("cap-x"); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	// Then Load returns not found
	_, found, _ := store.Load("cap-x")
	if found {
		t.Error("Load() found = true after Remove, want false")
	}
}

func TestFileStore_RemoveNotFound(t *testing.T) {
	// Given an empty store
	store := NewFileStore(t.TempDir())

	// When Remove is called for a nonexistent ID
	err := store.Remove("nonexistent")

	// Then no error (idempotent)
	if err != nil {
		t.Errorf("Remove(nonexistent) error = %v, want nil", err)
	}
}

func TestFileStore_PathTraversal(t *testing.T) {
	store := NewFileStore(t.TempDir())

	tests := []struct {
		name string
		id   string
	}{
		{name: "parent traversal", id: "../../etc/passwd"},
		{name: "slash in id", id: "foo/bar"},
		{name: "empty id", id: ""},
		{name: "dot dot", id: ".."},
		{name: "current dir", id: "."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given a malicious or invalid ID

			// When Save is called
			err := store.Save(campaign.State{ParentBeadID: tt.id, Status: campaign.CampaignRunning})

			// Then it returns ErrInvalidID
			if !errors.Is(err, ErrInvalidID) {
				t.Errorf("Save(%q) error = %v, want ErrInvalidID", tt.id, err)
			}

			// When Load is called
			_, _, err = store.Load(tt.id)

			// Then it returns ErrInvalidID
			if !errors.Is(err, ErrInvalidID) {
				t.Errorf("Load(%q) error = %v, want ErrInvalidID", tt.id, err)
			}

			// When Remove is called
			err = store.Remove(tt.id)

			// Then it returns ErrInvalidID
			if !errors.Is(err, ErrInvalidID) {
				t.Errorf("Remove(%q) error = %v, want ErrInvalidID", tt.id, err)
			}
		})
	}
}

func TestFileStore_ValidIDs(t *testing.T) {
	store := NewFileStore(t.TempDir())

	// Given IDs with dots and hyphens (valid bead ID formats)
	validIDs := []string{"cap-123", "cap-123.1", "cap-abc.def"}

	for _, id := range validIDs {
		t.Run(id, func(t *testing.T) {
			// When Save is called with a valid ID
			err := store.Save(campaign.State{ParentBeadID: id, Status: campaign.CampaignRunning})

			// Then no error
			if err != nil {
				t.Errorf("Save(%q) error = %v, want nil", id, err)
			}
		})
	}
}
