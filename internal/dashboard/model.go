package dashboard

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// helpBarHeight is the number of lines reserved for the help bar at the bottom.
const helpBarHeight = 1

// borderChrome is the number of lines consumed by top + bottom borders.
const borderChrome = 2

// Model is the root Bubble Tea model for the dashboard TUI.
// It manages a two-pane layout with mode-based routing and focus management.
type Model struct {
	mode     Mode
	focus    Focus
	width    int
	height   int
	viewport viewport.Model
	help     help.Model
}

// NewModel creates a dashboard Model in browse mode with left-pane focus.
func NewModel() Model {
	return Model{
		mode:     ModeBrowse,
		focus:    PaneLeft,
		viewport: viewport.New(0, 0),
		help:     help.New(),
	}
}

// Init returns the initial command.
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles incoming messages with mode-based routing.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		_, rightWidth := PaneWidths(msg.Width)
		vpWidth := rightWidth - borderChrome
		if vpWidth < 0 {
			vpWidth = 0
		}
		m.viewport.Width = vpWidth
		m.viewport.Height = m.contentHeight()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

// handleKey processes key messages with global and mode-specific routing.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		if m.mode == ModeBrowse {
			return m, tea.Quit
		}
	case "tab":
		if m.focus == PaneLeft {
			m.focus = PaneRight
		} else {
			m.focus = PaneLeft
		}
		return m, nil
	}

	return m, nil
}

// contentHeight returns the usable height for pane content,
// accounting for border chrome and the help bar.
func (m Model) contentHeight() int {
	h := m.height - borderChrome - helpBarHeight
	if h < 1 {
		return 1
	}
	return h
}

// View renders the two-pane layout with help bar.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	leftWidth, rightWidth := PaneWidths(m.width)
	contentHeight := m.contentHeight()

	var leftStyle, rightStyle lipgloss.Style
	if m.focus == PaneLeft {
		leftStyle = FocusedBorder()
		rightStyle = UnfocusedBorder()
	} else {
		leftStyle = UnfocusedBorder()
		rightStyle = FocusedBorder()
	}

	leftStyle = leftStyle.
		Width(leftWidth - borderChrome).
		Height(contentHeight)
	rightStyle = rightStyle.
		Width(rightWidth - borderChrome).
		Height(contentHeight)

	leftPane := leftStyle.Render(m.viewLeft())
	rightPane := rightStyle.Render(m.viewRight())
	panes := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, rightPane)
	helpView := m.help.View(HelpBindings(m.mode))

	return lipgloss.JoinVertical(lipgloss.Left, panes, helpView)
}

// viewLeft renders the left pane content based on mode.
func (m Model) viewLeft() string {
	switch m.mode {
	case ModePipeline:
		return "Pipeline phases"
	case ModeSummary:
		return "Summary"
	default:
		return "Beads"
	}
}

// viewRight renders the right pane content based on mode.
func (m Model) viewRight() string {
	switch m.mode {
	case ModePipeline:
		return "Phase report"
	case ModeSummary:
		return "Result details"
	default:
		return "Detail"
	}
}
