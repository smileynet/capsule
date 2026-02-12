// Package campaign orchestrates multi-task pipelines for features and epics.
package campaign

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/smileynet/capsule/internal/orchestrator"
	"github.com/smileynet/capsule/internal/prompt"
	"github.com/smileynet/capsule/internal/provider"
)

// Sentinel errors for caller-checkable conditions.
var (
	ErrCircuitBroken = errors.New("campaign: circuit breaker tripped")
	ErrNoTasks       = errors.New("campaign: no ready tasks found")
)

// PipelineRunner abstracts the orchestrator for campaign use.
type PipelineRunner interface {
	RunPipeline(ctx context.Context, input orchestrator.PipelineInput) (orchestrator.PipelineOutput, error)
}

// BeadInfo holds minimal bead metadata for campaign task sequencing.
type BeadInfo struct {
	ID          string
	Title       string
	Description string
	Priority    int
	Type        string
}

// BeadInput holds the fields needed to create a new bead.
type BeadInput struct {
	ParentID string
	Type     string
	Title    string
	Priority int
}

// BeadClient abstracts bead CLI operations for campaign use.
type BeadClient interface {
	ReadyChildren(parentID string) ([]BeadInfo, error)
	Show(id string) (BeadInfo, error)
	Close(id string) error
	Create(input BeadInput) (string, error)
}

// StateStore persists campaign state between runs.
type StateStore interface {
	Save(state State) error
	Load(id string) (State, bool, error)
	Remove(id string) error
}

// Callback receives campaign lifecycle events for display.
type Callback interface {
	OnCampaignStart(parentID string, tasks []BeadInfo)
	OnTaskStart(beadID string)
	OnTaskComplete(result TaskResult)
	OnTaskFail(beadID string, err error)
	OnDiscoveryFiled(finding provider.Finding, newBeadID string)
	OnValidationStart()
	OnValidationComplete(result TaskResult)
	OnCampaignComplete(state State)
}

// Config holds campaign-specific settings.
type Config struct {
	FailureMode      string // "abort" | "continue"
	CircuitBreaker   int    // Max consecutive failures before stopping.
	DiscoveryFiling  bool   // File findings as new beads.
	CrossRunContext  bool   // Include sibling context in prompts.
	ValidationPhases string // Phase set name for feature validation.
}

// State holds the complete campaign state for persistence.
type State struct {
	ID             string       `json:"id"`
	ParentBeadID   string       `json:"parent_bead_id"`
	Tasks          []TaskResult `json:"tasks"`
	CurrentTaskIdx int          `json:"current_task_idx"`
	ConsecFailures int          `json:"consecutive_failures"`
	StartedAt      time.Time    `json:"started_at"`
	Status         string       `json:"status"` // "running" | "completed" | "failed" | "paused"
}

// TaskResult records the outcome of a single task within a campaign.
type TaskResult struct {
	BeadID       string                     `json:"bead_id"`
	Status       string                     `json:"status"` // "pending" | "completed" | "failed" | "skipped"
	PhaseResults []orchestrator.PhaseResult `json:"phase_results"`
	Error        string                     `json:"error,omitempty"`
}

// Runner orchestrates a campaign: sequential task execution with circuit breaking,
// discovery filing, and state persistence.
type Runner struct {
	pipeline PipelineRunner
	beads    BeadClient
	store    StateStore
	config   Config
	callback Callback
}

// NewRunner creates a campaign Runner with the given dependencies.
func NewRunner(pipeline PipelineRunner, beads BeadClient, store StateStore, config Config, callback Callback) *Runner {
	return &Runner{
		pipeline: pipeline,
		beads:    beads,
		store:    store,
		config:   config,
		callback: callback,
	}
}

// Run executes a campaign for the given parent bead (feature or epic).
// It discovers ready children, runs pipelines sequentially, handles failures,
// files discoveries, and runs validation on completion.
func (r *Runner) Run(ctx context.Context, parentID string) error {
	children, err := r.beads.ReadyChildren(parentID)
	if err != nil {
		return fmt.Errorf("campaign: listing children of %s: %w", parentID, err)
	}
	if len(children) == 0 {
		return ErrNoTasks
	}

	r.callback.OnCampaignStart(parentID, children)

	state := r.initOrResumeState(parentID, children)
	state.Status = "running"

	for i := state.CurrentTaskIdx; i < len(state.Tasks); i++ {
		task := &state.Tasks[i]
		if task.Status == "completed" || task.Status == "skipped" {
			continue
		}

		if r.config.CircuitBreaker > 0 && state.ConsecFailures >= r.config.CircuitBreaker {
			state.Status = "failed"
			_ = r.store.Save(state)
			return ErrCircuitBroken
		}

		r.callback.OnTaskStart(task.BeadID)
		task.Status = "running"

		input := r.buildPipelineInput(task.BeadID, state)
		output, err := r.pipeline.RunPipeline(ctx, input)

		if err != nil {
			task.Status = "failed"
			task.Error = err.Error()
			state.ConsecFailures++
			r.callback.OnTaskFail(task.BeadID, err)

			if r.config.FailureMode == "abort" {
				state.Status = "failed"
				_ = r.store.Save(state)
				return fmt.Errorf("campaign: task %s failed: %w", task.BeadID, err)
			}
			state.CurrentTaskIdx = i + 1
			_ = r.store.Save(state)
			continue // "continue" mode
		}

		task.Status = "completed"
		task.PhaseResults = output.PhaseResults
		state.ConsecFailures = 0
		r.callback.OnTaskComplete(*task)

		r.fileDiscoveries(output, parentID)
		r.runPostPipeline(task.BeadID)

		state.CurrentTaskIdx = i + 1
		_ = r.store.Save(state)
	}

	// All tasks done — run feature validation if configured.
	if r.allComplete(state) && r.config.ValidationPhases != "" {
		r.callback.OnValidationStart()
		valResult := r.runValidation(ctx, parentID, state)
		r.callback.OnValidationComplete(valResult)
	}

	state.Status = "completed"
	_ = r.store.Save(state)
	r.callback.OnCampaignComplete(state)
	return nil
}

