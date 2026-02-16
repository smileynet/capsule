package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/smileynet/capsule/internal/prompt"
	"github.com/smileynet/capsule/internal/provider"
	"github.com/smileynet/capsule/internal/worklog"
)

// Compile-time check: provider.MockProvider satisfies the orchestrator's Provider interface.
var _ Provider = (*provider.MockProvider)(nil)

// --- Test mocks ---

// mockResponse holds a single provider call response.
type mockResponse struct {
	result provider.Result
	err    error
}

// sequenceProvider returns pre-configured responses in order.
type sequenceProvider struct {
	responses []mockResponse
	calls     []providerCall
	callIdx   int
}

type providerCall struct {
	prompt  string
	workDir string
}

func (m *sequenceProvider) Name() string { return "mock" }

func (m *sequenceProvider) Execute(_ context.Context, p, workDir string) (provider.Result, error) {
	m.calls = append(m.calls, providerCall{prompt: p, workDir: workDir})
	if m.callIdx >= len(m.responses) {
		return provider.Result{}, fmt.Errorf("unexpected provider call %d (have %d responses)",
			m.callIdx+1, len(m.responses))
	}
	resp := m.responses[m.callIdx]
	m.callIdx++
	return resp.result, resp.err
}

type mockPromptLoader struct {
	composeFunc func(phaseName string, ctx prompt.Context) (string, error)
}

func (m *mockPromptLoader) Compose(phaseName string, ctx prompt.Context) (string, error) {
	if m.composeFunc != nil {
		return m.composeFunc(phaseName, ctx)
	}
	return "prompt:" + phaseName, nil
}

type mockWorktreeMgr struct {
	createErr error
	path      string
	created   []string
}

func (m *mockWorktreeMgr) Create(id, _ string) error {
	m.created = append(m.created, id)
	return m.createErr
}

func (m *mockWorktreeMgr) Remove(string, bool) error { return nil }

func (m *mockWorktreeMgr) Path(string) string { return m.path }

type mockWorklogMgr struct {
	createErr  error
	appendErr  error
	archiveErr error
	entries    []worklog.PhaseEntry
	archived   bool
	created    bool
}

func (m *mockWorklogMgr) Create(string, worklog.BeadContext) error {
	m.created = true
	return m.createErr
}

func (m *mockWorklogMgr) AppendPhaseEntry(_ string, entry worklog.PhaseEntry) error {
	m.entries = append(m.entries, entry)
	return m.appendErr
}

func (m *mockWorklogMgr) Archive(string, string) error {
	m.archived = true
	return m.archiveErr
}

// --- Signal helpers ---

func makeSignalJSON(status provider.Status, feedback, summary string) string {
	s := provider.Signal{
		Status:       status,
		Feedback:     feedback,
		Summary:      summary,
		FilesChanged: []string{},
	}
	data, _ := json.Marshal(s)
	return string(data)
}

func passResponse() mockResponse {
	return mockResponse{
		result: provider.Result{Output: makeSignalJSON(provider.StatusPass, "ok", "passed")},
	}
}

func needsWorkResponse(feedback string) mockResponse {
	return mockResponse{
		result: provider.Result{Output: makeSignalJSON(provider.StatusNeedsWork, feedback, "needs work")},
	}
}

func errorResponse(feedback string) mockResponse {
	return mockResponse{
		result: provider.Result{Output: makeSignalJSON(provider.StatusError, feedback, "error occurred")},
	}
}

// nPassResponses returns n consecutive PASS mock responses.
func nPassResponses(n int) []mockResponse {
	responses := make([]mockResponse, n)
	for i := range responses {
		responses[i] = passResponse()
	}
	return responses
}

// --- Simple test phases ---

func twoPhases() []PhaseDefinition {
	return []PhaseDefinition{
		{Name: "worker", Kind: Worker, MaxRetries: 3},
		{Name: "reviewer", Kind: Reviewer, MaxRetries: 3, RetryTarget: "worker"},
	}
}

// --- PipelineError tests ---

