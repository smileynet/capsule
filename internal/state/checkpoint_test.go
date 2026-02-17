package state

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/smileynet/capsule/internal/orchestrator"
	"github.com/smileynet/capsule/internal/provider"
)

func TestCheckpointFileStore_SaveAndLoad(t *testing.T) {
	// Given a checkpoint to persist
	dir := t.TempDir()
	store := NewCheckpointFileStore(filepath.Join(dir, "checkpoints"))

	cp := orchestrator.PipelineCheckpoint{
		BeadID: "cap-42",
		PhaseResults: []orchestrator.PhaseResult{
			{
				PhaseName: "test-writer",
				Signal:    provider.Signal{Status: provider.StatusPass, Summary: "passed"},
				Attempt:   1,
				Duration:  2 * time.Second,
				Timestamp: time.Now().Truncate(time.Second),
			},
			{
				PhaseName: "test-review",
				Signal:    provider.Signal{Status: provider.StatusPass, Summary: "approved"},
				Attempt:   1,
				Duration:  3 * time.Second,
				Timestamp: time.Now().Truncate(time.Second),
			},
		},
		SavedAt: time.Now().Truncate(time.Second),
	}

	// When Save is called
	if err := store.SaveCheckpoint(cp); err != nil {
		t.Fatalf("SaveCheckpoint() error = %v", err)
	}

	// Then Load returns the same checkpoint
	loaded, found, err := store.LoadCheckpoint("cap-42")
	if err != nil {
		t.Fatalf("LoadCheckpoint() error = %v", err)
	}
	if !found {
		t.Fatal("LoadCheckpoint() found = false, want true")
	}
	if loaded.BeadID != cp.BeadID {
		t.Errorf("BeadID = %q, want %q", loaded.BeadID, cp.BeadID)
	}
	if len(loaded.PhaseResults) != 2 {
		t.Fatalf("PhaseResults len = %d, want 2", len(loaded.PhaseResults))
	}
	if loaded.PhaseResults[0].PhaseName != "test-writer" {
		t.Errorf("PhaseResults[0].PhaseName = %q, want %q", loaded.PhaseResults[0].PhaseName, "test-writer")
	}
	if loaded.PhaseResults[1].Signal.Status != provider.StatusPass {
		t.Errorf("PhaseResults[1].Signal.Status = %q, want %q", loaded.PhaseResults[1].Signal.Status, provider.StatusPass)
	}
	if !loaded.SavedAt.Equal(cp.SavedAt) {
		t.Errorf("SavedAt = %v, want %v", loaded.SavedAt, cp.SavedAt)
	}
}

func TestCheckpointFileStore_LoadNotFound(t *testing.T) {
	// Given an empty store
	store := NewCheckpointFileStore(t.TempDir())

	// When Load is called for a nonexistent ID
	_, found, err := store.LoadCheckpoint("nonexistent")

	// Then it returns not found
	if err != nil {
		t.Fatalf("LoadCheckpoint() error = %v", err)
	}
	if found {
		t.Error("LoadCheckpoint() found = true, want false")
	}
}

func TestCheckpointFileStore_Remove(t *testing.T) {
	// Given a saved checkpoint
	dir := t.TempDir()
	store := NewCheckpointFileStore(dir)
	cp := orchestrator.PipelineCheckpoint{BeadID: "cap-x", SavedAt: time.Now()}
	if err := store.SaveCheckpoint(cp); err != nil {
		t.Fatalf("SaveCheckpoint() error = %v", err)
	}

	// When Remove is called
	if err := store.RemoveCheckpoint("cap-x"); err != nil {
		t.Fatalf("RemoveCheckpoint() error = %v", err)
	}

	// Then Load returns not found
	_, found, _ := store.LoadCheckpoint("cap-x")
	if found {
		t.Error("LoadCheckpoint() found = true after Remove, want false")
	}
}

func TestCheckpointFileStore_RemoveNotFound(t *testing.T) {
	// Given an empty store
	store := NewCheckpointFileStore(t.TempDir())

	// When Remove is called for a nonexistent ID
	err := store.RemoveCheckpoint("nonexistent")

	// Then no error (idempotent)
	if err != nil {
		t.Errorf("RemoveCheckpoint(nonexistent) error = %v, want nil", err)
	}
}

func TestCheckpointFileStore_PathTraversal(t *testing.T) {
	store := NewCheckpointFileStore(t.TempDir())

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
			// When SaveCheckpoint is called with an invalid ID
			err := store.SaveCheckpoint(orchestrator.PipelineCheckpoint{BeadID: tt.id})
			if !errors.Is(err, ErrInvalidID) {
				t.Errorf("SaveCheckpoint(%q) error = %v, want ErrInvalidID", tt.id, err)
			}

			// When LoadCheckpoint is called
			_, _, err = store.LoadCheckpoint(tt.id)
			if !errors.Is(err, ErrInvalidID) {
				t.Errorf("LoadCheckpoint(%q) error = %v, want ErrInvalidID", tt.id, err)
			}

			// When RemoveCheckpoint is called
			err = store.RemoveCheckpoint(tt.id)
			if !errors.Is(err, ErrInvalidID) {
				t.Errorf("RemoveCheckpoint(%q) error = %v, want ErrInvalidID", tt.id, err)
			}
		})
	}
}

func TestCheckpointFileStore_OverwriteOnSave(t *testing.T) {
	// Given a saved checkpoint with 1 phase
	store := NewCheckpointFileStore(t.TempDir())
	cp1 := orchestrator.PipelineCheckpoint{
		BeadID: "cap-42",
		PhaseResults: []orchestrator.PhaseResult{
			{PhaseName: "phase-a", Signal: provider.Signal{Status: provider.StatusPass}},
		},
		SavedAt: time.Now(),
	}
	if err := store.SaveCheckpoint(cp1); err != nil {
		t.Fatalf("first SaveCheckpoint() error = %v", err)
	}

	// When Save is called again with 2 phases (simulating checkpoint update)
	cp2 := orchestrator.PipelineCheckpoint{
		BeadID: "cap-42",
		PhaseResults: []orchestrator.PhaseResult{
			{PhaseName: "phase-a", Signal: provider.Signal{Status: provider.StatusPass}},
			{PhaseName: "phase-b", Signal: provider.Signal{Status: provider.StatusPass}},
		},
		SavedAt: time.Now(),
	}
	if err := store.SaveCheckpoint(cp2); err != nil {
		t.Fatalf("second SaveCheckpoint() error = %v", err)
	}

	// Then Load returns the latest checkpoint with 2 phases
	loaded, found, err := store.LoadCheckpoint("cap-42")
	if err != nil {
		t.Fatalf("LoadCheckpoint() error = %v", err)
	}
	if !found {
		t.Fatal("LoadCheckpoint() found = false, want true")
	}
	if got := len(loaded.PhaseResults); got != 2 {
		t.Errorf("PhaseResults len = %d, want 2", got)
	}
}
