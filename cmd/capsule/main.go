package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"

	"github.com/smileynet/capsule/internal/bead"
	"github.com/smileynet/capsule/internal/campaign"
	"github.com/smileynet/capsule/internal/config"
	"github.com/smileynet/capsule/internal/dashboard"
	"github.com/smileynet/capsule/internal/gate"
	"github.com/smileynet/capsule/internal/orchestrator"
	"github.com/smileynet/capsule/internal/prompt"
	"github.com/smileynet/capsule/internal/provider"
	"github.com/smileynet/capsule/internal/state"
	"github.com/smileynet/capsule/internal/tui"
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
	Version   kong.VersionFlag `help:"Show version." short:"V"`
	Run       RunCmd           `cmd:"" help:"Run a capsule pipeline."`
	Campaign  CampaignCmd      `cmd:"" help:"Run a campaign for a feature or epic."`
	Dashboard DashboardCmd     `cmd:"" help:"Open interactive dashboard TUI."`
	Abort     AbortCmd         `cmd:"" help:"Abort a running capsule."`
	Clean     CleanCmd         `cmd:"" help:"Clean up capsule worktree and artifacts."`
}

// RunCmd executes a capsule pipeline for a given bead.
type RunCmd struct {
	BeadID   string `arg:"" help:"Bead ID to run."`
	Provider string `help:"Provider to use for completions." default:"claude"`
	Timeout  int    `help:"Timeout in seconds." default:"300"`
	NoTUI    bool   `help:"Force plain text output even if stdout is a TTY." default:"false"`
}

// CampaignCmd runs a campaign for a feature or epic bead.
type CampaignCmd struct {
	ParentID string `arg:"" help:"Feature or epic bead ID."`
	Provider string `help:"Provider to use for completions." default:"claude"`
	Timeout  int    `help:"Timeout in seconds." default:"300"`
}

// Run executes the campaign command.
func (c *CampaignCmd) Run() error {
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("campaign: %w", err)
	}

	cfg.Runtime.Provider = c.Provider
	cfg.Runtime.Timeout = time.Duration(c.Timeout) * time.Second

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("campaign: %w", err)
	}

	// Create provider.
	reg := provider.NewRegistry()
	reg.Register("claude", func() (provider.Executor, error) {
		return provider.NewClaudeProvider(provider.WithTimeout(cfg.Runtime.Timeout)), nil
	})
	p, err := reg.NewProvider(cfg.Runtime.Provider)
	if err != nil {
		return fmt.Errorf("campaign: %w", err)
	}

	// Resolve pipeline phases.
	phases, err := orchestrator.LoadPhases(cfg.Pipeline.Phases)
	if err != nil {
		return fmt.Errorf("campaign: loading phases: %w", err)
	}

	// Build orchestrator.
	promptLoader := prompt.NewLoader("prompts")
	wtMgr := worktree.NewManager(".", cfg.Worktree.BaseDir)
	wlMgr := worklog.NewManager("templates/worklog.md.template", ".capsule/logs")
	gateRunner := gate.NewRunner()

	orch := orchestrator.New(p,
		orchestrator.WithPromptLoader(promptLoader),
		orchestrator.WithWorktreeManager(wtMgr),
		orchestrator.WithWorklogManager(wlMgr),
		orchestrator.WithGateRunner(gateRunner),
		orchestrator.WithPhases(phases),
		orchestrator.WithStatusCallback(plainTextCallback(os.Stdout)),
	)

	// Build campaign dependencies.
	bdClient := newCampaignBeadClient(".")
	stateStore := state.NewFileStore(".capsule/campaigns")
	cb := &campaignPlainTextCallback{w: os.Stdout}

	campaignCfg := campaign.Config{
		FailureMode:      cfg.Campaign.FailureMode,
		CircuitBreaker:   cfg.Campaign.CircuitBreaker,
		DiscoveryFiling:  cfg.Campaign.DiscoveryFiling,
		CrossRunContext:  cfg.Campaign.CrossRunContext,
		ValidationPhases: cfg.Campaign.ValidationPhases,
	}

	runner := campaign.NewRunner(orch, bdClient, stateStore, campaignCfg, cb)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	return runner.Run(ctx, c.ParentID)
}

