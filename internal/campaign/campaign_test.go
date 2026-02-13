package campaign

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/smileynet/capsule/internal/orchestrator"
	"github.com/smileynet/capsule/internal/provider"
)

// --- Test mocks ---

type mockPipeline struct {
	outputs []orchestrator.PipelineOutput
	errs    []error
	calls   []orchestrator.PipelineInput
	idx     int
}

func (m *mockPipeline) RunPipeline(_ context.Context, input orchestrator.PipelineInput) (orchestrator.PipelineOutput, error) {
	m.calls = append(m.calls, input)
	if m.idx >= len(m.outputs) {
		return orchestrator.PipelineOutput{}, fmt.Errorf("unexpected pipeline call %d", m.idx+1)
	}
	out := m.outputs[m.idx]
	var err error
	if m.idx < len(m.errs) {
		err = m.errs[m.idx]
	}
	m.idx++
	return out, err
}

type mockBeadClient struct {
	children []BeadInfo
	childErr error
	showInfo map[string]BeadInfo
	showErr  error
	closed   []string
	closeErr error
	created  []BeadInput
	createID string
}

func (m *mockBeadClient) ReadyChildren(string) ([]BeadInfo, error) {
	return m.children, m.childErr
}

func (m *mockBeadClient) Show(id string) (BeadInfo, error) {
	if m.showInfo != nil {
		if info, ok := m.showInfo[id]; ok {
			return info, nil
		}
	}
	return BeadInfo{ID: id}, m.showErr
}

func (m *mockBeadClient) Close(id string) error {
	m.closed = append(m.closed, id)
	return m.closeErr
}

func (m *mockBeadClient) Create(input BeadInput) (string, error) {
	m.created = append(m.created, input)
	return m.createID, nil
}

type mockStateStore struct {
	saved  []State
	loaded map[string]State
}

func (m *mockStateStore) Save(state State) error {
	m.saved = append(m.saved, state)
	return nil
}

func (m *mockStateStore) Load(id string) (State, bool, error) {
	if m.loaded != nil {
		if s, ok := m.loaded[id]; ok {
			return s, true, nil
		}
	}
	return State{}, false, nil
}

func (m *mockStateStore) Remove(string) error { return nil }

type mockCallback struct {
	campaignStarted  bool
	tasksStarted     []string
	tasksCompleted   []TaskResult
	tasksFailed      []string
	discoveriesFiled []string
	validationStart  bool
	validationDone   bool
	campaignDone     bool
}

func (m *mockCallback) OnCampaignStart(string, []BeadInfo) { m.campaignStarted = true }
func (m *mockCallback) OnTaskStart(id string)              { m.tasksStarted = append(m.tasksStarted, id) }
func (m *mockCallback) OnTaskComplete(r TaskResult)        { m.tasksCompleted = append(m.tasksCompleted, r) }
func (m *mockCallback) OnTaskFail(id string, _ error)      { m.tasksFailed = append(m.tasksFailed, id) }
func (m *mockCallback) OnDiscoveryFiled(f provider.Finding, newID string) {
	m.discoveriesFiled = append(m.discoveriesFiled, newID)
}
func (m *mockCallback) OnValidationStart()              { m.validationStart = true }
func (m *mockCallback) OnValidationComplete(TaskResult) { m.validationDone = true }
func (m *mockCallback) OnCampaignComplete(State)        { m.campaignDone = true }

func passOutput() orchestrator.PipelineOutput {
	return orchestrator.PipelineOutput{Completed: true}
}

// --- Tests ---

func TestRun_HappyPath(t *testing.T) {
	// Given 3 tasks all succeed
	pipeline := &mockPipeline{
		outputs: []orchestrator.PipelineOutput{passOutput(), passOutput(), passOutput()},
		errs:    []error{nil, nil, nil},
	}
	beads := &mockBeadClient{
		children: []BeadInfo{
			{ID: "cap-1", Title: "Task 1"},
			{ID: "cap-2", Title: "Task 2"},
			{ID: "cap-3", Title: "Task 3"},
		},
	}
	store := &mockStateStore{}
	cb := &mockCallback{}
	config := Config{FailureMode: "abort", CircuitBreaker: 3}

	r := NewRunner(pipeline, beads, store, config, cb)

	// When Run is called
	err := r.Run(context.Background(), "cap-feature")

	// Then it completes without error
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// And all 3 tasks were started
	if len(cb.tasksStarted) != 3 {
		t.Errorf("tasks started = %d, want 3", len(cb.tasksStarted))
	}
	// And all 3 tasks completed
	if len(cb.tasksCompleted) != 3 {
		t.Errorf("tasks completed = %d, want 3", len(cb.tasksCompleted))
	}
	// And all 3 beads were closed
	if len(beads.closed) != 3 {
		t.Errorf("beads closed = %d, want 3", len(beads.closed))
	}
	// And campaign lifecycle events fired
	if !cb.campaignStarted {
		t.Error("campaign start callback not fired")
	}
	if !cb.campaignDone {
		t.Error("campaign complete callback not fired")
	}
	// And state was saved multiple times (after each task + final)
	if len(store.saved) < 3 {
		t.Errorf("state saves = %d, want >= 3", len(store.saved))
	}
	// And final state is completed
	last := store.saved[len(store.saved)-1]
	if last.Status != CampaignCompleted {
		t.Errorf("final state = %q, want %q", last.Status, CampaignCompleted)
	}
}

