package orchestrator

import (
	"time"

	"github.com/smileynet/capsule/internal/provider"
)

// PhaseKind distinguishes workers (produce artifacts) from reviewers (evaluate artifacts)
// and gates (shell commands).
type PhaseKind int

const (
	Worker   PhaseKind = iota // Worker phases produce or modify code.
	Reviewer                  // Reviewer phases evaluate worker output.
	Gate                      // Gate phases execute shell commands.
)

func (k PhaseKind) String() string {
	switch k {
	case Worker:
		return "worker"
	case Reviewer:
		return "reviewer"
	case Gate:
		return "gate"
	default:
		return "unknown"
	}
}

// PhaseDefinition describes a single pipeline phase.
type PhaseDefinition struct {
	Name        string        // Phase name (also used as prompt template name for Worker/Reviewer).
	Kind        PhaseKind     // Worker, Reviewer, or Gate.
	Prompt      string        // Template name override (defaults to Name for Worker/Reviewer).
	Command     string        // Shell command (required for Gate, ignored otherwise).
	MaxRetries  int           // Maximum retry attempts for this phase's pair.
	RetryTarget string        // Phase to re-run on NEEDS_WORK (empty for workers).
	Optional    bool          // If true, SKIP/ERROR → continue pipeline.
	Condition   string        // "files_match:<glob>" or empty (always run). Evaluated before phase execution.
	Provider    string        // TODO(cap-6vp): select alternate provider in executePhase. Override default provider for this phase.
	Timeout     time.Duration // TODO(cap-6vp): apply via context.WithTimeout in executePhase. Override default timeout for this phase.
}

// PromptName returns the prompt template name for this phase.
// Uses the explicit Prompt field if set, otherwise falls back to Name.
func (pd PhaseDefinition) PromptName() string {
	if pd.Prompt != "" {
		return pd.Prompt
	}
	return pd.Name
}

// PhaseStatus represents the current state of a phase execution.
type PhaseStatus string

const (
	PhasePending PhaseStatus = "pending"
	PhaseRunning PhaseStatus = "running"
	PhasePassed  PhaseStatus = "passed"
	PhaseFailed  PhaseStatus = "failed"
	PhaseError   PhaseStatus = "error"
	PhaseSkipped PhaseStatus = "skipped"
)

// StatusUpdate carries progress information for a single phase execution.
type StatusUpdate struct {
	BeadID   string           // The bead being processed.
	Phase    string           // Current phase name.
	Status   PhaseStatus      // Current phase status.
	Progress string           // Human-readable progress (e.g. "2/6").
	Attempt  int              // Current attempt number (1-based).
	MaxRetry int              // Maximum retries configured.
	Signal   *provider.Signal // Populated on phase completion (passed/failed/error), nil while running.
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

// MinimalPhases returns a simplified 3-phase pipeline.
func MinimalPhases() []PhaseDefinition {
	return []PhaseDefinition{
		{Name: "test-writer", Kind: Worker, MaxRetries: 3},
		{Name: "execute", Kind: Worker, MaxRetries: 3},
		{Name: "merge", Kind: Worker, MaxRetries: 1},
	}
}

// ThoroughPhases returns an extended pipeline with test quality review, lint gate, and security scan.
func ThoroughPhases() []PhaseDefinition {
	return []PhaseDefinition{
		{Name: "test-writer", Kind: Worker, MaxRetries: 3},
		{Name: "test-quality", Kind: Reviewer, MaxRetries: 2, RetryTarget: "test-writer", Prompt: "test-quality"},
		{Name: "execute", Kind: Worker, MaxRetries: 3},
		{Name: "lint", Kind: Gate, Command: "make lint", Optional: true},
		{Name: "execute-review", Kind: Reviewer, MaxRetries: 3, RetryTarget: "execute"},
		{Name: "sign-off", Kind: Reviewer, MaxRetries: 3, RetryTarget: "execute"},
		{Name: "merge", Kind: Worker, MaxRetries: 1},
	}
}

// PresetPhases returns phases for a named preset ("default", "minimal", "thorough").
// Returns nil if the preset name is not recognized.
func PresetPhases(name string) []PhaseDefinition {
	switch name {
	case "default", "":
		return DefaultPhases()
	case "minimal":
		return MinimalPhases()
	case "thorough":
		return ThoroughPhases()
	default:
		return nil
	}
}
