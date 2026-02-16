package dashboard

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFileArchiveReader_ReadWorklog(t *testing.T) {
	// Given: an archive directory with a worklog file for a bead
	baseDir := t.TempDir()
	beadID := "cap-abc123"
	beadDir := filepath.Join(baseDir, beadID)
	if err := os.MkdirAll(beadDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "# Worklog\n\nPhase 1: passed\n"
	if err := os.WriteFile(filepath.Join(beadDir, "worklog.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := NewFileArchiveReader(baseDir)

	// When: reading the worklog for that bead
	got, err := reader.ReadWorklog(beadID)

	// Then: the file content is returned without error
	if err != nil {
		t.Fatalf("ReadWorklog() error = %v", err)
	}
	if got != content {
		t.Errorf("ReadWorklog() = %q, want %q", got, content)
	}
}

func TestFileArchiveReader_ReadWorklog_NotFound(t *testing.T) {
	// Given: an empty archive directory
	baseDir := t.TempDir()
	reader := NewFileArchiveReader(baseDir)

	// When: reading a worklog for a nonexistent bead
	_, err := reader.ReadWorklog("nonexistent")

	// Then: os.ErrNotExist is returned
	if err == nil {
		t.Fatal("ReadWorklog() expected error for missing file, got nil")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("ReadWorklog() error = %v, want os.ErrNotExist", err)
	}
}

func TestFileArchiveReader_ReadSummary(t *testing.T) {
	// Given: an archive directory with a summary file for a bead
	baseDir := t.TempDir()
	beadID := "cap-xyz789"
	beadDir := filepath.Join(baseDir, beadID)
	if err := os.MkdirAll(beadDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "## Summary\n\nAll phases passed.\n"
	if err := os.WriteFile(filepath.Join(beadDir, "summary.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	reader := NewFileArchiveReader(baseDir)

	// When: reading the summary for that bead
	got, err := reader.ReadSummary(beadID)

	// Then: the file content is returned without error
	if err != nil {
		t.Fatalf("ReadSummary() error = %v", err)
	}
	if got != content {
		t.Errorf("ReadSummary() = %q, want %q", got, content)
	}
}

func TestFileArchiveReader_ReadSummary_NotFound(t *testing.T) {
	// Given: an empty archive directory
	baseDir := t.TempDir()
	reader := NewFileArchiveReader(baseDir)

	// When: reading a summary for a nonexistent bead
	_, err := reader.ReadSummary("nonexistent")

	// Then: os.ErrNotExist is returned
	if err == nil {
		t.Fatal("ReadSummary() expected error for missing file, got nil")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("ReadSummary() error = %v, want os.ErrNotExist", err)
	}
}

func TestFileArchiveReader_ImplementsInterface(t *testing.T) {
	// Given/When/Then: FileArchiveReader satisfies ArchiveReader at compile time
	var _ ArchiveReader = (*FileArchiveReader)(nil)
}

func TestFileArchiveReader_RejectsInvalidBeadID(t *testing.T) {
	reader := NewFileArchiveReader(t.TempDir())

	tests := []struct {
		name   string
		beadID string
	}{
		{"empty", ""},
		{"path traversal slash", "../escape"},
		{"path traversal backslash", `dir\escape`},
		{"dot", "."},
		{"dot-dot", ".."},
		{"flag-like", "-malicious"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := reader.ReadWorklog(tt.beadID)
			if !errors.Is(err, ErrInvalidBeadID) {
				t.Errorf("ReadWorklog(%q) error = %v, want ErrInvalidBeadID", tt.beadID, err)
			}
			_, err = reader.ReadSummary(tt.beadID)
			if !errors.Is(err, ErrInvalidBeadID) {
				t.Errorf("ReadSummary(%q) error = %v, want ErrInvalidBeadID", tt.beadID, err)
			}
		})
	}
}
