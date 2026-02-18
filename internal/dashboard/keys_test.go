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
	expected := []string{"up", "down", "enter", "tab", "r", "q"}
	for _, want := range expected {
		if !containsKey(allKeys, want) {
			t.Errorf("BrowseKeyMap missing key %q, got %v", want, allKeys)
		}
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

func TestConfirmKeys_ContainsExpected(t *testing.T) {
	// Given: the confirm key map
	km := ConfirmKeyMap()
	bindings := km.ShortHelp()
	allKeys := collectKeys(bindings)

	// Then: enter and esc are present
	expected := []string{"enter", "esc"}
	for _, want := range expected {
		if !containsKey(allKeys, want) {
			t.Errorf("ConfirmKeyMap missing key %q, got %v", want, allKeys)
		}
	}
}

func TestConfirmKeys_NoQuit(t *testing.T) {
	// Given: the confirm key map
	km := ConfirmKeyMap()
	bindings := km.ShortHelp()
	allKeys := collectKeys(bindings)

	// Then: q is not included (Esc is the cancel key)
	if containsKey(allKeys, "q") {
		t.Error("ConfirmKeyMap should not contain 'q' key")
	}
}

func TestBrowseKeyMapForBead_Task(t *testing.T) {
	// Given: a task bead type
	km := BrowseKeyMapForBead("task", 0)

	// Then: Enter label says "run pipeline"
	h := km.Enter.Help()
	if !containsText(h.Desc, "run pipeline") {
		t.Errorf("task Enter desc = %q, want 'run pipeline'", h.Desc)
	}
}

func TestBrowseKeyMapForBead_FeatureWithChildren(t *testing.T) {
	// Given: a feature bead type with 4 children
	km := BrowseKeyMapForBead("feature", 4)

	// Then: Enter label says "run campaign (4 tasks)"
	h := km.Enter.Help()
	if !containsText(h.Desc, "run campaign (4 tasks)") {
		t.Errorf("feature Enter desc = %q, want 'run campaign (4 tasks)'", h.Desc)
	}
}

func TestBrowseKeyMapForBead_EpicWithChildren(t *testing.T) {
	// Given: an epic bead type with 2 children
	km := BrowseKeyMapForBead("epic", 2)

	// Then: Enter label says "run campaign (2 tasks)"
	h := km.Enter.Help()
	if !containsText(h.Desc, "run campaign (2 tasks)") {
		t.Errorf("epic Enter desc = %q, want 'run campaign (2 tasks)'", h.Desc)
	}
}

func TestBrowseKeyMapForBead_FeatureNoChildren(t *testing.T) {
	// Given: a feature bead type with no children
	km := BrowseKeyMapForBead("feature", 0)

	// Then: Enter label says "run pipeline" (no children = pipeline)
	h := km.Enter.Help()
	if !containsText(h.Desc, "run pipeline") {
		t.Errorf("feature (no children) Enter desc = %q, want 'run pipeline'", h.Desc)
	}
}

func TestPipelineSummaryKeyMap_WithPostPipeline(t *testing.T) {
	// Given: summary key map with post-pipeline configured
	km := PipelineSummaryKeyMap(true)
	bindings := km.ShortHelp()

	// Then: the label includes "merge + close"
	h := bindings[0].Help()
	if !containsText(h.Desc, "merge + close") {
		t.Errorf("summary with postPipeline desc = %q, want 'merge + close'", h.Desc)
	}
}

func TestPipelineSummaryKeyMap_WithoutPostPipeline(t *testing.T) {
	// Given: summary key map without post-pipeline
	km := PipelineSummaryKeyMap(false)
	bindings := km.ShortHelp()

	// Then: the label just says "continue"
	h := bindings[0].Help()
	if h.Desc != "continue" {
		t.Errorf("summary without postPipeline desc = %q, want 'continue'", h.Desc)
	}
}

func containsKey(keys []string, want string) bool {
	for _, k := range keys {
		if k == want {
			return true
		}
	}
	return false
}