// initOrResumeState loads existing state or creates a new one.
func (r *Runner) initOrResumeState(parentID string, children []BeadInfo) State {
	existing, found, err := r.store.Load(parentID)
	if err == nil && found && existing.Status != "completed" {
		return existing
	}

	tasks := make([]TaskResult, len(children))
	for i, c := range children {
		tasks[i] = TaskResult{BeadID: c.ID, Status: "pending"}
	}

	return State{
		ID:           parentID,
		ParentBeadID: parentID,
		Tasks:        tasks,
		StartedAt:    time.Now(),
		Status:       "running",
	}
}

// buildPipelineInput creates a PipelineInput for a task, optionally including sibling context.
func (r *Runner) buildPipelineInput(beadID string, state State) orchestrator.PipelineInput {
	input := orchestrator.PipelineInput{BeadID: beadID}

	// Look up bead details for the title/description.
	info, err := r.beads.Show(beadID)
	if err == nil {
		input.Title = info.Title
		input.Description = info.Description
	}

	// Include sibling context from completed tasks.
	if r.config.CrossRunContext {
		input.SiblingContext = r.buildSiblingContext(state)
	}

	return input
}

// buildSiblingContext builds a slice of completed sibling summaries for cross-run context.
func (r *Runner) buildSiblingContext(state State) []prompt.SiblingContext {
	var siblings []prompt.SiblingContext
	for _, task := range state.Tasks {
		if task.Status != "completed" {
			continue
		}
		sc := prompt.SiblingContext{BeadID: task.BeadID}

		// Extract summary and files from the last phase result.
		if len(task.PhaseResults) > 0 {
			last := task.PhaseResults[len(task.PhaseResults)-1]
			sc.Summary = last.Signal.Summary
			sc.FilesChanged = last.Signal.FilesChanged
		}

		// Try to get the title from the bead client.
		info, err := r.beads.Show(task.BeadID)
		if err == nil {
			sc.Title = info.Title
		}

		siblings = append(siblings, sc)
	}
	return siblings
}

// fileDiscoveries creates new beads from findings in phase outputs.
func (r *Runner) fileDiscoveries(output orchestrator.PipelineOutput, parentID string) {
	if !r.config.DiscoveryFiling {
		return
	}

	for _, pr := range output.PhaseResults {
		for _, f := range pr.Signal.Findings {
			newID, err := r.beads.Create(BeadInput{
				ParentID: parentID,
				Type:     "task",
				Title:    f.Title,
				Priority: severityToPriority(f.Severity),
			})
			if err != nil {
				continue
			}
			r.callback.OnDiscoveryFiled(f, newID)
		}
	}
}

// runPostPipeline closes the bead after successful pipeline completion.
func (r *Runner) runPostPipeline(beadID string) {
	_ = r.beads.Close(beadID)
}

// allComplete checks if all tasks are completed.
func (r *Runner) allComplete(state State) bool {
	for _, task := range state.Tasks {
		if task.Status != "completed" && task.Status != "skipped" {
			return false
		}
	}
	return true
}

// runValidation runs a validation pipeline for the parent bead.
func (r *Runner) runValidation(ctx context.Context, parentID string, _ State) TaskResult {
	input := orchestrator.PipelineInput{
		BeadID: parentID,
		Title:  "Feature validation: " + parentID,
	}
	output, err := r.pipeline.RunPipeline(ctx, input)
	if err != nil {
		return TaskResult{
			BeadID: parentID,
			Status: "failed",
			Error:  err.Error(),
		}
	}
	return TaskResult{
		BeadID:       parentID,
		Status:       "completed",
		PhaseResults: output.PhaseResults,
	}
}

// severityToPriority maps finding severity to bead priority.
func severityToPriority(severity string) int {
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
