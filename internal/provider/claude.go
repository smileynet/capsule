package provider

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// defaultTimeout is used when no timeout option is provided.
const defaultTimeout = 5 * time.Minute

// ClaudeProvider executes the claude CLI as a subprocess.
type ClaudeProvider struct {
	timeout    time.Duration
	cmdBuilder func(ctx context.Context, prompt, workDir string) *exec.Cmd
}

// NewClaudeProvider creates a ClaudeProvider with the given options.
func NewClaudeProvider(opts ...ClaudeOption) *ClaudeProvider {
	p := &ClaudeProvider{
		timeout: defaultTimeout,
	}
	for _, opt := range opts {
		opt(p)
	}
	if p.cmdBuilder == nil {
		p.cmdBuilder = defaultCmdBuilder
	}
	return p
}

// ClaudeOption configures a ClaudeProvider.
type ClaudeOption func(*ClaudeProvider)

// WithTimeout sets the execution timeout.
func WithTimeout(d time.Duration) ClaudeOption {
	return func(p *ClaudeProvider) { p.timeout = d }
}

// Name returns "claude".
func (p *ClaudeProvider) Name() string { return "claude" }

// Execute runs the claude CLI with the given prompt in workDir.
// It captures stdout for signal parsing and returns stderr in errors.
func (p *ClaudeProvider) Execute(ctx context.Context, prompt, workDir string) (Result, error) {
	start := time.Now()

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	cmd := p.cmdBuilder(ctx, prompt, workDir)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return Result{}, &TimeoutError{
				Provider: "claude",
				Duration: p.timeout,
			}
		}

		// Context cancelled or non-zero exit.
		return Result{}, &ProviderError{
			Provider: "claude",
			Err:      fmt.Errorf("%w: %s", err, stderr.String()),
		}
	}

	return Result{
		Output:   stdout.String(),
		ExitCode: 0,
		Duration: duration,
	}, nil
}

// defaultCmdBuilder creates the real claude CLI command.
func defaultCmdBuilder(ctx context.Context, prompt, workDir string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "claude",
		"-p", prompt,
		"--dangerously-skip-permissions",
	)
	cmd.Dir = workDir
	cmd.WaitDelay = time.Second
	return cmd
}
