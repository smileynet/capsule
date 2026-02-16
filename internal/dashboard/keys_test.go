package dashboard

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
)

func TestBrowseKeys_ContainsExpected(t *testing.T) {
	// Given: the browse key map
	km := BrowseKeyMap()
	bindings := km.ShortHelp()
	allKeys := collectKeys(bindings)

	// Then: all expected navigation and action keys are present
	expected := []string{"up", "down", "enter", "tab", "r", "h", "q"}
	for _, want := range expected {
		if !containsKey(allKeys, want) {
			t.Errorf("BrowseKeyMap missing key %q, got %v", want, allKeys)
		}
	}
}

func TestBrowseKeys_HistoryHelp(t *testing.T) {
	// Given: the browse key map
	km := BrowseKeyMap()

	// Then: the History binding has appropriate help text
	h := km.History.Help()
	if h.Key != "h" {
		t.Errorf("History key help = %q, want %q", h.Key, "h")
	}
	if h.Desc != "history" {
		t.Errorf("History desc = %q, want %q", h.Desc, "history")
	}
}

func TestPipelineKeys_ContainsExpected(t *testing.T) {
	// Given: the pipeline key map
	km := PipelineKeyMap()
	bindings := km.ShortHelp()
	allKeys := collectKeys(bindings)

	// Then: all expected navigation keys are present
	expected := []string{"up", "down", "tab", "q"}
	for _, want := range expected {
		if !containsKey(allKeys, want) {
			t.Errorf("PipelineKeyMap missing key %q, got %v", want, allKeys)
		}
	}
}

func TestSummaryKeys_ContainsAnyKey(t *testing.T) {
	// Given: the summary key map
	km := SummaryKeyMap()
	bindings := km.ShortHelp()

	// Then: at least one binding exists
	if len(bindings) == 0 {
		t.Fatal("SummaryKeyMap returned no bindings")
	}

	// And: an "any key" binding is present in help text
	found := false
	for _, b := range bindings {
		h := b.Help()
		if containsText(h.Key, "any") || containsText(h.Desc, "any") {
			found = true
			break
		}
	}
	if !found {
		t.Error("SummaryKeyMap should contain an 'any key' binding")
	}
}

func TestPipelineKeys_NoEnter(t *testing.T) {
	// Given: the pipeline key map
	km := PipelineKeyMap()
	bindings := km.ShortHelp()
	allKeys := collectKeys(bindings)

	// Then: enter is not included (no dispatch in pipeline mode)
	if containsKey(allKeys, "enter") {
		t.Error("PipelineKeyMap should not contain 'enter' key")
	}
}

func TestCampaignKeys_ContainsExpected(t *testing.T) {
	// Given: the campaign key map
	km := CampaignKeyMap()
	bindings := km.ShortHelp()
	allKeys := collectKeys(bindings)

	// Then: all expected navigation keys are present (same as pipeline)
	expected := []string{"up", "down", "tab", "q"}
	for _, want := range expected {
		if !containsKey(allKeys, want) {
			t.Errorf("CampaignKeyMap missing key %q, got %v", want, allKeys)
		}
	}
}

func TestCampaignKeys_NoEnter(t *testing.T) {
	// Given: the campaign key map
	km := CampaignKeyMap()
	bindings := km.ShortHelp()
	allKeys := collectKeys(bindings)

	// Then: enter is not included (no dispatch in campaign mode)
	if containsKey(allKeys, "enter") {
		t.Error("CampaignKeyMap should not contain 'enter' key")
	}
}

// collectKeys extracts all key strings from a slice of key.Binding.
func collectKeys(bindings []key.Binding) []string {
	var keys []string
	for _, b := range bindings {
		keys = append(keys, b.Keys()...)
	}
	return keys
}

func containsKey(keys []string, want string) bool {
	for _, k := range keys {
		if k == want {
			return true
		}
	}
	return false
}
