package dashboard

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// campaignState manages the task queue, embedded pipeline state, and
// progress counters for campaign mode.
type campaignState struct {
	parentID      string
	parentTitle   string
	tasks         []CampaignTaskInfo
	taskStatuses  []CampaignTaskStatus
	taskDurations []time.Duration
	taskReports   map[string][]PhaseReport // Phase reports keyed by bead ID.
	currentIdx    int                      // -1 = no task running
	selectedIdx   int                      // Cursor for browsing tasks (independent of currentIdx).
	pipeline      pipelineState
	completed     int
	failed        int
}

// newCampaignState creates a campaignState for the given parent and tasks.
func newCampaignState(parentID, parentTitle string, tasks []CampaignTaskInfo) campaignState {
	statuses := make([]CampaignTaskStatus, len(tasks))
	for i := range statuses {
		statuses[i] = CampaignTaskPending
	}
	return campaignState{
		parentID:      parentID,
		parentTitle:   parentTitle,
		tasks:         tasks,
		taskStatuses:  statuses,
		taskDurations: make([]time.Duration, len(tasks)),
		taskReports:   make(map[string][]PhaseReport),
		currentIdx:    -1,
		pipeline:      newPipelineState(nil),
	}
}

// Update processes messages for the campaign state.
func (cs campaignState) Update(msg tea.Msg) (campaignState, tea.Cmd) {
	switch msg := msg.(type) {
	case CampaignTaskStartMsg:
		return cs.handleTaskStart(msg), nil
	case CampaignTaskDoneMsg:
		return cs.handleTaskDone(msg), nil
	case PhaseUpdateMsg, elapsedTickMsg, spinner.TickMsg:
		var cmd tea.Cmd
		cs.pipeline, cmd = cs.pipeline.Update(msg)
		return cs, cmd
	case tea.KeyMsg:
		return cs.handleKey(msg), nil
	}
	return cs, nil
}

func (cs campaignState) handleKey(msg tea.KeyMsg) campaignState {
	if len(cs.tasks) == 0 {
		return cs
	}
	switch msg.String() {
	case "up", "k":
		cs.selectedIdx--
		if cs.selectedIdx < 0 {
			cs.selectedIdx = len(cs.tasks) - 1
		}
	case "down", "j":
		cs.selectedIdx++
		if cs.selectedIdx >= len(cs.tasks) {
			cs.selectedIdx = 0
		}
	}
	return cs
}

func (cs campaignState) handleTaskStart(msg CampaignTaskStartMsg) campaignState {
	cs.currentIdx = msg.Index
	if msg.Index >= 0 && msg.Index < len(cs.taskStatuses) {
		cs.taskStatuses[msg.Index] = CampaignTaskRunning
	}
	cs.pipeline = newPipelineState(nil)
	return cs
}

func (cs campaignState) handleTaskDone(msg CampaignTaskDoneMsg) campaignState {
	if msg.Index >= 0 && msg.Index < len(cs.taskStatuses) {
		if msg.Success {
			cs.taskStatuses[msg.Index] = CampaignTaskPassed
			cs.completed++
		} else {
			cs.taskStatuses[msg.Index] = CampaignTaskFailed
			cs.failed++
		}
		cs.taskDurations[msg.Index] = msg.Duration
	}
	if len(msg.PhaseReports) > 0 {
		cs.taskReports[msg.BeadID] = msg.PhaseReports
	}
	return cs
}

