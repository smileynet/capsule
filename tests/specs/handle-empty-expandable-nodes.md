# Test Specification: Handle empty expandable nodes

## Tracer
Empty node handling - proves we handle epics/features with no open children

## Test Cases

| Input | Expected Output | Notes |
|-------|-----------------|-------|
| Expand empty epic | Shows "(no open tasks)" | Empty message |
| Empty epic child count | Badge shows [0] | Zero badge |
| Right arrow on empty epic | Expands but cursor stays | No children to move to |
| Collapse empty epic | Hides message | Collapse works |

## Edge Cases
- [ ] Epic with all closed children (should show as empty)
- [ ] Feature with no tasks created yet (empty)
- [ ] Empty epic at end of tree
- [ ] Multiple empty epics in a row

## Implementation Notes
- Detect when expandable node has zero open children
- Show "(no open tasks)" in tree view when expanded
- Right arrow on empty node: expand but don't move cursor
- Update child count badge to show [0]
- Handle in tree rendering and expand logic
