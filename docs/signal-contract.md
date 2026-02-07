# Signal Contract

Every phase in the capsule pipeline must produce a **signal** as its final output: a JSON object that the orchestrator parses to decide what happens next.

## Schema

```json
{
  "status": "PASS | NEEDS_WORK | ERROR",
  "feedback": "Human-readable explanation of the result",
  "files_changed": ["path/to/file1.go", "path/to/file2_test.go"],
  "summary": "One-line description of what happened"
}
```

## Fields

| Field           | Type       | Required | Description                                                     |
|-----------------|------------|----------|-----------------------------------------------------------------|
| `status`        | string     | yes      | One of `PASS`, `NEEDS_WORK`, or `ERROR`                        |
| `feedback`      | string     | yes      | Human-readable explanation. On `NEEDS_WORK`, describes what to fix. On `ERROR`, describes what went wrong. |
| `files_changed` | string[]   | yes      | Paths of files created or modified (relative to worktree root). Empty array `[]` if none. |
| `summary`       | string     | yes      | One-line description of what the phase did                      |

## Status Values

- **PASS** -- The phase completed successfully. The orchestrator advances to the next phase.
- **NEEDS_WORK** -- The phase found issues. The orchestrator re-runs the paired phase with `feedback` appended.
- **ERROR** -- The phase failed unexpectedly. The orchestrator stops the pipeline and reports the error.

## Output Convention

The signal JSON must appear as the **last JSON object** in the phase's stdout. Phases may produce other text output (logs, progress messages) before the signal. The parser (`scripts/parse-signal.sh`) extracts the last `{...}` block from the output.

### Valid output example

```
Reading worklog.md...
Found 3 acceptance criteria.
Writing test file: src/validation_test.go
Running tests... 2 failing (expected in RED phase)

{"status":"PASS","feedback":"Created 2 test files covering all 3 acceptance criteria. Tests fail as expected.","files_changed":["src/validation_test.go","src/format_test.go"],"summary":"Test files created for validation feature"}
```

### Invalid output (no JSON block)

If a phase produces no JSON block, `parse-signal.sh` returns a synthetic ERROR signal:

```json
{
  "status": "ERROR",
  "feedback": "No signal JSON found in phase output",
  "files_changed": [],
  "summary": "Phase did not produce a signal"
}
```

## Validation Rules

The parser checks that the last JSON block:

1. Is valid JSON
2. Contains all four required fields (`status`, `feedback`, `files_changed`, `summary`)
3. Has a `status` value that is one of `PASS`, `NEEDS_WORK`, or `ERROR`
4. Has `files_changed` as an array

If any check fails, the parser returns a synthetic ERROR signal with a description of what was wrong.
