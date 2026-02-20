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

// statusLineDuration is how long the transient status message is shown
// before being cleared automatically.
const statusLineDuration = 5 * time.Second

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
	lastDispatchedID string // Preserved across returnToBrowse so cursor snaps on next BeadListMsg.
	aborting         bool

	backgroundMode Mode // Non-zero when pipeline/campaign is running while user is in browse.

	campaign       campaignState
	campaignRunner CampaignRunner
	campaignDone   *CampaignDoneMsg // set on CampaignDoneMsg or synthesized on channel close
	campaignErr    error            // set on CampaignErrorMsg from runner failure

	confirm       confirmState
	hasValidation bool // true when campaign validation phases are configured

	archive ArchiveReader

	activeProvider string   // Currently selected provider name (default from config).
	providerNames  []string // Registered provider names for cycling.

	statusMsg string // Transient status shown between panes and help bar; cleared by statusClearMsg.
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

// WithCampaignValidation sets whether campaign validation phases are configured.
// When true, the confirmation screen shows a validation step after task execution.
func WithCampaignValidation(v bool) ModelOption {
	return func(m *Model) { m.hasValidation = v }
}

// WithArchiveReader sets the ArchiveReader used to fetch archived pipeline
// results for closed beads.
func WithArchiveReader(ar ArchiveReader) ModelOption {
	return func(m *Model) { m.archive = ar }
}

// WithProviderNames sets the list of registered provider names and the
// initially active provider. When more than one name is provided, the 'p'
// key toggles between them in browse mode.
func WithProviderNames(names []string, active string) ModelOption {
	return func(m *Model) {
		m.providerNames = names
		m.activeProvider = active
	}
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
	// Always deliver the final result. Unlike incremental status updates,
	// the completion message must reach the receiver so channelClosedMsg
	// processing can build the correct summary/status.
	if err != nil {
		ch <- PipelineErrorMsg{Err: err}
		return
	}
	ch <- PipelineDoneMsg{Output: output}
}

