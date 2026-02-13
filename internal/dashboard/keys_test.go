package dashboard

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
)

func TestBrowseKeys_ContainsExpected(t *testing.T) {
	km := BrowseKeyMap()
	bindings := km.ShortHelp()
	allKeys := collectKeys(bindings)

	expected := []string{"up", "down", "enter", "tab", "r", "q"}
	for _, want := range expected {
		if !containsKey(allKeys, want) {
			t.Errorf("BrowseKeyMap missing key %q, got %v", want, allKeys)
		}
	}
}

func TestPipelineKeys_ContainsExpected(t *testing.T) {
	km := PipelineKeyMap()
	bindings := km.ShortHelp()
	allKeys := collectKeys(bindings)

	expected := []string{"up", "down", "tab", "q"}
	for _, want := range expected {
		if !containsKey(allKeys, want) {
			t.Errorf("PipelineKeyMap missing key %q, got %v", want, allKeys)
		}
	}
}

func TestSummaryKeys_ContainsAnyKey(t *testing.T) {
	km := SummaryKeyMap()
	bindings := km.ShortHelp()
	if len(bindings) == 0 {
		t.Fatal("SummaryKeyMap returned no bindings")
	}
	// Summary mode should have an "any key" binding help text.
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
	km := PipelineKeyMap()
	bindings := km.ShortHelp()
	allKeys := collectKeys(bindings)

	if containsKey(allKeys, "enter") {
		t.Error("PipelineKeyMap should not contain 'enter' key")
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
