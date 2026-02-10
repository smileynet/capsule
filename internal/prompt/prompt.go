// Package prompt loads and composes phase prompt templates.
package prompt

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// ErrEmpty indicates a prompt file exists but contains no content.
var ErrEmpty = errors.New("prompt: empty prompt file")

// Context holds the values interpolated into prompt templates.
type Context struct {
	BeadID      string
	Title       string
	Description string
	Feedback    string
}

// Loader reads prompt templates from a directory.
type Loader struct {
	promptsDir string
}

// NewLoader creates a Loader that reads prompts from dir.
func NewLoader(dir string) *Loader {
	return &Loader{promptsDir: dir}
}

// Load reads the prompt file for the named phase.
// The file must exist at <promptsDir>/<phaseName>.md and be non-empty.
func (l *Loader) Load(phaseName string) (string, error) {
	if strings.ContainsAny(phaseName, `/\`) {
		return "", fmt.Errorf("prompt: invalid phase name %q", phaseName)
	}

	path := filepath.Join(l.promptsDir, phaseName+".md")
	data, err := os.ReadFile(path)
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
