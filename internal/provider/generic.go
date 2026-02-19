package provider

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"time"
)

// defaultTimeout is used when no timeout option is provided.
const defaultTimeout = 5 * time.Minute

// CommandConfig parameterizes CLI invocation for any AI CLI tool.
type CommandConfig struct {
	Name            string   // provider name for logs/errors
	Binary          string   // executable name
	Subcommand      string   // optional subcommand (e.g. "chat" for Kiro)
	PromptFlag      string   // how prompt is passed ("-p" for Claude, "" for positional)
	PermissionFlags []string // headless/trust flags
	ExtraFlags      []string // additional flags (e.g. --wrap never)
	StripANSI       bool     // whether to strip ANSI escape codes from output
}

// Verify GenericProvider satisfies Executor at compile time.
var _ Executor = (*GenericProvider)(nil)

// GenericProvider executes any AI CLI tool as a subprocess.
type GenericProvider struct {
	config     CommandConfig
	timeout    time.Duration
	cmdBuilder func(ctx context.Context, prompt, workDir string) *exec.Cmd
}

// Option configures a GenericProvider.
type Option func(*GenericProvider)

// WithTimeout sets the execution timeout.
func WithTimeout(d time.Duration) Option {
	return func(p *GenericProvider) { p.timeout = d }
}

// NewGenericProvider creates a GenericProvider from config and options.
func NewGenericProvider(cfg CommandConfig, opts ...Option) *GenericProvider {
	p := &GenericProvider{
		config:  cfg,
		timeout: defaultTimeout,
	}
	for _, opt := range opts {
		opt(p)
	}
	if p.cmdBuilder == nil {
		p.cmdBuilder = p.defaultCmdBuilder
	}
	return p
}

// Name returns the configured provider name.
func (p *GenericProvider) Name() string { return p.config.Name }

// Execute runs the CLI with the given prompt in workDir.
// It captures stdout for signal parsing and returns stderr in errors.
func (p *GenericProvider) Execute(ctx context.Context, prompt, workDir string) (Result, error) {
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
				Provider: p.config.Name,
				Duration: p.timeout,
			}
		}

		// Context cancelled or non-zero exit.
		return Result{}, &ProviderError{
			Provider: p.config.Name,
			Err:      fmt.Errorf("%w: %s", err, stderr.String()),
		}
	}

	output := stdout.String()
	if p.config.StripANSI {
		output = stripANSI(output)
	}

	return Result{
		Output:   output,
		ExitCode: 0,
		Duration: duration,
	}, nil
}

// defaultCmdBuilder creates the CLI command from config fields.
func (p *GenericProvider) defaultCmdBuilder(ctx context.Context, prompt, workDir string) *exec.Cmd {
	args := buildArgs(p.config, prompt)
	cmd := exec.CommandContext(ctx, p.config.Binary, args...)
	cmd.Dir = workDir
	cmd.WaitDelay = time.Second
	return cmd
}

// buildArgs constructs the argument list from a CommandConfig.
func buildArgs(cfg CommandConfig, prompt string) []string {
	var args []string
	if cfg.Subcommand != "" {
		args = append(args, cfg.Subcommand)
	}
	args = append(args, cfg.PermissionFlags...)
	args = append(args, cfg.ExtraFlags...)
	if cfg.PromptFlag != "" {
		args = append(args, cfg.PromptFlag, prompt)
	} else {
		args = append(args, prompt)
	}
	return args
}

// ansiPattern matches common ANSI escape sequences.
var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// stripANSI removes ANSI escape codes from a string.
func stripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}