// pipelineRunner abstracts orchestrator.RunPipeline for testing.
type pipelineRunner interface {
	RunPipeline(ctx context.Context, input orchestrator.PipelineInput) (orchestrator.PipelineOutput, error)
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

	// Resolve pipeline phases.
	phases, err := orchestrator.LoadPhases(cfg.Pipeline.Phases)
	if err != nil {
		return fmt.Errorf("run: loading phases: %w", err)
	}

	// Create a cancellable context for the pipeline. The cancel func is passed
	// to the TUI so keyboard abort (q / Ctrl+C) can cancel the pipeline gracefully.
	pipelineCtx, pipelineCancel := context.WithCancel(context.Background())
	defer pipelineCancel()

	// Build display bridge and display.
	bridge := tui.NewBridge()
	display := tui.NewDisplay(tui.DisplayOptions{
		Writer:     os.Stdout,
		ForcePlain: r.NoTUI,
		Phases:     phaseNames(phases),
		CancelFunc: pipelineCancel,
	})

	// Build orchestrator.
	promptLoader := prompt.NewLoader("prompts")
	wtMgr := worktree.NewManager(".", cfg.Worktree.BaseDir)
	wlMgr := worklog.NewManager("templates/worklog.md.template", ".capsule/logs")
	bdClient := bead.NewClient(".")
	gateRunner := gate.NewRunner()

	orch := orchestrator.New(p,
		orchestrator.WithPromptLoader(promptLoader),
		orchestrator.WithWorktreeManager(wtMgr),
		orchestrator.WithWorklogManager(wlMgr),
		orchestrator.WithGateRunner(gateRunner),
		orchestrator.WithPhases(phases),
		orchestrator.WithStatusCallback(bridgeStatusCallback(bridge)),
	)

	return r.run(os.Stdout, orch, wtMgr, bdClient, display, bridge, pipelineCtx)
}

// run executes the pipeline with display lifecycle management, enabling testable wiring.
func (r *RunCmd) run(w io.Writer, runner pipelineRunner, wt mergeOps, bd beadResolver, display tui.Display, bridge *tui.Bridge, pipelineCtx context.Context) error {
	// Start display goroutine.
	displayDone := make(chan error, 1)
	go func() {
		displayDone <- display.Run(context.Background(), bridge.Events())
	}()

	// Run the pipeline.
	pipelineErr := r.runPipeline(pipelineCtx, w, runner, bd)

	// Signal display completion.
	if pipelineErr != nil {
		bridge.Error(pipelineErr)
	} else {
		bridge.Done()
	}

	// Wait for display to finish (so it releases the terminal).
	<-displayDone

	if pipelineErr != nil {
		return pipelineErr
	}

	// Post-pipeline lifecycle: merge → cleanup → close bead.
	// Best-effort: pipeline success is the hard requirement.
	postPipeline(w, r.BeadID, wt, bd)
	return nil
}

// runPipeline resolves the bead and runs the pipeline, returning any pipeline error.
func (r *RunCmd) runPipeline(parentCtx context.Context, w io.Writer, runner pipelineRunner, bd beadResolver) error {
	// Wrap with OS signal handling so Ctrl+C in non-TUI mode still works.
	ctx, stop := signal.NotifyContext(parentCtx, os.Interrupt)
	defer stop()

	// Resolve bead context for worklog (best-effort; warnings only).
	beadCtx := r.resolveBeadContext(w, bd)

	input := orchestrator.PipelineInput{
		BeadID: r.BeadID,
		Title:  beadCtx.TaskTitle,
		Bead:   beadCtx,
	}

	_, pipelineErr := runner.RunPipeline(ctx, input)
	return pipelineErr
}

// resolveBeadContext attempts to resolve bead context, logging warnings on failure.
func (r *RunCmd) resolveBeadContext(w io.Writer, bd beadResolver) worklog.BeadContext {
	beadCtx, err := bd.Resolve(r.BeadID)
	if err != nil {
		if errors.Is(err, bead.ErrNotFound) {
			_, _ = fmt.Fprintf(w, "warning: bead %q not found (try: bd ready)\n", r.BeadID)
		} else {
			_, _ = fmt.Fprintf(w, "warning: bead resolve failed: %v\n", err)
		}
	}
	return beadCtx
}

