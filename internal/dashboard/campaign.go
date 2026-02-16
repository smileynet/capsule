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
	case PhaseUpdateMsg:
		var cmd tea.Cmd
		cs.pipeline, cmd = cs.pipeline.Update(msg)
		return cs, cmd
	case spinner.TickMsg:
		var cmd tea.Cmd
		cs.pipeline, cmd = cs.pipeline.Update(msg)
		return cs, cmd
	}
	return cs, nil
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

		indicator := cs.taskIndicator(status)
		fmt.Fprintf(&b, "  %s %s", indicator, task.Title)

		if cs.taskDurations[i] > 0 {
			fmt.Fprintf(&b, " %s", pipeDurationStyle.Render(fmt.Sprintf("%.1fs", cs.taskDurations[i].Seconds())))
		}

		// Running task: show indented phases below.
		if i == cs.currentIdx && status == CampaignTaskRunning && len(cs.pipeline.phases) > 0 {
			for _, phase := range cs.pipeline.phases {
				b.WriteByte('\n')
				pInd := pipeIndicator(phase.Status, cs.pipeline.spinner.View())
				pName := pipePhaseName(phase.Status, phase.Name)
				fmt.Fprintf(&b, "      %s %s", pInd, pName)
				if phase.Duration > 0 {
					fmt.Fprintf(&b, " %s", pipeDurationStyle.Render(fmt.Sprintf("%.1fs", phase.Duration.Seconds())))
				}
			}
		}
	}

	return b.String()
}

func (cs campaignState) taskIndicator(status CampaignTaskStatus) string {
	switch status {
	case CampaignTaskPending:
		return pipePendingStyle.Render("○")
	case CampaignTaskRunning:
		return cs.pipeline.spinner.View()
	case CampaignTaskPassed:
		return pipePassedStyle.Render("✓")
	case CampaignTaskFailed:
		return pipeFailedStyle.Render("✗")
	case CampaignTaskSkipped:
		return pipeSkippedStyle.Render("–")
	default:
		return "?"
	}
}

// ViewReport delegates to the embedded pipelineState's ViewReport.
func (cs campaignState) ViewReport(width, height int) string {
	if cs.currentIdx < 0 {
		return ""
	}
	return cs.pipeline.ViewReport(width, height)
}
