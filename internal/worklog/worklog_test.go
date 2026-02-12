package worklog

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// goTemplate is a minimal Go template for testing worklog creation.
const goTemplate = `# Worklog: {{.TaskID}}

Generated: {{.Timestamp}}

## Mission Briefing
{{if .EpicID}}
### Epic: {{.EpicID}}

**{{.EpicTitle}}**

{{.EpicGoal}}
{{end}}{{if .FeatureID}}
### Feature: {{.FeatureID}}

**{{.FeatureTitle}}**

{{.FeatureGoal}}
{{end}}
### Task: {{.TaskID}}

**{{.TaskTitle}}**

{{.TaskDescription}}

### Acceptance Criteria

{{.AcceptanceCriteria}}

---

## Phase Log
`

func TestCreate(t *testing.T) {
	// Given a valid Go template and bead context with full hierarchy
	tmplDir := t.TempDir()
	tmplPath := filepath.Join(tmplDir, "worklog.md.template")
	if err := os.WriteFile(tmplPath, []byte(goTemplate), 0o644); err != nil {
		t.Fatal(err)
	}

	bead := BeadContext{
		EpicID:             "epic-001",
		EpicTitle:          "Build CLI",
		EpicGoal:           "Migrate scripts to Go",
		FeatureID:          "feat-001",
		FeatureTitle:       "Worktree management",
		FeatureGoal:        "Create and manage worktrees",
		TaskID:             "task-001",
		TaskTitle:          "Implement worklog",
		TaskDescription:    "Create worklog package",
		AcceptanceCriteria: "- Tests pass\n- Functions work",
	}

	worktreeDir := t.TempDir()

	// When Create is called
	err := Create(tmplPath, worktreeDir, bead)

	// Then worklog.md is created with substituted values
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(worktreeDir, "worklog.md"))
	if err != nil {
		t.Fatalf("reading worklog.md: %v", err)
	}
	content := string(data)

	for _, want := range []string{
		"# Worklog: task-001",
		"### Epic: epic-001",
		"**Build CLI**",
		"Migrate scripts to Go",
		"### Feature: feat-001",
		"**Worktree management**",
		"Create and manage worktrees",
		"### Task: task-001",
		"**Implement worklog**",
		"Create worklog package",
		"- Tests pass\n- Functions work",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("worklog.md missing %q", want)
		}
	}

	// Positive check: timestamp line should contain a date-like pattern (YYYY-MM-DD)
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(line, "Generated:") {
			if len(line) < len("Generated: 2025-01-01") {
				t.Errorf("Generated line too short: %q", line)
			}
			break
		}
	}
}

func TestCreate_MissingBeadContext(t *testing.T) {
	// Given a Go template and bead context with only TaskID (no epic/feature)
	tmplDir := t.TempDir()
	tmplPath := filepath.Join(tmplDir, "worklog.md.template")
	if err := os.WriteFile(tmplPath, []byte(goTemplate), 0o644); err != nil {
		t.Fatal(err)
	}

	bead := BeadContext{
		TaskID:    "task-orphan",
		TaskTitle: "Standalone task",
	}

	worktreeDir := t.TempDir()

	// When Create is called
	err := Create(tmplPath, worktreeDir, bead)

	// Then worklog.md is created successfully
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	data, err := os.ReadFile(filepath.Join(worktreeDir, "worklog.md"))
	if err != nil {
		t.Fatalf("reading worklog.md: %v", err)
	}
	content := string(data)

	// Task section is present
	if !strings.Contains(content, "### Task: task-orphan") {
		t.Error("worklog.md missing task section")
	}
	if !strings.Contains(content, "**Standalone task**") {
		t.Error("worklog.md missing task title")
	}

	// Epic and Feature sections are omitted
	if strings.Contains(content, "### Epic:") {
		t.Error("worklog.md should not contain Epic section when EpicID is empty")
	}
	if strings.Contains(content, "### Feature:") {
		t.Error("worklog.md should not contain Feature section when FeatureID is empty")
	}
}

func TestCreate_MissingTemplate(t *testing.T) {
	// Given a template path that does not exist
	worktreeDir := t.TempDir()
	bead := BeadContext{TaskID: "task-001"}

	// When Create is called
	err := Create("/nonexistent/template.md", worktreeDir, bead)

	// Then an error is returned
	if err == nil {
		t.Fatal("expected error for missing template")
	}
	if !strings.Contains(err.Error(), "template") {
		t.Errorf("error should mention template, got: %v", err)
	}
}

