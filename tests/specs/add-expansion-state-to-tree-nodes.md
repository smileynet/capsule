# Test Specification: Add expansion state to tree nodes

## Tracer
Foundation - proves we can track which nodes are expanded

## Test Cases

| Input | Expected Output | Notes |
|-------|-----------------|-------|
| treeNode with expanded=true | Node tracks expansion state | Basic state tracking |
| treeNode with defaultExpanded=true | Node starts expanded | Default state |
| buildTree with epic node | Epic has expanded=true by default | Epic default |
| buildTree with feature node | Feature has expanded=false by default | Feature default |
| isExpandable(epic with children) | Returns true | Expandable check |
| isExpandable(task node) | Returns false | Leaf node check |

## Edge Cases
- [ ] Node with no children (should not be expandable)
- [ ] Node type not epic/feature/task (handle gracefully)
- [ ] Empty tree (no nodes)

## Implementation Notes
- Add `expanded` bool field to treeNode struct in internal/dashboard/tree.go
- Add `defaultExpanded` bool to control initial state
- Update buildTree to set default expansion based on node type
- Add helper function: isExpandable(node) returns true for epic/feature with children
