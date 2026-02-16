// Package dashboard implements a two-pane TUI for browsing beads and
// dispatching pipelines. Separate from internal/tui which handles
// the capsule run display.
package dashboard

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/smileynet/capsule/internal/prompt"
)

// Mode represents the current dashboard view mode.
type Mode int

const (
	ModeBrowse          Mode = iota // Browsing bead list with detail pane.
	ModePipeline                    // Pipeline running with phase list and reports.
	ModeSummary                     // Pipeline complete, showing result summary.
	ModeCampaign                    // Campaign running with task queue and inline phases.
	ModeCampaignSummary             // Campaign complete, showing aggregate results.
)

// Focus represents which pane has keyboard focus.
type Focus int

const (
	PaneLeft  Focus = iota // Left pane (bead list or phase list) has focus.
	PaneRight              // Right pane (detail or report viewport) has focus.
)

// BeadSummary is a minimal view of a bead for the list pane.
type BeadSummary struct {
	ID       string
	Title    string
	Priority int
	Type     string
}

// BeadDetail is the resolved detail of a single bead for the right pane.
type BeadDetail struct {
	ID           string
	Title        string
	Priority     int
	Type         string
	Description  string
	Acceptance   string
	EpicID       string
	EpicTitle    string
	FeatureID    string
	FeatureTitle string
}

// PhaseStatus represents the current state of a pipeline phase.
type PhaseStatus string

const (
	PhasePending PhaseStatus = "pending"
	PhaseRunning PhaseStatus = "running"
	PhasePassed  PhaseStatus = "passed"
	PhaseFailed  PhaseStatus = "failed"
	PhaseError   PhaseStatus = "error"
	PhaseSkipped PhaseStatus = "skipped"
)

// PhaseReport stores the result of a completed pipeline phase.
type PhaseReport struct {
	PhaseName    string
	Status       PhaseStatus
	Summary      string
	Feedback     string
	FilesChanged []string
	Duration     time.Duration
}

// PipelineInput is the input to start a pipeline run.
type PipelineInput struct {
	BeadID         string
	Provider       string
	SiblingContext []prompt.SiblingContext // Completed sibling tasks for cross-run context.
}

// PipelineOutput is the result of a completed pipeline run.
type PipelineOutput struct {
	Success      bool
	Error        error
	PhaseReports []PhaseReport
}

// --- Consumer-side interfaces ---

// BeadLister fetches bead lists for the browse pane.
type BeadLister interface {
	Ready() ([]BeadSummary, error)
	Closed(limit int) ([]BeadSummary, error)
}

// BeadResolver fetches full detail for a single bead.
type BeadResolver interface {
	Resolve(id string) (BeadDetail, error)
}

// PipelineRunner dispatches and runs a pipeline.
type PipelineRunner interface {
	RunPipeline(ctx context.Context, input PipelineInput, statusFn func(PhaseUpdateMsg)) (PipelineOutput, error)
}

// PostPipelineFunc runs post-pipeline lifecycle (merge, cleanup, close bead).
// Called in a background goroutine after a pipeline completes and the user
// returns to browse mode. Errors are surfaced via PostPipelineDoneMsg but
// not displayed in the UI.
type PostPipelineFunc func(beadID string) error

// --- tea.Msg types ---

// BeadListMsg carries the result of a BeadLister.Ready() call.
type BeadListMsg struct {
	Beads []BeadSummary
	Err   error
}

// ClosedBeadListMsg carries the result of a BeadLister.Closed() call.
type ClosedBeadListMsg struct {
	Beads []BeadSummary
	Err   error
}

// BeadResolvedMsg carries the result of a BeadResolver.Resolve() call.
type BeadResolvedMsg struct {
	ID     string
	Detail BeadDetail
	Err    error
}

// PhaseUpdateMsg carries a status update for a single pipeline phase.
type PhaseUpdateMsg struct {
	Phase        string
	Status       PhaseStatus
	Attempt      int
	MaxRetry     int
	Duration     time.Duration
	Summary      string
	FilesChanged []string
	Feedback     string
}

// PipelineDoneMsg signals successful pipeline completion.
type PipelineDoneMsg struct {
	Output PipelineOutput
}

// PipelineErrorMsg signals pipeline failure.
type PipelineErrorMsg struct {
	Err error
}

// DispatchMsg signals the user has selected a bead to run a pipeline on.
type DispatchMsg struct {
	BeadID    string
	BeadType  string
	BeadTitle string
}

// RefreshBeadsMsg signals that the bead list should be reloaded.
// browseState emits this on 'r'; Model.Update intercepts it and calls initBrowse.
type RefreshBeadsMsg struct{}

// ToggleHistoryMsg signals that the browse pane should switch to closed beads.
// browseState emits this on 'h'; Model.Update intercepts it and fetches closed beads.
type ToggleHistoryMsg struct{}

// PostPipelineDoneMsg signals that post-pipeline lifecycle completed.
// Best-effort: the UI does not display errors from post-pipeline.
type PostPipelineDoneMsg struct {
	BeadID string
	Err    error
}

// elapsedTickMsg is sent every second to update the elapsed time display
// for running pipeline phases.
type elapsedTickMsg struct{}

// channelClosedMsg signals that the pipeline event channel has been closed,
// indicating the pipeline goroutine has finished.
type channelClosedMsg struct{}

// --- Campaign types ---

// CampaignTaskStatus represents the state of a task within a campaign.
type CampaignTaskStatus string

const (
	CampaignTaskPending CampaignTaskStatus = "pending"
	CampaignTaskRunning CampaignTaskStatus = "running"
	CampaignTaskPassed  CampaignTaskStatus = "passed"
	CampaignTaskFailed  CampaignTaskStatus = "failed"
	CampaignTaskSkipped CampaignTaskStatus = "skipped"
)

// CampaignTaskInfo describes a child task in a campaign.
type CampaignTaskInfo struct {
	BeadID   string
	Title    string
	Priority int
}

// --- Campaign tea.Msg types ---

// CampaignStartMsg signals that a campaign has been discovered and is starting.
type CampaignStartMsg struct {
	ParentID    string
	ParentTitle string
	Tasks       []CampaignTaskInfo
}

// CampaignTaskStartMsg signals that a specific task within a campaign is starting.
type CampaignTaskStartMsg struct {
	BeadID string
	Index  int
	Total  int
}

// CampaignTaskDoneMsg signals that a specific task within a campaign has completed.
type CampaignTaskDoneMsg struct {
	BeadID       string
	Index        int
	Success      bool
	Duration     time.Duration
	PhaseReports []PhaseReport
}

// CampaignDoneMsg signals that the entire campaign has completed.
type CampaignDoneMsg struct {
	ParentID   string
	TotalTasks int
	Passed     int
	Failed     int
	Skipped    int
}

// CampaignErrorMsg signals that the campaign runner returned an error.
type CampaignErrorMsg struct {
	Err error
}

// CampaignRunner dispatches and runs a campaign (sequential child pipelines).
type CampaignRunner interface {
	RunCampaign(
		ctx context.Context,
		parentID string,
		statusFn func(tea.Msg),
		pipelineFn func(context.Context, PipelineInput, func(PhaseUpdateMsg)) (PipelineOutput, error),
	) error
}
