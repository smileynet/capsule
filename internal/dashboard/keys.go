package dashboard

import "github.com/charmbracelet/bubbles/key"

// browseKeys holds key bindings for browse mode.
type browseKeys struct {
	Up      key.Binding
	Down    key.Binding
	Enter   key.Binding
	Tab     key.Binding
	Refresh key.Binding
	Quit    key.Binding
}

// ShortHelp returns the browse mode bindings for the help bar.
func (k browseKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Enter, k.Tab, k.Refresh, k.Quit}
}

// FullHelp returns the browse mode bindings grouped for expanded help.
func (k browseKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Enter},
		{k.Tab, k.Refresh, k.Quit},
	}
}

// pipelineKeys holds key bindings for pipeline mode.
type pipelineKeys struct {
	Up   key.Binding
	Down key.Binding
	Tab  key.Binding
	Quit key.Binding
}

// ShortHelp returns the pipeline mode bindings for the help bar.
func (k pipelineKeys) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Tab, k.Quit}
}

// FullHelp returns the pipeline mode bindings grouped for expanded help.
func (k pipelineKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down},
		{k.Tab, k.Quit},
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
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "run pipeline"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch pane"),
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
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "abort"),
		),
	}
}

// SummaryKeyMap returns the key bindings for summary mode.
func SummaryKeyMap() summaryKeys {
	return summaryKeys{
		// "any" is a display-only key for the help bar; actual any-key
		// handling is done in the Update() switch on tea.KeyMsg.
		AnyKey: key.NewBinding(
			key.WithKeys("any"),
			key.WithHelp("any key", "continue"),
		),
	}
}