// postPipeline performs merge, cleanup, and bead closing after a successful pipeline.
// Callable from both RunCmd and DashboardCmd. Failures print warnings to w but are
// otherwise best-effort.
func postPipeline(w io.Writer, beadID string, wt mergeOps, bd beadResolver) {
	// Detect main branch.
	mainBranch, err := wt.DetectMainBranch()
	if err != nil {
		_, _ = fmt.Fprintf(w, "warning: cannot detect main branch: %v\n", err)
		return
	}

	// Merge to main.
	commitMsg := fmt.Sprintf("%s: pipeline complete", beadID)
	err = wt.MergeToMain(beadID, mainBranch, commitMsg)
	if err != nil {
		if errors.Is(err, worktree.ErrMergeConflict) {
			_, _ = fmt.Fprintf(w, "warning: merge conflict merging capsule-%s into %s\n", beadID, mainBranch)
			_, _ = fmt.Fprintf(w, "  To fix:\n")
			_, _ = fmt.Fprintf(w, "    git checkout %s\n", mainBranch)
			_, _ = fmt.Fprintf(w, "    git merge --no-ff capsule-%s\n", beadID)
			_, _ = fmt.Fprintf(w, "    # resolve conflicts, then:\n")
			_, _ = fmt.Fprintf(w, "    capsule clean %s\n", beadID)
			return
		}
		_, _ = fmt.Fprintf(w, "warning: merge failed: %v\n", err)
		return
	}
	_, _ = fmt.Fprintf(w, "Merged capsule-%s → %s\n", beadID, mainBranch)

	// Cleanup: remove worktree and branch.
	if err := wt.Remove(beadID, true); err != nil {
		_, _ = fmt.Fprintf(w, "warning: cleanup failed: %v\n", err)
	}
	if err := wt.Prune(); err != nil {
		_, _ = fmt.Fprintf(w, "warning: prune failed: %v\n", err)
	}

	// Close bead.
	if err := bd.Close(beadID); err != nil {
		_, _ = fmt.Fprintf(w, "warning: bead close failed: %v\n", err)
	} else {
		_, _ = fmt.Fprintf(w, "Closed %s\n", beadID)
	}

	_, _ = fmt.Fprintf(w, "Worklog: .capsule/logs/%s/worklog.md\n", beadID)
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

// --- Dashboard command ---

// DashboardCmd opens the interactive dashboard TUI.
type DashboardCmd struct{}

// teaRunner abstracts Bubble Tea program execution for testing.
type teaRunner interface {
	Run() (tea.Model, error)
}

// Run builds real dependencies and launches the dashboard TUI.
func (d *DashboardCmd) Run() error {
	if !isatty.IsTerminal(os.Stdout.Fd()) && !isatty.IsCygwinTerminal(os.Stdout.Fd()) {
		return fmt.Errorf("dashboard: requires a terminal (TTY)")
	}

	if _, err := exec.LookPath("bd"); err != nil {
		return fmt.Errorf("dashboard: bd is not installed (required for bead management)")
	}

	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("dashboard: %w", err)
	}

	// Create provider via registry.
	reg := provider.NewRegistry()
	reg.Register("claude", func() (provider.Executor, error) {
		return provider.NewClaudeProvider(provider.WithTimeout(cfg.Runtime.Timeout)), nil
	})
	p, err := reg.NewProvider(cfg.Runtime.Provider)
	if err != nil {
		return fmt.Errorf("dashboard: %w", err)
	}

	// Resolve pipeline phases.
	phases, err := orchestrator.LoadPhases(cfg.Pipeline.Phases)
	if err != nil {
		return fmt.Errorf("dashboard: loading phases: %w", err)
	}

	bdClient := bead.NewClient(".")
	lister := &beadListerAdapter{client: bdClient}
	resolver := &beadResolverAdapter{client: bdClient}
	wtMgr := worktree.NewManager(".", cfg.Worktree.BaseDir)

	ppFunc := func(beadID string) error {
		postPipeline(io.Discard, beadID, wtMgr, bdClient)
		return nil
	}

	pipelineAdapter := &dashboardPipelineAdapter{
		providerExec: p,
		promptLoader: prompt.NewLoader("prompts"),
		wtMgr:        wtMgr,
		wlMgr:        worklog.NewManager("templates/worklog.md.template", ".capsule/logs"),
		gateRunner:   gate.NewRunner(),
		phases:       phases,
		bdClient:     bdClient,
	}

	campaignAdapter := &dashboardCampaignAdapter{
		beadClient: newCampaignBeadClient("."),
		stateStore: state.NewFileStore(".capsule/campaigns"),
		campaignCfg: campaign.Config{
			FailureMode:      cfg.Campaign.FailureMode,
			CircuitBreaker:   cfg.Campaign.CircuitBreaker,
			DiscoveryFiling:  cfg.Campaign.DiscoveryFiling,
			CrossRunContext:  cfg.Campaign.CrossRunContext,
			ValidationPhases: cfg.Campaign.ValidationPhases,
		},
	}

	m := dashboard.NewModel(
		dashboard.WithBeadLister(lister),
		dashboard.WithBeadResolver(resolver),
		dashboard.WithPostPipelineFunc(ppFunc),
		dashboard.WithPipelineRunner(pipelineAdapter),
		dashboard.WithPhaseNames(phaseNames(phases)),
		dashboard.WithCampaignRunner(campaignAdapter),
	)

	prog := tea.NewProgram(m, tea.WithAltScreen())
	return d.run(true, prog)
}

