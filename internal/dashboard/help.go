package dashboard

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
)

// HelpBindings returns the help.KeyMap for the given mode,
// providing context-aware help bar content.
func HelpBindings(mode Mode, showClosed bool) help.KeyMap {
	switch mode {
	case ModePipeline:
		return PipelineKeyMap()
	case ModeSummary, ModeCampaignSummary:
		return SummaryKeyMap()
	case ModeCampaign:
		return CampaignKeyMap()
	default:
		km := BrowseKeyMap()
		if showClosed {
			km.History = key.NewBinding(
				key.WithKeys("h"),
				key.WithHelp("h", "ready"),
			)
		}
		return km
	}
}
