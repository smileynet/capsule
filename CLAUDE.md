# CLAUDE.md

## Build & Test

```bash
make build   # Build binary with version info
make test    # Run all tests
make lint    # Run golangci-lint
make clean   # Remove binary and test cache
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

- **Post-edit hook**: `goimports -w`, `go build ./...`, `go vet ./...` run after every `.go` file edit
- **Pre-commit**: `golangci-lint run ./...` and `go test ./...` via beads hook
- **Linter config**: `.golangci.yml` (errorlint, gocritic enabled)
