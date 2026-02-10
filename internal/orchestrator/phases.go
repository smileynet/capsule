package orchestrator

// PhaseKind distinguishes workers (produce artifacts) from reviewers (evaluate artifacts).
type PhaseKind int

const (
	Worker   PhaseKind = iota // Worker phases produce or modify code.
	Reviewer                  // Reviewer phases evaluate worker output.
)

func (k PhaseKind) String() string {
	switch k {
	case Worker:
		return "worker"
	case Reviewer:
		return "reviewer"
	default:
		return "unknown"
	}
}

// PhaseDefinition describes a single pipeline phase.
type PhaseDefinition struct {
	Name        string    // Prompt template name (e.g. "test-writer").
	Kind        PhaseKind // Worker or Reviewer.
	MaxRetries  int       // Maximum retry attempts for this phase's pair.
	RetryTarget string    // Phase to re-run on NEEDS_WORK (empty for workers).
}

// PhaseStatus represents the current state of a phase execution.
type PhaseStatus string

const (
	PhasePending PhaseStatus = "pending"
	PhaseRunning PhaseStatus = "running"
	PhasePassed  PhaseStatus = "passed"
	PhaseFailed  PhaseStatus = "failed"
	PhaseError   PhaseStatus = "error"
)

// StatusUpdate carries progress information for a single phase execution.
type StatusUpdate struct {
	BeadID   string      // The bead being processed.
	Phase    string      // Current phase name.
	Status   PhaseStatus // Current phase status.
	Progress string      // Human-readable progress (e.g. "2/6").
	Attempt  int         // Current attempt number (1-based).
	MaxRetry int         // Maximum retries configured.
}

// StatusCallback receives phase progress updates.
type StatusCallback func(StatusUpdate)

// DefaultPhases returns the standard 6-phase pipeline in execution order.
func DefaultPhases() []PhaseDefinition {
	return []PhaseDefinition{
		{Name: "test-writer", Kind: Worker, MaxRetries: 3},
		{Name: "test-review", Kind: Reviewer, MaxRetries: 3, RetryTarget: "test-writer"},
		{Name: "execute", Kind: Worker, MaxRetries: 3},
		{Name: "execute-review", Kind: Reviewer, MaxRetries: 3, RetryTarget: "execute"},
		{Name: "sign-off", Kind: Reviewer, MaxRetries: 3, RetryTarget: "execute"},
		{Name: "merge", Kind: Worker, MaxRetries: 1},
	}
}
