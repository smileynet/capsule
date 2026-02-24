# Demo Full — Agent Instructions

Template project used as a test subject for capsule's multi-epic campaign pipeline.

## Stack

- Go 1.22
- Standard library only

## Structure

```
src/
  go.mod         - Go module definition
  taskboard.go   - Entry point with Board/Task types and feature gaps
```

## Conventions

- All Go source lives under `src/`
- Tests use `_test.go` suffix in the same package
- CRUD functions take `*Board` as first parameter
- Validation functions follow `ValidateX(input) error`
- Serialization functions: `MarshalBoard` / `UnmarshalBoard`
- Statistics functions take `*Board` and return computed values
- The `Board` struct in `taskboard.go` is the central data type

## Test Command

```bash
cd src && go test ./...
```

## Feature Gaps

The following functions are referenced but not yet implemented:

### Task CRUD (Feature demo-1.1)
- `CreateTask(b *Board, title string, priority Priority) (*Task, error)` - Add new task
- `ListTasks(b *Board, status Status) []*Task` - List/filter tasks
- `UpdateStatus(b *Board, taskID int, status Status) error` - Transition task status

### Task Validation (Feature demo-1.2)
- `ValidateTitle(title string) error` - Title constraints
- `ValidatePriority(p Priority) error` - Priority range check

### JSON Serialization (Feature demo-2.1)
- `MarshalBoard(b *Board) ([]byte, error)` - Serialize board to JSON
- `UnmarshalBoard(data []byte) (*Board, error)` - Deserialize board from JSON

### Board Statistics (Feature demo-2.2)
- `CountByStatus(b *Board) map[Status]int` - Count tasks by status
- `CompletionPct(b *Board) float64` - Percentage of done tasks

These gaps are intended to be filled by bead tasks during campaign pipeline testing.
