package capsule

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestEmbeddedPrompts(t *testing.T) {
	// Verify that embedded prompts FS contains expected files.
	data, err := fs.ReadFile(Prompts, "execute.md")
	if err != nil {
		t.Fatalf("reading embedded execute.md: %v", err)
	}
	if len(data) == 0 {
		t.Error("embedded execute.md is empty")
	}
}

func TestEmbeddedTemplates(t *testing.T) {
	// Verify that embedded templates FS contains the worklog template.
	data, err := fs.ReadFile(Templates, "worklog.md.template")
	if err != nil {
		t.Fatalf("reading embedded worklog.md.template: %v", err)
	}
	if len(data) == 0 {
		t.Error("embedded worklog.md.template is empty")
	}
}

func TestOverlayFS_EmbeddedOnly(t *testing.T) {
	// Given: an embedded FS with a file and a local dir without it
	embedded := fstest.MapFS{
		"hello.txt": &fstest.MapFile{Data: []byte("from embedded")},
	}
	localDir := t.TempDir() // empty

	// When: opening the file via overlay
	ofs := OverlayFS(localDir, embedded)
	data, err := fs.ReadFile(ofs, "hello.txt")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	// Then: embedded content is returned
	if string(data) != "from embedded" {
		t.Errorf("got %q, want %q", string(data), "from embedded")
	}
}

func TestOverlayFS_LocalOverride(t *testing.T) {
	// Given: both local and embedded have the same file
	embedded := fstest.MapFS{
		"hello.txt": &fstest.MapFile{Data: []byte("from embedded")},
	}
	localDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(localDir, "hello.txt"), []byte("from local"), 0o644); err != nil {
		t.Fatal(err)
	}

	// When: opening the file via overlay
	ofs := OverlayFS(localDir, embedded)
	data, err := fs.ReadFile(ofs, "hello.txt")
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	// Then: local file takes precedence
	if string(data) != "from local" {
		t.Errorf("got %q, want %q", string(data), "from local")
	}
}

func TestOverlayFS_Mixed(t *testing.T) {
	// Given: local has one file, embedded has another
	embedded := fstest.MapFS{
		"a.txt": &fstest.MapFile{Data: []byte("embedded-a")},
		"b.txt": &fstest.MapFile{Data: []byte("embedded-b")},
	}
	localDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(localDir, "a.txt"), []byte("local-a"), 0o644); err != nil {
		t.Fatal(err)
	}

	ofs := OverlayFS(localDir, embedded)

	// When/Then: a.txt comes from local, b.txt comes from embedded
	dataA, err := fs.ReadFile(ofs, "a.txt")
	if err != nil {
		t.Fatalf("ReadFile(a.txt) error = %v", err)
	}
	if string(dataA) != "local-a" {
		t.Errorf("a.txt = %q, want %q", string(dataA), "local-a")
	}

	dataB, err := fs.ReadFile(ofs, "b.txt")
	if err != nil {
		t.Fatalf("ReadFile(b.txt) error = %v", err)
	}
	if string(dataB) != "embedded-b" {
		t.Errorf("b.txt = %q, want %q", string(dataB), "embedded-b")
	}
}

func TestOverlayFS_NotFound(t *testing.T) {
	// Given: neither local nor embedded has the file
	embedded := fstest.MapFS{}
	localDir := t.TempDir()

	ofs := OverlayFS(localDir, embedded)

	// When/Then: Open returns an error
	_, err := fs.ReadFile(ofs, "missing.txt")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestOverlayFS_RejectsInvalidPath(t *testing.T) {
	// Given: an overlay FS
	ofs := OverlayFS(t.TempDir(), fstest.MapFS{})

	// When/Then: invalid paths are rejected per fs.ValidPath contract
	for _, name := range []string{"../escape", "/absolute", "bad\\slash"} {
		_, err := ofs.Open(name)
		if err == nil {
			t.Errorf("Open(%q) should return error", name)
		}
	}
}
