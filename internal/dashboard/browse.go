package dashboard

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// CursorMarker is the prefix shown on the selected bead row.
const CursorMarker = "▸ "

// closedBeadLimit is the maximum number of closed beads to fetch.
const closedBeadLimit = 50

// browseState manages the bead list, cursor, and loading/error states
// for browse mode's left pane. Shows all beads (open + closed) in a tree.
type browseState struct {
	roots       []*treeNode // Tree structure for accessing children
	flatNodes   []flatNode
	cursor      int
	loading     bool
	err         error
	expandedIDs map[string]bool // Tracks which nodes are expanded
}

// newBrowseState returns a browseState in the loading state.
func newBrowseState() browseState {
	return browseState{
		loading:     true,
		expandedIDs: make(map[string]bool),
	}
}

// initBrowse returns a tea.Cmd that fetches both ready and closed beads,
// merges them, and wraps the result in a BeadListMsg.
func initBrowse(lister BeadLister) tea.Cmd {
	return func() tea.Msg {
		ready, err := lister.Ready()
		if err != nil {
			return BeadListMsg{Err: err}
		}
		closed, err := lister.Closed(closedBeadLimit)
		if err != nil {
			// Closed fetch failure is non-fatal; show ready beads only.
			return BeadListMsg{Beads: ready}
		}
		// Mark closed beads and merge.
		for i := range closed {
			closed[i].Closed = true
		}
		merged := mergeBeads(ready, closed)
		return BeadListMsg{Beads: merged}
	}
}

// mergeBeads combines ready and closed bead lists, deduplicating by ID.
// Ready beads take precedence over closed beads with the same ID.
func mergeBeads(ready, closed []BeadSummary) []BeadSummary {
	seen := make(map[string]bool, len(ready))
	merged := make([]BeadSummary, 0, len(ready)+len(closed))
	for _, b := range ready {
		seen[b.ID] = true
		merged = append(merged, b)
	}
	for _, b := range closed {
		if !seen[b.ID] {
			merged = append(merged, b)
		}
	}
	return merged
}

// Update processes messages for the browse state.
func (bs browseState) Update(msg tea.Msg) (browseState, tea.Cmd) {
	switch msg := msg.(type) {
	case BeadListMsg:
		return bs.applyBeadList(msg.Beads, msg.Err), nil

	case tea.KeyMsg:
		if bs.loading {
			return bs, nil
		}
		return bs.handleKey(msg)
	}

	return bs, nil
}

// applyBeadList builds a tree from the merged bead list and flattens it.
func (bs browseState) applyBeadList(beads []BeadSummary, err error) browseState {
	bs.loading = false
	if err != nil {
		bs.err = err
		bs.roots = nil
		bs.flatNodes = nil
		return bs
	}
	bs.err = nil
	bs.roots = buildTree(beads, bs.expandedIDs)
	bs.flatNodes = flattenTree(bs.roots)
	// Clamp cursor to valid range after tree rebuild
	if bs.cursor >= len(bs.flatNodes) {
		bs.cursor = len(bs.flatNodes) - 1
	}
	if bs.cursor < 0 && len(bs.flatNodes) > 0 {
		bs.cursor = 0
	}
	// Clean up expandedIDs for beads that no longer exist
	validIDs := make(map[string]bool)
	for _, b := range beads {
		validIDs[b.ID] = true
	}
	for id := range bs.expandedIDs {
		if !validIDs[id] {
			delete(bs.expandedIDs, id)
		}
	}
	return bs
}

func (bs browseState) handleKey(msg tea.KeyMsg) (browseState, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if len(bs.flatNodes) > 0 {
			bs.cursor--
			if bs.cursor < 0 {
				bs.cursor = len(bs.flatNodes) - 1
			}
		}
		return bs, nil

	case "down", "j":
		if len(bs.flatNodes) > 0 {
			bs.cursor++
			if bs.cursor >= len(bs.flatNodes) {
				bs.cursor = 0
			}
		}
		return bs, nil

	case "right", "l":
		if len(bs.flatNodes) > 0 && bs.cursor < len(bs.flatNodes) {
			node := bs.flatNodes[bs.cursor].Node
			if isExpandable(node) {
				if node.expanded {
					// Collapse: hide children
					bs.expandedIDs[node.Bead.ID] = false
					node.expanded = false
					bs.flatNodes = flattenTree(bs.roots)
					// Clamp cursor after collapse
					if bs.cursor >= len(bs.flatNodes) {
						bs.cursor = len(bs.flatNodes) - 1
					}
				} else {
					// Expand: show children
					bs.expandedIDs[node.Bead.ID] = true
					node.expanded = true
					bs.flatNodes = flattenTree(bs.roots)
					// Clamp cursor after expand
					if bs.cursor >= len(bs.flatNodes) {
						bs.cursor = len(bs.flatNodes) - 1
					}
					// Move cursor to first child if there are open children
					hasOpenChildren := openChildCount(node) > 0
					if hasOpenChildren && bs.cursor+1 < len(bs.flatNodes) {
						bs.cursor++
					}
				}
			}
		}
		return bs, nil

	case "left", "h":
		if len(bs.flatNodes) > 0 && bs.cursor < len(bs.flatNodes) {
			currentNode := bs.flatNodes[bs.cursor].Node
			currentID := currentNode.Bead.ID

			// Find parent ID
			parentID := findParentID(currentID)
			if parentID == "" {
				// Root node, no-op
				return bs, nil
			}

			// Find parent in flatNodes
			for i, fn := range bs.flatNodes {
				if fn.Node.Bead.ID == parentID {
					bs.cursor = i
					break
				}
			}
		}
		return bs, nil

	case "enter":
		if len(bs.flatNodes) > 0 && bs.cursor < len(bs.flatNodes) {
			node := bs.flatNodes[bs.cursor].Node
			if node.Bead.Closed {
				return bs, nil // Block dispatch on closed items.
			}
			selected := node.Bead
			return bs, func() tea.Msg {
				return ConfirmRequestMsg{BeadID: selected.ID, BeadType: selected.Type, BeadTitle: selected.Title}
			}
		}
		return bs, nil

	case "r":
		bs.loading = true
		bs.err = nil
		return bs, func() tea.Msg { return RefreshBeadsMsg{} }
	}

	return bs, nil
}

