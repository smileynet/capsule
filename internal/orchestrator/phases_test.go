package orchestrator

import (
	"testing"
)

func TestPhaseKind_Values(t *testing.T) {
	// Given phase kinds
	// Then Worker and Reviewer are distinct values
	if Worker == Reviewer {
		t.Fatal("Worker and Reviewer must be distinct PhaseKind values")
	}
}

func TestPhaseKind_String(t *testing.T) {
	tests := []struct {
		kind PhaseKind
		want string
	}{
		{Worker, "worker"},
		{Reviewer, "reviewer"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			// When String is called
			got := tt.kind.String()

			// Then the label matches the expected value
			if got != tt.want {
				t.Errorf("PhaseKind(%d).String() = %q, want %q", tt.kind, got, tt.want)
			}
		})
	}
}

func TestDefaultPhases_Count(t *testing.T) {
	// When the default phases are queried
	phases := DefaultPhases()

	// Then all 6 pipeline phases are registered
	if got := len(phases); got != 6 {
		t.Fatalf("DefaultPhases() returned %d phases, want 6", got)
	}
}

func TestDefaultPhases_Order(t *testing.T) {
	// When the default phases are queried
	phases := DefaultPhases()

	// Then phases are returned in pipeline order
	wantNames := []string{
		"test-writer",
		"test-review",
		"execute",
		"execute-review",
		"sign-off",
		"merge",
	}
	for i, want := range wantNames {
		if phases[i].Name != want {
			t.Errorf("DefaultPhases()[%d].Name = %q, want %q", i, phases[i].Name, want)
		}
	}
}

func TestDefaultPhases_Kinds(t *testing.T) {
	// When the default phases are queried
	phases := DefaultPhases()

	// Then worker and reviewer kinds are correctly assigned
	wantKinds := []PhaseKind{
		Worker,   // test-writer
		Reviewer, // test-review
		Worker,   // execute
		Reviewer, // execute-review
		Reviewer, // sign-off
		Worker,   // merge
	}
	for i, want := range wantKinds {
		if phases[i].Kind != want {
			t.Errorf("DefaultPhases()[%d].Kind = %v, want %v (phase: %s)",
				i, phases[i].Kind, want, phases[i].Name)
		}
	}
}

func TestDefaultPhases_RetryTargets(t *testing.T) {
	// When the default phases are queried
	phases := DefaultPhases()

	// Build name-to-phase map for readability.
	byName := make(map[string]PhaseDefinition, len(phases))
	for _, p := range phases {
		byName[p.Name] = p
	}

	// Then workers have no retry target (they don't issue NEEDS_WORK)
	for _, name := range []string{"test-writer", "execute", "merge"} {
		if byName[name].RetryTarget != "" {
			t.Errorf("phase %q: worker should have empty RetryTarget, got %q",
				name, byName[name].RetryTarget)
		}
	}

	// Then reviewers point to the correct retry targets
	if got := byName["test-review"].RetryTarget; got != "test-writer" {
		t.Errorf("test-review.RetryTarget = %q, want %q", got, "test-writer")
	}
	if got := byName["execute-review"].RetryTarget; got != "execute" {
		t.Errorf("execute-review.RetryTarget = %q, want %q", got, "execute")
	}
	if got := byName["sign-off"].RetryTarget; got != "execute" {
		t.Errorf("sign-off.RetryTarget = %q, want %q", got, "execute")
	}
}

func TestStatusUpdate_Fields(t *testing.T) {
	// Given a fully populated StatusUpdate
	su := StatusUpdate{
		BeadID:   "cap-123",
		Phase:    "test-writer",
		Status:   PhaseRunning,
		Progress: "1/6",
		Attempt:  1,
		MaxRetry: 3,
	}

	// Then all fields are accessible with expected values
	if su.BeadID != "cap-123" {
		t.Errorf("BeadID = %q, want %q", su.BeadID, "cap-123")
	}
	if su.Phase != "test-writer" {
		t.Errorf("Phase = %q, want %q", su.Phase, "test-writer")
	}
	if su.Status != PhaseRunning {
		t.Errorf("Status = %q, want %q", su.Status, PhaseRunning)
	}
	if su.Progress != "1/6" {
		t.Errorf("Progress = %q, want %q", su.Progress, "1/6")
	}
	if su.Attempt != 1 {
		t.Errorf("Attempt = %d, want %d", su.Attempt, 1)
	}
	if su.MaxRetry != 3 {
		t.Errorf("MaxRetry = %d, want %d", su.MaxRetry, 3)
	}
}

func TestPhaseStatus_Values(t *testing.T) {
	// Given all PhaseStatus constants
	statuses := []PhaseStatus{PhasePending, PhaseRunning, PhasePassed, PhaseFailed, PhaseError}

	// Then all values are distinct
	seen := make(map[PhaseStatus]bool, len(statuses))
	for _, s := range statuses {
		if seen[s] {
			t.Errorf("duplicate PhaseStatus value: %q", s)
		}
		seen[s] = true
	}
}

func TestStatusCallback_Invocation(t *testing.T) {
	// Given a StatusCallback that captures its argument
	var received StatusUpdate
	cb := StatusCallback(func(su StatusUpdate) {
		received = su
	})

	// When the callback is invoked
	want := StatusUpdate{
		BeadID:   "cap-456",
		Phase:    "execute",
		Status:   PhasePassed,
		Progress: "3/6",
		Attempt:  2,
		MaxRetry: 3,
	}
	cb(want)

	// Then it receives the StatusUpdate
	if received != want {
		t.Errorf("callback received %+v, want %+v", received, want)
	}
}
