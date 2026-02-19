# Epic Acceptance: cap-10s

## Multi-CLI Support

**Status:** Accepted
**Date:** 2026-02-19

## What This Delivers

Capsule now supports multiple AI CLI tools beyond Claude Code. A config-driven GenericProvider with built-in presets enables users to run pipelines with different providers via `capsule run <bead-id> --provider=kiro`. Provider-specific command building, ANSI stripping, and flag management are handled transparently.

## Features Accepted

| # | Feature | Summary |
|---|---------|---------|
| 1 | cap-10s.1: Generic CLI Provider with Kiro preset | Config-driven GenericProvider replacing Claude-specific provider, with built-in presets for Claude and Kiro CLI |
| 2 | cap-10s.2: Provider-specific prompt adaptation | Placeholder for future prompt adaptation per provider capabilities |

## End-to-End Verification

The following user journeys are validated at the binary level via smoke tests:

- **Kiro provider pipeline**: `capsule run <bead-id> --provider=kiro` with mock kiro-cli completes all 6 phases, merges to main, and closes bead (TestSmoke_KiroProvider)
- **Default provider pipeline**: `capsule run <bead-id>` with default claude provider completes full pipeline (TestSmoke_GreenfieldNarrative)
- **Unknown provider error**: `capsule run <bead-id> --provider=nonexistent` exits code 2 with error listing available providers including both "claude" and "kiro"
- **Provider flag with --no-tui**: `capsule run <bead-id> --no-tui --provider=nonexistent` accepts flag combination

Unit-level validation (internal/provider/):
- GenericProvider builds correct args for claude preset (flag-based prompt: `-p <prompt>`)
- GenericProvider builds correct args for kiro preset (positional prompt: `chat --trust-all-tools <prompt>`)
- Registry registers both claude and kiro via RegisterBuiltins
- ANSI stripping enabled for kiro, disabled for claude
- Timeout, error wrapping, and subprocess execution covered

```bash
# Run all verification
make lint && make test-full && make smoke
```

## Out of Scope

- Live testing with real Kiro CLI installation (validated via mock kiro-cli only)
- Provider-specific prompt adaptation beyond placeholder (cap-10s.2 is a future scope item)
- Configuration file-based provider selection (currently CLI flag only)

## Known Limitations

- Kiro prompt adaptation (cap-10s.2) is a placeholder — all providers currently receive identical prompts
- Full E2E with real kiro-cli requires Kiro to be installed; smoke tests use a mock script
