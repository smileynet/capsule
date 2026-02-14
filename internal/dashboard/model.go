package dashboard

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/spinner"
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
	browse   browseState
	pipeline pipelineState
	lister   BeadLister

	resolver    BeadResolver
	cache       *Cache
	detailID    string // ID currently displayed in right pane
	resolvingID string // ID of the bead currently being resolved ("" = idle)
	resolveErr  error  // last resolve error (nil on success)
}

// NewModel creates a dashboard Model in browse mode with left-pane focus.
// If a BeadLister is provided, Init will fire an async fetch for the bead list.
func NewModel(opts ...ModelOption) Model {
	m := Model{
		mode:     ModeBrowse,
		focus:    PaneLeft,
		viewport: viewport.New(0, 0),
		help:     help.New(),
		browse:   newBrowseState(),
		cache:    NewCache(),
	}
	for _, o := range opts {
		o(&m)
	}
	return m
}

// ModelOption configures a Model during construction.
type ModelOption func(*Model)

// WithBeadLister sets the BeadLister used to fetch the bead list.
func WithBeadLister(l BeadLister) ModelOption {
	return func(m *Model) { m.lister = l }
}

// WithBeadResolver sets the BeadResolver used to fetch bead details.
func WithBeadResolver(r BeadResolver) ModelOption {
	return func(m *Model) { m.resolver = r }
}

// resolveBeadCmd returns a tea.Cmd that calls resolver.Resolve(id)
// and wraps the result in a BeadResolvedMsg.
func resolveBeadCmd(resolver BeadResolver, id string) tea.Cmd {
	return func() tea.Msg {
		detail, err := resolver.Resolve(id)
		return BeadResolvedMsg{ID: id, Detail: detail, Err: err}
	}
}

// formatBeadDetail renders a BeadDetail as plain text for the viewport.
func formatBeadDetail(d BeadDetail) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s  %s  %s\n", d.ID, PriorityBadge(d.Priority), d.Type)
	b.WriteString(d.Title)
	b.WriteByte('\n')

	if d.EpicID != "" {
		fmt.Fprintf(&b, "\nEpic: %s — %s", d.EpicID, d.EpicTitle)
	}
	if d.FeatureID != "" {
		fmt.Fprintf(&b, "\nFeature: %s — %s", d.FeatureID, d.FeatureTitle)
	}

	if d.Description != "" {
		fmt.Fprintf(&b, "\n\n%s", d.Description)
	}

	if d.Acceptance != "" {
		fmt.Fprintf(&b, "\n\nAcceptance:\n%s", d.Acceptance)
	}

	return b.String()
}

// Init returns the initial command. If a BeadLister was provided,
// it fires an async fetch for the bead list.
func (m Model) Init() tea.Cmd {
	if m.lister != nil {
		return initBrowse(m.lister)
	}
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
		m.viewport.Width = max(rightWidth-borderChrome, 0)
		m.viewport.Height = m.contentHeight()
		return m, nil

	case BeadListMsg:
		m.browse, _ = m.browse.Update(msg)
		return m.maybeResolve()

	case BeadResolvedMsg:
		isCurrent := msg.ID == m.resolvingID
		if isCurrent {
			m.resolvingID = ""
		}
		if msg.Err != nil {
			if isCurrent {
				m.resolveErr = msg.Err
			}
			return m, nil
		}
		m.cache.Set(msg.ID, &msg.Detail)
		if isCurrent {
			m.resolveErr = nil
			m.viewport.SetContent(formatBeadDetail(msg.Detail))
			m.viewport.GotoTop()
		}
		return m, nil

	case RefreshBeadsMsg:
		m.cache.Invalidate()
		if m.lister != nil {
			return m, initBrowse(m.lister)
		}
		return m, nil

	case PhaseUpdateMsg:
		var cmd tea.Cmd
		m.pipeline, cmd = m.pipeline.Update(msg)
		return m, cmd

	case spinner.TickMsg:
		if m.mode == ModePipeline {
			var cmd tea.Cmd
			m.pipeline, cmd = m.pipeline.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

// handleKey processes key messages with global and mode-specific routing.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global keys.
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

	// Mode-specific keys.
	switch {
	case m.mode == ModeBrowse && m.focus == PaneLeft:
		var cmd tea.Cmd
		m.browse, cmd = m.browse.Update(msg)
		m, resolveCmd := m.maybeResolve()
		return m, tea.Batch(cmd, resolveCmd)

	case m.mode == ModeBrowse && m.focus == PaneRight:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	case m.mode == ModePipeline && m.focus == PaneLeft:
		var cmd tea.Cmd
		m.pipeline, cmd = m.pipeline.Update(msg)
		return m, cmd

	case m.mode == ModePipeline && m.focus == PaneRight:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

// maybeResolve checks if the selected bead changed and triggers a resolve
// if needed. On cache hit, the viewport is updated immediately. On cache miss,
// an async resolveBeadCmd is returned.
func (m Model) maybeResolve() (Model, tea.Cmd) {
	selected := m.browse.SelectedID()
	if selected == "" || selected == m.detailID {
		return m, nil
	}
	m.detailID = selected

	if detail, ok := m.cache.Get(selected); ok {
		m.resolvingID = ""
		m.resolveErr = nil
		m.viewport.SetContent(formatBeadDetail(*detail))
		m.viewport.GotoTop()
		return m, nil
	}

	if m.resolver != nil {
		m.resolvingID = selected
		m.resolveErr = nil
		return m, resolveBeadCmd(m.resolver, selected)
	}
	return m, nil
}

// contentHeight returns the usable height for pane content,
// accounting for border chrome and the help bar.
func (m Model) contentHeight() int {
	return max(m.height-borderChrome-helpBarHeight, 1)
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
		leftWidth, _ := PaneWidths(m.width)
		return m.pipeline.View(leftWidth-borderChrome, m.contentHeight())
	case ModeSummary:
		return "Summary"
	default:
		leftWidth, _ := PaneWidths(m.width)
		return m.browse.View(leftWidth-borderChrome, m.contentHeight())
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
		return m.viewBrowseDetail()
	}
}

// viewBrowseDetail renders the right pane in browse mode:
// loading indicator, error message, or resolved detail viewport.
func (m Model) viewBrowseDetail() string {
	if m.resolvingID != "" {
		return fmt.Sprintf("Loading %s...", m.resolvingID)
	}
	if m.resolveErr != nil {
		return fmt.Sprintf("Error: %s\n\nPress r to retry", m.resolveErr)
	}
	if m.detailID == "" {
		return "Select a bead"
	}
	return m.viewport.View()
}
