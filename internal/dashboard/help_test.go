package dashboard

import (
	"testing"

	"github.com/charmbracelet/bubbles/help"
)

func TestHelpBindings_BrowseMode(t *testing.T) {
	// Given: help bindings for browse mode
	km := HelpBindings(ModeBrowse)
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
	km := HelpBindings(ModePipeline)
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
	km := HelpBindings(ModeSummary)
	bindings := km.ShortHelp()

	// Then: at least one binding is present
	if len(bindings) == 0 {
		t.Fatal("summary help should have at least one binding")
	}
}

func TestHelpBindings_ImplementsKeyMap(t *testing.T) {
	// Given: all dashboard modes
	modes := []Mode{ModeBrowse, ModePipeline, ModeSummary}

	// Then: each returns a type satisfying help.KeyMap (ShortHelp + FullHelp)
	for _, mode := range modes {
		km := HelpBindings(mode)
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
