package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// detailHeaderHeight is the number of lines reserved for the phase list and
// chrome above the detail viewport. The viewport gets the remaining height.
const detailHeaderHeight = 6

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
	detailStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
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
	phases        []PhaseState
	spinner       spinner.Model
	currentIdx    int // Tracks active phase index for future scroll/focus support.
	done          bool
	aborting      bool
	err           error
	cancelFunc    context.CancelFunc // Called on first abort keypress; nil means immediate quit.
	startTime     time.Time          // Records model creation for future elapsed-time display.
	width         int                // Terminal width from WindowSizeMsg; 0 means not yet received.
	height        int                // Terminal height from WindowSizeMsg; 0 means not yet received.
	detailVisible bool               // Whether the detail panel is shown.
	detailContent string             // Raw output content for the detail panel.
	viewport      viewport.Model     // Scrollable viewport for the detail panel.
}

// ModelOption configures the Model.
type ModelOption func(*Model)

// WithCancelFunc sets a function called on the first abort keypress (q or Ctrl+C).
// When set, the first press triggers graceful abort; a second press forces immediate exit.
// When nil (default), any abort keypress immediately quits the program.
func WithCancelFunc(fn context.CancelFunc) ModelOption {
	return func(m *Model) {
		m.cancelFunc = fn
	}
}

// StatusUpdateMsg bridges orchestrator status updates to the TUI.
type StatusUpdateMsg struct {
	Phase        string
	Status       PhaseStatus
	Attempt      int
	MaxRetry     int
	Duration     time.Duration
	Progress     string   // Human-readable progress (e.g. "2/6").
	Summary      string   // Phase summary text.
	FilesChanged []string // Files modified in this phase.
	Feedback     string   // Feedback for retries (shown on failure).
}

func (StatusUpdateMsg) isDisplayEvent() {}

// PipelineDoneMsg signals that the pipeline completed successfully.
type PipelineDoneMsg struct{}

func (PipelineDoneMsg) isDisplayEvent() {}

// PipelineErrorMsg signals that the pipeline failed with an error.
type PipelineErrorMsg struct {
	Err error
}

func (PipelineErrorMsg) isDisplayEvent() {}

// OutputMsg delivers phase output content for the detail view.
type OutputMsg struct {
	Content string
}

func (OutputMsg) isDisplayEvent() {}

// NewModel creates a Model initialized with the given phase names.
func NewModel(phaseNames []string, opts ...ModelOption) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot

	phases := make([]PhaseState, len(phaseNames))
	for i, name := range phaseNames {
		phases[i] = PhaseState{Name: name, Status: StatusPending}
	}

	m := Model{
		phases:    phases,
		spinner:   s,
		startTime: time.Now(),
		viewport:  viewport.New(0, 0),
	}
	for _, opt := range opts {
		opt(&m)
	}
	return m
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

	case OutputMsg:
		m.detailContent = msg.Content
		m.viewport.SetContent(msg.Content)
		m.viewport.GotoBottom()
		return m, nil

	case PipelineDoneMsg:
		m.done = true
		m.aborting = false
		return m, tea.Quit

	case PipelineErrorMsg:
		m.done = true
		m.err = msg.Err
		return m, tea.Quit

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			if m.done {
				return m, nil
			}
			if m.aborting || m.cancelFunc == nil {
				m.done = true
				return m, tea.Quit
			}
			m.aborting = true
			m.cancelFunc()
			return m, nil
		case "d":
			if !m.done {
				m.detailVisible = !m.detailVisible
			}
			return m, nil
		}
		// Forward remaining keys to viewport when detail is visible.
		if m.detailVisible {
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = max(msg.Height-detailHeaderHeight, 1)
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

	if m.aborting && !m.done {
		s += "\n" + failedStyle.Render("  Aborting...") + " (press again to force quit)\n"
	}

	if m.detailVisible && !m.done {
		s += m.renderDetail()
	}

	if m.done {
		s += m.renderFooter()
	}

	return s
}

// renderDetail returns the detail panel with viewport content.
func (m Model) renderDetail() string {
	header := detailStyle.Render("\n  ── Detail (d to close) ──") + "\n"
	if m.detailContent == "" {
		return header + detailStyle.Render("  No output yet") + "\n"
	}
	return header + m.viewport.View() + "\n"
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