// dispatchCampaign runs a campaign in the calling goroutine, bridging
// status events to ch. It closes ch when done. The provider name is
// captured at dispatch time and injected into every task's PipelineInput.
func dispatchCampaign(ctx context.Context, cr CampaignRunner, pr PipelineRunner, parentID, providerName string, ch chan<- tea.Msg) {
	defer close(ch)
	statusFn := func(msg tea.Msg) {
		select {
		case ch <- msg:
		case <-ctx.Done():
		}
	}
	var pipelineFn func(context.Context, PipelineInput, func(PhaseUpdateMsg)) (PipelineOutput, error)
	if pr != nil {
		pipelineFn = func(ctx context.Context, input PipelineInput, statusFn func(PhaseUpdateMsg)) (PipelineOutput, error) {
			input.Provider = providerName
			return pr.RunPipeline(ctx, input, statusFn)
		}
	}
	// Always deliver the final error. Unlike incremental status updates,
	// the error must reach the receiver so channelClosedMsg processing
	// can build the correct status message.
	if err := cr.RunCampaign(ctx, parentID, statusFn, pipelineFn); err != nil {
		ch <- CampaignErrorMsg{Err: err}
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

// renderDetailContent formats a bead detail for the viewport. For closed beads
// with an archive reader, it appends archived summary and worklog data.
func (m Model) renderDetailContent(d BeadDetail) string {
	if m.archive == nil {
		return formatBeadDetail(d)
	}
	if bead, ok := m.browse.SelectedBead(); ok && bead.Closed {
		summary, _ := m.archive.ReadSummary(d.ID)
		worklog, _ := m.archive.ReadWorklog(d.ID)
		return formatClosedBeadDetail(d, summary, worklog)
	}
	return formatBeadDetail(d)
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
		if m.lastDispatchedID != "" {
			for i, fn := range m.browse.flatNodes {
				if fn.Node.Bead.ID == m.lastDispatchedID {
					m.browse.cursor = i
					break
				}
			}
			m.lastDispatchedID = ""
		}
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
		m.pendingResolveID = ""
		m.resolvingID = ""
		m.resolveErr = nil
		if m.lister != nil {
			return m, tea.Batch(initBrowse(m.lister), m.browseSpinner.Tick)
		}
		return m, nil

	case ProviderCycleMsg:
		if len(m.providerNames) > 1 {
			next := m.providerNames[0]
			for i, name := range m.providerNames {
				if name == m.activeProvider {
					next = m.providerNames[(i+1)%len(m.providerNames)]
					break
				}
			}
			m.activeProvider = next
		}
		return m, nil

	case ConfirmRequestMsg:
		return m.handleConfirmRequest(msg)

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

	case CampaignValidationStartMsg:
		m.campaign.validating = true
		return m, listenForEvents(m.eventCh)

	case CampaignValidationDoneMsg:
		m.campaign.validating = false
		m.campaign.validationResult = &msg
		return m, listenForEvents(m.eventCh)

	case PhaseUpdateMsg:
		if m.mode == ModeCampaign || m.backgroundMode == ModeCampaign {
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
		if msg.Err != nil {
			m.statusMsg = fmt.Sprintf("%s %s: post-pipeline failed: %s", SymbolCross, msg.BeadID, msg.Err)
		} else {
			m.statusMsg = fmt.Sprintf("%s %s: merged to main, bead closed, worktree removed", SymbolCheck, msg.BeadID)
		}
		return m, tea.Tick(statusLineDuration, func(time.Time) tea.Msg {
			return statusClearMsg{}
		})

	case statusClearMsg:
		m.statusMsg = ""
		return m, nil

	case channelClosedMsg:
		m.cancelPipeline = nil
		m.eventCh = nil
		if m.mode == ModeBrowse && m.backgroundMode != 0 {
			return m.handleBackgroundComplete()
		}
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
		switch {
		case m.mode == ModePipeline || m.backgroundMode == ModePipeline:
			m.pipeline, cmd = m.pipeline.Update(msg)
		case m.mode == ModeCampaign || m.backgroundMode == ModeCampaign:
			m.campaign, cmd = m.campaign.Update(msg)
		default:
			return m, nil
		}
		return m, cmd

	case spinner.TickMsg:
		var cmd tea.Cmd
		var cmds []tea.Cmd
		// Route spinner ticks to the foreground mode.
		switch m.mode {
		case ModePipeline:
			m.pipeline, cmd = m.pipeline.Update(msg)
			cmds = append(cmds, cmd)
		case ModeCampaign:
			m.campaign, cmd = m.campaign.Update(msg)
			cmds = append(cmds, cmd)
		case ModeBrowse:
			if m.browse.loading || m.resolvingID != "" {
				m.browseSpinner, cmd = m.browseSpinner.Update(msg)
				cmds = append(cmds, cmd)
			}
		default:
			return m, nil
		}
		// Also route to background mode to keep spinners alive for re-entry.
		switch m.backgroundMode {
		case ModePipeline:
			m.pipeline, cmd = m.pipeline.Update(msg)
			cmds = append(cmds, cmd)
		case ModeCampaign:
			m.campaign, cmd = m.campaign.Update(msg)
			cmds = append(cmds, cmd)
		}
		if len(cmds) == 0 {
			return m, nil
		}
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

// handleKey processes key messages with global and mode-specific routing.
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Summary modes: Enter/Esc/b returns to browse, other keys allow navigation.
	if m.mode == ModeSummary {
		switch msg.String() {
		case "enter", "esc", "b":
			return m.returnToBrowse()
		}
	}
	if m.mode == ModeCampaignSummary {
		switch msg.String() {
		case "enter", "esc", "b":
			return m.returnToBrowseFromCampaign()
		}
	}

	// Confirm mode: Enter dispatches, Esc/q returns to browse.
	if m.mode == ModeConfirm {
		switch msg.String() {
		case "enter":
			m.mode = ModeBrowse // Temporarily set back before dispatch routing.
			return m.handleDispatch(DispatchMsg{
				BeadID:    m.confirm.beadID,
				BeadType:  m.confirm.beadType,
				BeadTitle: m.confirm.beadTitle,
				Provider:  m.confirm.provider,
			})
		case "esc", "q":
			m.mode = ModeBrowse
			m.focus = PaneLeft
			return m, nil
		}
		return m, nil // Swallow all other keys in confirm mode.
	}

	// Global keys.
	switch msg.String() {
	case "esc":
		if m.mode == ModePipeline || m.mode == ModeCampaign {
			return m.sendToBackground()
		}
	case "q", "ctrl+c":
		switch {
		case m.mode == ModeBrowse && m.backgroundMode != 0:
			// Abort the background operation, don't quit the app.
			if m.cancelPipeline != nil {
				m.aborting = true
				m.cancelPipeline()
			}
			return m, nil
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
	case "p":
		if m.mode == ModeBrowse && len(m.providerNames) > 1 {
			return m, func() tea.Msg { return ProviderCycleMsg{} }
		}
	case "r":
		if m.mode == ModeBrowse {
			m.browse.loading = true
			m.browse.err = nil
			return m, func() tea.Msg { return RefreshBeadsMsg{} }
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

	case m.mode == ModeSummary && m.focus == PaneLeft:
		var cmd tea.Cmd
		m.pipeline, cmd = m.pipeline.Update(msg)
		return m, cmd

	case m.mode == ModeSummary && m.focus == PaneRight:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd

	case m.mode == ModeCampaignSummary && m.focus == PaneLeft:
		var cmd tea.Cmd
		m.campaign, cmd = m.campaign.Update(msg)
		return m, cmd

	case m.mode == ModeCampaignSummary && m.focus == PaneRight:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleConfirmRequest builds a confirmState and transitions to ModeConfirm.
// If the selected bead is already running in the background, re-enter that view.
func (m Model) handleConfirmRequest(msg ConfirmRequestMsg) (tea.Model, tea.Cmd) {
	if m.backgroundMode != 0 && msg.BeadID == m.dispatchedBeadID {
		m.mode = m.backgroundMode
		m.backgroundMode = 0
		m.statusMsg = ""
		return m, nil
	}
	cs := confirmState{
		beadID:        msg.BeadID,
		beadType:      msg.BeadType,
		beadTitle:     msg.BeadTitle,
		hasValidation: m.hasValidation,
		provider:      m.activeProvider,
	}
	// For features/epics, collect open children from the browse tree.
	if msg.BeadType == "feature" || msg.BeadType == "epic" {
		cs.children = collectOpenChildren(m.browse.roots, msg.BeadID)
	}
	m.confirm = cs
	m.mode = ModeConfirm
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
	if m.cancelPipeline != nil {
		m.cancelPipeline()
	}
	m.backgroundMode = 0
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelPipeline = cancel
	ch := make(chan tea.Msg, 16)
	m.eventCh = ch
	m.mode = ModePipeline
	m.focus = PaneLeft
	m.pipeline = newPipelineState(m.phaseNames)
	m.pipeline.beadID = msg.BeadID
	m.pipeline.beadTitle = msg.BeadTitle
	m.pipeline.provider = msg.Provider
	m.pipelineOutput = nil
	m.pipelineErr = nil
	m.aborting = false
	m.dispatchedBeadID = msg.BeadID
	input := PipelineInput{BeadID: msg.BeadID, Provider: msg.Provider}
	go dispatchPipeline(ctx, m.runner, input, ch)
	return m, tea.Batch(m.pipeline.spinner.Tick, elapsedTickCmd(), listenForEvents(ch))
}

// handleCampaignDispatch transitions to campaign mode and starts the campaign goroutine.
func (m Model) handleCampaignDispatch(msg DispatchMsg) (tea.Model, tea.Cmd) {
	if m.cancelPipeline != nil {
		m.cancelPipeline()
	}
	m.backgroundMode = 0
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelPipeline = cancel
	ch := make(chan tea.Msg, 16)
	m.eventCh = ch
	m.mode = ModeCampaign
	m.focus = PaneLeft
	m.campaign = newCampaignState(msg.BeadID, msg.BeadTitle, nil)
	m.campaign.provider = msg.Provider
	m.pipelineOutput = nil
	m.pipelineErr = nil
	m.aborting = false
	m.campaignDone = nil
	m.campaignErr = nil
	m.dispatchedBeadID = msg.BeadID
	go dispatchCampaign(ctx, m.campaignRunner, m.runner, msg.BeadID, msg.Provider, ch)
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

// handleBackgroundComplete cleans up after a backgrounded operation completes.
// Called when channelClosedMsg arrives while in browse mode with a background op.
func (m Model) handleBackgroundComplete() (Model, tea.Cmd) {
	bgMode := m.backgroundMode
	beadID := m.dispatchedBeadID
	m.lastDispatchedID = beadID // snap cursor on next bead list refresh
	m.backgroundMode = 0
	m.aborting = false
	m.dispatchedBeadID = ""

	switch {
	case bgMode == ModeCampaign && m.campaignDone != nil:
		m.statusMsg = fmt.Sprintf("%s Campaign complete: %d/%d passed",
			SymbolCheck, m.campaignDone.Passed, m.campaignDone.TotalTasks)
	case bgMode == ModeCampaign && m.campaignErr != nil:
		m.statusMsg = fmt.Sprintf("%s Campaign error: %s", SymbolCross, m.campaignErr)
	case bgMode == ModeCampaign:
		m.statusMsg = fmt.Sprintf("%s Background operation complete", SymbolCheck)
	case m.pipelineErr != nil:
		m.statusMsg = fmt.Sprintf("%s Pipeline failed: %s", SymbolCross, m.pipelineErr)
	default:
		m.statusMsg = fmt.Sprintf("%s Pipeline complete", SymbolCheck)
	}

	m.cache.Invalidate()
	m.campaignDone = nil
	m.campaignErr = nil
	var cmds []tea.Cmd

	// Fire post-pipeline lifecycle for non-campaign background completions.
	// Campaigns handle their own lifecycle, but standalone pipelines need
	// merge/close/cleanup to run even when they completed in the background.
	if bgMode != ModeCampaign && m.postPipeline != nil && beadID != "" && m.pipelineErr == nil {
		ppFn := m.postPipeline
		cmds = append(cmds, func() tea.Msg {
			err := ppFn(beadID)
			return PostPipelineDoneMsg{BeadID: beadID, Err: err}
		})
	}

	if m.lister != nil {
		cmds = append(cmds, initBrowse(m.lister), m.browseSpinner.Tick)
	}
	cmds = append(cmds, tea.Tick(statusLineDuration, func(time.Time) tea.Msg {
		return statusClearMsg{}
	}))
	return m, tea.Batch(cmds...)
}

// sendToBackground transitions from pipeline/campaign mode to browse mode
// while keeping the operation running. The event channel and cancel func
// are preserved so messages continue to be processed.
func (m Model) sendToBackground() (Model, tea.Cmd) {
	m.backgroundMode = m.mode
	m.mode = ModeBrowse
	m.focus = PaneLeft
	m.statusMsg = fmt.Sprintf("Running %s in background", m.dispatchedBeadID)
	if m.lister != nil {
		return m, tea.Batch(initBrowse(m.lister), m.browseSpinner.Tick)
	}
	return m, nil
}

// contentHeight returns the usable height for pane content,
// accounting for border chrome and the help bar.
func (m Model) contentHeight() int {
	return max(m.height-borderChrome-helpBarHeight, 1)
}

// helpBindings returns context-aware help bindings.
// In browse mode, the Enter label varies by selected bead type.
// In confirm mode, only Enter/Esc are shown.
// In summary mode with postPipeline, the continue label reflects lifecycle actions.
func (m Model) helpBindings() help.KeyMap {
	switch m.mode {
	case ModeConfirm:
		return ConfirmKeyMap()
	case ModeBrowse:
		var km browseKeys
		if m.backgroundMode != 0 {
			km = BrowseKeyMapWithBackground(m.dispatchedBeadID)
		} else if bead, ok := m.browse.SelectedBead(); ok && !bead.Closed {
			childCount := 0
			if bead.Type == "feature" || bead.Type == "epic" {
				childCount = countOpenChildren(m.browse.roots, bead.ID)
			}
			km = BrowseKeyMapForBead(bead.Type, childCount)
		} else {
			km = BrowseKeyMapForBead("", 0)
		}
		// Enable provider toggle when multiple providers are registered.
		if len(m.providerNames) > 1 {
			km.Provider = BrowseKeyMapWithProvider(m.activeProvider).Provider
		}
		return km
	case ModeSummary:
		return PipelineSummaryKeyMap(m.postPipeline != nil)
	default:
		return HelpBindings(m.mode)
	}
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
	helpView := m.help.View(m.helpBindings())

	if m.statusMsg != "" {
		statusLine := pipeHeaderStyle.Render(m.statusMsg)
		return lipgloss.JoinVertical(lipgloss.Left, panes, statusLine, helpView)
	}
	return lipgloss.JoinVertical(lipgloss.Left, panes, helpView)
}

// viewLeft renders the left pane content based on mode.
func (m Model) viewLeft() string {
	leftWidth, _ := PaneWidths(m.width)
	w := leftWidth - borderChrome
	h := m.contentHeight()

	switch m.mode {
	case ModeConfirm:
		return m.confirm.View(w, h)
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
	case ModeConfirm:
		return m.viewBrowseDetail() // Keep showing bead detail during confirmation.
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
