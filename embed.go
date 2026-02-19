// Package capsule provides embedded runtime resources (prompts, templates)
// and an overlay filesystem that checks local disk first, falling back to embedded.
package capsule

import (
	"embed"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed prompts/*.md
var rawPrompts embed.FS

//go:embed templates/worklog.md.template
var rawTemplates embed.FS

// Prompts is the embedded prompts filesystem with the "prompts/" prefix stripped.
var Prompts = mustSub(rawPrompts, "prompts")

// Templates is the embedded templates filesystem with the "templates/" prefix stripped.
var Templates = mustSub(rawTemplates, "templates")

func mustSub(fsys fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(fsys, dir)
	if err != nil {
		panic(err)
	}
	return sub
}

// OverlayFS returns a filesystem that checks localDir on disk first,
// falling back to the embedded filesystem for files not found locally.
// The returned FS validates names per the fs.ValidPath contract.
func OverlayFS(localDir string, embedded fs.FS) fs.FS {
	return overlayFS{localDir: localDir, embedded: embedded}
}

type overlayFS struct {
	localDir string
	embedded fs.FS
}

func (o overlayFS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	f, err := os.Open(filepath.Join(o.localDir, filepath.FromSlash(name)))
	if err == nil {
		return f, nil
	}
	return o.embedded.Open(name)
}
