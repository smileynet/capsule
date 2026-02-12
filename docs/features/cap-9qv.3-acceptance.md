# Feature Acceptance: cap-9qv.3

## Configuration loading and external prompt management

**Status:** Accepted
**Date:** 2026-02-10
**Parent Epic:** cap-9qv (Go CLI Tool)

## What Was Requested

Capsule loads config from YAML and prompts from external files so behavior can be customized per-project.

## Acceptance Criteria

| # | Criterion | Evidence | Verified |
|---|-----------|----------|----------|
| 1 | Given a config.yaml, when capsule loads config, then Runtime.Provider, Runtime.Timeout, and Worktree.BaseDir are set | `TestLoad_ValidFile` (config_test.go) | Yes |
| 2 | Given no config file, when capsule loads config, then sensible defaults are used | `TestDefaultConfig`, `TestLoad_MissingFile` | Yes |
| 3 | Given prompts/ directory, when prompt loader loads test-writer, then prompts/test-writer.md content is returned | `TestLoad_ReadsPromptFile` (prompt_test.go) | Yes |
| 4 | Given prompt context, when Compose is called, then bead info and feedback are interpolated into the prompt | `TestCompose_InterpolatesContext`, `TestCompose_InterpolatesFeedback` | Yes |

## How to Verify

```bash
go test ./internal/config/ ./internal/prompt/ -v -count=1
```

## Out of Scope

- Hot-reload of config changes
- Config file generation or migration tooling
- Prompt validation beyond template syntax

## Known Limitations

- Layered config uses pointer-based rawConfig structs for disambiguation — adds complexity but required by Go zero-value semantics
- Given-When-Then comments not yet added to some table-driven tests (filed as P4 bead)
