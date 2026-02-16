package dashboard

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// phaseEntry tracks the display state of a single pipeline phase.
type phaseEntry struct {
	Name     string
	Status   PhaseStatus
	Attempt  int
	MaxRetry int
	Duration time.Duration
}

// pipelineState manages the phase list, cursor, reports, and auto-follow for pipeline mode.
type pipelineState struct {
	phases     []phaseEntry
	cursor     int
	autoFollow bool
	spinner    spinner.Model
	running    bool
	reports    map[string]*PhaseReport
	aborting   bool
	beadID     string // Bead ID shown in header (optional).
	beadTitle  string // Bead title shown in header (optional).
}

// newPipelineState creates a pipelineState for the given phase names.
func newPipelineState(phaseNames []string) pipelineState {
	s := spinner.New()
	s.Spinner = spinner.Dot

	phases := make([]phaseEntry, len(phaseNames))
	for i, name := range phaseNames {
		phases[i] = phaseEntry{Name: name, Status: PhasePending}
	}
	return pipelineState{
		phases:     phases,
		autoFollow: true,
		spinner:    s,
		reports:    make(map[string]*PhaseReport),
	}
}

// Update processes messages for the pipeline state.
func (ps pipelineState) Update(msg tea.Msg) (pipelineState, tea.Cmd) {
	switch msg := msg.(type) {
	case PhaseUpdateMsg:
		return ps.handlePhaseUpdate(msg), nil
	case tea.KeyMsg:
		return ps.handleKey(msg), nil
	case spinner.TickMsg:
		var cmd tea.Cmd
		ps.spinner, cmd = ps.spinner.Update(msg)
		return ps, cmd
	}
	return ps, nil
}

func (ps pipelineState) handlePhaseUpdate(msg PhaseUpdateMsg) pipelineState {
	for i := range ps.phases {
		if ps.phases[i].Name == msg.Phase {
			ps.phases[i].Status = msg.Status
			if msg.Attempt > 0 {
				ps.phases[i].Attempt = msg.Attempt
			}
			if msg.MaxRetry > 0 {
				ps.phases[i].MaxRetry = msg.MaxRetry
			}
			if msg.Duration > 0 {
				ps.phases[i].Duration = msg.Duration
			}
			switch msg.Status {
			case PhaseRunning:
				ps.running = true
				if ps.autoFollow {
					ps.cursor = i
				}
			case PhasePassed, PhaseFailed, PhaseError:
				ps.reports[msg.Phase] = &PhaseReport{
					PhaseName:    msg.Phase,
					Status:       msg.Status,
					Summary:      msg.Summary,
					Feedback:     msg.Feedback,
					FilesChanged: msg.FilesChanged,
					Duration:     msg.Duration,
				}
			}
			break
		}
	}
	return ps
}

func (ps pipelineState) handleKey(msg tea.KeyMsg) pipelineState {
	switch msg.String() {
	case "up", "k":
		if len(ps.phases) > 0 {
			ps.autoFollow = false
			ps.cursor--
			if ps.cursor < 0 {
				ps.cursor = len(ps.phases) - 1
			}
		}
	case "down", "j":
		if len(ps.phases) > 0 {
			ps.autoFollow = false
			ps.cursor++
			if ps.cursor >= len(ps.phases) {
				ps.cursor = 0
			}
		}
	}
	return ps
}

// Status indicator styles (reimplemented from tui.Model to avoid coupling).
var (
	pipePassedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	pipeFailedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	pipeRunningStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	pipePendingStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	pipeSkippedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	pipeDurationStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	pipeRetryStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	pipeHeaderStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
)

func pipeIndicator(status PhaseStatus, spinnerView string) string {
	switch status {
	case PhasePending:
		return pipePendingStyle.Render("○")
	case PhaseRunning:
		return spinnerView
	case PhasePassed:
		return pipePassedStyle.Render("✓")
	case PhaseFailed, PhaseError:
		return pipeFailedStyle.Render("✗")
	case PhaseSkipped:
		return pipeSkippedStyle.Render("–")
	default:
		return "?"
	}
}

func pipePhaseName(status PhaseStatus, name string) string {
	switch status {
	case PhasePending:
		return pipePendingStyle.Render(name)
	case PhaseRunning:
		return pipeRunningStyle.Render(name)
	default:
		return name
	}
}

