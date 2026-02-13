// Package dashboard implements a two-pane TUI for browsing beads and
// dispatching pipelines. Separate from internal/tui which handles
// the capsule run display.
package dashboard

import "time"

// Mode represents the current dashboard view mode.
type Mode int

const (
	ModeBrowse   Mode = iota // Browsing bead list with detail pane.
	ModePipeline             // Pipeline running with phase list and reports.
	ModeSummary              // Pipeline complete, showing result summary.
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
	BeadID   string
	Provider string
}

// PipelineOutput is the result of a completed pipeline run.
type PipelineOutput struct {
	Success      bool
	Error        error
	PhaseReports []PhaseReport
}

// --- Consumer-side interfaces ---

// BeadLister fetches the list of ready beads.
type BeadLister interface {
	Ready() ([]BeadSummary, error)
}

// BeadResolver fetches full detail for a single bead.
type BeadResolver interface {
	Resolve(id string) (BeadDetail, error)
}

// PipelineRunner dispatches and runs a pipeline.
type PipelineRunner interface {
	RunPipeline(input PipelineInput, statusFn func(PhaseUpdateMsg)) (PipelineOutput, error)
}

// --- tea.Msg types ---

// BeadListMsg carries the result of a BeadLister.Ready() call.
type BeadListMsg struct {
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
	BeadID string
}

// RefreshBeadsMsg signals that the bead list should be reloaded.
// browseState emits this on 'r'; Model.Update intercepts it and calls initBrowse.
type RefreshBeadsMsg struct{}
