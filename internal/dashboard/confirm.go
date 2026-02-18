package dashboard

import (
	"fmt"
	"strings"
)

// confirmChild represents a child task in the confirmation screen.
type confirmChild struct {
	ID    string
	Title string
}

// confirmState holds the data needed for the confirmation screen.
type confirmState struct {
	beadID        string
	beadType      string
	beadTitle     string
	children      []confirmChild
	hasValidation bool
}

// View renders the confirmation screen for the given dimensions.
func (cs confirmState) View(width, height int) string {
	var b strings.Builder

	if cs.isCampaign() {
		cs.viewCampaign(&b)
	} else {
		cs.viewPipeline(&b)
	}

	b.WriteString("\n\n  [Enter] Confirm   [Esc] Cancel")
	return b.String()
}

func (cs confirmState) isCampaign() bool {
	return (cs.beadType == "feature" || cs.beadType == "epic") && len(cs.children) > 0
}

func (cs confirmState) viewPipeline(b *strings.Builder) {
	fmt.Fprintf(b, "Run pipeline for %s?\n", cs.beadID)
	fmt.Fprintf(b, "\n  %s\n", cs.beadTitle)
	b.WriteString("\n  This will:")
	b.WriteString("\n  • Create a worktree branch")
	b.WriteString("\n  • Run pipeline phases")
	b.WriteString("\n  • Auto-merge to main on success")
}

func (cs confirmState) viewCampaign(b *strings.Builder) {
	taskCount := len(cs.children)
	if cs.hasValidation {
		fmt.Fprintf(b, "Run campaign for %s? (%d tasks + validation)\n", cs.beadID, taskCount)
	} else {
		fmt.Fprintf(b, "Run campaign for %s? (%d tasks)\n", cs.beadID, taskCount)
	}
	fmt.Fprintf(b, "\n  %s\n", cs.beadTitle)

	if cs.hasValidation {
		b.WriteString("\n  Step 1 — Run open tasks sequentially:")
	} else {
		b.WriteString("\n  Run open tasks sequentially:")
	}
	for i, child := range cs.children {
		fmt.Fprintf(b, "\n    %d. %s — %s", i+1, child.ID, child.Title)
	}

	if cs.hasValidation {
		fmt.Fprintf(b, "\n\n  Step 2 — Feature validation:")
		fmt.Fprintf(b, "\n    Run acceptance pipeline on %s", cs.beadID)
	}
}

// countOpenChildren counts how many open direct children a parent has in the
// flat node list. Used by the help bar to show "run campaign (N tasks)".
func countOpenChildren(nodes []flatNode, parentID string) int {
	return len(collectOpenChildren(nodes, parentID))
}

// collectOpenChildren walks the browse tree's flatNodes to find open children
// of parentID. Only direct children (depth = parentDepth+1) are collected.
func collectOpenChildren(nodes []flatNode, parentID string) []confirmChild {
	// Find the parent node index.
	parentIdx := -1
	parentDepth := -1
	for i, fn := range nodes {
		if fn.Node.Bead.ID == parentID {
			parentIdx = i
			parentDepth = fn.Depth
			break
		}
	}
	if parentIdx < 0 {
		return nil
	}

	var children []confirmChild
	for i := parentIdx + 1; i < len(nodes); i++ {
		fn := nodes[i]
		if fn.Depth <= parentDepth {
			break // Exited the subtree.
		}
		// Only direct children, not grandchildren.
		if fn.Depth != parentDepth+1 {
			continue
		}
		if fn.Node.Bead.Closed {
			continue
		}
		children = append(children, confirmChild{
			ID:    fn.Node.Bead.ID,
			Title: fn.Node.Bead.Title,
		})
	}
	return children
}
