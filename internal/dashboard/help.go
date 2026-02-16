package dashboard

import "github.com/charmbracelet/bubbles/help"

// HelpBindings returns the help.KeyMap for the given mode,
// providing context-aware help bar content.
func HelpBindings(mode Mode) help.KeyMap {
	switch mode {
	case ModePipeline:
		return PipelineKeyMap()
	case ModeSummary, ModeCampaignSummary:
		return SummaryKeyMap()
	case ModeCampaign:
		return CampaignKeyMap()
	default:
		return BrowseKeyMap()
	}
}
