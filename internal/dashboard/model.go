package dashboard

import (
	"context"
	"fmt"
	"strings"
	"time"

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

// archiveSeparator is the visual divider between bead detail and archived data.
const archiveSeparator = "───────────────────────────────"

// resolveDebounce is the delay before dispatching an async resolve
// after the cursor moves to a bead not in cache. This prevents visual
// thrash when the user scrolls quickly through the bead list.
const resolveDebounce = 150 * time.Millisecond

// Model is the root Bubble Tea model for the dashboard TUI.
// It manages a two-pane layout with mode-based routing and focus management.
type Model struct {
	mode          Mode
	focus         Focus
	width         int
	height        int
	viewport      viewport.Model
	help          help.Model
	browse        browseState
	browseSpinner spinner.Model
	pipeline      pipelineState
	lister        BeadLister

	resolver         BeadResolver
	cache            *Cache
	detailID         string // ID currently displayed in right pane
	resolvingID      string // ID of the bead currently being resolved ("" = idle)
	resolveErr       error  // last resolve error (nil on success)
	pendingResolveID string // ID awaiting debounce expiry ("" = no pending debounce)

	runner           PipelineRunner
	phaseNames       []string
	cancelPipeline   context.CancelFunc
	eventCh          <-chan tea.Msg
	pipelineOutput   *PipelineOutput
	pipelineErr      error
	postPipeline     PostPipelineFunc
	dispatchedBeadID string
	aborting         bool

	campaign       campaignState
	campaignRunner CampaignRunner
	campaignDone   *CampaignDoneMsg // set on CampaignDoneMsg or synthesized on channel close
	campaignErr    error            // set on CampaignErrorMsg from runner failure

	archive ArchiveReader
}

// newBrowseSpinner returns a spinner for browse mode loading states.
func newBrowseSpinner() spinner.Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	return s
}

// NewModel creates a dashboard Model in browse mode with left-pane focus.
// If a BeadLister is provided, Init will fire an async fetch for the bead list.
func NewModel(opts ...ModelOption) Model {
	m := Model{
		mode:          ModeBrowse,
		focus:         PaneLeft,
		viewport:      viewport.New(0, 0),
		help:          help.New(),
		browse:        newBrowseState(),
		browseSpinner: newBrowseSpinner(),
		cache:         NewCache(),
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

// WithPipelineRunner sets the PipelineRunner used to dispatch pipelines.
func WithPipelineRunner(r PipelineRunner) ModelOption {
	return func(m *Model) { m.runner = r }
}

// WithPhaseNames sets the phase names displayed in pipeline mode.
func WithPhaseNames(names []string) ModelOption {
	return func(m *Model) { m.phaseNames = names }
}

// WithPostPipelineFunc sets the function called after a pipeline completes
// and the user returns to browse mode. It runs in a background goroutine.
func WithPostPipelineFunc(fn PostPipelineFunc) ModelOption {
	return func(m *Model) { m.postPipeline = fn }
}

// WithCampaignRunner sets the CampaignRunner used to dispatch campaigns.
func WithCampaignRunner(r CampaignRunner) ModelOption {
	return func(m *Model) { m.campaignRunner = r }
}

// WithArchiveReader sets the ArchiveReader used to fetch archived pipeline
// results for closed beads.
func WithArchiveReader(ar ArchiveReader) ModelOption {
	return func(m *Model) { m.archive = ar }
}

// listenForEvents returns a tea.Cmd that reads one message from ch.
// On channel close, it returns channelClosedMsg. Returns nil if ch is nil.
func listenForEvents(ch <-chan tea.Msg) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return channelClosedMsg{}
		}
		return msg
	}
}

// dispatchPipeline runs a pipeline in the calling goroutine, bridging
// status events to ch via statusFn. It sends PipelineDoneMsg or
// PipelineErrorMsg on completion and closes ch when done.
func dispatchPipeline(ctx context.Context, runner PipelineRunner, input PipelineInput, ch chan<- tea.Msg) {
	defer close(ch)
	statusFn := func(msg PhaseUpdateMsg) {
		select {
		case ch <- msg:
		case <-ctx.Done():
		}
	}
	output, err := runner.RunPipeline(ctx, input, statusFn)
	if err != nil {
		select {
		case ch <- PipelineErrorMsg{Err: err}:
		case <-ctx.Done():
		}
		return
	}
	select {
	case ch <- PipelineDoneMsg{Output: output}:
	case <-ctx.Done():
	}
}

