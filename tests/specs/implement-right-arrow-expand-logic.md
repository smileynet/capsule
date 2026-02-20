# Test Specification: Implement right arrow expand logic

## Tracer
Expand action - proves right arrow expands and navigates

## Test Cases

| Input | Expected Output | Notes |
|-------|-----------------|-------|
| Right arrow on collapsed epic | Epic expands, cursor moves to first child | Primary case |
| Right arrow on expanded epic | Cursor moves to first child | Already expanded |
| Right arrow on leaf task | No-op, cursor stays | Leaf node |
| 'l' key on collapsed epic | Same as right arrow | Vim binding |
| Right arrow on collapsed feature | Feature expands, cursor to child | Feature node |

## Edge Cases
- [ ] Right arrow on node with no children (expand but don't move cursor)
- [ ] Right arrow when cursor at last node
- [ ] Right arrow on closed bead (should not expand)
- [ ] Rapid right arrow presses (debounce not needed, but test)

## Implementation Notes
- Add "right", "l" case to browse handleKey in internal/dashboard/browse.go
- Check if node is expandable using isExpandable helper
- If collapsed: set expanded=true, move cursor to first child
- If expanded: just move cursor to first child
- If leaf: no-op
- Return updated browseState
