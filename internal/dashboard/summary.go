package dashboard

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// viewSummaryRight renders the right pane in summary mode:
// an overall result summary showing pass/fail status and phase count.
func (m Model) viewSummaryRight() string {
	var (
		passed        int
		totalDuration time.Duration
	)
	for _, p := range m.pipeline.phases {
		if p.Status == PhasePassed {
			passed++
		}
		totalDuration += p.Duration
	}
	total := len(m.pipeline.phases)

	var b strings.Builder

	success := m.pipelineErr == nil && (m.pipelineOutput == nil || m.pipelineOutput.Success)
	if success {
		fmt.Fprintf(&b, "%s  Pipeline Passed\n", pipePassedStyle.Render("✓"))
		fmt.Fprintf(&b, "\n%d/%d phases passed in %.1fs", passed, total, totalDuration.Seconds())
	} else {
		fmt.Fprintf(&b, "%s  Pipeline Failed\n", pipeFailedStyle.Render("✗"))
		if m.pipelineErr != nil {
			fmt.Fprintf(&b, "\nError: %s", m.pipelineErr)
		}
		fmt.Fprintf(&b, "\n\n%d/%d phases passed", passed, total)
	}

	return b.String()
}

// returnToBrowseAfterAbort transitions from pipeline mode to browse mode
// after an abort. Unlike returnToBrowse, it skips post-pipeline lifecycle
// since the pipeline was cancelled.
func (m Model) returnToBrowseAfterAbort() (Model, tea.Cmd) {
	m.mode = ModeBrowse
	m.focus = PaneLeft
	m.aborting = false
	m.dispatchedBeadID = ""
	m.cache.Invalidate()

	if m.lister != nil {
		return m, tea.Batch(initBrowse(m.lister), m.browseSpinner.Tick)
	}
	return m, nil
}

// returnToBrowseFromCampaign transitions from campaign summary to browse mode.
// Skips postPipeline since campaigns handle their own lifecycle.
func (m Model) returnToBrowseFromCampaign() (Model, tea.Cmd) {
	m.mode = ModeBrowse
	m.focus = PaneLeft
	m.cache.Invalidate()
	m.campaignDone = nil
	m.dispatchedBeadID = ""

	if m.lister != nil {
		return m, tea.Batch(initBrowse(m.lister), m.browseSpinner.Tick)
	}
	return m, nil
}

// viewCampaignSummaryRight renders the right pane in campaign summary mode.
func (m Model) viewCampaignSummaryRight() string {
	done := m.campaignDone
	if done == nil {
		return ""
	}

	var b strings.Builder

	if done.Failed == 0 {
		fmt.Fprintf(&b, "%s  Campaign Passed\n", pipePassedStyle.Render("✓"))
		fmt.Fprintf(&b, "\n%d/%d tasks passed", done.Passed, done.TotalTasks)
	} else {
		fmt.Fprintf(&b, "%s  Campaign Failed\n", pipeFailedStyle.Render("✗"))
		fmt.Fprintf(&b, "\n%d/%d tasks passed, %d failed", done.Passed, done.TotalTasks, done.Failed)
	}

	return b.String()
}

// returnToBrowse transitions from summary mode back to browse mode,
// invalidating the bead cache and triggering a refresh. If a post-pipeline
// function is configured, it fires in a background goroutine.
func (m Model) returnToBrowse() (Model, tea.Cmd) {
	m.mode = ModeBrowse
	m.focus = PaneLeft
	m.cache.Invalidate()

	var cmds []tea.Cmd

	// Fire post-pipeline lifecycle in background if configured.
	if m.postPipeline != nil && m.dispatchedBeadID != "" {
		beadID := m.dispatchedBeadID
		ppFn := m.postPipeline
		m.dispatchedBeadID = ""
		cmds = append(cmds, func() tea.Msg {
			err := ppFn(beadID)
			return PostPipelineDoneMsg{BeadID: beadID, Err: err}
		})
	}

	// Refresh bead list with spinner animation.
	if m.lister != nil {
		cmds = append(cmds, initBrowse(m.lister), m.browseSpinner.Tick)
	}

	return m, tea.Batch(cmds...)
}