// run executes the tea program, enabling testable wiring.
func (d *DashboardCmd) run(isTTY bool, prog teaRunner) error {
	if !isTTY {
		return fmt.Errorf("dashboard: requires a terminal (TTY)")
	}
	_, err := prog.Run()
	return err
}

// --- Dashboard adapter types ---

// dashboardPipelineAdapter implements dashboard.PipelineRunner by building
// a fresh orchestrator per run with the provided statusFn callback.
type dashboardPipelineAdapter struct {
	providerExec provider.Executor
	promptLoader *prompt.Loader
	wtMgr        *worktree.Manager
	wlMgr        *worklog.Manager
	gateRunner   *gate.Runner
	phases       []orchestrator.PhaseDefinition
	bdClient     *bead.Client
}

func (a *dashboardPipelineAdapter) RunPipeline(ctx context.Context, input dashboard.PipelineInput, statusFn func(dashboard.PhaseUpdateMsg)) (dashboard.PipelineOutput, error) {
	// Build status callback that converts orchestrator updates to dashboard messages.
	cb := func(su orchestrator.StatusUpdate) {
		msg := dashboard.PhaseUpdateMsg{
			Phase:    su.Phase,
			Status:   dashboard.PhaseStatus(su.Status),
			Attempt:  su.Attempt,
			MaxRetry: su.MaxRetry,
			Duration: su.Duration,
		}
		if su.Signal != nil {
			msg.Summary = su.Signal.Summary
			msg.FilesChanged = su.Signal.FilesChanged
			msg.Feedback = su.Signal.Feedback
		}
		statusFn(msg)
	}

	orch := orchestrator.New(a.providerExec,
		orchestrator.WithPromptLoader(a.promptLoader),
		orchestrator.WithWorktreeManager(a.wtMgr),
		orchestrator.WithWorklogManager(a.wlMgr),
		orchestrator.WithGateRunner(a.gateRunner),
		orchestrator.WithPhases(a.phases),
		orchestrator.WithStatusCallback(cb),
	)

	// Resolve bead context (best-effort).
	beadCtx, _ := a.bdClient.Resolve(input.BeadID)

	orchInput := orchestrator.PipelineInput{
		BeadID:         input.BeadID,
		Title:          beadCtx.TaskTitle,
		Bead:           beadCtx,
		SiblingContext: input.SiblingContext,
	}

	output, err := orch.RunPipeline(ctx, orchInput)
	if err != nil {
		return dashboard.PipelineOutput{}, err
	}

	// Convert phase results.
	reports := make([]dashboard.PhaseReport, len(output.PhaseResults))
	for i, pr := range output.PhaseResults {
		reports[i] = dashboard.PhaseReport{
			PhaseName:    pr.PhaseName,
			Status:       dashboard.PhaseStatus(pr.Signal.Status),
			Summary:      pr.Signal.Summary,
			FilesChanged: pr.Signal.FilesChanged,
			Feedback:     pr.Signal.Feedback,
			Duration:     pr.Duration,
		}
	}

	return dashboard.PipelineOutput{
		Success:      output.Completed,
		PhaseReports: reports,
	}, nil
}