func TestRun_NoTasks(t *testing.T) {
	// Given no ready children
	beads := &mockBeadClient{children: []BeadInfo{}}
	r := NewRunner(&mockPipeline{}, beads, &mockStateStore{}, Config{}, &mockCallback{})

	// When Run is called
	err := r.Run(context.Background(), "cap-feature")

	// Then it returns ErrNoTasks
	if !errors.Is(err, ErrNoTasks) {
		t.Errorf("expected ErrNoTasks, got %v", err)
	}
}

func TestRun_AbortOnFailure(t *testing.T) {
	// Given task 2 fails, failure_mode=abort
	pipeline := &mockPipeline{
		outputs: []orchestrator.PipelineOutput{passOutput(), {}},
		errs:    []error{nil, fmt.Errorf("task 2 failed")},
	}
	beads := &mockBeadClient{
		children: []BeadInfo{
			{ID: "cap-1", Title: "Task 1"},
			{ID: "cap-2", Title: "Task 2"},
			{ID: "cap-3", Title: "Task 3"},
		},
	}
	store := &mockStateStore{}
	cb := &mockCallback{}
	config := Config{FailureMode: "abort", CircuitBreaker: 3}

	r := NewRunner(pipeline, beads, store, config, cb)

	// When Run is called
	err := r.Run(context.Background(), "cap-feature")

	// Then it returns an error
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// And only 2 tasks were started (abort after task 2)
	if len(cb.tasksStarted) != 2 {
		t.Errorf("tasks started = %d, want 2", len(cb.tasksStarted))
	}
	// And task 2 was reported as failed
	if len(cb.tasksFailed) != 1 || cb.tasksFailed[0] != "cap-2" {
		t.Errorf("tasks failed = %v, want [cap-2]", cb.tasksFailed)
	}
}

func TestRun_ContinueOnFailure(t *testing.T) {
	// Given task 2 fails, failure_mode=continue
	pipeline := &mockPipeline{
		outputs: []orchestrator.PipelineOutput{passOutput(), {}, passOutput()},
		errs:    []error{nil, fmt.Errorf("task 2 failed"), nil},
	}
	beads := &mockBeadClient{
		children: []BeadInfo{
			{ID: "cap-1", Title: "Task 1"},
			{ID: "cap-2", Title: "Task 2"},
			{ID: "cap-3", Title: "Task 3"},
		},
	}
	store := &mockStateStore{}
	cb := &mockCallback{}
	config := Config{FailureMode: "continue", CircuitBreaker: 3}

	r := NewRunner(pipeline, beads, store, config, cb)

	// When Run is called
	err := r.Run(context.Background(), "cap-feature")

	// Then it completes (continue mode doesn't abort)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// And all 3 tasks were started
	if len(cb.tasksStarted) != 3 {
		t.Errorf("tasks started = %d, want 3", len(cb.tasksStarted))
	}
	// And task 2 was reported as failed
	if len(cb.tasksFailed) != 1 {
		t.Errorf("tasks failed = %d, want 1", len(cb.tasksFailed))
	}
}

func TestRun_CircuitBreaker(t *testing.T) {
	// Given 3 consecutive failures, circuit_breaker=2
	pipeline := &mockPipeline{
		outputs: []orchestrator.PipelineOutput{{}, {}},
		errs:    []error{fmt.Errorf("fail 1"), fmt.Errorf("fail 2")},
	}
	beads := &mockBeadClient{
		children: []BeadInfo{
			{ID: "cap-1", Title: "Task 1"},
			{ID: "cap-2", Title: "Task 2"},
			{ID: "cap-3", Title: "Task 3"},
		},
	}
	store := &mockStateStore{}
	cb := &mockCallback{}
	config := Config{FailureMode: "continue", CircuitBreaker: 2}

	r := NewRunner(pipeline, beads, store, config, cb)

	// When Run is called
	err := r.Run(context.Background(), "cap-feature")

	// Then it trips the circuit breaker
	if !errors.Is(err, ErrCircuitBroken) {
		t.Errorf("expected ErrCircuitBroken, got %v", err)
	}
	// And only 2 tasks were started (breaker trips before 3rd)
	if len(cb.tasksStarted) != 2 {
		t.Errorf("tasks started = %d, want 2", len(cb.tasksStarted))
	}
}

