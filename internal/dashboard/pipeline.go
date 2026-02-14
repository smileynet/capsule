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

// pipelineState manages the phase list, cursor, and auto-follow for pipeline mode.
type pipelineState struct {
	phases     []phaseEntry
	cursor     int
	autoFollow bool
	spinner    spinner.Model
	running    bool
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
			if msg.Status == PhaseRunning {
				ps.running = true
				if ps.autoFollow {
					ps.cursor = i
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
)

func pipeIndicator(status PhaseStatus, spinnerView string) string {
	switch status {
	case PhasePending:
		return pipePendingStyle.Render("○")
	case PhaseRunning:
		return spinnerView
	case PhasePassed:
		return pipePassedStyle.Render("✓")
	case PhaseFailed:
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
	for i, phase := range ps.phases {
		if i > 0 {
			b.WriteByte('\n')
		}
		if i == ps.cursor {
			b.WriteString(CursorMarker)
		} else {
			b.WriteString("  ")
		}

		indicator := pipeIndicator(phase.Status, ps.spinner.View())
		name := pipePhaseName(phase.Status, phase.Name)
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

// SelectedPhase returns the name of the phase at the current cursor position,
// or "" if the list is empty.
func (ps pipelineState) SelectedPhase() string {
	if len(ps.phases) == 0 || ps.cursor < 0 || ps.cursor >= len(ps.phases) {
		return ""
	}
	return ps.phases[ps.cursor].Name
}
