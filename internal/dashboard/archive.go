package dashboard

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrInvalidBeadID indicates a bead ID failed path-safety validation.
var ErrInvalidBeadID = errors.New("archive: invalid bead id")

// ArchiveReader reads archived pipeline results for a given bead.
type ArchiveReader interface {
	ReadWorklog(beadID string) (string, error)
	ReadSummary(beadID string) (string, error)
}

// FileArchiveReader reads archived worklog and summary files from a base
// directory with the layout: <baseDir>/<beadID>/worklog.md and summary.md.
type FileArchiveReader struct {
	baseDir string
}

// NewFileArchiveReader creates a FileArchiveReader rooted at baseDir.
func NewFileArchiveReader(baseDir string) *FileArchiveReader {
	return &FileArchiveReader{baseDir: baseDir}
}

// ReadWorklog returns the contents of <baseDir>/<beadID>/worklog.md.
// Returns os.ErrNotExist if the file does not exist.
func (r *FileArchiveReader) ReadWorklog(beadID string) (string, error) {
	return r.readFile(beadID, "worklog.md")
}

// ReadSummary returns the contents of <baseDir>/<beadID>/summary.md.
// Returns os.ErrNotExist if the file does not exist.
func (r *FileArchiveReader) ReadSummary(beadID string) (string, error) {
	return r.readFile(beadID, "summary.md")
}

// validateBeadID checks that beadID is safe for use as a path component.
// Rejects empty, path traversal (/ \ . ..), null bytes, and flag-like IDs (starting with -).
func validateBeadID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: cannot be empty", ErrInvalidBeadID)
	}
	if strings.HasPrefix(id, "-") {
		return fmt.Errorf("%w: %q (must not start with -)", ErrInvalidBeadID, id)
	}
	if strings.ContainsAny(id, "/\\\x00") || id == "." || id == ".." {
		return fmt.Errorf("%w: %q", ErrInvalidBeadID, id)
	}
	return nil
}

func (r *FileArchiveReader) readFile(beadID, filename string) (string, error) {
	if err := validateBeadID(beadID); err != nil {
		return "", err
	}
	data, err := os.ReadFile(filepath.Join(r.baseDir, beadID, filename))
	if err != nil {
		return "", fmt.Errorf("archive: read %s for %s: %w", filename, beadID, err)
	}
	return string(data), nil
}
