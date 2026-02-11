package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"time"

	"github.com/alecthomas/kong"

	"github.com/smileynet/capsule/internal/config"
	"github.com/smileynet/capsule/internal/orchestrator"
	"github.com/smileynet/capsule/internal/prompt"
	"github.com/smileynet/capsule/internal/provider"
	"github.com/smileynet/capsule/internal/worklog"
	"github.com/smileynet/capsule/internal/worktree"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

// CLI is the top-level command structure for capsule.
type CLI struct {
	Version kong.VersionFlag `help:"Show version." short:"V"`
	Run     RunCmd           `cmd:"" help:"Run a capsule pipeline."`
	Abort   AbortCmd         `cmd:"" help:"Abort a running capsule."`
	Clean   CleanCmd         `cmd:"" help:"Clean up capsule worktree and artifacts."`
}

// RunCmd executes a capsule pipeline for a given bead.
type RunCmd struct {
	BeadID   string `arg:"" help:"Bead ID to run."`
	Provider string `help:"Provider to use for completions." default:"claude"`
	Timeout  int    `help:"Timeout in seconds." default:"300"`
	Debug    bool   `help:"Enable debug output."`
}

// pipelineRunner abstracts orchestrator.RunPipeline for testing.
type pipelineRunner interface {
	RunPipeline(ctx context.Context, input orchestrator.PipelineInput) error
}

// loadConfig loads layered config from user and project paths with env overrides.
func loadConfig() (*config.Config, error) {
	cfg, err := config.LoadLayered(
		os.ExpandEnv("$HOME/.config/capsule/config.yaml"),
		".capsule/config.yaml",
	)
	if err != nil {
		return nil, err
	}
	if err := cfg.ApplyEnv(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Run executes the run command.
func (r *RunCmd) Run() error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("run: %w", err)
	}

	// Apply CLI flag overrides.
	cfg.Runtime.Provider = r.Provider
	cfg.Runtime.Timeout = time.Duration(r.Timeout) * time.Second

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("run: %w", err)
	}

	// Create provider via registry.
	reg := provider.NewRegistry()
	reg.Register("claude", func() (provider.Executor, error) {
		return provider.NewClaudeProvider(provider.WithTimeout(cfg.Runtime.Timeout)), nil
	})

	p, err := reg.NewProvider(cfg.Runtime.Provider)
	if err != nil {
		return fmt.Errorf("run: %w", err)
	}

	// Build orchestrator.
	promptLoader := prompt.NewLoader("prompts")
	wtMgr := worktree.NewManager(".", cfg.Worktree.BaseDir)
	wlMgr := worklog.NewManager("templates/worklog.md.template", ".capsule/logs")

	orch := orchestrator.New(p,
		orchestrator.WithPromptLoader(promptLoader),
		orchestrator.WithWorktreeManager(wtMgr),
		orchestrator.WithWorklogManager(wlMgr),
		orchestrator.WithStatusCallback(plainTextCallback(os.Stdout)),
	)

	return r.run(os.Stdout, orch)
}

// run executes the pipeline with the given runner, enabling testable wiring.
func (r *RunCmd) run(_ io.Writer, runner pipelineRunner) error {
	// Set up Ctrl+C handling.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	input := orchestrator.PipelineInput{
		BeadID: r.BeadID,
	}

	return runner.RunPipeline(ctx, input)
}

// AbortCmd aborts a running capsule by removing the worktree.
// The branch is preserved so work can be inspected. Use clean to remove everything.
type AbortCmd struct {
	BeadID string `arg:"" help:"Bead ID to abort."`
}

// worktreeOps abstracts worktree operations for testing abort and clean commands.
type worktreeOps interface {
	Exists(id string) bool
	Remove(id string, deleteBranch bool) error
	Prune() error
}

// Run executes the abort command by removing the worktree.
func (a *AbortCmd) Run() error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("abort: %w", err)
	}

	mgr := worktree.NewManager(".", cfg.Worktree.BaseDir)
	return a.run(os.Stdout, mgr)
}

// run executes the abort with the given worktree manager, enabling testable wiring.
func (a *AbortCmd) run(w io.Writer, mgr worktreeOps) error {
	if !mgr.Exists(a.BeadID) {
		return fmt.Errorf("abort: no worktree found for %q", a.BeadID)
	}

	// Preserve branch for inspection; use clean to remove branch.
	if err := mgr.Remove(a.BeadID, false); err != nil {
		return fmt.Errorf("abort: %w", err)
	}

	_, _ = fmt.Fprintf(w, "Aborted capsule %s (branch preserved)\n", a.BeadID)
	return nil
}

// CleanCmd cleans up capsule worktree and artifacts.
type CleanCmd struct {
	BeadID string `arg:"" help:"Bead ID to clean."`
}

// Run executes the clean command by removing worktree, branch, and pruning.
func (c *CleanCmd) Run() error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("clean: %w", err)
	}

	mgr := worktree.NewManager(".", cfg.Worktree.BaseDir)
	return c.run(os.Stdout, mgr)
}

// run executes the clean with the given worktree manager, enabling testable wiring.
func (c *CleanCmd) run(w io.Writer, mgr worktreeOps) error {
	if !mgr.Exists(c.BeadID) {
		return fmt.Errorf("clean: no worktree found for %q", c.BeadID)
	}

	if err := mgr.Remove(c.BeadID, true); err != nil {
		return fmt.Errorf("clean: %w", err)
	}

	if err := mgr.Prune(); err != nil {
		return fmt.Errorf("clean: prune: %w", err)
	}

	_, _ = fmt.Fprintf(w, "Cleaned capsule %s\n", c.BeadID)
	return nil
}

// Exit codes.
const (
	exitSuccess  = 0
	exitPipeline = 1
	exitSetup    = 2
)

// exitCode maps an error to the appropriate exit code.
func exitCode(err error) int {
	if err == nil {
		return exitSuccess
	}
	var pe *orchestrator.PipelineError
	if errors.As(err, &pe) {
		return exitPipeline
	}
	return exitSetup
}

// plainTextCallback returns a StatusCallback that prints timestamped phase lines.
func plainTextCallback(w io.Writer) orchestrator.StatusCallback {
	return func(su orchestrator.StatusUpdate) {
		ts := time.Now().Format("15:04:05")
		retry := ""
		if su.Attempt > 1 {
			retry = fmt.Sprintf(" (attempt %d/%d)", su.Attempt, su.MaxRetry)
		}
		_, _ = fmt.Fprintf(w, "[%s] [%s] %s %s%s\n", ts, su.Progress, su.Phase, su.Status, retry)
	}
}

func main() {
	var cli CLI
	ctx := kong.Parse(&cli, kong.Vars{"version": version + " " + commit + " " + date})
	err := ctx.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s\n", err)
		os.Exit(exitCode(err))
	}
}