// View renders the pipeline phase list for the given dimensions.
func (ps pipelineState) View(width, height int) string {
	if len(ps.phases) == 0 {
		return "No phases"
	}

	var b strings.Builder

	// Bead header: muted ID + title line above the phase list.
	if ps.beadID != "" {
		b.WriteString(pipeHeaderStyle.Render(ps.beadID + "  " + ps.beadTitle))
		b.WriteByte('\n')
	}

	for i, phase := range ps.phases {
		if i > 0 {
			b.WriteByte('\n')
		}
		if i == ps.cursor {
			b.WriteString(CursorMarker)
		} else {
			b.WriteString("  ")
		}

		var indicator, name string
		if phase.Status == PhaseRunning && ps.aborting {
			indicator = pipeFailedStyle.Render("⚠")
			name = pipeRunningStyle.Render(phase.Name + " Aborting...")
		} else {
			indicator = pipeIndicator(phase.Status, ps.spinner.View())
			name = pipePhaseName(phase.Status, phase.Name)
		}
		fmt.Fprintf(&b, "%s %s", indicator, name)

		if phase.Attempt > 1 {
			fmt.Fprintf(&b, " %s", pipeRetryStyle.Render(fmt.Sprintf("(%d/%d)", phase.Attempt, phase.MaxRetry)))
		}

		if phase.Duration > 0 {
			fmt.Fprintf(&b, " %s", pipeDurationStyle.Render(fmt.Sprintf("%.1fs", phase.Duration.Seconds())))
		}
	}
	return b.String()
}

// ViewReport renders the right-pane content for the currently selected phase.
func (ps pipelineState) ViewReport(width, height int) string {
	if len(ps.phases) == 0 || ps.cursor < 0 || ps.cursor >= len(ps.phases) {
		return ""
	}

	phase := ps.phases[ps.cursor]

	switch phase.Status {
	case PhasePending:
		return pipePendingStyle.Render("Waiting...")

	case PhaseRunning:
		var b strings.Builder
		if ps.aborting {
			fmt.Fprintf(&b, "%s  %s\n", pipeRunningStyle.Render(phase.Name), pipeFailedStyle.Render("Aborting"))
			fmt.Fprintf(&b, "\n%s %s", pipeFailedStyle.Render("⚠"), pipeFailedStyle.Render("Waiting for cleanup..."))
		} else {
			fmt.Fprintf(&b, "%s  %s\n", pipeRunningStyle.Render(phase.Name), pipeRunningStyle.Render("Running"))
			fmt.Fprintf(&b, "\n%s %s", ps.spinner.View(), pipeRunningStyle.Render("In progress..."))
		}
		return b.String()

	case PhaseSkipped:
		return pipeSkippedStyle.Render("Skipped")

	default:
		report := ps.reports[phase.Name]
		if report == nil {
			return ""
		}
		return ps.formatReport(report)
	}
}

func (ps pipelineState) formatReport(r *PhaseReport) string {
	var b strings.Builder

	// Header: phase name + status.
	statusText := "Passed"
	statusStyle := pipePassedStyle
	if r.Status == PhaseFailed || r.Status == PhaseError {
		statusText = "Failed"
		statusStyle = pipeFailedStyle
	}
	fmt.Fprintf(&b, "%s  %s\n", r.PhaseName, statusStyle.Render(statusText))

	// Duration.
	if r.Duration > 0 {
		fmt.Fprintf(&b, "\n%s", pipeDurationStyle.Render(fmt.Sprintf("Duration: %.1fs", r.Duration.Seconds())))
	}

	// Summary.
	if r.Summary != "" {
		fmt.Fprintf(&b, "\n\n%s", r.Summary)
	}

	// Files changed.
	if len(r.FilesChanged) > 0 {
		b.WriteString("\n\nFiles changed:")
		for _, f := range r.FilesChanged {
			fmt.Fprintf(&b, "\n  %s", f)
		}
	}

	// Feedback (typically present for failed/error phases).
	if r.Feedback != "" {
		fmt.Fprintf(&b, "\n\nFeedback:\n%s", r.Feedback)
	}

	return b.String()
}

// SelectedPhase returns the name of the phase at the current cursor position,
// or "" if the list is empty.
func (ps pipelineState) SelectedPhase() string {
	if len(ps.phases) == 0 || ps.cursor < 0 || ps.cursor >= len(ps.phases) {
		return ""
	}
	return ps.phases[ps.cursor].Name
}
