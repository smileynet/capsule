# Test Specification: Restore expansion state on refresh

## Tracer
Refresh preservation - proves state survives bead list reload

## Test Cases

| Input | Expected Output | Notes |
|-------|-----------------|-------|
| Refresh with expanded nodes | Nodes remain expanded | State preserved |
| Refresh with collapsed nodes | Nodes remain collapsed | State preserved |
| Refresh with mixed states | All states preserved | Mixed state |
| Refresh after dispatch | Cursor snaps to dispatched bead | Cursor restore |

## Edge Cases
- [ ] Refresh when bead list is empty
- [ ] Refresh when all beads are closed
- [ ] Refresh when cursor bead no longer exists (move to first)
- [ ] Refresh during pipeline run (should still work)

## Implementation Notes
- Update BeadListMsg handler in internal/dashboard/model.go
- Preserve expandedIDs map across bead list reload
- Rebuild tree with preserved expansion state
- Restore cursor to lastDispatchedID if set
- Test with mixed expanded/collapsed nodes
