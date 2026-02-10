// Package orchestrator sequences pipeline phases with retry logic.
package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/smileynet/capsule/internal/prompt"
	"github.com/smileynet/capsule/internal/provider"
	"github.com/smileynet/capsule/internal/worklog"
)

// Provider executes AI completions against a configured backend.
// Defined here (the consumer) per Go convention: accept interfaces, return structs.
type Provider interface {
	// Name returns the provider identifier (e.g. "claude").
	Name() string
	// Execute runs a prompt in the given working directory and returns the raw result.
	Execute(ctx context.Context, prompt, workDir string) (provider.Result, error)
}

// PromptLoader composes prompts for pipeline phases.
type PromptLoader interface {
	Compose(phaseName string, ctx prompt.Context) (string, error)
}

// WorktreeManager manages git worktrees for pipeline isolation.
type WorktreeManager interface {
	Create(id, baseBranch string) error
	Remove(id string, deleteBranch bool) error
	Path(id string) string
}

// WorklogManager tracks phase execution in a worklog.
type WorklogManager interface {
	Create(worktreePath string, bead worklog.BeadContext) error
	AppendPhaseEntry(worktreePath string, entry worklog.PhaseEntry) error
	Archive(worktreePath, beadID string) error
}

// PipelineInput provides the context needed to run a pipeline.
type PipelineInput struct {
	BeadID      string
	Title       string
	Description string
	BaseBranch  string
	Bead        worklog.BeadContext
}

// PipelineError indicates a pipeline failure with phase context.
type PipelineError struct {
	Phase   string          // Phase that failed.
	Attempt int             // Attempt number when failure occurred (0 if not in retry).
	Signal  provider.Signal // Last signal received (zero value if error before signal).
	Err     error           // Underlying error (nil if failure was from signal status).
}

func (e *PipelineError) Error() string {
	if e.Attempt == 0 {
		if e.Err != nil {
			return fmt.Sprintf("pipeline: phase %q: %s", e.Phase, e.Err)
		}
		return fmt.Sprintf("pipeline: phase %q: status %s: %s",
			e.Phase, e.Signal.Status, e.Signal.Feedback)
	}
	if e.Err != nil {
		return fmt.Sprintf("pipeline: phase %q attempt %d: %s", e.Phase, e.Attempt, e.Err)
	}
	return fmt.Sprintf("pipeline: phase %q attempt %d: status %s: %s",
		e.Phase, e.Attempt, e.Signal.Status, e.Signal.Feedback)
}

func (e *PipelineError) Unwrap() error {
	return e.Err
}

// Orchestrator sequences pipeline phases with retry logic.
type Orchestrator struct {
	provider       Provider
	promptLoader   PromptLoader
	worktreeMgr    WorktreeManager
	worklogMgr     WorklogManager
	phases         []PhaseDefinition
	statusCallback StatusCallback
	baseBranch     string
}

// Option configures an Orchestrator.
type Option func(*Orchestrator)

