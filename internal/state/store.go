// Package state implements campaign state persistence to the filesystem.
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/smileynet/capsule/internal/campaign"
)

// FileStore persists campaign state as JSON files under a base directory.
type FileStore struct {
	baseDir string
}

// NewFileStore creates a FileStore that saves state under baseDir.
func NewFileStore(baseDir string) *FileStore {
	return &FileStore{baseDir: baseDir}
}

// Save writes the campaign state to a JSON file named by the campaign's ParentBeadID.
func (s *FileStore) Save(state campaign.State) error {
	p, err := s.path(state.ParentBeadID)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(s.baseDir, 0o755); err != nil {
		return fmt.Errorf("state: creating directory: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("state: marshaling: %w", err)
	}

	if err := os.WriteFile(p, data, 0o644); err != nil {
		return fmt.Errorf("state: writing %s: %w", p, err)
	}
	return nil
}

// Load reads campaign state for the given parent bead ID.
// Returns (state, true, nil) if found, (zero, false, nil) if not found.
func (s *FileStore) Load(id string) (campaign.State, bool, error) {
	p, err := s.path(id)
	if err != nil {
		return campaign.State{}, false, err
	}

	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return campaign.State{}, false, nil
		}
		return campaign.State{}, false, fmt.Errorf("state: reading %s: %w", p, err)
	}

	var state campaign.State
	if err := json.Unmarshal(data, &state); err != nil {
		return campaign.State{}, false, fmt.Errorf("state: parsing %s: %w", p, err)
	}
	return state, true, nil
}

// Remove deletes the campaign state file for the given ID.
func (s *FileStore) Remove(id string) error {
	p, err := s.path(id)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("state: removing %s: %w", p, err)
	}
	return nil
}

// ErrInvalidID indicates a campaign ID is empty or contains path traversal components.
var ErrInvalidID = errors.New("state: invalid campaign ID")

// path returns the filesystem path for a campaign state file.
// It rejects IDs that are empty, dot-segments, or contain path separators.
func (s *FileStore) path(id string) (string, error) {
	if id == "" || id == "." || id == ".." || id != filepath.Base(id) {
		return "", fmt.Errorf("%w: %q", ErrInvalidID, id)
	}
	return filepath.Join(s.baseDir, id+".json"), nil
}
