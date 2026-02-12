// Package gate executes shell commands as pipeline gate phases.
package gate

import (
	"context"
	"os/exec"

	"github.com/smileynet/capsule/internal/provider"
)

// Runner executes shell commands and returns a provider.Signal based on the exit code.
type Runner struct{}

// NewRunner creates a Runner.
func NewRunner() *Runner {
	return &Runner{}
}

// Run executes command in workDir via sh -c. A zero exit code produces StatusPass;
// a non-zero exit code produces StatusError with the combined output as feedback.
func (r *Runner) Run(ctx context.Context, command, workDir string) (provider.Signal, error) {
	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = workDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return provider.Signal{
			Status:       provider.StatusError,
			Feedback:     string(output),
			Summary:      err.Error(),
			FilesChanged: []string{},
			Findings:     []provider.Finding{},
		}, nil
	}
	return provider.Signal{
		Status:       provider.StatusPass,
		Summary:      string(output),
		Feedback:     "gate passed",
		FilesChanged: []string{},
		Findings:     []provider.Finding{},
	}, nil
}