func TestRun_CircuitBreakerResets(t *testing.T) {
	// Given: fail, pass, fail (circuit_breaker=2 — should NOT trip because pass resets)
	pipeline := &mockPipeline{
		outputs: []orchestrator.PipelineOutput{{}, passOutput(), {}},
		errs:    []error{fmt.Errorf("fail"), nil, fmt.Errorf("fail")},
	}
	beads := &mockBeadClient{
		children: []BeadInfo{
			{ID: "cap-1", Title: "Task 1"},
			{ID: "cap-2", Title: "Task 2"},
			{ID: "cap-3", Title: "Task 3"},
		},
	}
	config := Config{FailureMode: "continue", CircuitBreaker: 2}

	r := NewRunner(pipeline, beads, &mockStateStore{}, config, &mockCallback{})

	// When Run is called
	err := r.Run(context.Background(), "cap-feature")

	// Then it completes (breaker was reset by success)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_DiscoveryFiling(t *testing.T) {
	// Given a pipeline output with findings
	pipeline := &mockPipeline{
		outputs: []orchestrator.PipelineOutput{{
			Completed: true,
			PhaseResults: []orchestrator.PhaseResult{{
				PhaseName: "code-review",
				Signal: provider.Signal{
					Status:   provider.StatusPass,
					Feedback: "ok",
					Summary:  "done",
					Findings: []provider.Finding{
						{Title: "Missing nil check", Severity: "minor", Description: "line 47"},
						{Title: "SQL injection", Severity: "critical", Description: "unsafe query"},
					},
					FilesChanged: []string{},
				},
			}},
		}},
		errs: []error{nil},
	}
	beads := &mockBeadClient{
		children: []BeadInfo{{ID: "cap-1", Title: "Task 1"}},
		createID: "cap-new",
	}
	cb := &mockCallback{}
	config := Config{
		FailureMode:     "abort",
		CircuitBreaker:  3,
		DiscoveryFiling: true,
	}

	r := NewRunner(pipeline, beads, &mockStateStore{}, config, cb)

	// When Run is called
	err := r.Run(context.Background(), "cap-feature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Then 2 beads were created for the 2 findings
	if len(beads.created) != 2 {
		t.Fatalf("beads created = %d, want 2", len(beads.created))
	}
	// And the priority maps from severity
	if beads.created[0].Priority != 2 { // minor → P2
		t.Errorf("finding 1 priority = %d, want 2", beads.created[0].Priority)
	}
	if beads.created[1].Priority != 0 { // critical → P0
		t.Errorf("finding 2 priority = %d, want 0", beads.created[1].Priority)
	}
	// And discovery callbacks fired
	if len(cb.discoveriesFiled) != 2 {
		t.Errorf("discoveries filed = %d, want 2", len(cb.discoveriesFiled))
	}
}

func TestRun_DiscoveryFilingDisabled(t *testing.T) {
	// Given discovery filing is disabled
	pipeline := &mockPipeline{
		outputs: []orchestrator.PipelineOutput{{
			Completed: true,
			PhaseResults: []orchestrator.PhaseResult{{
				Signal: provider.Signal{
					Status: provider.StatusPass, Feedback: "ok", Summary: "done",
					Findings:     []provider.Finding{{Title: "Issue", Severity: "minor"}},
					FilesChanged: []string{},
				},
			}},
		}},
		errs: []error{nil},
	}
	beads := &mockBeadClient{
		children: []BeadInfo{{ID: "cap-1", Title: "Task 1"}},
	}
	config := Config{DiscoveryFiling: false}

	r := NewRunner(pipeline, beads, &mockStateStore{}, config, &mockCallback{})
	err := r.Run(context.Background(), "cap-feature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Then no beads were created
	if len(beads.created) != 0 {
		t.Errorf("beads created = %d, want 0", len(beads.created))
	}
}

func TestRun_Resume(t *testing.T) {
	// Given a saved state with first task completed
	pipeline := &mockPipeline{
		outputs: []orchestrator.PipelineOutput{passOutput()},
		errs:    []error{nil},
	}
	beads := &mockBeadClient{
		children: []BeadInfo{
			{ID: "cap-1", Title: "Task 1"},
			{ID: "cap-2", Title: "Task 2"},
		},
	}
	store := &mockStateStore{
		loaded: map[string]State{
			"cap-feature": {
				ID:             "cap-feature",
				ParentBeadID:   "cap-feature",
				Status:         CampaignRunning,
				CurrentTaskIdx: 1,
				Tasks: []TaskResult{
					{BeadID: "cap-1", Status: TaskCompleted},
					{BeadID: "cap-2", Status: TaskPending},
				},
			},
		},
	}
	cb := &mockCallback{}
	config := Config{FailureMode: "abort", CircuitBreaker: 3}

	r := NewRunner(pipeline, beads, store, config, cb)

	// When Run resumes
	err := r.Run(context.Background(), "cap-feature")

	// Then only task 2 was started (task 1 already completed)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cb.tasksStarted) != 1 {
		t.Errorf("tasks started = %d, want 1", len(cb.tasksStarted))
	}
	if cb.tasksStarted[0] != "cap-2" {
		t.Errorf("started task = %q, want %q", cb.tasksStarted[0], "cap-2")
	}
}

func TestRun_Validation(t *testing.T) {
	// Given all tasks pass and validation is configured
	pipeline := &mockPipeline{
		outputs: []orchestrator.PipelineOutput{
			passOutput(), // task 1
			passOutput(), // validation
		},
		errs: []error{nil, nil},
	}
	beads := &mockBeadClient{
		children: []BeadInfo{{ID: "cap-1", Title: "Task 1"}},
	}
	cb := &mockCallback{}
	config := Config{
		FailureMode:      "abort",
		CircuitBreaker:   3,
		ValidationPhases: "default",
	}

	r := NewRunner(pipeline, beads, &mockStateStore{}, config, cb)

	// When Run is called
	err := r.Run(context.Background(), "cap-feature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Then validation ran
	if !cb.validationStart {
		t.Error("validation start callback not fired")
	}
	if !cb.validationDone {
		t.Error("validation complete callback not fired")
	}
	// And 2 pipeline calls were made (1 task + 1 validation)
	if len(pipeline.calls) != 2 {
		t.Errorf("pipeline calls = %d, want 2", len(pipeline.calls))
	}
}

func TestSeverityToPriority(t *testing.T) {
	tests := []struct {
		severity string
		want     int
	}{
		{"critical", 0},
		{"major", 1},
		{"minor", 2},
		{"nit", 3},
		{"unknown", 3},
	}
	for _, tt := range tests {
		t.Run(tt.severity, func(t *testing.T) {
			if got := severityToPriority(tt.severity); got != tt.want {
				t.Errorf("severityToPriority(%q) = %d, want %d", tt.severity, got, tt.want)
			}
		})
	}
}

func TestRun_CrossRunContext(t *testing.T) {
	// Given cross-run context is enabled and task 1 passes with a summary
	pipeline := &mockPipeline{
		outputs: []orchestrator.PipelineOutput{
			{
				Completed: true,
				PhaseResults: []orchestrator.PhaseResult{{
					PhaseName: "merge",
					Signal: provider.Signal{
						Status: provider.StatusPass, Feedback: "ok",
						Summary: "Implemented user login", FilesChanged: []string{"auth.go"},
						Findings: []provider.Finding{},
					},
				}},
			},
			passOutput(), // task 2
		},
		errs: []error{nil, nil},
	}
	beads := &mockBeadClient{
		children: []BeadInfo{
			{ID: "cap-1", Title: "Login feature"},
			{ID: "cap-2", Title: "Dashboard feature"},
		},
		showInfo: map[string]BeadInfo{
			"cap-1": {ID: "cap-1", Title: "Login feature"},
			"cap-2": {ID: "cap-2", Title: "Dashboard feature"},
		},
	}
	config := Config{
		FailureMode:     "abort",
		CircuitBreaker:  3,
		CrossRunContext: true,
	}

	r := NewRunner(pipeline, beads, &mockStateStore{}, config, &mockCallback{})

	// When Run is called
	err := r.Run(context.Background(), "cap-feature")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Then task 2's pipeline input includes sibling context from task 1
	if len(pipeline.calls) != 2 {
		t.Fatalf("pipeline calls = %d, want 2", len(pipeline.calls))
	}
	task2Input := pipeline.calls[1]
	if len(task2Input.SiblingContext) != 1 {
		t.Fatalf("sibling context len = %d, want 1", len(task2Input.SiblingContext))
	}
	sibling := task2Input.SiblingContext[0]
	if sibling.BeadID != "cap-1" {
		t.Errorf("sibling BeadID = %q, want %q", sibling.BeadID, "cap-1")
	}
	if sibling.Summary != "Implemented user login" {
		t.Errorf("sibling Summary = %q, want %q", sibling.Summary, "Implemented user login")
	}
	if len(sibling.FilesChanged) != 1 || sibling.FilesChanged[0] != "auth.go" {
		t.Errorf("sibling FilesChanged = %v, want [auth.go]", sibling.FilesChanged)
	}
}

func TestRun_ReadyChildrenError(t *testing.T) {
	// Given ReadyChildren returns an error
	beads := &mockBeadClient{childErr: fmt.Errorf("bd not found")}
	r := NewRunner(&mockPipeline{}, beads, &mockStateStore{}, Config{}, &mockCallback{})

	err := r.Run(context.Background(), "cap-feature")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRun_PipelinePausedSetsCampaignPaused(t *testing.T) {
	// Given task 1 passes and task 2 returns ErrPipelinePaused
	pipeline := &mockPipeline{
		outputs: []orchestrator.PipelineOutput{passOutput(), {}},
		errs:    []error{nil, orchestrator.ErrPipelinePaused},
	}
	beads := &mockBeadClient{
		children: []BeadInfo{
			{ID: "cap-1", Title: "Task 1"},
			{ID: "cap-2", Title: "Task 2"},
			{ID: "cap-3", Title: "Task 3"},
		},
	}
	store := &mockStateStore{}
	cb := &mockCallback{}
	config := Config{FailureMode: "abort", CircuitBreaker: 3}

	r := NewRunner(pipeline, beads, store, config, cb)

	// When Run is called
	err := r.Run(context.Background(), "cap-feature")

	// Then it returns ErrCampaignPaused
	if !errors.Is(err, ErrCampaignPaused) {
		t.Fatalf("expected ErrCampaignPaused, got %v", err)
	}
	// And state was saved as paused
	if len(store.saved) == 0 {
		t.Fatal("expected state to be saved")
	}
	last := store.saved[len(store.saved)-1]
	if last.Status != CampaignPaused {
		t.Errorf("saved state = %q, want %q", last.Status, CampaignPaused)
	}
	// And the paused task is set back to pending (not failed)
	for _, task := range last.Tasks {
		if task.BeadID == "cap-2" {
			if task.Status != TaskPending {
				t.Errorf("paused task status = %q, want %q", task.Status, TaskPending)
			}
		}
	}
	// And no tasks were reported as failed
	if len(cb.tasksFailed) != 0 {
		t.Errorf("tasks failed = %d, want 0 (pause is not a failure)", len(cb.tasksFailed))
	}
}

func TestRun_ResumeFromPausedState(t *testing.T) {
	// Given a saved paused state with task 1 completed, task 2 pending
	pipeline := &mockPipeline{
		outputs: []orchestrator.PipelineOutput{passOutput(), passOutput()},
		errs:    []error{nil, nil},
	}
	beads := &mockBeadClient{
		children: []BeadInfo{
			{ID: "cap-1", Title: "Task 1"},
			{ID: "cap-2", Title: "Task 2"},
			{ID: "cap-3", Title: "Task 3"},
		},
	}
	store := &mockStateStore{
		loaded: map[string]State{
			"cap-feature": {
				ID:             "cap-feature",
				ParentBeadID:   "cap-feature",
				Status:         CampaignPaused,
				CurrentTaskIdx: 1,
				Tasks: []TaskResult{
					{BeadID: "cap-1", Status: TaskCompleted},
					{BeadID: "cap-2", Status: TaskPending},
					{BeadID: "cap-3", Status: TaskPending},
				},
			},
		},
	}
	cb := &mockCallback{}
	config := Config{FailureMode: "abort", CircuitBreaker: 3}

	r := NewRunner(pipeline, beads, store, config, cb)

	// When Run resumes from paused state
	err := r.Run(context.Background(), "cap-feature")

	// Then it completes without error
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// And only tasks 2 and 3 were started
	if len(cb.tasksStarted) != 2 {
		t.Errorf("tasks started = %d, want 2", len(cb.tasksStarted))
	}
	if cb.tasksStarted[0] != "cap-2" {
		t.Errorf("first started task = %q, want %q", cb.tasksStarted[0], "cap-2")
	}
	if cb.tasksStarted[1] != "cap-3" {
		t.Errorf("second started task = %q, want %q", cb.tasksStarted[1], "cap-3")
	}
}
