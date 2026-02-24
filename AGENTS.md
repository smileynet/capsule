# Agent Instructions

This project uses **bd** (beads) for issue tracking. Run `bd onboard` to get started.

## Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --status in_progress  # Claim work
bd close <id>         # Complete work
bd sync               # Sync with git
```

## Build & Test

```bash
make build         # Build binary with version info
make test          # Run unit tests (-short)
make test-full     # Run all tests (no -short) + shell tests
make test-scripts  # Run shell tests in tests/scripts/
make smoke         # Run end-to-end smoke tests (builds binary)
make lint          # Run golangci-lint
make fmt           # Run goimports on all Go files
make gate-feature  # Quality gate for feature close (lint + test-full)
make gate-epic     # Quality gate for epic close (lint + test-full + smoke)
make demo          # Create demo playground (TEMPLATE=campaign by default)
make dev-setup     # One-shot new contributor setup
make clean         # Remove binary and test cache
make hooks         # Install pre-commit hook for bd chaining
```

## Project Structure

```
cmd/capsule/main.go      # CLI entry point (Kong). Wiring only.
internal/                 # All logic. One responsibility per package.
  config/                 # Configuration loading
  orchestrator/           # Pipeline phase sequencing
  prompt/                 # External prompt management
  provider/               # AI provider interface + implementations
  tui/                    # Terminal UI (rich and plain text output)
  worklog/                # Worklog lifecycle
  worktree/               # Git worktree management
scripts/                  # Shell scripts (pipeline, hooks)
prompts/                  # External prompt templates
templates/                # Project templates
docs/                     # Architecture and conventions
```

## Conventions

- **Go style**: See `docs/go-conventions.md` for full reference
- **Interfaces**: Define where consumed, not where implemented
- **Errors**: Wrap with `%w`, use sentinel errors for caller-checkable conditions
- **Testing**: Table-driven tests, mock at package boundaries, `t.TempDir()` for filesystem
- **CLI**: Kong with nested struct commands and `Run()` methods
- **Packages**: Short, lowercase, singular nouns. No `utils` or `helpers`
- **Imports**: `goimports` enforced on every edit via post-edit hook

## Quality Gates

| Scope | Gate | What Runs | Command |
|-------|------|-----------|---------|
| Every edit | PostToolUse hook | `goimports` + `go build` + `go vet` (scoped to edited package) | (automatic) |
| Every commit | Pre-commit hook | Incremental lint + `-short` tests (staged packages only) | (automatic) |
| Feature close | Manual before `bd close` | lint + test-full (includes shell tests) | `make gate-feature` |
| Epic close | Manual before `bd close` | lint + test-full + smoke | `make gate-epic` |

- **Linter config**: `.golangci.yml` (errorlint, gocritic enabled)

## Hook Setup (Fresh Clone)

After cloning, install the Go quality pre-commit hook:

```bash
make hooks
```

The bd pre-commit shim at `.git/hooks/pre-commit` calls `bd hook pre-commit`, which discovers and runs `.git/hooks/pre-commit.old` before its own export logic.

## Data Handling in Pipeline Scripts

When editing shell scripts that process untrusted content (task descriptions, LLM output, user feedback), refer to `docs/data-handling.md` for safe patterns. Key rules: use `printf '%s\n'` not `echo`, use `jq -r` not grep/sed for JSON, use `awk` not `${//}` for template substitution with untrusted content.

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds

