// Package worklog handles worklog template instantiation, phase entry append, and archival.
package worklog

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

// Manager wraps the package-level worklog functions with config for template and archive paths.
type Manager struct {
	templatePath string
	archiveDir   string
}

// NewManager creates a Manager with the given template and archive directory paths.
func NewManager(templatePath, archiveDir string) *Manager {
	return &Manager{templatePath: templatePath, archiveDir: archiveDir}
}

// Create instantiates a worklog from the configured template into worktreePath/worklog.md.
func (m *Manager) Create(worktreePath string, bead BeadContext) error {
	return Create(m.templatePath, worktreePath, bead)
}

// AppendPhaseEntry appends a phase result to the worklog at worktreePath/worklog.md.
func (m *Manager) AppendPhaseEntry(worktreePath string, entry PhaseEntry) error {
	return AppendPhaseEntry(worktreePath, entry)
}

// Archive copies the worklog to the configured archive directory under beadID.
func (m *Manager) Archive(worktreePath, beadID string) error {
	return Archive(worktreePath, m.archiveDir, beadID)
}

// Sentinel errors for caller-checkable conditions.
var (
	ErrAlreadyExists = errors.New("worklog: already exists")
	ErrNotFound      = errors.New("worklog: not found")
	ErrInvalidID     = errors.New("worklog: invalid id")
)

// validateBeadID checks that beadID is safe for use as a path component.
// Rejects empty, path traversal (/ \ . ..), and flag-like IDs (starting with -).
func validateBeadID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: cannot be empty", ErrInvalidID)
	}
	if strings.HasPrefix(id, "-") {
		return fmt.Errorf("%w: %q (must not start with -)", ErrInvalidID, id)
	}
	if strings.ContainsAny(id, `/\`) || id == "." || id == ".." {
		return fmt.Errorf("%w: %q", ErrInvalidID, id)
	}
	return nil
}

// BeadContext holds the bead hierarchy data used to instantiate a worklog template.
type BeadContext struct {
	EpicID             string
	EpicTitle          string
	EpicGoal           string
	FeatureID          string
	FeatureTitle       string
	FeatureGoal        string
	TaskID             string
	TaskTitle          string
	TaskDescription    string
	AcceptanceCriteria string
}

// PhaseEntry records the result of a single pipeline phase.
type PhaseEntry struct {
	Name      string
	Status    string
	Verdict   string
	Timestamp time.Time
}

// templateData holds all fields available to the worklog Go template.
type templateData struct {
	BeadContext
	Timestamp string
}

// Create instantiates a worklog from templatePath into worktreePath/worklog.md,
// executing the Go template with values from bead.
func Create(templatePath, worktreePath string, bead BeadContext) error {
	tmplBytes, err := os.ReadFile(templatePath)
	if err != nil {
		return fmt.Errorf("worklog: reading template: %w", err)
	}

	outPath := filepath.Join(worktreePath, "worklog.md")
	if _, err := os.Stat(outPath); err == nil {
		return fmt.Errorf("%w: %s", ErrAlreadyExists, outPath)
	}

	tmpl, err := template.New("worklog").Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("worklog: parsing template: %w", err)
	}

	data := templateData{
		BeadContext: bead,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("worklog: executing template: %w", err)
	}

	if err := os.WriteFile(outPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("worklog: writing %s: %w", outPath, err)
	}
	return nil
}

// AppendPhaseEntry appends a phase result entry to the worklog at worktreePath/worklog.md.
func AppendPhaseEntry(worktreePath string, entry PhaseEntry) error {
	worklogPath := filepath.Join(worktreePath, "worklog.md")

	existing, err := os.ReadFile(worklogPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrNotFound, worklogPath)
		}
		return fmt.Errorf("worklog: reading %s: %w", worklogPath, err)
	}

	ts := entry.Timestamp.UTC().Format("2006-01-02T15:04:05Z")
	text := fmt.Sprintf("\n### %s\n\n- Status: %s\n- Verdict: %s\n- Timestamp: %s\n",
		entry.Name, entry.Status, entry.Verdict, ts)

	return os.WriteFile(worklogPath, append(existing, []byte(text)...), 0o644)
}

// Archive copies worktreePath/worklog.md to archiveDir/<beadID>/worklog.md.
// The archive subdirectory is created if it does not exist.
func Archive(worktreePath, archiveDir, beadID string) error {
	if err := validateBeadID(beadID); err != nil {
		return err
	}

	src := filepath.Join(worktreePath, "worklog.md")
	data, err := os.ReadFile(src)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrNotFound, src)
		}
		return fmt.Errorf("worklog: reading %s: %w", src, err)
	}

	destDir := filepath.Join(archiveDir, beadID)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return fmt.Errorf("worklog: creating archive dir %s: %w", destDir, err)
	}

	dest := filepath.Join(destDir, "worklog.md")
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return fmt.Errorf("worklog: writing %s: %w", dest, err)
	}
	return nil
}
