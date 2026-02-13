package dashboard

import (
	"testing"

	"github.com/charmbracelet/bubbles/help"
)

func TestHelpBindings_BrowseMode(t *testing.T) {
	km := HelpBindings(ModeBrowse)
	bindings := km.ShortHelp()
	allKeys := collectKeys(bindings)

	if !containsKey(allKeys, "enter") {
		t.Error("browse help should contain 'enter' key")
	}
	if !containsKey(allKeys, "q") {
		t.Error("browse help should contain 'q' key")
	}
}

func TestHelpBindings_PipelineMode(t *testing.T) {
	km := HelpBindings(ModePipeline)
	bindings := km.ShortHelp()
	allKeys := collectKeys(bindings)

	if containsKey(allKeys, "enter") {
		t.Error("pipeline help should not contain 'enter' key")
	}
	if !containsKey(allKeys, "q") {
		t.Error("pipeline help should contain 'q' key")
	}
}

func TestHelpBindings_SummaryMode(t *testing.T) {
	km := HelpBindings(ModeSummary)
	bindings := km.ShortHelp()

	if len(bindings) == 0 {
		t.Fatal("summary help should have at least one binding")
	}
}

func TestHelpBindings_ImplementsKeyMap(t *testing.T) {
	// All modes should return types that satisfy help.KeyMap interface
	// by having both ShortHelp and FullHelp methods.
	modes := []Mode{ModeBrowse, ModePipeline, ModeSummary}
	for _, mode := range modes {
		km := HelpBindings(mode)
		// ShortHelp and FullHelp are the help.KeyMap interface methods.
		_ = km.ShortHelp()
		_ = km.FullHelp()
	}
}

// Verify our key map types satisfy help.KeyMap at compile time.
var (
	_ help.KeyMap = browseKeys{}
	_ help.KeyMap = pipelineKeys{}
	_ help.KeyMap = summaryKeys{}
)
