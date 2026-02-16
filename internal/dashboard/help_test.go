package dashboard

import (
	"testing"

	"github.com/charmbracelet/bubbles/help"
)

func TestHelpBindings_BrowseMode(t *testing.T) {
	// Given: help bindings for browse mode
	km := HelpBindings(ModeBrowse, false)
	bindings := km.ShortHelp()
	allKeys := collectKeys(bindings)

	// Then: enter and quit keys are present
	if !containsKey(allKeys, "enter") {
		t.Error("browse help should contain 'enter' key")
	}
	if !containsKey(allKeys, "q") {
		t.Error("browse help should contain 'q' key")
	}
}

func TestHelpBindings_PipelineMode(t *testing.T) {
	// Given: help bindings for pipeline mode
	km := HelpBindings(ModePipeline, false)
	bindings := km.ShortHelp()
	allKeys := collectKeys(bindings)

	// Then: enter is absent (no dispatch) but quit is present
	if containsKey(allKeys, "enter") {
		t.Error("pipeline help should not contain 'enter' key")
	}
	if !containsKey(allKeys, "q") {
		t.Error("pipeline help should contain 'q' key")
	}
}

func TestHelpBindings_SummaryMode(t *testing.T) {
	// Given: help bindings for summary mode
	km := HelpBindings(ModeSummary, false)
	bindings := km.ShortHelp()

	// Then: at least one binding is present
	if len(bindings) == 0 {
		t.Fatal("summary help should have at least one binding")
	}
}

func TestHelpBindings_CampaignMode(t *testing.T) {
	// Given: help bindings for campaign mode
	km := HelpBindings(ModeCampaign, false)
	bindings := km.ShortHelp()
	allKeys := collectKeys(bindings)

	// Then: enter is absent (no dispatch) but quit is present
	if containsKey(allKeys, "enter") {
		t.Error("campaign help should not contain 'enter' key")
	}
	if !containsKey(allKeys, "q") {
		t.Error("campaign help should contain 'q' key")
	}
}

func TestHelpBindings_CampaignSummaryMode(t *testing.T) {
	// Given: help bindings for campaign summary mode
	km := HelpBindings(ModeCampaignSummary, false)
	bindings := km.ShortHelp()

	// Then: at least one binding is present
	if len(bindings) == 0 {
		t.Fatal("campaign summary help should have at least one binding")
	}
}

func TestHelpBindings_ImplementsKeyMap(t *testing.T) {
	// Given: all dashboard modes
	modes := []Mode{ModeBrowse, ModePipeline, ModeSummary, ModeCampaign, ModeCampaignSummary}

	// Then: each returns a type satisfying help.KeyMap (ShortHelp + FullHelp)
	for _, mode := range modes {
		km := HelpBindings(mode, false)
		_ = km.ShortHelp()
		_ = km.FullHelp()
	}
}

func TestHelpBindings_BrowseShowsHistoryLabel(t *testing.T) {
	// Given: help bindings for browse mode showing ready beads
	km := HelpBindings(ModeBrowse, false)
	bindings := km.ShortHelp()

	// Then: h key shows "history" label
	for _, b := range bindings {
		if containsKey(b.Keys(), "h") {
			if b.Help().Desc != "history" {
				t.Errorf("h key desc = %q, want %q", b.Help().Desc, "history")
			}
			return
		}
	}
	t.Error("browse help should contain 'h' key")
}

func TestHelpBindings_BrowseClosedShowsReadyLabel(t *testing.T) {
	// Given: help bindings for browse mode showing closed beads
	km := HelpBindings(ModeBrowse, true)
	bindings := km.ShortHelp()

	// Then: h key shows "ready" label
	for _, b := range bindings {
		if containsKey(b.Keys(), "h") {
			if b.Help().Desc != "ready" {
				t.Errorf("h key desc = %q, want %q", b.Help().Desc, "ready")
			}
			return
		}
	}
	t.Error("browse help should contain 'h' key")
}

// Verify our key map types satisfy help.KeyMap at compile time.
var (
	_ help.KeyMap = browseKeys{}
	_ help.KeyMap = pipelineKeys{}
	_ help.KeyMap = summaryKeys{}
	_ help.KeyMap = campaignKeys{}
)
