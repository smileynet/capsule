# Test Specification: Add cursor bounds checking

## Tracer
Safety - proves cursor never goes out of bounds

## Test Cases

| Input | Expected Output | Notes |
|-------|-----------------|-------|
| Expand with cursor at end | Cursor clamped to valid range | Bounds check |
| Collapse last visible node | Cursor moves to valid position | Edge case |
| Collapse node cursor is on | Cursor stays on collapsed node | Self-collapse |
| Navigate after collapse | Cursor within [0, len-1] | Always valid |

## Edge Cases
- [ ] Collapse when cursor is on last child (cursor should move to parent)
- [ ] Expand when cursor at last position
- [ ] Collapse all nodes (cursor should be on first root)
- [ ] Single node tree (cursor always at 0)

## Implementation Notes
- Add bounds check after expand/collapse operations
- Clamp cursor to valid range [0, len(flatNodes)-1]
- Handle edge case: collapsing last visible node
- Add test: collapse node that cursor is on
- Ensure cursor is always valid after any operation
