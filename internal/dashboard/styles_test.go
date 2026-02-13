package dashboard

import "testing"

func TestPriorityBadge_DoesNotPanic(t *testing.T) {
	// Verify badge rendering for all priorities doesn't panic.
	for p := 0; p <= 4; p++ {
		got := PriorityBadge(p)
		if got == "" {
			t.Errorf("PriorityBadge(%d) returned empty string", p)
		}
	}
}

func TestPriorityBadge_OutOfRange(t *testing.T) {
	// Out-of-range priority should still render without panicking.
	got := PriorityBadge(99)
	if got == "" {
		t.Error("PriorityBadge(99) returned empty string")
	}
}

func TestPriorityBadge_ContainsLabel(t *testing.T) {
	tests := []struct {
		priority int
		want     string
	}{
		{0, "P0"},
		{1, "P1"},
		{2, "P2"},
		{3, "P3"},
		{4, "P4"},
	}
	for _, tt := range tests {
		got := PriorityBadge(tt.priority)
		if !containsPlainText(got, tt.want) {
			t.Errorf("PriorityBadge(%d) = %q, want to contain %q", tt.priority, got, tt.want)
		}
	}
}

func TestPaneWidths_Normal(t *testing.T) {
	left, right := PaneWidths(90)
	if left != 30 {
		t.Errorf("left = %d, want 30 (1/3 of 90)", left)
	}
	if right != 60 {
		t.Errorf("right = %d, want 60 (2/3 of 90)", right)
	}
}

func TestPaneWidths_MinLeft(t *testing.T) {
	// When total width is small, left pane should be at minimum.
	left, right := PaneWidths(40)
	if left < MinLeftWidth {
		t.Errorf("left = %d, want >= %d", left, MinLeftWidth)
	}
	if left+right != 40 {
		t.Errorf("left+right = %d, want 40", left+right)
	}
}

func TestPaneWidths_VerySmall(t *testing.T) {
	// Width less than minimum: left gets the minimum, right gets clamped to 0.
	left, right := PaneWidths(20)
	if left != MinLeftWidth {
		t.Errorf("left = %d, want %d", left, MinLeftWidth)
	}
	if right != 0 {
		t.Errorf("right = %d, want 0", right)
	}
}

func TestPaneWidths_Zero(t *testing.T) {
	left, right := PaneWidths(0)
	if left != 0 {
		t.Errorf("left = %d, want 0", left)
	}
	if right != 0 {
		t.Errorf("right = %d, want 0", right)
	}
}

func TestFocusedBorder_DoesNotPanic(t *testing.T) {
	_ = FocusedBorder()
}

func TestUnfocusedBorder_DoesNotPanic(t *testing.T) {
	_ = UnfocusedBorder()
}