func TestPipelineError_Error_WithErr(t *testing.T) {
	// Given a PipelineError with an underlying error
	pe := &PipelineError{Phase: "execute", Attempt: 2, Err: fmt.Errorf("connection refused")}

	// When Error is called
	got := pe.Error()

	// Then the message includes phase, attempt, and error
	want := `pipeline: phase "execute" attempt 2: connection refused`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPipelineError_Error_WithSignal(t *testing.T) {
	// Given a PipelineError with a signal (no underlying error)
	pe := &PipelineError{
		Phase: "test-review", Attempt: 1,
		Signal: provider.Signal{Status: provider.StatusError, Feedback: "tests fail"},
	}

	// When Error is called
	got := pe.Error()

	// Then the message includes status and feedback
	want := `pipeline: phase "test-review" attempt 1: status ERROR: tests fail`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPipelineError_Unwrap(t *testing.T) {
	// Given a PipelineError wrapping a sentinel error
	sentinel := fmt.Errorf("wrapped error")
	pe := &PipelineError{Phase: "setup", Err: sentinel}

	// When Unwrap is called
	unwrapped := pe.Unwrap()

	// Then the underlying error is returned
	if !errors.Is(unwrapped, sentinel) {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, sentinel)
	}
}

func TestPipelineError_Error_SetupPhase(t *testing.T) {
	// Given a PipelineError with attempt 0 (setup/teardown)
	pe := &PipelineError{Phase: "setup", Err: fmt.Errorf("worktree failed")}

	// When Error is called
	got := pe.Error()

	// Then the message omits the attempt number
	want := `pipeline: phase "setup": worktree failed`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// --- Constructor tests ---

func TestNew_DefaultPhases(t *testing.T) {
	// Given a provider
	p := &provider.MockProvider{NameVal: "test"}

	// When an Orchestrator is created with defaults
	o := New(p)

	// Then it has the default 6 phases
	if got := len(o.phases); got != 6 {
		t.Fatalf("default phases = %d, want 6", got)
	}
	if o.phases[0].Name != "test-writer" {
		t.Errorf("first phase = %q, want %q", o.phases[0].Name, "test-writer")
	}
}

func TestNew_WithOptions(t *testing.T) {
	// Given mocks and custom phases
	p := &provider.MockProvider{NameVal: "test"}
	pl := &mockPromptLoader{}
	wt := &mockWorktreeMgr{}
	wl := &mockWorklogMgr{}
	phases := []PhaseDefinition{{Name: "custom", Kind: Worker, MaxRetries: 1}}
	var captured StatusUpdate
	cb := func(su StatusUpdate) { captured = su }

	// When options are applied
	o := New(p,
		WithPromptLoader(pl),
		WithWorktreeManager(wt),
		WithWorklogManager(wl),
		WithPhases(phases),
		WithStatusCallback(cb),
		WithBaseBranch("develop"),
	)

	// Then all options are set correctly
	if o.promptLoader != pl {
		t.Error("promptLoader not set")
	}
	if o.worktreeMgr != wt {
		t.Error("worktreeMgr not set")
	}
	if o.worklogMgr != wl {
		t.Error("worklogMgr not set")
	}
	if len(o.phases) != 1 || o.phases[0].Name != "custom" {
		t.Errorf("phases = %v, want [custom]", o.phases)
	}
	if o.baseBranch != "develop" {
		t.Errorf("baseBranch = %q, want %q", o.baseBranch, "develop")
	}

	// Verify callback is wired
	o.statusCallback(StatusUpdate{Phase: "test"})
	if captured.Phase != "test" {
		t.Error("statusCallback not set")
	}
}

func TestRunPipeline_NilPromptLoader(t *testing.T) {
	// Given an Orchestrator without a promptLoader
	o := New(&provider.MockProvider{NameVal: "test"})

	// When RunPipeline is called
	_, err := o.RunPipeline(context.Background(), PipelineInput{BeadID: "cap-1"})

	// Then it returns a setup PipelineError
	var pe *PipelineError
	if !errors.As(err, &pe) {
		t.Fatalf("expected PipelineError, got %T: %v", err, err)
	}
	if pe.Phase != "setup" {
		t.Errorf("Phase = %q, want %q", pe.Phase, "setup")
	}
}

// --- runPhasePair tests ---

func TestRunPhasePair_HappyPath(t *testing.T) {
	// Given a worker that PASSes and a reviewer that PASSes
	sp := &sequenceProvider{responses: []mockResponse{passResponse(), passResponse()}}
	o := New(sp,
		WithPromptLoader(&mockPromptLoader{}),
		WithPhases(twoPhases()),
	)

	worker := o.phases[0]
	reviewer := o.phases[1]
	pCtx := prompt.Context{BeadID: "cap-1"}

	// When runPhasePair executes
	signal, err := o.runPhasePair(context.Background(), worker, reviewer, pCtx, "/tmp/wt", "1/1", "", 1)

	// Then it succeeds with a PASS signal
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if signal.Status != provider.StatusPass {
		t.Errorf("signal.Status = %q, want %q", signal.Status, provider.StatusPass)
	}
	// And both phases executed exactly once
	if got := len(sp.calls); got != 2 {
		t.Errorf("provider called %d times, want 2", got)
	}
}

func TestRunPhasePair_RetryOnNeedsWork(t *testing.T) {
	// Given: worker PASSes, reviewer NEEDS_WORK, then worker PASSes, reviewer PASSes
	sp := &sequenceProvider{responses: []mockResponse{
		passResponse(),                      // attempt 1: worker
		needsWorkResponse("fix formatting"), // attempt 1: reviewer
		passResponse(),                      // attempt 2: worker (retry)
		passResponse(),                      // attempt 2: reviewer
	}}
	o := New(sp,
		WithPromptLoader(&mockPromptLoader{}),
		WithPhases(twoPhases()),
	)

	worker := o.phases[0]
	reviewer := o.phases[1]
	pCtx := prompt.Context{BeadID: "cap-1"}

	// When runPhasePair executes
	signal, err := o.runPhasePair(context.Background(), worker, reviewer, pCtx, "/tmp/wt", "1/1", "", 1)

	// Then it succeeds after retry
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if signal.Status != provider.StatusPass {
		t.Errorf("signal.Status = %q, want %q", signal.Status, provider.StatusPass)
	}
	// And 4 provider calls were made (2 per attempt)
	if got := len(sp.calls); got != 4 {
		t.Errorf("provider called %d times, want 4", got)
	}
}

func TestRunPhasePair_FeedbackPassedToWorker(t *testing.T) {
	// Given a prompt loader that captures the feedback
	var capturedFeedback []string
	pl := &mockPromptLoader{
		composeFunc: func(phaseName string, ctx prompt.Context) (string, error) {
			capturedFeedback = append(capturedFeedback, ctx.Feedback)
			return "prompt:" + phaseName, nil
		},
	}

	sp := &sequenceProvider{responses: []mockResponse{
		passResponse(),                       // attempt 1: worker
		needsWorkResponse("fix indentation"), // attempt 1: reviewer
		passResponse(),                       // attempt 2: worker (retry with feedback)
		passResponse(),                       // attempt 2: reviewer
	}}
	o := New(sp,
		WithPromptLoader(pl),
		WithPhases(twoPhases()),
	)

	worker := o.phases[0]
	reviewer := o.phases[1]
	pCtx := prompt.Context{BeadID: "cap-1"}

	// When runPhasePair executes
	_, err := o.runPhasePair(context.Background(), worker, reviewer, pCtx, "/tmp/wt", "1/1", "", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Then the retry worker receives the reviewer's feedback
	// Calls: worker(""), reviewer(""), worker("fix indentation"), reviewer("")
	if len(capturedFeedback) != 4 {
		t.Fatalf("got %d compose calls, want 4", len(capturedFeedback))
	}
	if capturedFeedback[0] != "" {
		t.Errorf("first worker feedback = %q, want empty", capturedFeedback[0])
	}
	if capturedFeedback[2] != "fix indentation" {
		t.Errorf("retry worker feedback = %q, want %q", capturedFeedback[2], "fix indentation")
	}
}

func TestRunPhasePair_WorkerError(t *testing.T) {
	// Given a worker that returns ERROR
	sp := &sequenceProvider{responses: []mockResponse{errorResponse("compilation failed")}}
	o := New(sp,
		WithPromptLoader(&mockPromptLoader{}),
		WithPhases(twoPhases()),
	)

	worker := o.phases[0]
	reviewer := o.phases[1]
	pCtx := prompt.Context{BeadID: "cap-1"}

	// When runPhasePair executes
	_, err := o.runPhasePair(context.Background(), worker, reviewer, pCtx, "/tmp/wt", "1/1", "", 1)

	// Then it returns a PipelineError for the worker phase
	var pe *PipelineError
	if !errors.As(err, &pe) {
		t.Fatalf("expected PipelineError, got %T: %v", err, err)
	}
	if pe.Phase != "worker" {
		t.Errorf("Phase = %q, want %q", pe.Phase, "worker")
	}
	if pe.Signal.Status != provider.StatusError {
		t.Errorf("Signal.Status = %q, want %q", pe.Signal.Status, provider.StatusError)
	}
	// And the reviewer never ran
	if got := len(sp.calls); got != 1 {
		t.Errorf("provider called %d times, want 1", got)
	}
}

func TestRunPhasePair_ReviewerError(t *testing.T) {
	// Given a worker PASSes but reviewer returns ERROR
	sp := &sequenceProvider{responses: []mockResponse{
		passResponse(),
		errorResponse("internal error"),
	}}
	o := New(sp,
		WithPromptLoader(&mockPromptLoader{}),
		WithPhases(twoPhases()),
	)

	worker := o.phases[0]
	reviewer := o.phases[1]
	pCtx := prompt.Context{BeadID: "cap-1"}

	// When runPhasePair executes
	_, err := o.runPhasePair(context.Background(), worker, reviewer, pCtx, "/tmp/wt", "1/1", "", 1)

	// Then it returns a PipelineError for the reviewer phase
	var pe *PipelineError
	if !errors.As(err, &pe) {
		t.Fatalf("expected PipelineError, got %T: %v", err, err)
	}
	if pe.Phase != "reviewer" {
		t.Errorf("Phase = %q, want %q", pe.Phase, "reviewer")
	}
}

func TestRunPhasePair_MaxRetriesExceeded(t *testing.T) {
	// Given a reviewer that always returns NEEDS_WORK (MaxRetries = 3)
	sp := &sequenceProvider{responses: []mockResponse{
		passResponse(), needsWorkResponse("fix 1"), // attempt 1
		passResponse(), needsWorkResponse("fix 2"), // attempt 2
		passResponse(), needsWorkResponse("fix 3"), // attempt 3
	}}
	o := New(sp,
		WithPromptLoader(&mockPromptLoader{}),
		WithPhases(twoPhases()),
	)

	worker := o.phases[0]
	reviewer := o.phases[1]
	pCtx := prompt.Context{BeadID: "cap-1"}

	// When runPhasePair executes
	_, err := o.runPhasePair(context.Background(), worker, reviewer, pCtx, "/tmp/wt", "1/1", "", 1)

	// Then it fails with retries exhausted
	var pe *PipelineError
	if !errors.As(err, &pe) {
		t.Fatalf("expected PipelineError, got %T: %v", err, err)
	}
	if pe.Attempt != 3 {
		t.Errorf("Attempt = %d, want 3", pe.Attempt)
	}
	if want := `pipeline: phase "reviewer" attempt 3: failed after 3 attempts`; pe.Error() != want {
		t.Errorf("Error() = %q, want %q", pe.Error(), want)
	}
	// And all 6 provider calls were made (3 attempts x 2 phases)
	if got := len(sp.calls); got != 6 {
		t.Errorf("provider called %d times, want 6", got)
	}
}

func TestRunPhasePair_ProviderError(t *testing.T) {
	// Given the provider returns an execution error
	sp := &sequenceProvider{responses: []mockResponse{
		{err: fmt.Errorf("network timeout")},
	}}
	o := New(sp,
		WithPromptLoader(&mockPromptLoader{}),
		WithPhases(twoPhases()),
	)

	worker := o.phases[0]
	reviewer := o.phases[1]
	pCtx := prompt.Context{BeadID: "cap-1"}

	// When runPhasePair executes
	_, err := o.runPhasePair(context.Background(), worker, reviewer, pCtx, "/tmp/wt", "1/1", "", 1)

	// Then it returns a PipelineError wrapping the provider error
	var pe *PipelineError
	if !errors.As(err, &pe) {
		t.Fatalf("expected PipelineError, got %T: %v", err, err)
	}
	if pe.Err == nil {
		t.Fatal("expected underlying error, got nil")
	}
}

func TestRunPhasePair_StatusCallbacks(t *testing.T) {
	// Given a callback that records all updates
	var updates []StatusUpdate
	cb := func(su StatusUpdate) { updates = append(updates, su) }

	sp := &sequenceProvider{responses: []mockResponse{passResponse(), passResponse()}}
	o := New(sp,
		WithPromptLoader(&mockPromptLoader{}),
		WithPhases(twoPhases()),
		WithStatusCallback(cb),
	)

	worker := o.phases[0]
	reviewer := o.phases[1]
	pCtx := prompt.Context{BeadID: "cap-42"}

	// When runPhasePair executes
	_, err := o.runPhasePair(context.Background(), worker, reviewer, pCtx, "/tmp/wt", "1/2", "", 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Then callbacks fire for: worker Running, worker Passed, reviewer Running, reviewer Passed
	if len(updates) != 4 {
		t.Fatalf("got %d updates, want 4", len(updates))
	}
	if updates[0].Phase != "worker" || updates[0].Status != PhaseRunning {
		t.Errorf("update[0] = %s/%s, want worker/running", updates[0].Phase, updates[0].Status)
	}
	if updates[1].Phase != "worker" || updates[1].Status != PhasePassed {
		t.Errorf("update[1] = %s/%s, want worker/passed", updates[1].Phase, updates[1].Status)
	}
	if updates[2].Phase != "reviewer" || updates[2].Status != PhaseRunning {
		t.Errorf("update[2] = %s/%s, want reviewer/running", updates[2].Phase, updates[2].Status)
	}
	if updates[3].Phase != "reviewer" || updates[3].Status != PhasePassed {
		t.Errorf("update[3] = %s/%s, want reviewer/passed", updates[3].Phase, updates[3].Status)
	}
	// Running updates have nil Signal and zero Duration; completion updates have non-nil Signal and non-zero Duration
	if updates[0].Signal != nil {
		t.Error("update[0] (running) should have nil Signal")
	}
	if updates[0].Duration != 0 {
		t.Errorf("update[0] (running) Duration = %v, want 0", updates[0].Duration)
	}
	if updates[1].Signal == nil {
		t.Error("update[1] (passed) should have non-nil Signal")
	}
	if updates[1].Duration == 0 {
		t.Error("update[1] (passed) should have non-zero Duration")
	}
	if updates[2].Signal != nil {
		t.Error("update[2] (running) should have nil Signal")
	}
	if updates[3].Signal == nil {
		t.Error("update[3] (passed) should have non-nil Signal")
	}
	if updates[3].Duration == 0 {
		t.Error("update[3] (passed) should have non-zero Duration")
	}
	// And all updates carry the bead ID
	for i, u := range updates {
		if u.BeadID != "cap-42" {
			t.Errorf("update[%d].BeadID = %q, want %q", i, u.BeadID, "cap-42")
		}
	}
}

// --- RunPipeline tests ---

func TestRunPipeline_AllPhasesPass(t *testing.T) {
	// Given all 6 default phases return PASS
	sp := &sequenceProvider{responses: []mockResponse{
		passResponse(), // test-writer
		passResponse(), // test-review
		passResponse(), // execute
		passResponse(), // execute-review
		passResponse(), // sign-off
		passResponse(), // merge
	}}
	wt := &mockWorktreeMgr{path: "/tmp/worktrees/cap-1"}
	wl := &mockWorklogMgr{}

	o := New(sp,
		WithPromptLoader(&mockPromptLoader{}),
		WithWorktreeManager(wt),
		WithWorklogManager(wl),
	)

	input := PipelineInput{BeadID: "cap-1", Title: "Test task"}

	// When RunPipeline executes
	_, err := o.RunPipeline(context.Background(), input)

	// Then it completes without error
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// And the worktree was created
	if len(wt.created) != 1 || wt.created[0] != "cap-1" {
		t.Errorf("worktree created = %v, want [cap-1]", wt.created)
	}
	// And the worklog was created and archived
	if !wl.created {
		t.Error("worklog was not created")
	}
	if !wl.archived {
		t.Error("worklog was not archived")
	}
	// And worklog has entries for all 6 phases
	if got := len(wl.entries); got != 6 {
		t.Errorf("worklog entries = %d, want 6", got)
	}
}

func TestRunPipeline_PhaseErrorAborts(t *testing.T) {
	// Given execute-review returns ERROR (4th phase)
	sp := &sequenceProvider{responses: []mockResponse{
		passResponse(),                    // test-writer
		passResponse(),                    // test-review
		passResponse(),                    // execute
		errorResponse("tests are broken"), // execute-review
	}}

	o := New(sp,
		WithPromptLoader(&mockPromptLoader{}),
	)

	input := PipelineInput{BeadID: "cap-1"}

	// When RunPipeline executes
	_, err := o.RunPipeline(context.Background(), input)

	// Then it returns a PipelineError for execute-review
	var pe *PipelineError
	if !errors.As(err, &pe) {
		t.Fatalf("expected PipelineError, got %T: %v", err, err)
	}
	if pe.Phase != "execute-review" {
		t.Errorf("Phase = %q, want %q", pe.Phase, "execute-review")
	}
	// And only 4 provider calls were made (pipeline aborted)
	if got := len(sp.calls); got != 4 {
		t.Errorf("provider called %d times, want 4", got)
	}
}

func TestRunPipeline_ReviewerRetryFlow(t *testing.T) {
	// Given test-review says NEEDS_WORK, then PASS on retry
	sp := &sequenceProvider{responses: []mockResponse{
		passResponse(),                 // test-writer (initial)
		needsWorkResponse("add tests"), // test-review (initial -> NEEDS_WORK)
		passResponse(),                 // test-writer (retry attempt 2)
		passResponse(),                 // test-review (retry attempt 2 -> PASS)
		passResponse(),                 // execute
		passResponse(),                 // execute-review
		passResponse(),                 // sign-off
		passResponse(),                 // merge
	}}

	o := New(sp,
		WithPromptLoader(&mockPromptLoader{}),
	)

	input := PipelineInput{BeadID: "cap-1"}

	// When RunPipeline executes
	_, err := o.RunPipeline(context.Background(), input)

	// Then it completes successfully after retry
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// And 8 provider calls were made
	if got := len(sp.calls); got != 8 {
		t.Errorf("provider called %d times, want 8", got)
	}
}

func TestRunPipeline_StandaloneReviewerRetry(t *testing.T) {
	// Given sign-off (standalone reviewer) says NEEDS_WORK, then PASS on retry
	sp := &sequenceProvider{responses: []mockResponse{
		passResponse(),                     // test-writer
		passResponse(),                     // test-review
		passResponse(),                     // execute
		passResponse(),                     // execute-review
		needsWorkResponse("needs cleanup"), // sign-off (initial -> NEEDS_WORK)
		passResponse(),                     // execute (retry)
		passResponse(),                     // sign-off (retry -> PASS)
		passResponse(),                     // merge
	}}

	o := New(sp,
		WithPromptLoader(&mockPromptLoader{}),
	)

	input := PipelineInput{BeadID: "cap-1"}

	// When RunPipeline executes
	_, err := o.RunPipeline(context.Background(), input)

	// Then it completes successfully
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// And 8 calls: 5 initial + 2 retry + 1 merge
	if got := len(sp.calls); got != 8 {
		t.Errorf("provider called %d times, want 8", got)
	}
}

func TestRunPipeline_WorktreeCreationFailure(t *testing.T) {
	// Given worktree creation fails
	wt := &mockWorktreeMgr{createErr: fmt.Errorf("branch already exists")}
	o := New(&provider.MockProvider{NameVal: "test"},
		WithPromptLoader(&mockPromptLoader{}),
		WithWorktreeManager(wt),
	)

	input := PipelineInput{BeadID: "cap-1"}

	// When RunPipeline executes
	_, err := o.RunPipeline(context.Background(), input)

	// Then it returns a setup PipelineError
	var pe *PipelineError
	if !errors.As(err, &pe) {
		t.Fatalf("expected PipelineError, got %T: %v", err, err)
	}
	if pe.Phase != "setup" {
		t.Errorf("Phase = %q, want %q", pe.Phase, "setup")
	}
}

func TestRunPipeline_WorklogCreationFailure(t *testing.T) {
	// Given worklog creation fails
	wl := &mockWorklogMgr{createErr: fmt.Errorf("template missing")}
	o := New(&provider.MockProvider{NameVal: "test"},
		WithPromptLoader(&mockPromptLoader{}),
		WithWorklogManager(wl),
	)

	input := PipelineInput{BeadID: "cap-1"}

	// When RunPipeline executes
	_, err := o.RunPipeline(context.Background(), input)

	// Then it returns a setup PipelineError
	var pe *PipelineError
	if !errors.As(err, &pe) {
		t.Fatalf("expected PipelineError, got %T: %v", err, err)
	}
	if pe.Phase != "setup" {
		t.Errorf("Phase = %q, want %q", pe.Phase, "setup")
	}
}

func TestRunPipeline_ArchiveFailure(t *testing.T) {
	// Given all phases pass but archive fails
	sp := &sequenceProvider{responses: nPassResponses(6)}
	wl := &mockWorklogMgr{archiveErr: fmt.Errorf("disk full")}

	o := New(sp,
		WithPromptLoader(&mockPromptLoader{}),
		WithWorklogManager(wl),
	)

	input := PipelineInput{BeadID: "cap-1"}

	// When RunPipeline executes
	_, err := o.RunPipeline(context.Background(), input)

	// Then it returns a teardown PipelineError
	var pe *PipelineError
	if !errors.As(err, &pe) {
		t.Fatalf("expected PipelineError, got %T: %v", err, err)
	}
	if pe.Phase != "teardown" {
		t.Errorf("Phase = %q, want %q", pe.Phase, "teardown")
	}
}

func TestRunPipeline_StatusCallbacks(t *testing.T) {
	// Given a callback that records all updates
	var updates []StatusUpdate
	cb := func(su StatusUpdate) { updates = append(updates, su) }

	sp := &sequenceProvider{responses: nPassResponses(6)}

	o := New(sp,
		WithPromptLoader(&mockPromptLoader{}),
		WithStatusCallback(cb),
	)

	input := PipelineInput{BeadID: "cap-1"}

	// When RunPipeline executes
	_, err := o.RunPipeline(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Then callbacks fire for each phase (Running + Passed = 2 per phase x 6 phases)
	if got := len(updates); got != 12 {
		t.Fatalf("got %d status updates, want 12", got)
	}
	// And the first update is test-writer Running
	if updates[0].Phase != "test-writer" || updates[0].Status != PhaseRunning {
		t.Errorf("first update = %s/%s, want test-writer/running", updates[0].Phase, updates[0].Status)
	}
	// And the last update is merge Passed
	if updates[11].Phase != "merge" || updates[11].Status != PhasePassed {
		t.Errorf("last update = %s/%s, want merge/passed", updates[11].Phase, updates[11].Status)
	}
	// And completion updates carry Signal data and Duration
	for i, u := range updates {
		if u.Status == PhaseRunning {
			if u.Signal != nil {
				t.Errorf("update[%d] (running) should have nil Signal", i)
			}
			if u.Duration != 0 {
				t.Errorf("update[%d] (running) Duration = %v, want 0", i, u.Duration)
			}
		} else {
			if u.Signal == nil {
				t.Errorf("update[%d] (%s) should have non-nil Signal", i, u.Status)
			}
			if u.Duration == 0 {
				t.Errorf("update[%d] (%s) should have non-zero Duration", i, u.Status)
			}
		}
	}
}

func TestRunPipeline_ContextCancelled(t *testing.T) {
	// Given a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	sp := &sequenceProvider{responses: []mockResponse{
		{err: context.Canceled},
	}}

	o := New(sp,
		WithPromptLoader(&mockPromptLoader{}),
		WithPhases(twoPhases()),
	)

	input := PipelineInput{BeadID: "cap-1"}

	// When RunPipeline executes with cancelled context
	_, err := o.RunPipeline(ctx, input)

	// Then it returns a PipelineError
	var pe *PipelineError
	if !errors.As(err, &pe) {
		t.Fatalf("expected PipelineError, got %T: %v", err, err)
	}
	// And the underlying error wraps context.Canceled
	if !errors.Is(pe.Err, context.Canceled) {
		t.Errorf("expected context.Canceled in chain, got %v", pe.Err)
	}
}

func TestRunPipeline_BaseBranchFromInput(t *testing.T) {
	// Given a worktree manager that captures the base branch
	wt := &branchCapturingWorktreeMgr{path: "/tmp/wt"}
	sp := &sequenceProvider{responses: nPassResponses(6)}

	o := New(sp,
		WithPromptLoader(&mockPromptLoader{}),
		WithWorktreeManager(wt),
		WithBaseBranch("main"),
	)

	input := PipelineInput{BeadID: "cap-1", BaseBranch: "develop"}

	// When RunPipeline executes with a custom base branch
	_, err := o.RunPipeline(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Then the input's BaseBranch is used (not the Orchestrator's default)
	if wt.lastBaseBranch != "develop" {
		t.Errorf("base branch = %q, want %q", wt.lastBaseBranch, "develop")
	}
}

type branchCapturingWorktreeMgr struct {
	path           string
	lastBaseBranch string
}

func (m *branchCapturingWorktreeMgr) Create(id, baseBranch string) error {
	m.lastBaseBranch = baseBranch
	return nil
}

func (m *branchCapturingWorktreeMgr) Remove(string, bool) error { return nil }

func (m *branchCapturingWorktreeMgr) Path(string) string { return m.path }

func TestRunPipeline_PromptContextCarriesMetadata(t *testing.T) {
	// Given a prompt loader that captures context
	var capturedCtx []prompt.Context
	pl := &mockPromptLoader{
		composeFunc: func(phaseName string, ctx prompt.Context) (string, error) {
			capturedCtx = append(capturedCtx, ctx)
			return "prompt:" + phaseName, nil
		},
	}

	// Simple 1-phase pipeline for this test
	phases := []PhaseDefinition{{Name: "worker", Kind: Worker, MaxRetries: 1}}
	sp := &sequenceProvider{responses: []mockResponse{passResponse()}}

	o := New(sp,
		WithPromptLoader(pl),
		WithPhases(phases),
	)

	input := PipelineInput{
		BeadID:      "cap-42",
		Title:       "Fix the bug",
		Description: "There is a null pointer",
	}

	// When RunPipeline executes
	_, err := o.RunPipeline(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Then the prompt context carries BeadID, Title, and Description
	if len(capturedCtx) != 1 {
		t.Fatalf("got %d compose calls, want 1", len(capturedCtx))
	}
	if capturedCtx[0].BeadID != "cap-42" {
		t.Errorf("BeadID = %q, want %q", capturedCtx[0].BeadID, "cap-42")
	}
	if capturedCtx[0].Title != "Fix the bug" {
		t.Errorf("Title = %q, want %q", capturedCtx[0].Title, "Fix the bug")
	}
	if capturedCtx[0].Description != "There is a null pointer" {
		t.Errorf("Description = %q, want %q", capturedCtx[0].Description, "There is a null pointer")
	}
}

// --- executePhase tests ---

func TestExecutePhase_PromptError(t *testing.T) {
	// Given a prompt loader that returns an error
	pl := &mockPromptLoader{
		composeFunc: func(string, prompt.Context) (string, error) {
			return "", fmt.Errorf("template not found")
		},
	}
	sp := &sequenceProvider{}
	o := New(sp, WithPromptLoader(pl), WithPhases(twoPhases()))

	phase := o.phases[0]
	pCtx := prompt.Context{BeadID: "cap-1"}

	// When executePhase is called
	_, err := o.executePhase(context.Background(), phase, pCtx, "/tmp/wt")

	// Then it returns an error mentioning the phase
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "composing prompt for worker") {
		t.Errorf("error = %q, want mention of composing prompt", got)
	}
}

func TestExecutePhase_ParseSignalError(t *testing.T) {
	// Given the provider returns unparseable output
	sp := &sequenceProvider{responses: []mockResponse{
		{result: provider.Result{Output: "not json at all"}},
	}}
	o := New(sp, WithPromptLoader(&mockPromptLoader{}), WithPhases(twoPhases()))

	phase := o.phases[0]
	pCtx := prompt.Context{BeadID: "cap-1"}

	// When executePhase is called
	_, err := o.executePhase(context.Background(), phase, pCtx, "/tmp/wt")

	// Then it returns a parse error
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); !strings.Contains(got, "parsing signal for worker") {
		t.Errorf("error = %q, want mention of parsing signal", got)
	}
}

// --- Gate and optional phase tests ---

type mockGateRunner struct {
	calls   []gateCall
	signals []provider.Signal
	errs    []error
	idx     int
}

type gateCall struct {
	command string
	workDir string
}

func (m *mockGateRunner) Run(_ context.Context, command, workDir string) (provider.Signal, error) {
	m.calls = append(m.calls, gateCall{command: command, workDir: workDir})
	if m.idx >= len(m.signals) {
		return provider.Signal{}, fmt.Errorf("unexpected gate call %d", m.idx+1)
	}
	sig := m.signals[m.idx]
	var err error
	if m.idx < len(m.errs) {
		err = m.errs[m.idx]
	}
	m.idx++
	return sig, err
}

func TestRunPipeline_GatePhasePass(t *testing.T) {
	// Given a pipeline with a gate phase
	gr := &mockGateRunner{
		signals: []provider.Signal{{
			Status: provider.StatusPass, Feedback: "ok", Summary: "lint passed",
			FilesChanged: []string{}, Findings: []provider.Finding{},
		}},
	}
	sp := &sequenceProvider{responses: []mockResponse{
		passResponse(), // worker
		passResponse(), // reviewer
	}}

	phases := []PhaseDefinition{
		{Name: "worker", Kind: Worker, MaxRetries: 3},
		{Name: "lint", Kind: Gate, Command: "make lint"},
		{Name: "reviewer", Kind: Reviewer, MaxRetries: 3, RetryTarget: "worker"},
	}

	o := New(sp,
		WithPromptLoader(&mockPromptLoader{}),
		WithPhases(phases),
		WithGateRunner(gr),
	)

	input := PipelineInput{BeadID: "cap-1"}

	// When RunPipeline executes
	_, err := o.RunPipeline(context.Background(), input)

	// Then it completes without error
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// And the gate was called
	if len(gr.calls) != 1 {
		t.Errorf("gate called %d times, want 1", len(gr.calls))
	}
	if gr.calls[0].command != "make lint" {
		t.Errorf("gate command = %q, want %q", gr.calls[0].command, "make lint")
	}
}

func TestRunPipeline_GatePhaseError_Optional(t *testing.T) {
	// Given a pipeline with an optional gate that fails
	gr := &mockGateRunner{
		signals: []provider.Signal{{
			Status: provider.StatusError, Feedback: "lint error", Summary: "exit 1",
			FilesChanged: []string{}, Findings: []provider.Finding{},
		}},
	}
	sp := &sequenceProvider{responses: []mockResponse{
		passResponse(), // worker
		passResponse(), // merge
	}}

	phases := []PhaseDefinition{
		{Name: "worker", Kind: Worker, MaxRetries: 1},
		{Name: "lint", Kind: Gate, Command: "make lint", Optional: true},
		{Name: "merge", Kind: Worker, MaxRetries: 1},
	}

	o := New(sp,
		WithPromptLoader(&mockPromptLoader{}),
		WithPhases(phases),
		WithGateRunner(gr),
	)

	input := PipelineInput{BeadID: "cap-1"}

	// When RunPipeline executes
	_, err := o.RunPipeline(context.Background(), input)

	// Then it completes (optional failure doesn't abort)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// And the merge phase still ran
	if len(sp.calls) != 2 {
		t.Errorf("provider called %d times, want 2", len(sp.calls))
	}
}

func TestRunPipeline_GatePhaseError_Required(t *testing.T) {
	// Given a pipeline with a required gate that fails
	gr := &mockGateRunner{
		signals: []provider.Signal{{
			Status: provider.StatusError, Feedback: "lint error", Summary: "exit 1",
			FilesChanged: []string{}, Findings: []provider.Finding{},
		}},
	}
	sp := &sequenceProvider{responses: []mockResponse{
		passResponse(), // worker
	}}

	phases := []PhaseDefinition{
		{Name: "worker", Kind: Worker, MaxRetries: 1},
		{Name: "lint", Kind: Gate, Command: "make lint"}, // not optional
		{Name: "merge", Kind: Worker, MaxRetries: 1},
	}

	o := New(sp,
		WithPromptLoader(&mockPromptLoader{}),
		WithPhases(phases),
		WithGateRunner(gr),
	)

	input := PipelineInput{BeadID: "cap-1"}

	// When RunPipeline executes
	_, err := o.RunPipeline(context.Background(), input)

	// Then it fails with a PipelineError for the gate phase
	var pe *PipelineError
	if !errors.As(err, &pe) {
		t.Fatalf("expected PipelineError, got %T: %v", err, err)
	}
	if pe.Phase != "lint" {
		t.Errorf("Phase = %q, want %q", pe.Phase, "lint")
	}
}

func TestRunPipeline_GateNoRunner(t *testing.T) {
	// Given a pipeline with a gate but no GateRunner
	sp := &sequenceProvider{responses: []mockResponse{
		passResponse(), // worker
	}}

	phases := []PhaseDefinition{
		{Name: "worker", Kind: Worker, MaxRetries: 1},
		{Name: "lint", Kind: Gate, Command: "make lint"},
	}

	o := New(sp,
		WithPromptLoader(&mockPromptLoader{}),
		WithPhases(phases),
		// No WithGateRunner
	)

	input := PipelineInput{BeadID: "cap-1"}

	// When RunPipeline executes
	_, err := o.RunPipeline(context.Background(), input)

	// Then it returns a PipelineError
	var pe *PipelineError
	if !errors.As(err, &pe) {
		t.Fatalf("expected PipelineError, got %T: %v", err, err)
	}
	if !strings.Contains(pe.Err.Error(), "GateRunner") {
		t.Errorf("error should mention GateRunner, got %q", pe.Err.Error())
	}
}

func TestRunPipeline_SkipStatus(t *testing.T) {
	// Given a phase that returns SKIP
	sp := &sequenceProvider{responses: []mockResponse{
		{result: provider.Result{Output: makeSignalJSON(provider.StatusPass, "ok", "passed")}},
		{result: provider.Result{
			Output: `{"status":"SKIP","feedback":"not applicable","files_changed":[],"summary":"skipped"}`,
		}},
		{result: provider.Result{Output: makeSignalJSON(provider.StatusPass, "ok", "passed")}},
	}}

	var updates []StatusUpdate
	cb := func(su StatusUpdate) { updates = append(updates, su) }

	phases := []PhaseDefinition{
		{Name: "worker", Kind: Worker, MaxRetries: 1},
		{Name: "reviewer", Kind: Worker, MaxRetries: 1}, // returns SKIP
		{Name: "merge", Kind: Worker, MaxRetries: 1},
	}

	o := New(sp,
		WithPromptLoader(&mockPromptLoader{}),
		WithPhases(phases),
		WithStatusCallback(cb),
	)

	input := PipelineInput{BeadID: "cap-1"}

	// When RunPipeline executes
	_, err := o.RunPipeline(context.Background(), input)

	// Then it completes (SKIP doesn't abort)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// And the skipped phase got a PhaseSkipped callback
	var foundSkipped bool
	for _, u := range updates {
		if u.Phase == "reviewer" && u.Status == PhaseSkipped {
			foundSkipped = true
		}
	}
	if !foundSkipped {
		t.Error("expected PhaseSkipped callback for reviewer")
	}
}

// --- findPhase tests ---

func TestFindPhase_Found(t *testing.T) {
	o := New(&provider.MockProvider{NameVal: "test"})

	// When looking up an existing phase
	phase, ok := o.findPhase("execute")

	// Then it is found
	if !ok {
		t.Fatal("expected to find phase")
	}
	if phase.Name != "execute" {
		t.Errorf("Name = %q, want %q", phase.Name, "execute")
	}
}

func TestFindPhase_NotFound(t *testing.T) {
	o := New(&provider.MockProvider{NameVal: "test"})

	// When looking up a non-existent phase
	_, ok := o.findPhase("nonexistent")

	// Then it is not found
	if ok {
		t.Fatal("expected not to find phase")
	}
}
