package prompt

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewLoader(t *testing.T) {
	l := NewLoader("/some/dir")
	if l == nil {
		t.Fatal("NewLoader() returned nil")
	}
}

func TestLoad_ReadsPromptFile(t *testing.T) {
	dir := t.TempDir()
	content := "# Test Writer\n\nYou are a test writer."
	if err := os.WriteFile(filepath.Join(dir, "test-writer.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	l := NewLoader(dir)
	got, err := l.Load("test-writer")
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got != content {
		t.Errorf("Load() = %q, want %q", got, content)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	l := NewLoader(t.TempDir())
	_, err := l.Load("nonexistent")
	if err == nil {
		t.Fatal("Load(missing) should return error")
	}
	if !strings.Contains(err.Error(), "prompt") {
		t.Errorf("error should mention 'prompt', got: %v", err)
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "empty.md"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	l := NewLoader(dir)
	_, err := l.Load("empty")
	if err == nil {
		t.Fatal("Load(empty) should return error")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error should mention 'empty', got: %v", err)
	}
}

func TestCompose_InterpolatesContext(t *testing.T) {
	dir := t.TempDir()
	tmpl := `# Phase for {{.BeadID}}
Task: {{.Title}}
Description: {{.Description}}
`
	if err := os.WriteFile(filepath.Join(dir, "test-writer.md"), []byte(tmpl), 0o644); err != nil {
		t.Fatal(err)
	}

	l := NewLoader(dir)
	ctx := Context{
		BeadID:      "cap-123",
		Title:       "Add widget",
		Description: "Implement the widget feature",
	}

	got, err := l.Compose("test-writer", ctx)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	want := "# Phase for cap-123\nTask: Add widget\nDescription: Implement the widget feature\n"
	if got != want {
		t.Errorf("Compose() =\n%s\nwant:\n%s", got, want)
	}
}

func TestCompose_InterpolatesFeedback(t *testing.T) {
	dir := t.TempDir()
	tmpl := `# Phase
{{if .Feedback}}Previous feedback:
{{.Feedback}}{{end}}
`
	if err := os.WriteFile(filepath.Join(dir, "execute.md"), []byte(tmpl), 0o644); err != nil {
		t.Fatal(err)
	}

	l := NewLoader(dir)
	ctx := Context{
		BeadID:   "cap-456",
		Title:    "Fix bug",
		Feedback: "Tests need more coverage",
	}

	got, err := l.Compose("execute", ctx)
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if !strings.Contains(got, "Tests need more coverage") {
		t.Errorf("result should contain Feedback, got: %s", got)
	}
}

func TestCompose_NoTemplateSyntax(t *testing.T) {
	dir := t.TempDir()
	plain := "# Plain prompt with no template markers\n\nJust regular text."
	if err := os.WriteFile(filepath.Join(dir, "plain.md"), []byte(plain), 0o644); err != nil {
		t.Fatal(err)
	}

	l := NewLoader(dir)
	got, err := l.Compose("plain", Context{BeadID: "cap-789"})
	if err != nil {
		t.Fatalf("Compose() error = %v", err)
	}
	if got != plain {
		t.Errorf("Compose(plain) = %q, want %q", got, plain)
	}
}

func TestCompose_MissingPrompt(t *testing.T) {
	l := NewLoader(t.TempDir())
	_, err := l.Compose("missing", Context{BeadID: "cap-000"})
	if err == nil {
		t.Fatal("Compose(missing) should return error")
	}
	if !strings.Contains(err.Error(), "prompt") {
		t.Errorf("error should mention 'prompt', got: %v", err)
	}
}

func TestCompose_InvalidTemplate(t *testing.T) {
	dir := t.TempDir()
	bad := "# Bad template {{.Undefined | badFunc}}"
	if err := os.WriteFile(filepath.Join(dir, "bad.md"), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}

	l := NewLoader(dir)
	_, err := l.Compose("bad", Context{BeadID: "cap-000"})
	if err == nil {
		t.Fatal("Compose(invalid template) should return error")
	}
	if !strings.Contains(err.Error(), "prompt") {
		t.Errorf("error should mention 'prompt', got: %v", err)
	}
}

func TestLoad_EmptyFile_SentinelError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "blank.md"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	l := NewLoader(dir)
	_, err := l.Load("blank")
	if !errors.Is(err, ErrEmpty) {
		t.Errorf("Load(empty) error should wrap ErrEmpty, got: %v", err)
	}
}

func TestLoad_PathTraversal(t *testing.T) {
	l := NewLoader(t.TempDir())
	_, err := l.Load("../../etc/passwd")
	if err == nil {
		t.Fatal("Load(path traversal) should return error")
	}
	if !strings.Contains(err.Error(), "invalid phase name") {
		t.Errorf("error should mention 'invalid phase name', got: %v", err)
	}
}

func TestCompose_MissingKeyError(t *testing.T) {
	dir := t.TempDir()
	tmpl := "# Template with typo: {{.Titl}}"
	if err := os.WriteFile(filepath.Join(dir, "typo.md"), []byte(tmpl), 0o644); err != nil {
		t.Fatal(err)
	}

	l := NewLoader(dir)
	_, err := l.Compose("typo", Context{BeadID: "cap-000", Title: "Real title"})
	if err == nil {
		t.Fatal("Compose(missing key) should return error with missingkey=error")
	}
}
