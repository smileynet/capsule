package dashboard

import (
	"fmt"

	"github.com/charmbracelet/bubbles/key"
)

// browseKeys holds key bindings for browse mode.
type browseKeys struct {
	Up          key.Binding
	Down        key.Binding
	Right       key.Binding
	Left        key.Binding
	Enter       key.Binding
	Tab         key.Binding
	Provider    key.Binding
	CollapseAll key.Binding
	Refresh     key.Binding
	Quit        key.Binding
}

// ShortHelp returns the browse mode bindings for the help bar.
func (k browseKeys) ShortHelp() []key.Binding {
	bindings := []key.Binding{k.Up, k.Down, k.Right, k.Left, k.Enter, k.Tab}
	if k.Provider.Enabled() {
		bindings = append(bindings, k.Provider)
	}
	return append(bindings, k.CollapseAll, k.Refresh, k.Quit)
}

// FullHelp returns the browse mode bindings grouped for expanded help.
func (k browseKeys) FullHelp() [][]key.Binding {
	row2 := []key.Binding{k.Tab}
	if k.Provider.Enabled() {
		row2 = append(row2, k.Provider)
	}
	row2 = append(row2, k.CollapseAll, k.Refresh, k.Quit)
	return [][]key.Binding{
		{k.Up, k.Down, k.Right, k.Left, k.Enter},
		row2,
	}
}

// pipelineKeys holds key bindings for pipeline mode.
type pipelineKeys struct {
	Up   key.Binding
	Down key.Binding
	Tab  key.Binding
	Esc  key.Binding
	Quit key.Binding
}

// ShortHelp returns the pipeline mode bindings for the help bar.
func (k pipelineKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Tab, k.Esc, k.Quit}
}

// FullHelp returns the pipeline mode bindings grouped for expanded help.
func (k pipelineKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.Tab, k.Esc, k.Quit},
	}
}

// summaryKeys holds key bindings for summary mode.
type summaryKeys struct {
	AnyKey key.Binding
}

// ShortHelp returns the summary mode bindings for the help bar.
func (k summaryKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.AnyKey}
}

// FullHelp returns the summary mode bindings grouped for expanded help.
func (k summaryKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.AnyKey}}
}

// BrowseKeyMap returns the key bindings for browse mode.
// The Provider binding is disabled by default; use BrowseKeyMapWithProvider
// to enable it with a dynamic label showing the current provider name.
func BrowseKeyMap() browseKeys {
	return browseKeys{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "expand"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "collapse"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "run pipeline"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch pane"),
		),
		Provider: key.NewBinding(
			key.WithKeys("p"),
			key.WithHelp("p", "provider"),
			key.WithDisabled(),
		),
		CollapseAll: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "collapse all"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// BrowseKeyMapWithProvider returns browse key bindings with the provider
// toggle key enabled and its help text showing the current provider name.
func BrowseKeyMapWithProvider(providerName string) browseKeys {
	km := BrowseKeyMap()
	km.Provider = key.NewBinding(
		key.WithKeys("p"),
		key.WithHelp("p", fmt.Sprintf("provider: %s", providerName)),
	)
	return km
}

// PipelineKeyMap returns the key bindings for pipeline mode.
func PipelineKeyMap() pipelineKeys {
	return pipelineKeys{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch pane"),
		),
		Esc: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "browse"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "abort"),
		),
	}
}

// campaignKeys holds key bindings for campaign mode.
type campaignKeys struct {
	Up   key.Binding
	Down key.Binding
	Tab  key.Binding
	Esc  key.Binding
	Quit key.Binding
}

// ShortHelp returns the campaign mode bindings for the help bar.
func (k campaignKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Tab, k.Esc, k.Quit}
}

// FullHelp returns the campaign mode bindings grouped for expanded help.
func (k campaignKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.Tab, k.Esc, k.Quit},
	}
}

// CampaignKeyMap returns the key bindings for campaign mode.
func CampaignKeyMap() campaignKeys {
	return campaignKeys{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch pane"),
		),
		Esc: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "browse"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "abort"),
		),
	}
}

// SummaryKeyMap returns the key bindings for summary mode.
func SummaryKeyMap() summaryKeys {
	return summaryKeys{
		AnyKey: key.NewBinding(
			key.WithKeys("enter", "esc", "b"),
			key.WithHelp("enter/esc/b", "back to browse"),
		),
	}
}

// PipelineSummaryKeyMap returns summary key bindings with a context-aware label.
// When hasPostPipeline is true, the label reflects the lifecycle actions.
func PipelineSummaryKeyMap(hasPostPipeline bool) summaryKeys {
	desc := "back to browse"
	if hasPostPipeline {
		desc = "back (merge + close)"
	}
	return summaryKeys{
		AnyKey: key.NewBinding(
			key.WithKeys("enter", "esc", "b"),
			key.WithHelp("enter/esc/b", desc),
		),
	}
}

// confirmKeys holds key bindings for confirm mode.
type confirmKeys struct {
	Enter key.Binding
	Esc   key.Binding
}

// ShortHelp returns the confirm mode bindings for the help bar.
func (k confirmKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Enter, k.Esc}
}

// FullHelp returns the confirm mode bindings grouped for expanded help.
func (k confirmKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{{k.Enter, k.Esc}}
}

// ConfirmKeyMap returns the key bindings for confirm mode.
func ConfirmKeyMap() confirmKeys {
	return confirmKeys{
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		Esc: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
	}
}

// BrowseKeyMapWithBackground returns browse key bindings when a background
// operation is running. q aborts the background op, Enter on the running
// bead re-enters the view.
func BrowseKeyMapWithBackground(beadID string) browseKeys {
	km := BrowseKeyMap()
	km.Enter = key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", fmt.Sprintf("re-enter %s", beadID)),
	)
	km.Quit = key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "abort background"),
	)
	return km
}

// BrowseKeyMapForBead returns browse key bindings with a dynamic Enter label
// based on the selected bead type and its child count.
func BrowseKeyMapForBead(beadType string, childCount int) browseKeys {
	km := BrowseKeyMap()
	if (beadType == "feature" || beadType == "epic") && childCount > 0 {
		taskWord := "tasks"
		if childCount == 1 {
			taskWord = "task"
		}
		km.Enter = key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", fmt.Sprintf("run campaign (%d %s)", childCount, taskWord)),
		)
	}
	return km
}
