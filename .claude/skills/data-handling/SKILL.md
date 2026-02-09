---
name: data-handling
description: >-
  Safe shell scripting patterns for untrusted data in capsule pipeline scripts.
  Auto-applied when editing scripts/, templates/, or bash code that handles
  LLM output, task descriptions, or user feedback.
user-invocable: false
---

# Data Handling

Pipeline scripts process untrusted content from beads, LLM output, and user feedback. Every format crossing requires sanitization.

## Quick-Reference Trap Table

| Dangerous pattern | Safe alternative | Why |
|---|---|---|
| `${var//pattern/$untrusted}` | awk file-read substitution | `&` and `\` in replacement expand as backreferences |
| `echo "$var"` | `printf '%s\n' "$var"` | `echo` interprets `-n`, `-e` flags and backslash escapes |
| `grep`/`sed` on JSON | `jq -r '.field // empty'` | Regex on JSON breaks on nested quotes, escapes, and Unicode |
| `envsubst` with self-referencing values | Explicit whitelist + guard or awk rendering | Values containing `${LISTED_VAR}` trigger recursive expansion |
| `awk gsub()` with untrusted replacement | awk file-read approach | `&` in `gsub()` replacement expands as backreference |

## Boundary Principle

Sanitize at every format crossing:
- **JSON to shell**: `jq -r` with `// empty`, never grep/sed
- **Shell to template rendering**: awk or guarded envsubst, never raw `${//}`
- **Shell to git commands**: Quote arguments, use `--` to separate flags from paths

## Reference

For full patterns, examples, and the boundary checklist, see `docs/data-handling.md`.
