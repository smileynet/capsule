// Package orchestrator sequences pipeline phases with retry logic.
package orchestrator

import (
	"context"
	"errors"
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

// GateRunner executes shell commands as pipeline gate phases.
type GateRunner interface {
	Run(ctx context.Context, command, workDir string) (provider.Signal, error)
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
	BeadID         string
	Title          string
	Description    string
	BaseBranch     string
	Bead           worklog.BeadContext
	SkipPhases     []string                // Phases to skip (for resume from checkpoint).
	SiblingContext []prompt.SiblingContext // Completed sibling tasks for cross-run context.
}

// PhaseResult records the outcome of a single phase execution with timing metadata.
type PhaseResult struct {
	PhaseName string
	Signal    provider.Signal
	Attempt   int
	Duration  time.Duration
	Timestamp time.Time
}

// PipelineOutput is the result of running a pipeline.
type PipelineOutput struct {
	PhaseResults []PhaseResult
	Completed    bool
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

// RetryStrategy holds resolved retry settings for a phase.
// TODO(cap-6vp): Wire into runPhasePair to replace direct MaxRetries usage.
type RetryStrategy struct {
	MaxAttempts      int
	BackoffFactor    float64 // TODO(cap-6vp): apply as timeout multiplier per retry attempt.
	EscalateProvider string  // TODO(cap-6vp): switch provider after EscalateAfter attempts.
	EscalateAfter    int
}

// Orchestrator sequences pipeline phases with retry logic.
type Orchestrator struct {
	provider       Provider
	promptLoader   PromptLoader
	worktreeMgr    WorktreeManager
	worklogMgr     WorklogManager
	gateRunner     GateRunner
	phases         []PhaseDefinition
	statusCallback StatusCallback
	baseBranch     string
	retryDefaults  RetryStrategy
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
		retryDefaults: RetryStrategy{
			MaxAttempts:   3,
			BackoffFactor: 1.0,
		},
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

// WithGateRunner sets the gate runner for executing shell commands.
func WithGateRunner(r GateRunner) Option {
	return func(o *Orchestrator) { o.gateRunner = r }
}

// WithRetryDefaults sets the pipeline-wide retry defaults.
func WithRetryDefaults(rs RetryStrategy) Option {
	return func(o *Orchestrator) { o.retryDefaults = rs }
}

// WithBaseBranch sets the base branch for worktree creation.
func WithBaseBranch(branch string) Option {
	return func(o *Orchestrator) { o.baseBranch = branch }
}

// RunPipeline executes all pipeline phases for the given bead.
// It creates a worktree and worklog, executes phases sequentially,
// retries on NEEDS_WORK, and archives the worklog on completion.
// Returns PipelineOutput with phase results for the caller to persist if needed.
func (o *Orchestrator) RunPipeline(ctx context.Context, input PipelineInput) (PipelineOutput, error) {
	var output PipelineOutput

	if o.promptLoader == nil {
		return output, &PipelineError{Phase: "setup", Err: errors.New("promptLoader is required")}
	}

	beadID := input.BeadID
	baseBranch := input.BaseBranch
	if baseBranch == "" {
		baseBranch = o.baseBranch
	}

	// Build skip set for resume.
	skipSet := make(map[string]bool, len(input.SkipPhases))
	for _, name := range input.SkipPhases {
		skipSet[name] = true
	}

	// Create worktree.
	// Note: worktrees are not cleaned up on failure so they can be inspected
	// for debugging. The CLI layer (cap-9qv.5.3) handles cleanup policy.
	var wtPath string
	if o.worktreeMgr != nil {
		if err := o.worktreeMgr.Create(beadID, baseBranch); err != nil {
			return output, &PipelineError{Phase: "setup", Err: fmt.Errorf("creating worktree: %w", err)}
		}
		wtPath = o.worktreeMgr.Path(beadID)
	}

	// Create worklog.
	if o.worklogMgr != nil {
		if err := o.worklogMgr.Create(wtPath, input.Bead); err != nil {
			return output, &PipelineError{Phase: "setup", Err: fmt.Errorf("creating worklog: %w", err)}
		}
	}

	// Build base prompt context from input.
	basePCtx := prompt.Context{
		BeadID:         input.BeadID,
		Title:          input.Title,
		Description:    input.Description,
		SiblingContext: input.SiblingContext,
	}

	// Execute phases sequentially.
	for i, phase := range o.phases {
		// Skip phases for resume.
		if skipSet[phase.Name] {
			continue
		}

		progress := fmt.Sprintf("%d/%d", i+1, len(o.phases))

		o.notify(StatusUpdate{
			BeadID: beadID, Phase: phase.Name,
			Status: PhaseRunning, Progress: progress,
			Attempt: 1, MaxRetry: phase.MaxRetries,
		})

		phaseStart := time.Now()
		signal, err := o.executePhase(ctx, phase, basePCtx, wtPath)
		phaseDuration := time.Since(phaseStart)
		if err != nil {
			return output, &PipelineError{Phase: phase.Name, Attempt: 1, Err: err}
		}
		o.logPhaseEntry(wtPath, phase.Name, signal)

		output.PhaseResults = append(output.PhaseResults, PhaseResult{
			PhaseName: phase.Name,
			Signal:    signal,
			Attempt:   1,
			Duration:  phaseDuration,
			Timestamp: phaseStart,
		})

		switch signal.Status {
		case provider.StatusPass:
			o.notify(StatusUpdate{
				BeadID: beadID, Phase: phase.Name,
				Status: PhasePassed, Progress: progress,
				Attempt: 1, MaxRetry: phase.MaxRetries,
				Signal: &signal,
			})

		case provider.StatusSkip:
			o.notify(StatusUpdate{
				BeadID: beadID, Phase: phase.Name,
				Status: PhaseSkipped, Progress: progress,
				Attempt: 1, MaxRetry: phase.MaxRetries,
				Signal: &signal,
			})

		case provider.StatusError:
			if phase.Optional {
				o.notify(StatusUpdate{
					BeadID: beadID, Phase: phase.Name,
					Status: PhaseSkipped, Progress: progress,
					Attempt: 1, MaxRetry: phase.MaxRetries,
					Signal: &signal,
				})
				continue
			}
			o.notify(StatusUpdate{
				BeadID: beadID, Phase: phase.Name,
				Status: PhaseError, Progress: progress,
				Attempt: 1, MaxRetry: phase.MaxRetries,
				Signal: &signal,
			})
			return output, &PipelineError{Phase: phase.Name, Attempt: 1, Signal: signal}

		case provider.StatusNeedsWork:
			if phase.RetryTarget == "" {
				return output, &PipelineError{
					Phase: phase.Name, Attempt: 1, Signal: signal,
					Err: fmt.Errorf("phase %q returned NEEDS_WORK but has no retry target", phase.Name),
				}
			}
			target, ok := o.findPhase(phase.RetryTarget)
			if !ok {
				return output, &PipelineError{
					Phase: phase.Name, Attempt: 1,
					Err: fmt.Errorf("retry target %q not found", phase.RetryTarget),
				}
			}
			o.notify(StatusUpdate{
				BeadID: beadID, Phase: phase.Name,
				Status: PhaseFailed, Progress: progress,
				Attempt: 1, MaxRetry: phase.MaxRetries,
				Signal: &signal,
			})
			_, err := o.runPhasePair(ctx, target, phase, basePCtx, wtPath, progress, signal.Feedback, 2)
			if err != nil {
				return output, err
			}
		}
	}

	// Archive worklog.
	if o.worklogMgr != nil {
		if err := o.worklogMgr.Archive(wtPath, beadID); err != nil {
			return output, &PipelineError{Phase: "teardown", Err: fmt.Errorf("archiving worklog: %w", err)}
		}
	}

	output.Completed = true
	return output, nil
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
				Signal: &workerSignal,
			})
			return workerSignal, &PipelineError{Phase: worker.Name, Attempt: attempt, Signal: workerSignal}
		}

		o.notify(StatusUpdate{
			BeadID: basePCtx.BeadID, Phase: worker.Name,
			Status: PhasePassed, Progress: progress,
			Attempt: attempt, MaxRetry: reviewer.MaxRetries,
			Signal: &workerSignal,
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
				Signal: &reviewerSignal,
			})
			return reviewerSignal, nil

		case provider.StatusError:
			o.notify(StatusUpdate{
				BeadID: basePCtx.BeadID, Phase: reviewer.Name,
				Status: PhaseError, Progress: progress,
				Attempt: attempt, MaxRetry: reviewer.MaxRetries,
				Signal: &reviewerSignal,
			})
			return reviewerSignal, &PipelineError{Phase: reviewer.Name, Attempt: attempt, Signal: reviewerSignal}

		case provider.StatusNeedsWork:
			o.notify(StatusUpdate{
				BeadID: basePCtx.BeadID, Phase: reviewer.Name,
				Status: PhaseFailed, Progress: progress,
				Attempt: attempt, MaxRetry: reviewer.MaxRetries,
				Signal: &reviewerSignal,
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
// For Gate phases, it delegates to the GateRunner.
// For Worker and Reviewer phases, it composes a prompt and calls the provider.
func (o *Orchestrator) executePhase(ctx context.Context, phase PhaseDefinition,
	pCtx prompt.Context, wtPath string) (provider.Signal, error) {

	if phase.Kind == Gate {
		return o.executeGate(ctx, phase, wtPath)
	}

	promptName := phase.PromptName()
	composed, err := o.promptLoader.Compose(promptName, pCtx)
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

// executeGate runs a gate phase via the GateRunner.
func (o *Orchestrator) executeGate(ctx context.Context, phase PhaseDefinition, wtPath string) (provider.Signal, error) {
	if o.gateRunner == nil {
		return provider.Signal{}, fmt.Errorf("gate phase %q requires a GateRunner", phase.Name)
	}
	return o.gateRunner.Run(ctx, phase.Command, wtPath)
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

// ResolveRetryStrategy returns the effective retry strategy for a phase.
// Phase-level MaxRetries override pipeline-level defaults.
func (o *Orchestrator) ResolveRetryStrategy(phase PhaseDefinition) RetryStrategy {
	rs := o.retryDefaults
	if phase.MaxRetries > 0 {
		rs.MaxAttempts = phase.MaxRetries
	}

	return rs
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
