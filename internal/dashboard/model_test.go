package dashboard

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func newSizedModel(w, h int) Model {
	m := NewModel()
	updated, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return updated.(Model)
}

func TestNewModel_DefaultMode(t *testing.T) {
	m := NewModel()
	if m.mode != ModeBrowse {
		t.Errorf("mode = %d, want ModeBrowse (%d)", m.mode, ModeBrowse)
	}
}

func TestNewModel_DefaultFocus(t *testing.T) {
	m := NewModel()
	if m.focus != PaneLeft {
		t.Errorf("focus = %d, want PaneLeft (%d)", m.focus, PaneLeft)
	}
}

func TestModel_TabTogglesFocus(t *testing.T) {
	m := newSizedModel(90, 40)

	// Tab should switch from left to right.
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.focus != PaneRight {
		t.Errorf("after first Tab: focus = %d, want PaneRight (%d)", m.focus, PaneRight)
	}

	// Tab again should switch back to left.
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyTab})
	m = updated.(Model)
	if m.focus != PaneLeft {
		t.Errorf("after second Tab: focus = %d, want PaneLeft (%d)", m.focus, PaneLeft)
	}
}

func TestModel_QuitInBrowseMode(t *testing.T) {
	m := newSizedModel(90, 40)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("q in browse mode should return a quit command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("q command produced %T, want tea.QuitMsg", msg)
	}
}

func TestModel_CtrlCQuits(t *testing.T) {
	m := newSizedModel(90, 40)

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Fatal("ctrl+c should return a quit command")
	}
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("ctrl+c command produced %T, want tea.QuitMsg", msg)
	}
}

func TestModel_WindowSizeMsg(t *testing.T) {
	m := NewModel()

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 50})
	m = updated.(Model)

	if m.width != 120 {
		t.Errorf("width = %d, want 120", m.width)
	}
	if m.height != 50 {
		t.Errorf("height = %d, want 50", m.height)
	}
}

func TestModel_ModeRouting(t *testing.T) {
	tests := []struct {
		name string
		mode Mode
	}{
		{"browse", ModeBrowse},
		{"pipeline", ModePipeline},
		{"summary", ModeSummary},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newSizedModel(90, 40)
			m.mode = tt.mode

			view := m.View()
			if view == "" {
				t.Error("View() returned empty string")
			}
		})
	}
}

func TestModel_WindowResizeUpdatesLayout(t *testing.T) {
	m := NewModel()

	// Set initial size.
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	m = updated.(Model)
	if m.width != 80 || m.height != 30 {
		t.Errorf("after first resize: %dx%d, want 80x30", m.width, m.height)
	}

	// Resize again.
	updated, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 50})
	m = updated.(Model)
	if m.width != 120 || m.height != 50 {
		t.Errorf("after second resize: %dx%d, want 120x50", m.width, m.height)
	}
}

func TestModel_HelpBarReflectsMode(t *testing.T) {
	tests := []struct {
		name     string
		mode     Mode
		wantText string
	}{
		{"browse", ModeBrowse, "run pipeline"},
		{"pipeline", ModePipeline, "abort"},
		{"summary", ModeSummary, "continue"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newSizedModel(90, 40)
			m.mode = tt.mode

			view := m.View()
			if !containsPlainText(view, tt.wantText) {
				t.Errorf("View() should contain %q", tt.wantText)
			}
		})
	}
}
