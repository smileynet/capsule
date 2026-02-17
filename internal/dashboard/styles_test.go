package dashboard

import "testing"

func TestPriorityBadge_DoesNotPanic(t *testing.T) {
	// Given: all valid priority values (0-4)
	// When: PriorityBadge is called for each
	// Then: none panic and all return non-empty strings
	for p := 0; p <= 4; p++ {
		got := PriorityBadge(p)
		if got == "" {
			t.Errorf("PriorityBadge(%d) returned empty string", p)
		}
	}
}

func TestPriorityBadge_OutOfRange(t *testing.T) {
	// Given: an out-of-range priority value
	// When: PriorityBadge is called
	// Then: it returns a non-empty string without panicking
	got := PriorityBadge(99)
	if got == "" {
		t.Error("PriorityBadge(99) returned empty string")
	}
}

func TestPriorityBadge_ContainsLabel(t *testing.T) {
	// Given: each valid priority value
	// When: PriorityBadge is called
	// Then: the result contains the corresponding P<n> label
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
	// Given: a normal terminal width of 90
	// When: PaneWidths is computed
	left, right := PaneWidths(90)

	// Then: left is 1/3 and right is 2/3
	if left != 30 {
		t.Errorf("left = %d, want 30 (1/3 of 90)", left)
	}
	if right != 60 {
		t.Errorf("right = %d, want 60 (2/3 of 90)", right)
	}
}

func TestPaneWidths_MinLeft(t *testing.T) {
	// Given: a small terminal width of 40
	// When: PaneWidths is computed
	left, right := PaneWidths(40)

	// Then: left pane is at least MinLeftWidth and total equals input
	if left < MinLeftWidth {
		t.Errorf("left = %d, want >= %d", left, MinLeftWidth)
	}
	if left+right != 40 {
		t.Errorf("left+right = %d, want 40", left+right)
	}
}

func TestPaneWidths_VerySmall(t *testing.T) {
	// Given: a terminal width smaller than MinLeftWidth
	// When: PaneWidths is computed
	left, right := PaneWidths(20)

	// Then: left gets MinLeftWidth and right is clamped to 0
	if left != MinLeftWidth {
		t.Errorf("left = %d, want %d", left, MinLeftWidth)
	}
	if right != 0 {
		t.Errorf("right = %d, want 0", right)
	}
}

func TestPaneWidths_Zero(t *testing.T) {
	// Given: a zero terminal width
	// When: PaneWidths is computed
	left, right := PaneWidths(0)

	// Then: both panes are 0
	if left != 0 {
		t.Errorf("left = %d, want 0", left)
	}
	if right != 0 {
		t.Errorf("right = %d, want 0", right)
	}
}

func TestFocusedBorder_DoesNotPanic(t *testing.T) {
	// Given/When: FocusedBorder is called
	// Then: it does not panic
	_ = FocusedBorder()
}

func TestUnfocusedBorder_DoesNotPanic(t *testing.T) {
	// Given/When: UnfocusedBorder is called
	// Then: it does not panic
	_ = UnfocusedBorder()
}
