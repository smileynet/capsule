package state

import (
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
			{BeadID: "cap-1", Status: "completed"},
			{BeadID: "cap-2", Status: "pending"},
		},
		CurrentTaskIdx: 1,
		StartedAt:      time.Now().Truncate(time.Second),
		Status:         "running",
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
	if loaded.Status != "running" {
		t.Errorf("Status = %q, want %q", loaded.Status, "running")
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
	state := campaign.State{ID: "cap-x", ParentBeadID: "cap-x", Status: "running"}
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
