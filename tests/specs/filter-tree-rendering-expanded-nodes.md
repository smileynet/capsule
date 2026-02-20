# Test Specification: Filter tree rendering to only show expanded nodes

## Tracer
Rendering - proves collapsed nodes hide their children

## Test Cases

| Input | Expected Output | Notes |
|-------|-----------------|-------|
| flattenTree with all expanded | All nodes in output | Baseline |
| flattenTree with collapsed epic | Epic shown, children hidden | Collapse works |
| flattenTree with collapsed feature | Feature shown, children hidden | Feature collapse |
| flattenTree with mixed states | Only expanded subtrees visible | Mixed state |
| Depth tracking after collapse | Correct indentation preserved | Depth tracking |

## Edge Cases
- [ ] Collapsed node at end of tree
- [ ] All nodes collapsed (only roots visible)
- [ ] Deeply nested collapsed nodes (3+ levels)
- [ ] Collapsed node with no children (should still appear)

## Implementation Notes
- Update flattenTree in internal/dashboard/tree.go
- Skip children of collapsed nodes during traversal
- Preserve depth tracking for proper indentation
- Update prefix generation for collapsed subtrees
