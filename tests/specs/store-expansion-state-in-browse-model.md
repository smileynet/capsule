# Test Specification: Store expansion state in browse model

## Tracer
State persistence - proves we can remember expansion across rebuilds

## Test Cases

| Input | Expected Output | Notes |
|-------|-----------------|-------|
| Expand node, rebuild tree | Node remains expanded | State persists |
| Collapse node, rebuild tree | Node remains collapsed | State persists |
| Expand multiple nodes | All expansions tracked | Multiple state |
| Remove bead, rebuild tree | Removed bead ID cleaned from map | Cleanup |

## Edge Cases
- [ ] Bead ID no longer exists after refresh (remove from expandedIDs)
- [ ] New bead added (use default expansion)
- [ ] Bead ID changes (treat as new bead)
- [ ] Empty expandedIDs map (all use defaults)

## Implementation Notes
- Add expandedIDs map[string]bool to browseState in internal/dashboard/browse.go
- Update expand handler to set expandedIDs[beadID] = true
- Update collapse handler to set expandedIDs[beadID] = false
- Update buildTree to read from expandedIDs when setting initial expanded state
- Clean up expandedIDs for beads that no longer exist