// beadListerAdapter wraps *bead.Client to implement dashboard.BeadLister.
type beadListerAdapter struct {
	client *bead.Client
}

func (a *beadListerAdapter) Ready() ([]dashboard.BeadSummary, error) {
	summaries, err := a.client.Ready()
	if err != nil {
		return nil, err
	}
	beads := make([]dashboard.BeadSummary, len(summaries))
	for i, s := range summaries {
		beads[i] = dashboard.BeadSummary{
			ID:       s.ID,
			Title:    s.Title,
			Priority: s.Priority,
			Type:     s.Type,
		}
	}
	return beads, nil
}

// beadResolverAdapter wraps *bead.Client to implement dashboard.BeadResolver.
type beadResolverAdapter struct {
	client *bead.Client
}

func (a *beadResolverAdapter) Resolve(id string) (dashboard.BeadDetail, error) {
	ctx, err := a.client.Resolve(id)
	if err != nil {
		return dashboard.BeadDetail{}, err
	}
	// Priority and Type are zero-valued: worklog.BeadContext does not carry them.
	return dashboard.BeadDetail{
		ID:           ctx.TaskID,
		Title:        ctx.TaskTitle,
		Description:  ctx.TaskDescription,
		Acceptance:   ctx.AcceptanceCriteria,
		EpicID:       ctx.EpicID,
		EpicTitle:    ctx.EpicTitle,
		FeatureID:    ctx.FeatureID,
		FeatureTitle: ctx.FeatureTitle,
	}, nil
}

// --- Campaign adapter types ---

// campaignBeadClient adapts bead.Client to campaign.BeadClient.
type campaignBeadClient struct {
	client *bead.Client
}

func newCampaignBeadClient(dir string) *campaignBeadClient {
	return &campaignBeadClient{client: bead.NewClient(dir)}
}

func (c *campaignBeadClient) ReadyChildren(parentID string) ([]campaign.BeadInfo, error) {
	summaries, err := c.client.Ready()
	if err != nil {
		return nil, err
	}
	// Filter to children of the parent.
	var children []campaign.BeadInfo
	for _, s := range summaries {
		if isChildOf(s.ID, parentID) {
			children = append(children, campaign.BeadInfo{
				ID:       s.ID,
				Title:    s.Title,
				Priority: s.Priority,
				Type:     s.Type,
			})
		}
	}
	return children, nil
}

// isChildOf checks if childID is a direct child of parentID (e.g. "cap-123.1" is child of "cap-123").
func isChildOf(childID, parentID string) bool {
	return strings.HasPrefix(childID, parentID+".") && len(childID) > len(parentID)+1
}

func (c *campaignBeadClient) Show(id string) (campaign.BeadInfo, error) {
	ctx, err := c.client.Resolve(id)
	if err != nil {
		return campaign.BeadInfo{}, err
	}
	return campaign.BeadInfo{
		ID:          id,
		Title:       ctx.TaskTitle,
		Description: ctx.TaskDescription,
	}, nil
}

func (c *campaignBeadClient) Close(id string) error {
	return c.client.Close(id)
}

func (c *campaignBeadClient) Create(input campaign.BeadInput) (string, error) {
	// TODO: implement bead creation via bd CLI when needed.
	return "", fmt.Errorf("bead creation not yet implemented")
}

// campaignPlainTextCallback implements campaign.Callback with plain text output.
type campaignPlainTextCallback struct {
	w io.Writer
}

func (c *campaignPlainTextCallback) OnCampaignStart(parentID string, tasks []campaign.BeadInfo) {
	_, _ = fmt.Fprintf(c.w, "[campaign] %s (%d tasks)\n", parentID, len(tasks))
}

func (c *campaignPlainTextCallback) OnTaskStart(beadID string) {
	ts := time.Now().Format("15:04:05")
	_, _ = fmt.Fprintf(c.w, "[%s] [%s] starting...\n", ts, beadID)
}

func (c *campaignPlainTextCallback) OnTaskComplete(result campaign.TaskResult) {
	ts := time.Now().Format("15:04:05")
	_, _ = fmt.Fprintf(c.w, "[%s] [%s] complete\n", ts, result.BeadID)
}

