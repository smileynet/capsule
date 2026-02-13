// Package state implements campaign state persistence to the filesystem.
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/smileynet/capsule/internal/orchestrator"
)

// Compile-time check: CheckpointFileStore satisfies orchestrator.CheckpointStore.
var _ orchestrator.CheckpointStore = (*CheckpointFileStore)(nil)

// CheckpointFileStore persists pipeline checkpoints as JSON files under a base directory.
type CheckpointFileStore struct {
	baseDir string
}

// NewCheckpointFileStore creates a CheckpointFileStore that saves checkpoints under baseDir.
func NewCheckpointFileStore(baseDir string) *CheckpointFileStore {
	return &CheckpointFileStore{baseDir: baseDir}
}

// SaveCheckpoint writes the pipeline checkpoint to a JSON file named by the bead ID.
func (s *CheckpointFileStore) SaveCheckpoint(cp orchestrator.PipelineCheckpoint) error {
	p, err := s.path(cp.BeadID)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(s.baseDir, 0o755); err != nil {
		return fmt.Errorf("checkpoint: creating directory: %w", err)
	}

	data, err := json.MarshalIndent(cp, "", "  ")
	if err != nil {
		return fmt.Errorf("checkpoint: marshaling: %w", err)
	}

	if err := os.WriteFile(p, data, 0o644); err != nil {
		return fmt.Errorf("checkpoint: writing %s: %w", p, err)
	}
	return nil
}

// LoadCheckpoint reads a pipeline checkpoint for the given bead ID.
// Returns (checkpoint, true, nil) if found, (zero, false, nil) if not found.
func (s *CheckpointFileStore) LoadCheckpoint(beadID string) (orchestrator.PipelineCheckpoint, bool, error) {
	p, err := s.path(beadID)
	if err != nil {
		return orchestrator.PipelineCheckpoint{}, false, err
	}

	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return orchestrator.PipelineCheckpoint{}, false, nil
		}
		return orchestrator.PipelineCheckpoint{}, false, fmt.Errorf("checkpoint: reading %s: %w", p, err)
	}

	var cp orchestrator.PipelineCheckpoint
	if err := json.Unmarshal(data, &cp); err != nil {
		return orchestrator.PipelineCheckpoint{}, false, fmt.Errorf("checkpoint: parsing %s: %w", p, err)
	}
	return cp, true, nil
}

// RemoveCheckpoint deletes the checkpoint file for the given bead ID.
func (s *CheckpointFileStore) RemoveCheckpoint(beadID string) error {
	p, err := s.path(beadID)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("checkpoint: removing %s: %w", p, err)
	}
	return nil
}

// path returns the filesystem path for a checkpoint file.
// It rejects IDs that are empty, dot-segments, or contain path separators.
func (s *CheckpointFileStore) path(id string) (string, error) {
	if id == "" || id == "." || id == ".." || id != filepath.Base(id) {
		return "", fmt.Errorf("%w: %q", ErrInvalidID, id)
	}
	return filepath.Join(s.baseDir, id+".checkpoint.json"), nil
}
