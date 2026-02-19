// Package prompt loads and composes phase prompt templates.
package prompt

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"strings"
	"text/template"
)

// ErrEmpty indicates a prompt file exists but contains no content.
var ErrEmpty = errors.New("prompt: empty prompt file")

// SiblingContext holds a summary of a completed sibling task for cross-run context.
type SiblingContext struct {
	BeadID       string
	Title        string
	Summary      string
	FilesChanged []string
}

// Context holds the values interpolated into prompt templates.
type Context struct {
	BeadID         string
	Title          string
	Description    string
	Feedback       string
	SiblingContext []SiblingContext
}

// Loader reads prompt templates from a filesystem.
type Loader struct {
	fsys fs.FS
}

// NewLoader creates a Loader that reads prompts from the given filesystem.
func NewLoader(fsys fs.FS) *Loader {
	return &Loader{fsys: fsys}
}

// Load reads the prompt file for the named phase.
// The file must exist at <phaseName>.md in the filesystem and be non-empty.
func (l *Loader) Load(phaseName string) (string, error) {
	if strings.ContainsAny(phaseName, `/\`) {
		return "", fmt.Errorf("prompt: invalid phase name %q", phaseName)
	}

	data, err := fs.ReadFile(l.fsys, phaseName+".md")
	if err != nil {
		return "", fmt.Errorf("prompt: loading %s: %w", phaseName, err)
	}
	if len(data) == 0 {
		return "", fmt.Errorf("%w: %s", ErrEmpty, phaseName)
	}
	return string(data), nil
}

// Compose loads a prompt template and interpolates ctx into it.
// Templates use Go text/template syntax (e.g. {{.BeadID}}).
// Prompts without template markers are returned unchanged.
func (l *Loader) Compose(phaseName string, ctx Context) (string, error) {
	raw, err := l.Load(phaseName)
	if err != nil {
		return "", err
	}

	tmpl, err := template.New(phaseName).Option("missingkey=error").Parse(raw)
	if err != nil {
		return "", fmt.Errorf("prompt: parsing template %s: %w", phaseName, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", fmt.Errorf("prompt: executing template %s: %w", phaseName, err)
	}

	return buf.String(), nil
}