func (c *campaignPlainTextCallback) OnTaskFail(beadID string, err error) {
	ts := time.Now().Format("15:04:05")
	_, _ = fmt.Fprintf(c.w, "[%s] [%s] failed: %v\n", ts, beadID, err)
}

func (c *campaignPlainTextCallback) OnDiscoveryFiled(f provider.Finding, newBeadID string) {
	_, _ = fmt.Fprintf(c.w, "  Filed: %s [P%d]: %s\n", newBeadID, severityToPriorityCLI(f.Severity), f.Title)
}

func (c *campaignPlainTextCallback) OnValidationStart() {
	_, _ = fmt.Fprintf(c.w, "[campaign] Running feature validation...\n")
}

func (c *campaignPlainTextCallback) OnValidationComplete(result campaign.TaskResult) {
	_, _ = fmt.Fprintf(c.w, "[campaign] Validation %s\n", result.Status)
}

func (c *campaignPlainTextCallback) OnCampaignComplete(s campaign.State) {
	_, _ = fmt.Fprintf(c.w, "[campaign] Complete: %d tasks\n", len(s.Tasks))
}

func severityToPriorityCLI(severity string) int {
	switch severity {
	case "critical":
		return 0
	case "major":
		return 1
	case "minor":
		return 2
	default:
		return 3
	}
}

// --- Dashboard campaign adapter types ---

// dashboardCampaignAdapter implements dashboard.CampaignRunner by building a
// campaign.Runner and bridging its lifecycle events to tea.Msg via statusFn.
type dashboardCampaignAdapter struct {
	beadClient  campaign.BeadClient
	stateStore  campaign.StateStore
	campaignCfg campaign.Config
}

func (a *dashboardCampaignAdapter) RunCampaign(
	ctx context.Context,
	parentID string,
	statusFn func(tea.Msg),
	pipelineFn func(context.Context, dashboard.PipelineInput, func(dashboard.PhaseUpdateMsg)) (dashboard.PipelineOutput, error),
) error {
	cb := &dashboardCampaignCallback{statusFn: statusFn}
	pr := &dashboardCampaignPipelineRunner{pipelineFn: pipelineFn}
	runner := campaign.NewRunner(pr, a.beadClient, a.stateStore, a.campaignCfg, cb)
	return runner.Run(ctx, parentID)
}

// dashboardCampaignPipelineRunner implements campaign.PipelineRunner by
// bridging dashboard's pipelineFn (which accepts dashboard types) to the
// campaign's orchestrator-typed interface.
type dashboardCampaignPipelineRunner struct {
	pipelineFn func(context.Context, dashboard.PipelineInput, func(dashboard.PhaseUpdateMsg)) (dashboard.PipelineOutput, error)
}

func (r *dashboardCampaignPipelineRunner) RunPipeline(ctx context.Context, input orchestrator.PipelineInput) (orchestrator.PipelineOutput, error) {
	if r.pipelineFn == nil {
		return orchestrator.PipelineOutput{}, fmt.Errorf("no pipeline runner configured")
	}

	// Convert orchestrator input to dashboard input.
	dashInput := dashboard.PipelineInput{
		BeadID:         input.BeadID,
		SiblingContext: input.SiblingContext,
	}

	output, err := r.pipelineFn(ctx, dashInput, func(dashboard.PhaseUpdateMsg) {})
	if err != nil {
		return orchestrator.PipelineOutput{}, err
	}

	// Convert dashboard output to orchestrator output.
	results := make([]orchestrator.PhaseResult, len(output.PhaseReports))
	for i, pr := range output.PhaseReports {
		results[i] = orchestrator.PhaseResult{
			PhaseName: pr.PhaseName,
			Signal: provider.Signal{
				Status:       provider.Status(pr.Status),
				Summary:      pr.Summary,
				FilesChanged: pr.FilesChanged,
				Feedback:     pr.Feedback,
			},
			Duration: pr.Duration,
		}
	}

	return orchestrator.PipelineOutput{
		PhaseResults: results,
		Completed:    output.Success,
	}, nil
}

