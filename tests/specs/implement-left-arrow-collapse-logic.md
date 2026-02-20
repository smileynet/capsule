# Test Specification: Implement left arrow collapse logic

## Tracer
Collapse action - proves left arrow collapses and navigates to parent

## Test Cases

| Input | Expected Output | Notes |
|-------|-----------------|-------|
| Left arrow on expanded epic | Epic collapses, cursor stays | Primary case |
| Left arrow on collapsed epic | Cursor moves to parent | Navigate up |
| Left arrow on leaf task | Cursor moves to parent | Navigate up |
| 'h' key on expanded epic | Same as left arrow | Vim binding |
| Left arrow on root node | No-op, cursor stays | No parent |

## Edge Cases
- [ ] Left arrow on root-level node (no parent)
- [ ] Left arrow when cursor on first child of parent
- [ ] Left arrow on deeply nested node (should find parent)
- [ ] Left arrow when parent is not visible (shouldn't happen, but handle)

## Implementation Notes
- Add "left", "h" case to browse handleKey in internal/dashboard/browse.go
- If node is expanded: set expanded=false, cursor stays
- If node is collapsed or leaf: find parent in flatNodes, move cursor to it
- Handle edge case: root node has no parent (no-op)
- Return updated browseState