// View renders the campaign task queue for the given dimensions.
func (cs campaignState) View(width, height int) string {
	if len(cs.tasks) == 0 {
		return "No tasks"
	}

	var b strings.Builder

	// Header line.
	done := cs.completed + cs.failed
	fmt.Fprintf(&b, "%s  %s  %d/%d", cs.parentID, cs.parentTitle, done, len(cs.tasks))

	// Task queue.
	for i, task := range cs.tasks {
		b.WriteByte('\n')
		status := cs.taskStatuses[i]

		// Cursor marker on selected task.
		if i == cs.selectedIdx {
			b.WriteString(CursorMarker)
		} else {
			b.WriteString("  ")
		}

		indicator := cs.taskIndicator(status)
		fmt.Fprintf(&b, "%s %s", indicator, task.Title)

		if cs.taskDurations[i] > 0 {
			fmt.Fprintf(&b, " %s", pipeDurationStyle.Render(fmt.Sprintf("%.1fs", cs.taskDurations[i].Seconds())))
		}

		// Running task: show indented live phases below.
		if i == cs.currentIdx && status == CampaignTaskRunning && len(cs.pipeline.phases) > 0 {
			for _, phase := range cs.pipeline.phases {
				b.WriteByte('\n')
				pInd := pipeIndicator(phase.Status, cs.pipeline.spinner.View())
				pName := pipePhaseName(phase.Status, phase.Name)
				fmt.Fprintf(&b, "      %s %s", pInd, pName)
				if phase.Status == PhaseRunning && !cs.pipeline.phaseStartedAt.IsZero() && !cs.pipeline.aborting {
					elapsed := int(time.Since(cs.pipeline.phaseStartedAt).Seconds())
					fmt.Fprintf(&b, " %s", pipeDurationStyle.Render(fmt.Sprintf("(%ds)", elapsed)))
				}
				if phase.Duration > 0 {
					fmt.Fprintf(&b, " %s", pipeDurationStyle.Render(fmt.Sprintf("%.1fs", phase.Duration.Seconds())))
				}
			}
		}

		// Selected completed/failed task: expand stored phase reports below.
		if i == cs.selectedIdx && (status == CampaignTaskPassed || status == CampaignTaskFailed) {
			if reports, ok := cs.taskReports[task.BeadID]; ok {
				for _, r := range reports {
					b.WriteByte('\n')
					ind := pipeIndicator(r.Status, "")
					fmt.Fprintf(&b, "      %s %s", ind, r.PhaseName)
					if r.Duration > 0 {
						fmt.Fprintf(&b, " %s", pipeDurationStyle.Render(fmt.Sprintf("%.1fs", r.Duration.Seconds())))
					}
				}
			}
		}
	}

	return b.String()
}

func (cs campaignState) taskIndicator(status CampaignTaskStatus) string {
	switch status {
	case CampaignTaskPending:
		return pipePendingStyle.Render(SymbolPending)
	case CampaignTaskRunning:
		return cs.pipeline.spinner.View()
	case CampaignTaskPassed:
		return pipePassedStyle.Render(SymbolCheck)
	case CampaignTaskFailed:
		return pipeFailedStyle.Render(SymbolCross)
	case CampaignTaskSkipped:
		return pipeSkippedStyle.Render(SymbolSkipped)
	default:
		return "?"
	}
}

// ViewReport renders the right-pane content for the selected task.
// For the running task, it delegates to the live pipeline. For completed
// tasks, it renders stored phase reports. For pending tasks, returns empty.
func (cs campaignState) ViewReport(width, height int) string {
	if len(cs.tasks) == 0 || cs.selectedIdx < 0 || cs.selectedIdx >= len(cs.tasks) {
		return ""
	}

	status := cs.taskStatuses[cs.selectedIdx]

	// Running task: delegate to live pipeline.
	if cs.selectedIdx == cs.currentIdx && status == CampaignTaskRunning {
		return cs.pipeline.ViewReport(width, height)
	}

	// Completed/failed task: render stored phase reports.
	if status == CampaignTaskPassed || status == CampaignTaskFailed {
		task := cs.tasks[cs.selectedIdx]
		reports, ok := cs.taskReports[task.BeadID]
		if !ok || len(reports) == 0 {
			return ""
		}
		return cs.formatTaskReport(task, reports)
	}

	return ""
}

// formatTaskReport renders the stored phase reports for a completed task.
func (cs campaignState) formatTaskReport(task CampaignTaskInfo, reports []PhaseReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n", task.Title)

	for _, r := range reports {
		var statusText string
		var renderedStatus string
		switch r.Status {
		case PhaseFailed, PhaseError:
			statusText = "Failed"
			renderedStatus = pipeFailedStyle.Render(statusText)
		case PhaseSkipped:
			statusText = "Skipped"
			renderedStatus = pipeSkippedStyle.Render(statusText)
		default:
			statusText = "Passed"
			renderedStatus = pipePassedStyle.Render(statusText)
		}
		fmt.Fprintf(&b, "\n%s  %s", r.PhaseName, renderedStatus)
		if r.Duration > 0 {
			fmt.Fprintf(&b, "  %s", pipeDurationStyle.Render(fmt.Sprintf("%.1fs", r.Duration.Seconds())))
		}
		if r.Summary != "" {
			fmt.Fprintf(&b, "\n  %s", r.Summary)
		}
	}

	return b.String()
}
