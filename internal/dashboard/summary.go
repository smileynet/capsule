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
	var passed int
	var totalDuration time.Duration
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

// returnToBrowse transitions from summary mode back to browse mode,
// invalidating the bead cache and triggering a refresh.
func (m Model) returnToBrowse() (Model, tea.Cmd) {
	m.mode = ModeBrowse
	m.focus = PaneLeft
	m.cache.Invalidate()
	if m.lister != nil {
		return m, initBrowse(m.lister)
	}
	return m, nil
}
