package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// PhaseStatus represents the current state of a phase in the TUI.
// Values intentionally mirror orchestrator.PhaseStatus for straightforward bridging
// via StatusUpdateMsg, keeping the tui package decoupled from orchestrator.
type PhaseStatus string

const (
	StatusPending PhaseStatus = "pending"
	StatusRunning PhaseStatus = "running"
	StatusPassed  PhaseStatus = "passed"
	StatusFailed  PhaseStatus = "failed"
	StatusError   PhaseStatus = "error"
	StatusSkipped PhaseStatus = "skipped"
)

// Lipgloss styles for phase status display.
var (
	passedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	failedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	runningStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	pendingStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	skippedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	durationStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	retryStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
)

// PhaseState tracks the display state of a single pipeline phase.
type PhaseState struct {
	Name     string
	Status   PhaseStatus
	Attempt  int
	MaxRetry int
	Duration time.Duration
}

// Model is the Bubble Tea model for pipeline phase status display.
type Model struct {
	phases     []PhaseState
	spinner    spinner.Model
	currentIdx int // Tracks active phase index for future scroll/focus support.
	done       bool
	err        error
	startTime  time.Time // Records model creation for future elapsed-time display.
	width      int       // Terminal width from WindowSizeMsg; 0 means not yet received.
}

// StatusUpdateMsg bridges orchestrator status updates to the TUI.
type StatusUpdateMsg struct {
	Phase    string
	Status   PhaseStatus
	Attempt  int
	MaxRetry int
	Duration time.Duration
}

// PipelineDoneMsg signals that the pipeline completed successfully.
type PipelineDoneMsg struct{}

// PipelineErrorMsg signals that the pipeline failed with an error.
type PipelineErrorMsg struct {
	Err error
}

// NewModel creates a Model initialized with the given phase names.
func NewModel(phaseNames []string) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot

	phases := make([]PhaseState, len(phaseNames))
	for i, name := range phaseNames {
		phases[i] = PhaseState{Name: name, Status: StatusPending}
	}

	return Model{
		phases:    phases,
		spinner:   s,
		startTime: time.Now(),
	}
}

// Init starts the spinner tick.
func (m Model) Init() tea.Cmd {
	return m.spinner.Tick
}

// Update handles incoming messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case StatusUpdateMsg:
		for i := range m.phases {
			if m.phases[i].Name == msg.Phase {
				m.phases[i].Status = msg.Status
				if msg.Attempt > 0 {
					m.phases[i].Attempt = msg.Attempt
				}
				if msg.MaxRetry > 0 {
					m.phases[i].MaxRetry = msg.MaxRetry
				}
				if msg.Duration > 0 {
					m.phases[i].Duration = msg.Duration
				}
				if msg.Status == StatusRunning {
					m.currentIdx = i
				}
				break
			}
		}
		return m, nil

	case PipelineDoneMsg:
		m.done = true
		return m, tea.Quit

	case PipelineErrorMsg:
		m.done = true
		m.err = msg.Err
		return m, tea.Quit

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.done = true
			return m, tea.Quit
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// View renders the phase list with status indicators.
func (m Model) View() string {
	var s string

	for _, phase := range m.phases {
		indicator := styledIndicator(phase.Status, m.spinner.View())
		name := styledPhaseName(phase.Status, phase.Name)
		line := fmt.Sprintf("  %s %s", indicator, name)

		if phase.Attempt > 1 {
			line += retryStyle.Render(fmt.Sprintf(" (%d/%d)", phase.Attempt, phase.MaxRetry))
		}

		if phase.Duration > 0 {
			line += durationStyle.Render(fmt.Sprintf(" %.1fs", phase.Duration.Seconds()))
		}

		s += line + "\n"
	}

	if m.done {
		s += m.renderFooter()
	}

	return s
}

// renderFooter returns the summary footer for a completed pipeline.
func (m Model) renderFooter() string {
	passed, total := m.phaseCounts()
	totalDur := m.totalDuration()

	var footer string
	if m.err != nil {
		footer = fmt.Sprintf("\n  %s %d/%d passed",
			failedStyle.Render("✗"), passed, total)
		if totalDur > 0 {
			footer += durationStyle.Render(fmt.Sprintf(" in %.1fs", totalDur.Seconds()))
		}
		footer += fmt.Sprintf("\n  Error: %s\n", m.err)
	} else {
		footer = fmt.Sprintf("\n  %s %d/%d passed",
			passedStyle.Render("✓"), passed, total)
		if totalDur > 0 {
			footer += durationStyle.Render(fmt.Sprintf(" in %.1fs", totalDur.Seconds()))
		}
		footer += "\n"
	}

	return footer
}

// phaseCounts returns the number of passed phases and total phases.
func (m Model) phaseCounts() (passed, total int) {
	total = len(m.phases)
	for _, p := range m.phases {
		if p.Status == StatusPassed {
			passed++
		}
	}
	return
}

// totalDuration sums reported phase durations.
func (m Model) totalDuration() time.Duration {
	var total time.Duration
	for _, p := range m.phases {
		total += p.Duration
	}
	return total
}

// styledIndicator returns the styled Unicode indicator for a phase status.
func styledIndicator(status PhaseStatus, spinnerView string) string {
	switch status {
	case StatusPending:
		return pendingStyle.Render("○")
	case StatusRunning:
		return spinnerView // Already styled by spinner.
	case StatusPassed:
		return passedStyle.Render("✓")
	case StatusFailed, StatusError:
		return failedStyle.Render("✗")
	case StatusSkipped:
		return skippedStyle.Render("–")
	default:
		return "?"
	}
}

// styledPhaseName applies the appropriate style to a phase name.
func styledPhaseName(status PhaseStatus, name string) string {
	switch status {
	case StatusPending:
		return pendingStyle.Render(name)
	case StatusRunning:
		return runningStyle.Render(name)
	default:
		return name
	}
}
