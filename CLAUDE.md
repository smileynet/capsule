# CLAUDE.md

## Build & Test

```bash
make build      # Build binary with version info
make test       # Run unit tests (-short, skips slow tests)
make test-full  # Run all tests including slow ones
make smoke      # Run end-to-end smoke tests (builds binary)
make lint       # Run golangci-lint
make clean      # Remove binary and test cache
make hooks      # Install pre-commit hook for bd chaining
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

| Scope | Gate | What Runs |
|-------|------|-----------|
| Every edit | PostToolUse hook | `goimports` + `go build` + `go vet` (scoped to edited package) |
| Every commit | Pre-commit hook | Incremental lint + `-short` tests (staged packages only) |
| Feature close | Manual before `bd close` | `make lint && make test-full` |
| Epic close | Manual before `bd close` | `make lint && make test-full && make smoke` |

- **Linter config**: `.golangci.yml` (errorlint, gocritic enabled)

## Hook Setup (Fresh Clone)

After cloning, install the Go quality pre-commit hook:

```bash
make hooks
```

The bd pre-commit shim at `.git/hooks/pre-commit` calls `bd hook pre-commit`, which discovers and runs `.git/hooks/pre-commit.old` before its own export logic.