// New creates an Orchestrator with the given provider and options.
func New(p Provider, opts ...Option) *Orchestrator {
	o := &Orchestrator{
		provider:       p,
		phases:         DefaultPhases(),
		statusCallback: func(StatusUpdate) {},
		baseBranch:     "main",
	}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// WithPromptLoader sets the prompt loader.
func WithPromptLoader(l PromptLoader) Option {
	return func(o *Orchestrator) { o.promptLoader = l }
}

// WithWorktreeManager sets the worktree manager.
func WithWorktreeManager(m WorktreeManager) Option {
	return func(o *Orchestrator) { o.worktreeMgr = m }
}

// WithWorklogManager sets the worklog manager.
func WithWorklogManager(m WorklogManager) Option {
	return func(o *Orchestrator) { o.worklogMgr = m }
}

// WithPhases overrides the default phase definitions.
func WithPhases(phases []PhaseDefinition) Option {
	return func(o *Orchestrator) { o.phases = phases }
}

// WithStatusCallback sets the callback for progress updates.
func WithStatusCallback(cb StatusCallback) Option {
	return func(o *Orchestrator) { o.statusCallback = cb }
}

// WithBaseBranch sets the base branch for worktree creation.
func WithBaseBranch(branch string) Option {
	return func(o *Orchestrator) { o.baseBranch = branch }
}

// RunPipeline executes all pipeline phases for the given bead.
// It creates a worktree and worklog, executes phases sequentially,
// retries on NEEDS_WORK, and archives the worklog on completion.
func (o *Orchestrator) RunPipeline(ctx context.Context, input PipelineInput) error {
	if o.promptLoader == nil {
		return &PipelineError{Phase: "setup", Err: fmt.Errorf("promptLoader is required")}
	}

	beadID := input.BeadID
	baseBranch := input.BaseBranch
	if baseBranch == "" {
		baseBranch = o.baseBranch
	}

	// Create worktree.
	// Note: worktrees are not cleaned up on failure so they can be inspected
	// for debugging. The CLI layer (cap-9qv.5.3) handles cleanup policy.
	var wtPath string
	if o.worktreeMgr != nil {
		if err := o.worktreeMgr.Create(beadID, baseBranch); err != nil {
			return &PipelineError{Phase: "setup", Err: fmt.Errorf("creating worktree: %w", err)}
		}
		wtPath = o.worktreeMgr.Path(beadID)
	}

	// Create worklog.
	if o.worklogMgr != nil {
		if err := o.worklogMgr.Create(wtPath, input.Bead); err != nil {
			return &PipelineError{Phase: "setup", Err: fmt.Errorf("creating worklog: %w", err)}
		}
	}

	// Build base prompt context from input.
	basePCtx := prompt.Context{
		BeadID:      input.BeadID,
		Title:       input.Title,
		Description: input.Description,
	}

	// Execute phases sequentially.
	for i, phase := range o.phases {
		progress := fmt.Sprintf("%d/%d", i+1, len(o.phases))

		o.notify(StatusUpdate{
			BeadID: beadID, Phase: phase.Name,
			Status: PhaseRunning, Progress: progress,
			Attempt: 1, MaxRetry: phase.MaxRetries,
		})

		signal, err := o.executePhase(ctx, phase, basePCtx, wtPath)
		if err != nil {
			return &PipelineError{Phase: phase.Name, Attempt: 1, Err: err}
		}
		o.logPhaseEntry(wtPath, phase.Name, signal)

		switch signal.Status {
		case provider.StatusPass:
			o.notify(StatusUpdate{
				BeadID: beadID, Phase: phase.Name,
				Status: PhasePassed, Progress: progress,
				Attempt: 1, MaxRetry: phase.MaxRetries,
			})

		case provider.StatusError:
			o.notify(StatusUpdate{
				BeadID: beadID, Phase: phase.Name,
				Status: PhaseError, Progress: progress,
				Attempt: 1, MaxRetry: phase.MaxRetries,
			})
			return &PipelineError{Phase: phase.Name, Attempt: 1, Signal: signal}

		case provider.StatusNeedsWork:
			if phase.RetryTarget == "" {
				return &PipelineError{
					Phase: phase.Name, Attempt: 1, Signal: signal,
					Err: fmt.Errorf("phase %q returned NEEDS_WORK but has no retry target", phase.Name),
				}
			}
			target, ok := o.findPhase(phase.RetryTarget)
			if !ok {
				return &PipelineError{
					Phase: phase.Name, Attempt: 1,
					Err: fmt.Errorf("retry target %q not found", phase.RetryTarget),
				}
			}
			o.notify(StatusUpdate{
				BeadID: beadID, Phase: phase.Name,
				Status: PhaseFailed, Progress: progress,
				Attempt: 1, MaxRetry: phase.MaxRetries,
			})
			_, err := o.runPhasePair(ctx, target, phase, basePCtx, wtPath, progress, signal.Feedback, 2)
			if err != nil {
				return err
			}
		}
	}

	// Archive worklog.
	if o.worklogMgr != nil {
		if err := o.worklogMgr.Archive(wtPath, beadID); err != nil {
			return &PipelineError{Phase: "teardown", Err: fmt.Errorf("archiving worklog: %w", err)}
		}
	}

	return nil
}

// runPhasePair retries a worker-reviewer pair. On each attempt, the worker
// executes with feedback, then the reviewer evaluates. Returns the final
// reviewer signal on PASS, or an error on ERROR or max retries exhausted.
func (o *Orchestrator) runPhasePair(ctx context.Context, worker, reviewer PhaseDefinition,
	basePCtx prompt.Context, wtPath, progress, feedback string, startAttempt int) (provider.Signal, error) {

	for attempt := startAttempt; attempt <= reviewer.MaxRetries; attempt++ {
		// Run worker with feedback.
		workerCtx := basePCtx
		workerCtx.Feedback = feedback

		o.notify(StatusUpdate{
			BeadID: basePCtx.BeadID, Phase: worker.Name,
			Status: PhaseRunning, Progress: progress,
			Attempt: attempt, MaxRetry: reviewer.MaxRetries,
		})

		workerSignal, err := o.executePhase(ctx, worker, workerCtx, wtPath)
		if err != nil {
			return provider.Signal{}, &PipelineError{Phase: worker.Name, Attempt: attempt, Err: err}
		}
		o.logPhaseEntry(wtPath, worker.Name, workerSignal)

		// Workers return PASS or ERROR. NEEDS_WORK from a worker is treated
		// as PASS (the reviewer will evaluate the output quality).
		if workerSignal.Status == provider.StatusError {
			o.notify(StatusUpdate{
				BeadID: basePCtx.BeadID, Phase: worker.Name,
				Status: PhaseError, Progress: progress,
				Attempt: attempt, MaxRetry: reviewer.MaxRetries,
			})
			return workerSignal, &PipelineError{Phase: worker.Name, Attempt: attempt, Signal: workerSignal}
		}

		o.notify(StatusUpdate{
			BeadID: basePCtx.BeadID, Phase: worker.Name,
			Status: PhasePassed, Progress: progress,
			Attempt: attempt, MaxRetry: reviewer.MaxRetries,
		})

		// Run reviewer.
		o.notify(StatusUpdate{
			BeadID: basePCtx.BeadID, Phase: reviewer.Name,
			Status: PhaseRunning, Progress: progress,
			Attempt: attempt, MaxRetry: reviewer.MaxRetries,
		})

		reviewerSignal, err := o.executePhase(ctx, reviewer, basePCtx, wtPath)
		if err != nil {
			return provider.Signal{}, &PipelineError{Phase: reviewer.Name, Attempt: attempt, Err: err}
		}
		o.logPhaseEntry(wtPath, reviewer.Name, reviewerSignal)

		switch reviewerSignal.Status {
		case provider.StatusPass:
			o.notify(StatusUpdate{
				BeadID: basePCtx.BeadID, Phase: reviewer.Name,
				Status: PhasePassed, Progress: progress,
				Attempt: attempt, MaxRetry: reviewer.MaxRetries,
			})
			return reviewerSignal, nil

		case provider.StatusError:
			o.notify(StatusUpdate{
				BeadID: basePCtx.BeadID, Phase: reviewer.Name,
				Status: PhaseError, Progress: progress,
				Attempt: attempt, MaxRetry: reviewer.MaxRetries,
			})
			return reviewerSignal, &PipelineError{Phase: reviewer.Name, Attempt: attempt, Signal: reviewerSignal}

		case provider.StatusNeedsWork:
			o.notify(StatusUpdate{
				BeadID: basePCtx.BeadID, Phase: reviewer.Name,
				Status: PhaseFailed, Progress: progress,
				Attempt: attempt, MaxRetry: reviewer.MaxRetries,
			})
			feedback = reviewerSignal.Feedback
		}
	}

	return provider.Signal{}, &PipelineError{
		Phase:   reviewer.Name,
		Attempt: reviewer.MaxRetries,
		Err:     fmt.Errorf("max retries (%d) exceeded", reviewer.MaxRetries),
	}
}

// executePhase composes a prompt and executes a single phase.
func (o *Orchestrator) executePhase(ctx context.Context, phase PhaseDefinition,
	pCtx prompt.Context, wtPath string) (provider.Signal, error) {

	composed, err := o.promptLoader.Compose(phase.Name, pCtx)
	if err != nil {
		return provider.Signal{}, fmt.Errorf("composing prompt for %s: %w", phase.Name, err)
	}

	result, err := o.provider.Execute(ctx, composed, wtPath)
	if err != nil {
		return provider.Signal{}, fmt.Errorf("executing %s: %w", phase.Name, err)
	}

	signal, err := result.ParseSignal()
	if err != nil {
		return provider.Signal{}, fmt.Errorf("parsing signal for %s: %w", phase.Name, err)
	}

	return signal, nil
}

// findPhase looks up a phase definition by name.
func (o *Orchestrator) findPhase(name string) (PhaseDefinition, bool) {
	for _, p := range o.phases {
		if p.Name == name {
			return p, true
		}
	}
	return PhaseDefinition{}, false
}

// notify fires the status callback.
func (o *Orchestrator) notify(su StatusUpdate) {
	o.statusCallback(su)
}

// logPhaseEntry records a phase result in the worklog (best-effort).
func (o *Orchestrator) logPhaseEntry(wtPath, phaseName string, signal provider.Signal) {
	if o.worklogMgr == nil {
		return
	}
	// Best-effort: worklog failures don't abort the pipeline.
	_ = o.worklogMgr.AppendPhaseEntry(wtPath, worklog.PhaseEntry{
		Name:      phaseName,
		Status:    string(signal.Status),
		Verdict:   signal.Summary,
		Timestamp: time.Now(),
	})
}
