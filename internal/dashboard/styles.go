package dashboard

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// MinLeftWidth is the minimum character width for the left pane.
const MinLeftWidth = 28

// Priority badge colors indexed by priority level (0-4).
// P0=red, P1=orange, P2=yellow, P3=blue, P4=gray.
var priorityColors = [5]lipgloss.AdaptiveColor{
	{Light: "1", Dark: "9"},     // P0: red
	{Light: "208", Dark: "208"}, // P1: orange
	{Light: "3", Dark: "11"},    // P2: yellow
	{Light: "4", Dark: "12"},    // P3: blue
	{Light: "240", Dark: "245"}, // P4: gray
}

// PriorityBadge returns a styled priority label like "P0", "P2", etc.
func PriorityBadge(priority int) string {
	label := fmt.Sprintf("P%d", priority)
	if priority < 0 || priority > 4 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "240", Dark: "245"}).
			Render(label)
	}
	return lipgloss.NewStyle().
		Foreground(priorityColors[priority]).
		Render(label)
}

// FocusedBorder returns a lipgloss style with an accent-colored rounded border.
func FocusedBorder() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.AdaptiveColor{Light: "4", Dark: "12"})
}

// UnfocusedBorder returns a lipgloss style with a dim rounded border.
func UnfocusedBorder() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.AdaptiveColor{Light: "240", Dark: "240"})
}

// PaneWidths calculates the left and right pane widths from a total width.
// Left pane gets 1/3 (minimum MinLeftWidth), right pane gets the rest.
func PaneWidths(totalWidth int) (left, right int) {
	if totalWidth <= 0 {
		return 0, 0
	}
	left = totalWidth / 3
	if left < MinLeftWidth {
		left = MinLeftWidth
	}
	right = totalWidth - left
	if right < 0 {
		right = 0
	}
	return left, right
}
