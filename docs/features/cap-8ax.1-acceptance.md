# Feature Acceptance: cap-8ax.1

## Reusable template project with bead fixtures

**Status:** Accepted
**Date:** 2026-02-07
**Parent Epic:** cap-8ax (Tracer Bullet: Scripts & Direct Claude CLI)

## What Was Requested

A reusable template project with pre-built beads so the pipeline can be repeatedly tested against a known starting state.

## Acceptance Criteria

| # | Criterion | Evidence | Verified |
|---|-----------|----------|----------|
| 1 | setup-template.sh creates a git repo with source files, CLAUDE.md, and .beads/ | test-setup-template.sh [1/7, 6/7], test-template-source-files.sh [1/6..6/6] | Yes |
| 2 | `bd ready` lists available task beads | test-setup-template.sh [3/7] | Yes |
| 3 | `bd show <task-id>` returns title, description, acceptance criteria, and parent hierarchy | test-setup-template.sh [4/7] | Yes |
| 4 | Setup produces identical state across repeated runs | test-setup-template.sh [5/7] | Yes |

## How to Verify

```bash
bash tests/scripts/test-setup-template.sh
bash tests/scripts/test-template-source-files.sh
```

## Out of Scope

- Multi-language template support
- Template selection UI
- Complete pipeline bead fixtures

## Known Limitations

- Template uses `example.com/` module path (safe namespace, not a real module)