// dashboardCampaignCallback implements campaign.Callback by converting
// campaign lifecycle events to dashboard tea.Msg types.
//
// taskIndex and taskTotal are mutated during callback invocations.
// This struct must only be called from the campaign runner goroutine.
type dashboardCampaignCallback struct {
	statusFn  func(tea.Msg)
	taskIndex int
	taskTotal int
}

func (c *dashboardCampaignCallback) OnCampaignStart(parentID string, tasks []campaign.BeadInfo) {
	c.taskTotal = len(tasks)
	c.taskIndex = 0
	infos := make([]dashboard.CampaignTaskInfo, len(tasks))
	for i, t := range tasks {
		infos[i] = dashboard.CampaignTaskInfo{
			BeadID:   t.ID,
			Title:    t.Title,
			Priority: t.Priority,
		}
	}
	c.statusFn(dashboard.CampaignStartMsg{
		ParentID: parentID,
		Tasks:    infos,
	})
}

func (c *dashboardCampaignCallback) OnTaskStart(beadID string) {
	c.statusFn(dashboard.CampaignTaskStartMsg{
		BeadID: beadID,
		Index:  c.taskIndex,
		Total:  c.taskTotal,
	})
}

func (c *dashboardCampaignCallback) OnTaskComplete(result campaign.TaskResult) {
	var totalDuration time.Duration
	for _, pr := range result.PhaseResults {
		totalDuration += pr.Duration
	}
	c.statusFn(dashboard.CampaignTaskDoneMsg{
		BeadID:   result.BeadID,
		Index:    c.taskIndex,
		Success:  result.Status == campaign.TaskCompleted,
		Duration: totalDuration,
	})
	c.taskIndex++
}

func (c *dashboardCampaignCallback) OnTaskFail(beadID string, _ error) {
	c.statusFn(dashboard.CampaignTaskDoneMsg{
		BeadID:  beadID,
		Index:   c.taskIndex,
		Success: false,
	})
	c.taskIndex++
}

func (c *dashboardCampaignCallback) OnDiscoveryFiled(_ provider.Finding, _ string) {
	// Discovery filing is silent in dashboard mode.
}

func (c *dashboardCampaignCallback) OnValidationStart() {
	// Validation is not surfaced in dashboard campaign mode.
}

func (c *dashboardCampaignCallback) OnValidationComplete(_ campaign.TaskResult) {
	// Validation is not surfaced in dashboard campaign mode.
}

func (c *dashboardCampaignCallback) OnCampaignComplete(s campaign.State) {
	passed, failed, skipped := 0, 0, 0
	for _, t := range s.Tasks {
		switch t.Status {
		case campaign.TaskCompleted:
			passed++
		case campaign.TaskFailed:
			failed++
		case campaign.TaskSkipped:
			skipped++
		}
	}
	c.statusFn(dashboard.CampaignDoneMsg{
		ParentID:   s.ParentBeadID,
		TotalTasks: len(s.Tasks),
		Passed:     passed,
		Failed:     failed,
		Skipped:    skipped,
	})
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
	// Campaign runtime errors map to pipeline exit code (not setup).
	if errors.Is(err, campaign.ErrNoTasks) || errors.Is(err, campaign.ErrCircuitBroken) {
		return exitPipeline
	}
	return exitSetup
}

// bridgeStatusCallback returns a StatusCallback that converts orchestrator
// StatusUpdates to tui.StatusUpdateMsg and sends them through the bridge.
func bridgeStatusCallback(bridge *tui.Bridge) orchestrator.StatusCallback {
	return func(su orchestrator.StatusUpdate) {
		msg := tui.StatusUpdateMsg{
			Phase:    su.Phase,
			Status:   tui.PhaseStatus(su.Status),
			Progress: su.Progress,
			Attempt:  su.Attempt,
			MaxRetry: su.MaxRetry,
			Duration: su.Duration,
		}
		if su.Signal != nil {
			msg.Summary = su.Signal.Summary
			msg.FilesChanged = su.Signal.FilesChanged
			msg.Feedback = su.Signal.Feedback
		}
		bridge.Send(msg)
	}
}

// phaseNames extracts phase names from a slice of PhaseDefinitions.
func phaseNames(phases []orchestrator.PhaseDefinition) []string {
	names := make([]string, len(phases))
	for i, p := range phases {
		names[i] = p.Name
	}
	return names
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