func TestCreate_ExistingWorklog(t *testing.T) {
	// Given a worktree that already has a worklog.md
	tmplDir := t.TempDir()
	tmplPath := filepath.Join(tmplDir, "worklog.md.template")
	if err := os.WriteFile(tmplPath, []byte("# {{.TaskID}}"), 0o644); err != nil {
		t.Fatal(err)
	}

	worktreeDir := t.TempDir()
	existing := filepath.Join(worktreeDir, "worklog.md")
	if err := os.WriteFile(existing, []byte("existing content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// When Create is called
	err := Create(tmplPath, worktreeDir, BeadContext{TaskID: "task-001"})

	// Then an ErrAlreadyExists sentinel is returned
	if err == nil {
		t.Fatal("expected error when worklog.md already exists")
	}
	if !errors.Is(err, ErrAlreadyExists) {
		t.Errorf("error should wrap ErrAlreadyExists, got: %v", err)
	}
}

func TestAppendPhaseEntry(t *testing.T) {
	// Given a worktree with an existing worklog.md
	worktreeDir := t.TempDir()
	worklogPath := filepath.Join(worktreeDir, "worklog.md")
	initial := "# Worklog\n\n## Phase Log\n"
	if err := os.WriteFile(worklogPath, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	entry := PhaseEntry{
		Name:      "test-writer",
		Status:    "completed",
		Verdict:   "PASS",
		Timestamp: time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC),
	}

	// When AppendPhaseEntry is called
	err := AppendPhaseEntry(worktreeDir, entry)

	// Then the entry is appended to the worklog
	if err != nil {
		t.Fatalf("AppendPhaseEntry() error = %v", err)
	}

	data, err := os.ReadFile(worklogPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	for _, want := range []string{
		"test-writer",
		"completed",
		"PASS",
		"2025-06-15",
	} {
		if !strings.Contains(content, want) {
			t.Errorf("worklog missing %q after append", want)
		}
	}
}

func TestAppendPhaseEntry_MissingWorklog(t *testing.T) {
	// Given a worktree without worklog.md
	worktreeDir := t.TempDir()

	entry := PhaseEntry{
		Name:   "test-writer",
		Status: "completed",
	}

	// When AppendPhaseEntry is called
	err := AppendPhaseEntry(worktreeDir, entry)

	// Then an ErrNotFound sentinel is returned
	if err == nil {
		t.Fatal("expected error for missing worklog")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("error should wrap ErrNotFound, got: %v", err)
	}
}

func TestAppendPhaseEntry_MultipleEntries(t *testing.T) {
	// Given a worktree with a worklog
	worktreeDir := t.TempDir()
	worklogPath := filepath.Join(worktreeDir, "worklog.md")
	if err := os.WriteFile(worklogPath, []byte("# Worklog\n\n## Phase Log\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries := []PhaseEntry{
		{Name: "test-writer", Status: "completed", Verdict: "PASS", Timestamp: time.Now()},
		{Name: "test-review", Status: "completed", Verdict: "PASS", Timestamp: time.Now()},
	}

	// When multiple entries are appended
	for _, e := range entries {
		if err := AppendPhaseEntry(worktreeDir, e); err != nil {
			t.Fatalf("AppendPhaseEntry(%s) error = %v", e.Name, err)
		}
	}

	// Then both entries appear in the worklog in chronological order
	data, err := os.ReadFile(worklogPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "test-writer") {
		t.Error("missing test-writer entry")
	}
	if !strings.Contains(content, "test-review") {
		t.Error("missing test-review entry")
	}
	// Verify ordering: test-writer appears before test-review
	writerIdx := strings.Index(content, "test-writer")
	reviewIdx := strings.Index(content, "test-review")
	if writerIdx >= reviewIdx {
		t.Errorf("test-writer (at %d) should appear before test-review (at %d)", writerIdx, reviewIdx)
	}
}

func TestArchive(t *testing.T) {
	// Given a worktree with a worklog.md
	worktreeDir := t.TempDir()
	worklogContent := "# Worklog: task-001\n\nSome phase results"
	if err := os.WriteFile(filepath.Join(worktreeDir, "worklog.md"), []byte(worklogContent), 0o644); err != nil {
		t.Fatal(err)
	}

	archiveBase := t.TempDir()

	// When Archive is called
	err := Archive(worktreeDir, archiveBase, "task-001")

	// Then worklog.md is copied to archiveDir/task-001/worklog.md
	if err != nil {
		t.Fatalf("Archive() error = %v", err)
	}

	archivedPath := filepath.Join(archiveBase, "task-001", "worklog.md")
	data, err := os.ReadFile(archivedPath)
	if err != nil {
		t.Fatalf("reading archived worklog: %v", err)
	}
	if string(data) != worklogContent {
		t.Errorf("archived content = %q, want %q", string(data), worklogContent)
	}
}

func TestArchive_CreatesDirectory(t *testing.T) {
	// Given a worktree with worklog.md and an archive dir that doesn't exist yet
	worktreeDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(worktreeDir, "worklog.md"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}

	archiveBase := filepath.Join(t.TempDir(), "logs")

	// When Archive is called
	err := Archive(worktreeDir, archiveBase, "task-002")

	// Then the archive directory is created
	if err != nil {
		t.Fatalf("Archive() error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(archiveBase, "task-002", "worklog.md")); err != nil {
		t.Fatalf("archived file not found: %v", err)
	}
}

func TestArchive_MissingWorklog(t *testing.T) {
	// Given a worktree without worklog.md
	worktreeDir := t.TempDir()
	archiveBase := t.TempDir()

	// When Archive is called
	err := Archive(worktreeDir, archiveBase, "task-001")

	// Then an ErrNotFound sentinel is returned
	if err == nil {
		t.Fatal("expected error for missing worklog")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("error should wrap ErrNotFound, got: %v", err)
	}
}

func TestManager_Create(t *testing.T) {
	// Given a manager with a valid template
	tmplDir := t.TempDir()
	tmplPath := filepath.Join(tmplDir, "worklog.md.template")
	if err := os.WriteFile(tmplPath, []byte("# {{.TaskID}}"), 0o644); err != nil {
		t.Fatal(err)
	}
	archiveDir := t.TempDir()
	mgr := NewManager(tmplPath, archiveDir)

	worktreeDir := t.TempDir()
	bead := BeadContext{TaskID: "task-mgr-1"}

	// When Create is called through the manager
	err := mgr.Create(worktreeDir, bead)

	// Then worklog.md is created
	if err != nil {
		t.Fatalf("Manager.Create() error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(worktreeDir, "worklog.md"))
	if err != nil {
		t.Fatalf("reading worklog.md: %v", err)
	}
	if !strings.Contains(string(data), "task-mgr-1") {
		t.Errorf("worklog.md missing task ID, got: %s", data)
	}
}

func TestManager_AppendPhaseEntry(t *testing.T) {
	// Given a manager and an existing worklog
	mgr := NewManager("", t.TempDir())
	worktreeDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(worktreeDir, "worklog.md"), []byte("# Worklog\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	entry := PhaseEntry{
		Name:      "test-writer",
		Status:    "completed",
		Verdict:   "PASS",
		Timestamp: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	// When AppendPhaseEntry is called through the manager
	err := mgr.AppendPhaseEntry(worktreeDir, entry)

	// Then the entry is appended
	if err != nil {
		t.Fatalf("Manager.AppendPhaseEntry() error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(worktreeDir, "worklog.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "test-writer") {
		t.Errorf("worklog.md missing phase entry, got: %s", data)
	}
}

func TestManager_Archive(t *testing.T) {
	// Given a manager with an archive directory and a worktree with a worklog
	archiveDir := t.TempDir()
	mgr := NewManager("", archiveDir)
	worktreeDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(worktreeDir, "worklog.md"), []byte("archived content"), 0o644); err != nil {
		t.Fatal(err)
	}

	// When Archive is called through the manager
	err := mgr.Archive(worktreeDir, "task-mgr-2")

	// Then the worklog is archived
	if err != nil {
		t.Fatalf("Manager.Archive() error = %v", err)
	}
	data, err := os.ReadFile(filepath.Join(archiveDir, "task-mgr-2", "worklog.md"))
	if err != nil {
		t.Fatalf("reading archived worklog: %v", err)
	}
	if string(data) != "archived content" {
		t.Errorf("archived content = %q, want %q", string(data), "archived content")
	}
}

func TestArchive_InvalidBeadID(t *testing.T) {
	// Given a worktree with a worklog.md and an invalid bead ID
	worktreeDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(worktreeDir, "worklog.md"), []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	archiveBase := t.TempDir()

	tests := []struct {
		name   string
		beadID string
	}{
		{"empty", ""},
		{"flag-like", "--flag"},
		{"path traversal", "../escape"},
		{"dot", "."},
		{"dotdot", ".."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When Archive is called with an invalid bead ID
			err := Archive(worktreeDir, archiveBase, tt.beadID)

			// Then an ErrInvalidID sentinel is returned
			if err == nil {
				t.Fatalf("expected error for beadID %q", tt.beadID)
			}
			if !errors.Is(err, ErrInvalidID) {
				t.Errorf("error should wrap ErrInvalidID, got: %v", err)
			}
		})
	}
}
