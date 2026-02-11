# Capsule

A deterministic TDD pipeline tool that uses AI to implement tasks. Capsule orchestrates a structured sequence — write tests, implement code, review, and merge — driven by JSON signal contracts between phases.

Given a task (bead), capsule creates an isolated git worktree, runs AI agents through each phase with retry loops, and merges the result back to your main branch.

## Prerequisites

- **Go 1.25+** (building from source)
- **git**
- **[bd](https://github.com/smileynet/beads)** (beads CLI) for task management
- **[claude](https://docs.anthropic.com/en/docs/claude-code)** CLI for pipeline execution

## Installation

```bash
git clone https://github.com/smileynet/capsule.git
cd capsule
make build
# Binary: ./capsule
```

## Project Setup

Capsule runs against a target project that must have:

| Requirement | Path |
|-------------|------|
| Prompt templates (7 files) | `prompts/{test-writer,test-review,execute,execute-review,sign-off,merge,summary}.md` |
| Worklog template | `templates/worklog.md.template` |
| Beads initialized | `.beads/` (via `bd init`) |
| Git repository | `.git/` |

## Quick Start

Set up a demo project using the included template:

```bash
# Create demo project
scripts/setup-template.sh --template=demo-brownfield /tmp/capsule-demo

# Copy pipeline prompts and templates into it
cp -r prompts/ /tmp/capsule-demo/prompts/
cp templates/worklog.md.template /tmp/capsule-demo/templates/worklog.md.template

# Run the pipeline
cd /tmp/capsule-demo
/path/to/capsule run demo-1.1.1
```

## Claude Code Commands

If you use [Claude Code](https://docs.anthropic.com/en/docs/claude-code), capsule ships slash commands for common workflows:

| Command | Purpose |
|---------|---------|
| `/capsule-setup [dir]` | Verify a project is ready for capsule |
| `/capsule-run <bead-id>` | Run a pipeline for a bead |
| `/capsule-inspect [bead-id]` | Dashboard overview or deep-dive into a run |
| `/capsule-cleanup [bead-id]` | Clean up worktrees when done |
| `/capsule-gate <feature\|epic>` | Run quality gates before closing |

**Typical session:**

```
/capsule-setup              # verify project has prompts, templates, beads
/capsule-run demo-1.1.1     # execute the pipeline
/capsule-inspect demo-1.1.1 # if it failed, diagnose why
/capsule-cleanup demo-1.1.1 # clean up when done
/capsule-gate feature       # before closing the feature in beads
```

## CLI Reference

### `capsule run <bead-id>`

Run the full TDD pipeline for a bead.

| Flag | Default | Description |
|------|---------|-------------|
| `--provider` | `claude` | AI provider for completions |
| `--timeout` | `300` | Timeout in seconds |

Exit codes: `0` success, `1` pipeline error, `2` setup error.

### `capsule abort <bead-id>`

Remove the worktree but preserve the branch for inspection.

### `capsule clean <bead-id>`

Remove worktree, delete branch, and prune stale metadata.

### `capsule --version`

Print version, commit, and build date.

## Configuration

Capsule loads config from (in precedence order):

1. `~/.config/capsule/config.yaml` (user)
2. `.capsule/config.yaml` (project)
3. Environment variables (`CAPSULE_*`)
4. CLI flags

See [docs/config-schema.md](docs/config-schema.md) for the full schema.

## Documentation

| Document | Description |
|----------|-------------|
| [docs/commands.md](docs/commands.md) | Full command and script reference |
| [docs/signal-contract.md](docs/signal-contract.md) | JSON signal contract between phases |
| [docs/config-schema.md](docs/config-schema.md) | Configuration schema and options |
| [docs/manual-verification.md](docs/manual-verification.md) | Manual verification walkthrough |
| [docs/go-conventions.md](docs/go-conventions.md) | Go coding conventions |

## Development

```bash
make build      # Build binary with version info
make test       # Run unit tests (-short)
make test-full  # Run all tests
make smoke      # End-to-end smoke tests
make lint       # Run golangci-lint
make hooks      # Install pre-commit hook
```
