package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/alecthomas/kong"

	"github.com/smileynet/capsule/internal/bead"
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

// beadResolver abstracts bead context resolution for testing.
type beadResolver interface {
	Resolve(id string) (worklog.BeadContext, error)
	Close(id string) error
}

// mergeOps abstracts worktree merge operations for testing.
type mergeOps interface {
	MergeToMain(id, mainBranch, commitMsg string) error
	DetectMainBranch() (string, error)
	Remove(id string, deleteBranch bool) error
	Prune() error
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
	bdClient := bead.NewClient(".")

	orch := orchestrator.New(p,
		orchestrator.WithPromptLoader(promptLoader),
		orchestrator.WithWorktreeManager(wtMgr),
		orchestrator.WithWorklogManager(wlMgr),
		orchestrator.WithStatusCallback(plainTextCallback(os.Stdout)),
	)

	return r.run(os.Stdout, orch, wtMgr, bdClient)
}

// run executes the pipeline with the given runner, enabling testable wiring.
func (r *RunCmd) run(w io.Writer, runner pipelineRunner, wt mergeOps, bd beadResolver) error {
	// Set up Ctrl+C handling.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// Resolve bead context for worklog.
	beadCtx, _ := bd.Resolve(r.BeadID)

	input := orchestrator.PipelineInput{
		BeadID: r.BeadID,
		Title:  beadCtx.TaskTitle,
		Bead:   beadCtx,
	}

	// Run the pipeline.
	if err := runner.RunPipeline(ctx, input); err != nil {
		return err
	}

	// Post-pipeline lifecycle: merge → cleanup → close bead.
	// Best-effort: pipeline success is the hard requirement.
	r.runPostPipeline(w, wt, bd)
	return nil
}

// runPostPipeline performs merge, cleanup, and bead closing after a successful pipeline.
// Failures print warnings but don't affect the exit code.
func (r *RunCmd) runPostPipeline(w io.Writer, wt mergeOps, bd beadResolver) {
	id := r.BeadID

	// Detect main branch.
	mainBranch, err := wt.DetectMainBranch()
	if err != nil {
		_, _ = fmt.Fprintf(w, "warning: cannot detect main branch: %v\n", err)
		return
	}

	// Merge to main.
	commitMsg := fmt.Sprintf("%s: pipeline complete", id)
	err = wt.MergeToMain(id, mainBranch, commitMsg)
	if err != nil {
		if errors.Is(err, worktree.ErrMergeConflict) {
			_, _ = fmt.Fprintf(w, "warning: merge conflict merging capsule-%s into %s\n", id, mainBranch)
			_, _ = fmt.Fprintf(w, "  To fix:\n")
			_, _ = fmt.Fprintf(w, "    git checkout %s\n", mainBranch)
			_, _ = fmt.Fprintf(w, "    git merge --no-ff capsule-%s\n", id)
			_, _ = fmt.Fprintf(w, "    # resolve conflicts, then:\n")
			_, _ = fmt.Fprintf(w, "    capsule clean %s\n", id)
			return
		}
		_, _ = fmt.Fprintf(w, "warning: merge failed: %v\n", err)
		return
	}
	_, _ = fmt.Fprintf(w, "Merged capsule-%s → %s\n", id, mainBranch)

	// Cleanup: remove worktree and branch.
	if err := wt.Remove(id, true); err != nil {
		_, _ = fmt.Fprintf(w, "warning: cleanup failed: %v\n", err)
	}
	if err := wt.Prune(); err != nil {
		_, _ = fmt.Fprintf(w, "warning: prune failed: %v\n", err)
	}

	// Close bead.
	if err := bd.Close(id); err != nil {
		_, _ = fmt.Fprintf(w, "warning: bead close failed: %v\n", err)
	} else {
		_, _ = fmt.Fprintf(w, "Closed %s\n", id)
	}

	_, _ = fmt.Fprintf(w, "Worklog: .capsule/logs/%s/worklog.md\n", id)
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

// plainTextCallback returns a StatusCallback that prints timestamped phase lines
// with enriched signal data on phase completion.
func plainTextCallback(w io.Writer) orchestrator.StatusCallback {
	return func(su orchestrator.StatusUpdate) {
		ts := time.Now().Format("15:04:05")
		retry := ""
		if su.Attempt > 1 {
			retry = fmt.Sprintf(" (attempt %d/%d)", su.Attempt, su.MaxRetry)
		}
		_, _ = fmt.Fprintf(w, "[%s] [%s] %s %s%s\n", ts, su.Progress, su.Phase, su.Status, retry)

		// Phase completion report.
		if su.Signal != nil && su.Status != orchestrator.PhaseRunning {
			if len(su.Signal.FilesChanged) > 0 {
				_, _ = fmt.Fprintf(w, "         files: %s\n", strings.Join(su.Signal.FilesChanged, ", "))
			}
			if su.Signal.Summary != "" {
				_, _ = fmt.Fprintf(w, "         summary: %s\n", su.Signal.Summary)
			}
			if su.Signal.Feedback != "" && su.Status == orchestrator.PhaseFailed {
				_, _ = fmt.Fprintf(w, "         feedback: %s\n", su.Signal.Feedback)
			}
		}
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
