# bd JSON Schema Contract

Fields the capsule pipeline depends on from `bd show` and `bd list` output.

## `bd show <id> --json`

Returns a JSON array with one element. Key fields:

| Field | Type | Notes |
|-------|------|-------|
| `.[0].title` | string | Required. Empty means bead not found. |
| `.[0].description` | string\|null | May be null or empty. |
| `.[0].acceptance_criteria` | string\|null | Often null. Fallback: extract from description headings. |
| `.[0].issue_type` | string | `"task"`, `"feature"`, `"epic"`, `"bug"` |
| `.[0].status` | string | `"open"`, `"in_progress"`, `"closed"` |
| `.[0].parent` | string\|null | Direct parent ID. May be null for root issues. |
| `.[0].dependencies` | array | Dependency objects (see below). |

### Parent resolution

Primary: `.[0].parent` returns the parent ID directly.

Fallback (for compatibility): extract from dependencies array:
```jq
[.[0].dependencies[]? | select(.dependency_type == "parent-child")][0].id // empty
```

### Dependency objects

Each element in `.[0].dependencies[]`:
```json
{
  "id": "cap-8ax.1",
  "title": "...",
  "status": "closed",
  "dependency_type": "parent-child"
}
```

## `bd list --parent=<id>`

By default, excludes closed issues. Use `--all` to include closed issues in results.

This matters for progress calculations — without `--all`, the denominator only
counts open issues and the progress fraction is meaningless.

```bash
# Wrong: only returns open children
bd list --parent="$ID" --json

# Correct: returns all children including closed
bd list --parent="$ID" --all --json
```

## Acceptance criteria extraction

The pipeline checks these sources in order:
1. `.[0].acceptance_criteria` field (explicit)
2. `## Acceptance Criteria` heading in description
3. `## Requirements` heading in description (fallback)
