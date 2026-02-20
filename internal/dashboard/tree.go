package dashboard

import (
	"sort"
)

// treeNode represents a bead and its children in a hierarchical tree.
type treeNode struct {
	Bead     BeadSummary
	Children []*treeNode
	IsLast   bool // true if this is the last child of its parent
	expanded bool // true if this node's children should be visible
}

// flatNode is a treeNode with pre-computed prefix strings for rendering.
type flatNode struct {
	Node   *treeNode
	Prefix string // box-drawing prefix, e.g. "├── ", "│   └── "
	Depth  int
}

// treeStats holds progress counts for a subtree.
type treeStats struct {
	Total  int
	Closed int
}

// buildTree groups beads into a hierarchy using ID prefix matching.
// A bead is a child of the bead with the longest matching ID prefix.
// Beads with no parent in the list become roots.
// Children are sorted by ID within each parent.
// expandedIDs is used to restore expansion state from previous builds.
func buildTree(beads []BeadSummary, expandedIDs map[string]bool) []*treeNode {
	// Build nodes indexed by ID.
	nodes := make(map[string]*treeNode, len(beads))
	for _, b := range beads {
		nodes[b.ID] = &treeNode{Bead: b}
	}

	// Collect sorted IDs for deterministic parent assignment.
	ids := make([]string, 0, len(beads))
	for _, b := range beads {
		ids = append(ids, b.ID)
	}
	sort.Strings(ids)

	// Assign children to their nearest parent.
	var roots []*treeNode
	for _, id := range ids {
		node := nodes[id]
		parentID := findParent(id, nodes)
		if parentID == "" {
			roots = append(roots, node)
		} else {
			parent := nodes[parentID]
			parent.Children = append(parent.Children, node)
		}
	}

	// Mark last children and sort children by ID at each level.
	for _, root := range roots {
		finalizeNode(root, expandedIDs)
	}
	return roots
}

// findParent returns the ID of the nearest ancestor in nodes, or "" if none.
func findParent(childID string, nodes map[string]*treeNode) string {
	// Walk up the ID hierarchy: "a.b.c" → check "a.b" → check "a".
	for i := len(childID) - 1; i >= 0; i-- {
		if childID[i] == '.' {
			candidate := childID[:i]
			if _, ok := nodes[candidate]; ok {
				return candidate
			}
		}
	}
	return ""
}

// finalizeNode sorts children by ID and marks the last child at each level.
// It also sets expansion state: restored from expandedIDs if present, otherwise
// defaults to epics expanded, features collapsed.
func finalizeNode(n *treeNode, expandedIDs map[string]bool) {
	// Set expansion state: check expandedIDs first, then fall back to default
	if len(n.Children) > 0 {
		if expanded, ok := expandedIDs[n.Bead.ID]; ok {
			n.expanded = expanded
		} else {
			n.expanded = n.Bead.Type == "epic"
		}
	}

	if len(n.Children) == 0 {
		return
	}
	sort.Slice(n.Children, func(i, j int) bool {
		return n.Children[i].Bead.ID < n.Children[j].Bead.ID
	})
	for i, child := range n.Children {
		child.IsLast = i == len(n.Children)-1
		finalizeNode(child, expandedIDs)
	}
}

// flattenTree converts a tree into a flat list with box-drawing prefixes.
func flattenTree(roots []*treeNode) []flatNode {
	var result []flatNode
	for i, root := range roots {
		root.IsLast = i == len(roots)-1
		result = flattenNode(root, "", 0, result)
	}
	return result
}

func flattenNode(n *treeNode, parentPrefix string, depth int, result []flatNode) []flatNode {
	var prefix string
	if depth == 0 {
		prefix = ""
	} else {
		if n.IsLast {
			prefix = parentPrefix + "└── "
		} else {
			prefix = parentPrefix + "├── "
		}
	}

	result = append(result, flatNode{
		Node:   n,
		Prefix: prefix,
		Depth:  depth,
	})

	// Skip children if node is collapsed
	if !n.expanded {
		return result
	}

	// Build the continuation prefix for children.
	var childPrefix string
	if depth == 0 {
		childPrefix = ""
	} else {
		if n.IsLast {
			childPrefix = parentPrefix + "    "
		} else {
			childPrefix = parentPrefix + "│   "
		}
	}

	for _, child := range n.Children {
		result = flattenNode(child, childPrefix, depth+1, result)
	}
	return result
}

// treeProgress computes the total and closed counts for a subtree rooted at node.
func treeProgress(node *treeNode) treeStats {
	if len(node.Children) == 0 {
		// Leaf nodes don't have progress — they are the items being counted.
		return treeStats{}
	}
	var stats treeStats
	for _, child := range node.Children {
		stats.Total++
		if child.Bead.Closed {
			stats.Closed++
		}
		// Add grandchildren counts.
		childStats := treeProgress(child)
		stats.Total += childStats.Total
		stats.Closed += childStats.Closed
	}
	return stats
}

// openChildCount returns the number of open (non-closed) direct children.
func openChildCount(node *treeNode) int {
	count := 0
	for _, child := range node.Children {
		if !child.Bead.Closed {
			count++
		}
	}
	return count
}

// isExpandable returns true if the node has children.
func isExpandable(node *treeNode) bool {
	return len(node.Children) > 0
}
