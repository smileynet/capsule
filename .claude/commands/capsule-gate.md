---
allowed-tools: Bash, Read, Glob, Grep
argument-hint: "<feature|epic>"
description: Run quality gates before closing a feature or epic
---

Run the quality gates required before closing a feature or epic in beads.

**Gate level:** `$ARGUMENTS` (required — must be `feature` or `epic`).

If `$ARGUMENTS` is empty or not one of `feature`/`epic`, ask the user which gate level to run.

## Gate definitions

**Feature gate:** `make lint` then `make test-full`

**Epic gate:** `make lint` then `make test-full` then `make smoke`

## Execution

Run each gate command sequentially. Stop on first failure.

Use a 300-second timeout for each gate command.

## Output format

~~~
CAPSULE GATE: $ARGUMENTS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Running <feature|epic> quality gates...

  [1/<N>] make lint .............. ✓ (0 issues)
  [2/<N>] make test-full ........ ✓ (<N> tests, <N> packages)
  [3/<N>] make smoke ............ ✓ (epic only)

───────────────────────────────────────────
RESULT
───────────────────────────────────────────

All <N> gates passed. ✓
~~~

Then find the relevant bead to close:
```bash
bd list --status=in_progress
```

~~~
Ready to close:
  <id> — <title> [<feature|epic>]

NEXT STEP: bd close <id>
~~~

**On failure:**
~~~
  [1/2] make lint .............. ✓
  [2/2] make test-full ......... ✗

───────────────────────────────────────────
RESULT
───────────────────────────────────────────

Gate failed: make test-full ✗

<Show the relevant test failure output — not the full log,
 just the failing test names and error messages.>

Fix the issues and re-run: /capsule-gate $ARGUMENTS
~~~
