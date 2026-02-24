package main

import (
	"fmt"
	"time"
)

// Status represents the state of a task.
type Status string

const (
	StatusTodo       Status = "todo"
	StatusInProgress Status = "in_progress"
	StatusDone       Status = "done"
)

// Priority represents task urgency (0 = critical, 4 = backlog).
type Priority int

const (
	PriorityCritical Priority = 0
	PriorityHigh     Priority = 1
	PriorityMedium   Priority = 2
	PriorityLow      Priority = 3
	PriorityBacklog  Priority = 4
)

// Task represents a single work item on the board.
type Task struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Status    Status    `json:"status"`
	Priority  Priority  `json:"priority"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Board holds a collection of tasks.
type Board struct {
	Name  string  `json:"name"`
	Tasks []*Task `json:"tasks"`
}

// NewBoard creates an empty board with the given name.
func NewBoard(name string) *Board {
	return &Board{
		Name:  name,
		Tasks: make([]*Task, 0),
	}
}

// FEATURE_GAP: CreateTask(b *Board, title string, priority Priority) (*Task, error)
// Add a new task with a unique sequential ID, StatusTodo, and current timestamp.

// FEATURE_GAP: ListTasks(b *Board, status Status) []*Task
// Return tasks matching the given status, or all tasks if status is empty.

// FEATURE_GAP: UpdateStatus(b *Board, taskID int, status Status) error
// Transition a task to the given status, updating the timestamp.

// FEATURE_GAP: ValidateTitle(title string) error
// Check that title is non-empty, non-whitespace, and <= 200 characters.

// FEATURE_GAP: ValidatePriority(p Priority) error
// Check that priority is in the range 0-4.

// FEATURE_GAP: MarshalBoard(b *Board) ([]byte, error)
// Serialize the board to JSON.

// FEATURE_GAP: UnmarshalBoard(data []byte) (*Board, error)
// Deserialize a board from JSON.

// FEATURE_GAP: CountByStatus(b *Board) map[Status]int
// Count tasks grouped by status.

// FEATURE_GAP: CompletionPct(b *Board) float64
// Return the percentage of tasks in StatusDone.

func main() {
	b := NewBoard("My Project")

	fmt.Printf("Board: %s (%d tasks)\n", b.Name, len(b.Tasks))
	fmt.Println("Note: task management functions are not yet implemented")
}
