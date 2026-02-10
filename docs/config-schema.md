# Configuration Schema

Capsule uses layered YAML configuration. See `capsule.example.yaml` at the project root for a quick-start template.

## File Locations

| Layer | Path | Purpose |
|-------|------|---------|
| User | `~/.config/capsule/config.yaml` | Personal defaults across all projects |
| Project | `.capsule/config.yaml` | Project-specific overrides |

Missing files are silently skipped (defaults apply). Invalid YAML or unknown fields produce an error.

## Precedence

Later sources override earlier ones:

1. Compiled defaults (lowest)
2. User config (`~/.config/capsule/config.yaml`)
3. Project config (`.capsule/config.yaml`)
4. Environment variables (`CAPSULE_*`)
5. CLI flags (highest)

Only fields explicitly set in a layer override the previous layer. Omitted fields are left unchanged — they do not reset to zero values.

## Fields

### `runtime`

| Field | Type | Default | Env Var | Description |
|-------|------|---------|---------|-------------|
| `provider` | string | `claude` | `CAPSULE_PROVIDER` | AI provider name. Must match a registered provider. |
| `timeout` | duration | `5m` | `CAPSULE_TIMEOUT` | Max execution time per phase. Go duration format: `ns`, `us`, `ms`, `s`, `m`, `h`. |

### `worktree`

| Field | Type | Default | Env Var | Description |
|-------|------|---------|---------|-------------|
| `base_dir` | string | `.capsule/worktrees` | `CAPSULE_WORKTREE_BASE_DIR` | Base directory for git worktrees, relative to project root. |

## Validation Rules

After all layers are merged, the final config is validated:

- `runtime.provider` — must be non-empty
- `runtime.timeout` — must be positive (> 0)
- `worktree.base_dir` — must be non-empty

## Duration Format

The `timeout` field accepts Go's `time.ParseDuration` format:

| Unit | Suffix | Example |
|------|--------|---------|
| Nanoseconds | `ns` | `500ns` |
| Microseconds | `us` | `100us` |
| Milliseconds | `ms` | `250ms` |
| Seconds | `s` | `30s` |
| Minutes | `m` | `5m` |
| Hours | `h` | `1h` |

Combinations are allowed: `1h30m`, `2m30s`.

**Not supported:** days (`d`), weeks (`w`), or years (`y`).

## Strict Parsing

Unknown fields are rejected. A config file containing `provder: openai` (typo) will produce an error rather than silently using the default provider. This catches common mistakes early.

## Source of Truth

The canonical struct definitions live in `internal/config/config.go`. This document must stay in sync with those types.