// dispatchCampaign runs a campaign in the calling goroutine, bridging
// status events to ch. It closes ch when done.
func dispatchCampaign(ctx context.Context, cr CampaignRunner, pr PipelineRunner, parentID string, ch chan<- tea.Msg) {
	defer close(ch)
	statusFn := func(msg tea.Msg) {
		select {
		case ch <- msg:
		case <-ctx.Done():
		}
	}
	var pipelineFn func(context.Context, PipelineInput, func(PhaseUpdateMsg)) (PipelineOutput, error)
	if pr != nil {
		pipelineFn = pr.RunPipeline
	}
	if err := cr.RunCampaign(ctx, parentID, statusFn, pipelineFn); err != nil {
		select {
		case ch <- CampaignErrorMsg{Err: err}:
		case <-ctx.Done():
		}
	}
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

// renderDetailContent formats a bead detail for the viewport. In closed mode
// with an archive reader, it appends archived summary and worklog data.
func (m Model) renderDetailContent(d BeadDetail) string {
	if !m.browse.showClosed || m.archive == nil {
		return formatBeadDetail(d)
	}
	summary, _ := m.archive.ReadSummary(d.ID)
	worklog, _ := m.archive.ReadWorklog(d.ID)
	return formatClosedBeadDetail(d, summary, worklog)
}

// formatClosedBeadDetail renders a closed bead's detail with archived summary
// and worklog below a separator. If both summary and worklog are empty, renders
// as a normal bead detail without a separator.
func formatClosedBeadDetail(d BeadDetail, summary, worklog string) string {
	base := formatBeadDetail(d)
	if summary == "" && worklog == "" {
		return base
	}

	var b strings.Builder
	b.WriteString(base)
	b.WriteString("\n\n" + archiveSeparator + "\n")

	if summary != "" {
		fmt.Fprintf(&b, "\n%s", summary)
	}

	if worklog != "" {
		fmt.Fprintf(&b, "\n\nWorklog:\n%s", worklog)
	}

	return b.String()
}

// Init returns the initial command. If a BeadLister was provided,
// it fires an async fetch for the bead list with spinner animation.
func (m Model) Init() tea.Cmd {
	if m.lister != nil {
		return tea.Batch(initBrowse(m.lister), m.browseSpinner.Tick)
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

	case resolveDebounceMsg:
		if msg.ID != m.pendingResolveID {
			return m, nil
		}
		m.pendingResolveID = ""
		m.resolvingID = msg.ID
		m.resolveErr = nil
		return m, tea.Batch(resolveBeadCmd(m.resolver, msg.ID), m.browseSpinner.Tick)

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
			m.viewport.SetContent(m.renderDetailContent(msg.Detail))
			m.viewport.GotoTop()
		}
		return m, nil

	case RefreshBeadsMsg:
		m.cache.Invalidate()
		if m.lister != nil {
			return m, tea.Batch(initBrowse(m.lister), m.browseSpinner.Tick)
		}
		return m, nil

	case ToggleHistoryMsg:
		if m.lister != nil {
			return m, tea.Batch(initClosedBrowse(m.lister), m.browseSpinner.Tick)
		}
		return m, nil

	case ClosedBeadListMsg:
		m.browse, _ = m.browse.Update(msg)
		return m.maybeResolve()

	case DispatchMsg:
		return m.handleDispatch(msg)

	case CampaignStartMsg:
		title := msg.ParentTitle
		if title == "" {
			title = m.campaign.parentTitle // Preserve title set during dispatch.
		}
		m.campaign = newCampaignState(msg.ParentID, title, msg.Tasks)
		return m, listenForEvents(m.eventCh)

	case CampaignTaskStartMsg, CampaignTaskDoneMsg:
		var cmd tea.Cmd
		m.campaign, cmd = m.campaign.Update(msg)
		return m, tea.Batch(cmd, listenForEvents(m.eventCh))

	case CampaignDoneMsg:
		m.campaignDone = &msg
		return m, listenForEvents(m.eventCh)

	case CampaignErrorMsg:
		m.campaignErr = msg.Err
		return m, listenForEvents(m.eventCh)

	case PhaseUpdateMsg:
		if m.mode == ModeCampaign {
			var cmd tea.Cmd
			m.campaign, cmd = m.campaign.Update(msg)
			return m, tea.Batch(cmd, listenForEvents(m.eventCh))
		}
		var cmd tea.Cmd
		m.pipeline, cmd = m.pipeline.Update(msg)
		return m, tea.Batch(cmd, listenForEvents(m.eventCh))

	case PipelineDoneMsg:
		m.pipelineOutput = &msg.Output
		return m, listenForEvents(m.eventCh)

	case PipelineErrorMsg:
		m.pipelineErr = msg.Err
		return m, listenForEvents(m.eventCh)

	case PostPipelineDoneMsg:
		// Post-pipeline is best-effort; no UI update needed.
		return m, nil

	case channelClosedMsg:
		m.cancelPipeline = nil
		m.eventCh = nil
		if m.aborting {
			return m.returnToBrowseAfterAbort()
		}
		if m.mode == ModeCampaign {
			if m.campaignDone == nil {
				m.campaignDone = &CampaignDoneMsg{
					ParentID:   m.campaign.parentID,
					TotalTasks: len(m.campaign.tasks),
					Passed:     m.campaign.completed,
					Failed:     m.campaign.failed,
				}
			}
			m.mode = ModeCampaignSummary
			return m, nil
		}
		m.mode = ModeSummary
		return m, nil

	case elapsedTickMsg:
		var cmd tea.Cmd
		switch m.mode {
		case ModePipeline:
			m.pipeline, cmd = m.pipeline.Update(msg)
		case ModeCampaign:
			m.campaign, cmd = m.campaign.Update(msg)
		default:
			return m, nil
		}
		return m, cmd

	case spinner.TickMsg:
		var cmd tea.Cmd
		switch {
		case m.mode == ModePipeline:
			m.pipeline, cmd = m.pipeline.Update(msg)
		case m.mode == ModeCampaign:
			m.campaign, cmd = m.campaign.Update(msg)
		case m.mode == ModeBrowse && (m.browse.loading || m.resolvingID != ""):
			m.browseSpinner, cmd = m.browseSpinner.Update(msg)
		default:
			return m, nil
		}
		return m, cmd

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

// handleKey processes key messages with global and mode-specific routing.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Summary modes: any key returns to browse with cache refresh.
	if m.mode == ModeSummary {
		return m.returnToBrowse()
	}
	if m.mode == ModeCampaignSummary {
		return m.returnToBrowseFromCampaign()
	}

	// Global keys.
	switch msg.String() {
	case "q", "ctrl+c":
		switch {
		case m.mode == ModeBrowse:
			return m, tea.Quit
		case (m.mode == ModePipeline || m.mode == ModeCampaign) && m.aborting:
			return m, tea.Quit
		case m.mode == ModePipeline && m.cancelPipeline != nil:
			m.aborting = true
			m.pipeline.aborting = true
			m.cancelPipeline()
			return m, nil
		case m.mode == ModeCampaign && m.cancelPipeline != nil:
			m.aborting = true
			m.campaign.pipeline.aborting = true
			m.cancelPipeline()
			return m, nil
		}
	case "tab":
		if m.focus == PaneLeft {
			m.focus = PaneRight
		} else {
			m.focus = PaneLeft
		}
		return m, nil
	case "r":
		if m.mode == ModeBrowse {
			m.browse.loading = true
			m.browse.err = nil
			m.browse.showClosed = false
			m.browse.readyBeads = nil
			return m, func() tea.Msg { return RefreshBeadsMsg{} }
		}
	case "h":
		if m.mode == ModeBrowse && !m.browse.loading {
			var cmd tea.Cmd
			m.browse, cmd = m.browse.Update(msg)
			return m, cmd
		}
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

	case m.mode == ModeCampaign && m.focus == PaneLeft:
		var cmd tea.Cmd
		m.campaign, cmd = m.campaign.Update(msg)
		return m, cmd

	case m.mode == ModeCampaign && m.focus == PaneRight:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleDispatch branches on BeadType: feature/epic → campaign, else → pipeline.
func (m Model) handleDispatch(msg DispatchMsg) (tea.Model, tea.Cmd) {
	if (msg.BeadType == "feature" || msg.BeadType == "epic") && m.campaignRunner != nil {
		return m.handleCampaignDispatch(msg)
	}
	return m.handlePipelineDispatch(msg)
}

// handlePipelineDispatch transitions to pipeline mode and starts the pipeline goroutine.
func (m Model) handlePipelineDispatch(msg DispatchMsg) (tea.Model, tea.Cmd) {
	if m.runner == nil {
		return m, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelPipeline = cancel
	ch := make(chan tea.Msg, 16)
	m.eventCh = ch
	m.mode = ModePipeline
	m.focus = PaneLeft
	m.pipeline = newPipelineState(m.phaseNames)
	m.pipeline.beadID = msg.BeadID
	m.pipeline.beadTitle = msg.BeadTitle
	m.pipelineOutput = nil
	m.pipelineErr = nil
	m.aborting = false
	m.dispatchedBeadID = msg.BeadID
	input := PipelineInput{BeadID: msg.BeadID}
	go dispatchPipeline(ctx, m.runner, input, ch)
	return m, tea.Batch(m.pipeline.spinner.Tick, elapsedTickCmd(), listenForEvents(ch))
}

// handleCampaignDispatch transitions to campaign mode and starts the campaign goroutine.
func (m Model) handleCampaignDispatch(msg DispatchMsg) (tea.Model, tea.Cmd) {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelPipeline = cancel
	ch := make(chan tea.Msg, 16)
	m.eventCh = ch
	m.mode = ModeCampaign
	m.focus = PaneLeft
	m.campaign = newCampaignState(msg.BeadID, msg.BeadTitle, nil)
	m.pipelineOutput = nil
	m.pipelineErr = nil
	m.aborting = false
	m.campaignDone = nil
	m.campaignErr = nil
	m.dispatchedBeadID = msg.BeadID
	go dispatchCampaign(ctx, m.campaignRunner, m.runner, msg.BeadID, ch)
	return m, tea.Batch(m.campaign.pipeline.spinner.Tick, elapsedTickCmd(), listenForEvents(ch))
}

// maybeResolve checks if the selected bead changed and triggers a resolve
// if needed. On cache hit, the viewport is updated immediately (bypassing
// debounce). On cache miss, a debounce tick is started; the actual resolve
// is dispatched only when the tick fires with a matching pendingResolveID.
func (m Model) maybeResolve() (Model, tea.Cmd) {
	selected := m.browse.SelectedID()
	if selected == "" || selected == m.detailID {
		return m, nil
	}
	m.detailID = selected

	if detail, ok := m.cache.Get(selected); ok {
		m.resolvingID = ""
		m.resolveErr = nil
		m.pendingResolveID = ""
		m.viewport.SetContent(m.renderDetailContent(*detail))
		m.viewport.GotoTop()
		return m, nil
	}

	if m.resolver != nil {
		m.pendingResolveID = selected
		m.resolveErr = nil
		id := selected
		return m, tea.Tick(resolveDebounce, func(time.Time) tea.Msg {
			return resolveDebounceMsg{ID: id}
		})
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
	helpView := m.help.View(HelpBindings(m.mode, m.browse.showClosed))

	return lipgloss.JoinVertical(lipgloss.Left, panes, helpView)
}

// viewLeft renders the left pane content based on mode.
func (m Model) viewLeft() string {
	leftWidth, _ := PaneWidths(m.width)
	w := leftWidth - borderChrome
	h := m.contentHeight()

	switch m.mode {
	case ModePipeline, ModeSummary:
		return m.pipeline.View(w, h)
	case ModeCampaign, ModeCampaignSummary:
		return m.campaign.View(w, h)
	default:
		return m.browse.View(w, h, m.browseSpinner.View())
	}
}

// viewRight renders the right pane content based on mode.
func (m Model) viewRight() string {
	switch m.mode {
	case ModePipeline:
		_, rightWidth := PaneWidths(m.width)
		return m.pipeline.ViewReport(rightWidth-borderChrome, m.contentHeight())
	case ModeSummary:
		return m.viewSummaryRight()
	case ModeCampaign:
		_, rightWidth := PaneWidths(m.width)
		return m.campaign.ViewReport(rightWidth-borderChrome, m.contentHeight())
	case ModeCampaignSummary:
		return m.viewCampaignSummaryRight()
	default:
		return m.viewBrowseDetail()
	}
}

// viewBrowseDetail renders the right pane in browse mode:
// loading spinner, error message, or resolved detail viewport.
func (m Model) viewBrowseDetail() string {
	if m.resolvingID != "" {
		return fmt.Sprintf("%s Loading %s...", m.browseSpinner.View(), m.resolvingID)
	}
	if m.resolveErr != nil {
		return fmt.Sprintf("Could not load bead detail\n\n%s", m.resolveErr)
	}
	if m.detailID == "" {
		return "Select a bead"
	}
	return m.viewport.View()
}
