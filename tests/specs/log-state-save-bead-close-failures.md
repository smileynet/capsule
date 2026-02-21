# Test Specification: Log state save and bead close failures

## Bead: cap-9f0.3.1

## Tracer
Foundation — proves errors are no longer silently swallowed.

## Context
- Add logger (`io.Writer`) to `campaign.Runner` via `Config`
- Replace all 7 `_ = r.store.Save(state)` with logged warnings
- Replace `_ = r.beads.Close(beadID)` in `runPostPipeline` with logged warning
- Campaign continues after save/close failure (best-effort, not fatal)

## Test Cases

| Input | Expected Output | Notes |
|-------|-----------------|-------|
| store.Save returns error | Warning written to logger | No longer silent |
| beads.Close returns error | Warning written to logger | No longer silent |
| store.Save fails, campaign continues | Next task still runs | Best-effort save |
| All 7 Save call sites fail | 7 distinct warnings logged | Full coverage |
| Logger is nil | No panic, warnings silently dropped | Nil-safe |
| Logger set to bytes.Buffer | Buffer contains warning text | Testable output |

## Edge Cases
- [ ] Save failure at circuit breaker trip (line 193)
- [ ] Save failure at context cancellation (line 218)
- [ ] Save failure at pipeline pause (line 225)
- [ ] Save failure at task failure in abort mode (line 236)
- [ ] Save failure at task failure in continue mode (line 240)
- [ ] Save failure after successful task state advance (line 251)
- [ ] Save failure at campaign completion (line 262)
- [ ] Warning message includes state ID and error text

## Implementation Notes
- Add `Logger io.Writer` field to `campaign.Config`
- Add helper: `func (r *Runner) logWarning(format string, args ...any)` that writes to Logger if non-nil
- Replace each `_ = r.store.Save(state)` with `if err := r.store.Save(state); err != nil { r.logWarning(...) }`
- Replace `_ = r.beads.Close(beadID)` similarly
- Test: inject a mock StateStore that returns errors, verify logger output
- Warning format: `"campaign: warning: save state %s: %v\n"` / `"campaign: warning: close bead %s: %v\n"`