// findParentID returns the parent ID for a given bead ID, or "" if it's a root.
// Example: "demo-1.1.2" -> "demo-1.1", "demo-1" -> ""
func findParentID(id string) string {
	for i := len(id) - 1; i >= 0; i-- {
		if id[i] == '.' {
			return id[:i]
		}
	}
	return ""
}

// SelectedID returns the bead ID at the current cursor position,
// or "" if the list is empty or still loading.
func (bs browseState) SelectedID() string {
	if len(bs.flatNodes) == 0 || bs.cursor < 0 || bs.cursor >= len(bs.flatNodes) {
		return ""
	}
	return bs.flatNodes[bs.cursor].Node.Bead.ID
}

// SelectedBead returns the BeadSummary at the current cursor position.
func (bs browseState) SelectedBead() (BeadSummary, bool) {
	if len(bs.flatNodes) == 0 || bs.cursor < 0 || bs.cursor >= len(bs.flatNodes) {
		return BeadSummary{}, false
	}
	return bs.flatNodes[bs.cursor].Node.Bead, true
}

// View renders the browse pane content for the given dimensions.
// spinnerView is the current spinner frame (may be empty when spinner is inactive).
func (bs browseState) View(width, height int, spinnerView string) string {
	if bs.loading {
		return fmt.Sprintf("%s Loading beads...", spinnerView)
	}

	if bs.err != nil {
		return fmt.Sprintf("Error: %s\n\nPress r to retry", bs.err)
	}

	if len(bs.flatNodes) == 0 {
		return "No beads — press r to refresh"
	}

	var b strings.Builder
	for i, fn := range bs.flatNodes {
		if i > 0 {
			b.WriteByte('\n')
		}

		// Cursor marker.
		if i == bs.cursor {
			b.WriteString(CursorMarker)
		} else {
			b.WriteString("  ")
		}

		bead := fn.Node.Bead
		hasChildren := len(fn.Node.Children) > 0

		// Tree prefix (box-drawing).
		b.WriteString(fn.Prefix)

		// Expand/collapse indicator
		if hasChildren {
			if fn.Node.expanded {
				b.WriteString("▼ ")
			} else {
				b.WriteString("▶ ")
			}
			// Child count badge [N]
			openCount := openChildCount(fn.Node)
			b.WriteString(fmt.Sprintf("[%d] ", openCount))
		} else {
			b.WriteString("• ")
		}

		if bead.Closed {
			// Closed items: dim text with check symbol, no priority badge.
			line := fmt.Sprintf("%s %s %s", bead.ID, SymbolCheck, bead.Title)
			if bead.Type != "" {
				line += " [" + bead.Type + "]"
			}
			if hasChildren {
				stats := treeProgress(fn.Node)
				line += fmt.Sprintf(" %d/%d", stats.Closed, stats.Total)
			}
			b.WriteString(dimStyle.Render(line))
		} else {
			// Open items: normal text with priority badge.
			b.WriteString(bead.ID)
			b.WriteString(" ")
			b.WriteString(PriorityBadge(bead.Priority))
			b.WriteString(" ")
			b.WriteString(bead.Title)
			if bead.Type != "" {
				b.WriteString(" [" + bead.Type + "]")
			}
			if hasChildren {
				stats := treeProgress(fn.Node)
				progress := fmt.Sprintf(" %d/%d", stats.Closed, stats.Total)
				if stats.Closed == stats.Total && stats.Total > 0 {
					progress += " " + successStyle.Render(SymbolCheck)
				}
				b.WriteString(progress)
			}
		}

		// Add placeholder if this node is expanded with no open children
		if hasChildren && fn.Node.expanded && openChildCount(fn.Node) == 0 {
			b.WriteByte('\n')
			b.WriteString("  ") // No cursor marker for placeholder

			// Build child prefix
			var childPrefix string
			if fn.Depth == 0 {
				childPrefix = ""
			} else {
				if fn.Node.IsLast {
					childPrefix = fn.Prefix[:len(fn.Prefix)-4] + "    "
				} else {
					childPrefix = fn.Prefix[:len(fn.Prefix)-4] + "│   "
				}
			}
			b.WriteString(childPrefix)
			b.WriteString("└── ")
			b.WriteString(dimStyle.Render("(no open tasks)"))
		}
	}
	return b.String()
}
