package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/alecthomas/kong"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/smileynet/capsule/internal/bead"
	"github.com/smileynet/capsule/internal/campaign"
	"github.com/smileynet/capsule/internal/dashboard"
	"github.com/smileynet/capsule/internal/orchestrator"
	"github.com/smileynet/capsule/internal/prompt"
	"github.com/smileynet/capsule/internal/provider"
	"github.com/smileynet/capsule/internal/tui"
	"github.com/smileynet/capsule/internal/worklog"
	"github.com/smileynet/capsule/internal/worktree"
)

// errExitCalled is a sentinel used to catch kong's os.Exit calls in tests.
var errExitCalled = errors.New("exit called")

func TestFeature_GoProjectSkeleton(t *testing.T) {
	t.Run("version flag prints version commit and date", func(t *testing.T) {
		// Given: a CLI parser with version, commit, and date fields
		var cli CLI
		var buf bytes.Buffer
		versionStr := "v1.0.0 abc1234 2026-01-01T00:00:00Z"
		k, err := kong.New(&cli,
			kong.Vars{"version": versionStr},
			kong.Writers(&buf, &buf),
			kong.Exit(func(int) { panic(errExitCalled) }),
		)
		if err != nil {
			t.Fatal(err)
		}

		// When: --version flag is passed
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic from --version flag")
			}
			err, ok := r.(error)
			if !ok || !errors.Is(err, errExitCalled) {
				panic(r)
			}

			// Then: version, commit, and date are all present in output
			output := buf.String()
			for _, want := range []string{"v1.0.0", "abc1234", "2026-01-01T00:00:00Z"} {
				if !strings.Contains(output, want) {
					t.Errorf("version output = %q, want to contain %q", output, want)
				}
			}
		}()

		k.Parse([]string{"--version"}) //nolint:errcheck // --version triggers panic via Exit hook
	})

	t.Run("no args shows usage and errors", func(t *testing.T) {
		// Given: a CLI parser
		var cli CLI
		k, err := kong.New(&cli, kong.Vars{"version": "test"})
		if err != nil {
			t.Fatal(err)
		}

		// When: no arguments are provided
		_, err = k.Parse([]string{})

		// Then: an error is returned (usage printed)
		if err == nil {
			t.Fatal("expected error when no command provided")
		}
	})

	t.Run("run command parses bead ID", func(t *testing.T) {
		// Given: a CLI parser
		var cli CLI
		k, err := kong.New(&cli, kong.Vars{"version": "test"})
		if err != nil {
			t.Fatal(err)
		}

		// When: run command is invoked with a bead ID
		kctx, err := k.Parse([]string{"run", "some-bead-id"})
		if err != nil {
			t.Fatal(err)
		}

		// Then: the command and bead ID are parsed correctly
		if kctx.Command() != "run <bead-id>" {
			t.Errorf("got command %q, want %q", kctx.Command(), "run <bead-id>")
		}
		if cli.Run.BeadID != "some-bead-id" {
			t.Errorf("got bead-id %q, want %q", cli.Run.BeadID, "some-bead-id")
		}
	})

	t.Run("run command accepts flags", func(t *testing.T) {
		// Given: a CLI parser
		var cli CLI
		k, err := kong.New(&cli, kong.Vars{"version": "test"})
		if err != nil {
			t.Fatal(err)
		}

		// When: run command is invoked with all flags
		_, err = k.Parse([]string{
			"run", "bead-123",
			"--provider", "claude",
			"--timeout", "120",
		})
		if err != nil {
			t.Fatal(err)
		}

		// Then: all flags are parsed correctly
		if cli.Run.Provider != "claude" {
			t.Errorf("provider = %q, want %q", cli.Run.Provider, "claude")
		}
		if cli.Run.Timeout != 120 {
			t.Errorf("timeout = %d, want %d", cli.Run.Timeout, 120)
		}
	})

	t.Run("run command accepts --no-tui flag", func(t *testing.T) {
		// Given: a CLI parser
		var cli CLI
		k, err := kong.New(&cli, kong.Vars{"version": "test"})
		if err != nil {
			t.Fatal(err)
		}

		// When: run command is invoked with --no-tui
		_, err = k.Parse([]string{"run", "bead-123", "--no-tui"})
		if err != nil {
			t.Fatal(err)
		}

		// Then: NoTUI flag is set
		if !cli.Run.NoTUI {
			t.Error("NoTUI = false, want true")
		}
	})

	t.Run("run command defaults no-tui to false", func(t *testing.T) {
		// Given: a CLI parser
		var cli CLI
		k, err := kong.New(&cli, kong.Vars{"version": "test"})
		if err != nil {
			t.Fatal(err)
		}

		// When: run command is invoked without --no-tui
		_, err = k.Parse([]string{"run", "bead-123"})
		if err != nil {
			t.Fatal(err)
		}

		// Then: NoTUI defaults to false
		if cli.Run.NoTUI {
			t.Error("NoTUI = true, want false (default)")
		}
	})

	t.Run("run command has sensible defaults", func(t *testing.T) {
		// Given: a CLI parser
		var cli CLI
		k, err := kong.New(&cli, kong.Vars{"version": "test"})
		if err != nil {
			t.Fatal(err)
		}

		// When: run command is invoked with no optional flags
		_, err = k.Parse([]string{"run", "bead-456"})
		if err != nil {
			t.Fatal(err)
		}

		// Then: defaults are applied
		if cli.Run.Provider != "claude" {
			t.Errorf("default provider = %q, want %q", cli.Run.Provider, "claude")
		}
		if cli.Run.Timeout != 300 {
			t.Errorf("default timeout = %d, want %d", cli.Run.Timeout, 300)
		}
	})

	t.Run("run command requires bead ID", func(t *testing.T) {
		// Given: a CLI parser
		var cli CLI
		k, err := kong.New(&cli, kong.Vars{"version": "test"})
		if err != nil {
			t.Fatal(err)
		}

		// When: run command is invoked without a bead ID
		_, err = k.Parse([]string{"run"})

		// Then: an error is returned
		if err == nil {
			t.Fatal("expected error when bead-id missing")
		}
	})

	t.Run("abort command parses bead ID", func(t *testing.T) {
		// Given: a CLI parser
		var cli CLI
		k, err := kong.New(&cli, kong.Vars{"version": "test"})
		if err != nil {
			t.Fatal(err)
		}

		// When: abort command is invoked with a bead ID
		kctx, err := k.Parse([]string{"abort", "bead-789"})
		if err != nil {
			t.Fatal(err)
		}

		// Then: the command and bead ID are parsed correctly
		if kctx.Command() != "abort <bead-id>" {
			t.Errorf("got command %q, want %q", kctx.Command(), "abort <bead-id>")
		}
		if cli.Abort.BeadID != "bead-789" {
			t.Errorf("got bead-id %q, want %q", cli.Abort.BeadID, "bead-789")
		}
	})

	t.Run("clean command parses bead ID", func(t *testing.T) {
		// Given: a CLI parser
		var cli CLI
		k, err := kong.New(&cli, kong.Vars{"version": "test"})
		if err != nil {
			t.Fatal(err)
		}

		// When: clean command is invoked with a bead ID
		kctx, err := k.Parse([]string{"clean", "bead-abc"})
		if err != nil {
			t.Fatal(err)
		}

		// Then: the command and bead ID are parsed correctly
		if kctx.Command() != "clean <bead-id>" {
			t.Errorf("got command %q, want %q", kctx.Command(), "clean <bead-id>")
		}
		if cli.Clean.BeadID != "bead-abc" {
			t.Errorf("got bead-id %q, want %q", cli.Clean.BeadID, "bead-abc")
		}
	})
}

func TestFeature_OrchestratorWiring(t *testing.T) {
	t.Run("plainTextCallback formats timestamped lines", func(t *testing.T) {
		// Given a buffer and a plain text callback
		var buf bytes.Buffer
		cb := plainTextCallback(&buf)

		// When a status update is sent
		cb(orchestrator.StatusUpdate{
			BeadID:   "cap-42",
			Phase:    "test-writer",
			Status:   orchestrator.PhaseRunning,
			Progress: "1/6",
			Attempt:  1,
			MaxRetry: 3,
		})

		// Then output contains the phase name and status
		output := buf.String()
		if !strings.Contains(output, "test-writer") {
			t.Errorf("output missing phase name, got: %q", output)
		}
		if !strings.Contains(output, "running") {
			t.Errorf("output missing status, got: %q", output)
		}
		if !strings.Contains(output, "1/6") {
			t.Errorf("output missing progress, got: %q", output)
		}
	})

	t.Run("plainTextCallback shows attempt on retry", func(t *testing.T) {
		// Given a buffer and a plain text callback
		var buf bytes.Buffer
		cb := plainTextCallback(&buf)

		// When a retry status update is sent
		cb(orchestrator.StatusUpdate{
			BeadID:   "cap-42",
			Phase:    "test-writer",
			Status:   orchestrator.PhaseRunning,
			Progress: "1/6",
			Attempt:  2,
			MaxRetry: 3,
		})

		// Then output includes attempt info
		output := buf.String()
		if !strings.Contains(output, "attempt 2/3") {
			t.Errorf("output missing retry info, got: %q", output)
		}
	})

	t.Run("plainTextCallback shows signal data on completion", func(t *testing.T) {
		// Given a buffer and a plain text callback
		var buf bytes.Buffer
		cb := plainTextCallback(&buf)

		// When a passed update with signal data is sent
		cb(orchestrator.StatusUpdate{
			Phase:    "test-writer",
			Status:   orchestrator.PhasePassed,
			Progress: "1/6",
			Attempt:  1,
			MaxRetry: 3,
			Signal: &provider.Signal{
				Status:       provider.StatusPass,
				FilesChanged: []string{"src/validate_email_test.go"},
				Summary:      "Wrote 7 failing tests",
				Feedback:     "ok",
			},
		})

		// Then output includes files and summary
		output := buf.String()
		if !strings.Contains(output, "files: src/validate_email_test.go") {
			t.Errorf("output missing files, got: %q", output)
		}
		if !strings.Contains(output, "summary: Wrote 7 failing tests") {
			t.Errorf("output missing summary, got: %q", output)
		}
		// Feedback is only shown for failed phases
		if strings.Contains(output, "feedback:") {
			t.Errorf("output should not show feedback for passed phase, got: %q", output)
		}
	})

	t.Run("plainTextCallback shows feedback on failure", func(t *testing.T) {
		// Given a buffer and a plain text callback
		var buf bytes.Buffer
		cb := plainTextCallback(&buf)

		// When a failed update with feedback is sent
		cb(orchestrator.StatusUpdate{
			Phase:    "test-review",
			Status:   orchestrator.PhaseFailed,
			Progress: "2/6",
			Attempt:  1,
			MaxRetry: 3,
			Signal: &provider.Signal{
				Status:   provider.StatusNeedsWork,
				Feedback: "Missing edge case tests",
				Summary:  "needs work",
			},
		})

		// Then output includes feedback
		output := buf.String()
		if !strings.Contains(output, "feedback: Missing edge case tests") {
			t.Errorf("output missing feedback, got: %q", output)
		}
	})

	t.Run("plainTextCallback omits signal data for running status", func(t *testing.T) {
		// Given a buffer and a plain text callback
		var buf bytes.Buffer
		cb := plainTextCallback(&buf)

		// When a running update is sent (Signal should be nil)
		cb(orchestrator.StatusUpdate{
			Phase:    "execute",
			Status:   orchestrator.PhaseRunning,
			Progress: "3/6",
			Attempt:  1,
			MaxRetry: 3,
		})

		// Then output is just the status line with no signal data
		output := buf.String()
		if strings.Contains(output, "files:") || strings.Contains(output, "summary:") {
			t.Errorf("output should not contain signal data for running, got: %q", output)
		}
	})

	t.Run("exitCode returns 0 for nil error", func(t *testing.T) {
		// Given no error
		// When exitCode is called
		code := exitCode(nil)
		// Then it returns 0
		if code != 0 {
			t.Errorf("exitCode(nil) = %d, want 0", code)
		}
	})

	t.Run("exitCode returns 1 for pipeline error", func(t *testing.T) {
		// Given a PipelineError
		err := &orchestrator.PipelineError{Phase: "execute", Attempt: 1, Signal: provider.Signal{Status: provider.StatusError}}
		// When exitCode is called
		code := exitCode(err)
		// Then it returns 1
		if code != 1 {
			t.Errorf("exitCode(PipelineError) = %d, want 1", code)
		}
	})

	t.Run("exitCode returns 2 for setup error", func(t *testing.T) {
		// Given a non-pipeline error (setup/config issue)
		err := fmt.Errorf("config: provider not found")
		// When exitCode is called
		code := exitCode(err)
		// Then it returns 2
		if code != 2 {
			t.Errorf("exitCode(generic) = %d, want 2", code)
		}
	})

	t.Run("exitCode returns 1 for context cancellation", func(t *testing.T) {
		// Given a context.Canceled error wrapped in PipelineError
		err := &orchestrator.PipelineError{Phase: "execute", Err: context.Canceled}
		// When exitCode is called
		code := exitCode(err)
		// Then it returns 1 (pipeline failure, not setup error)
		if code != 1 {
			t.Errorf("exitCode(context.Canceled) = %d, want 1", code)
		}
	})

	t.Run("exitCode returns 1 for campaign ErrNoTasks", func(t *testing.T) {
		// Given a campaign.ErrNoTasks error
		err := campaign.ErrNoTasks
		// When exitCode is called
		code := exitCode(err)
		// Then it returns 1 (runtime failure, not setup error)
		if code != 1 {
			t.Errorf("exitCode(ErrNoTasks) = %d, want 1", code)
		}
	})

	t.Run("exitCode returns 1 for campaign ErrCircuitBroken", func(t *testing.T) {
		// Given a campaign.ErrCircuitBroken error
		err := campaign.ErrCircuitBroken
		// When exitCode is called
		code := exitCode(err)
		// Then it returns 1 (runtime failure, not setup error)
		if code != 1 {
			t.Errorf("exitCode(ErrCircuitBroken) = %d, want 1", code)
		}
	})

	t.Run("RunCmd wires pipeline and returns nil on success", func(t *testing.T) {
		// Given a RunCmd with mocks that succeed
		var buf bytes.Buffer
		cmd := &RunCmd{BeadID: "cap-test", Provider: "claude", Timeout: 60}
		runner := &mockPipelineRunner{err: nil}
		wt := &mockMergeOps{mainBranch: "main"}
		bd := &mockBeadResolver{ctx: worklog.BeadContext{TaskID: "cap-test", TaskTitle: "Test task"}}
		bridge := tui.NewBridge()
		display := tui.NewDisplay(tui.DisplayOptions{Writer: &buf, ForcePlain: true})

		// When run is called with mocks
		err := cmd.run(&buf, runner, wt, bd, display, bridge, context.Background())

		// Then no error is returned
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// And the runner was called with bead context
		if runner.input.BeadID != "cap-test" {
			t.Errorf("BeadID = %q, want %q", runner.input.BeadID, "cap-test")
		}
		if runner.input.Title != "Test task" {
			t.Errorf("Title = %q, want %q", runner.input.Title, "Test task")
		}
		// And post-pipeline ran: merge + close
		if !wt.merged {
			t.Error("merge was not called")
		}
		if !bd.closed {
			t.Error("bead close was not called")
		}
		// And output shows merge and close messages
		output := buf.String()
		if !strings.Contains(output, "Merged capsule-cap-test") {
			t.Errorf("output missing merge message, got: %q", output)
		}
		if !strings.Contains(output, "Closed cap-test") {
			t.Errorf("output missing close message, got: %q", output)
		}
	})

	t.Run("RunCmd returns pipeline error on failure", func(t *testing.T) {
		// Given a RunCmd with a mock runner that fails
		var buf bytes.Buffer
		pipeErr := &orchestrator.PipelineError{Phase: "execute", Attempt: 1, Err: fmt.Errorf("broken")}
		cmd := &RunCmd{BeadID: "cap-fail", Provider: "claude", Timeout: 60}
		runner := &mockPipelineRunner{err: pipeErr}
		wt := &mockMergeOps{mainBranch: "main"}
		bd := &mockBeadResolver{ctx: worklog.BeadContext{TaskID: "cap-fail"}}
		bridge := tui.NewBridge()
		display := tui.NewDisplay(tui.DisplayOptions{Writer: &buf, ForcePlain: true})

		// When run is called
		err := cmd.run(&buf, runner, wt, bd, display, bridge, context.Background())

		// Then the pipeline error is returned
		var pe *orchestrator.PipelineError
		if !errors.As(err, &pe) {
			t.Fatalf("expected PipelineError, got %T: %v", err, err)
		}
		// And post-pipeline did NOT run
		if wt.merged {
			t.Error("merge should not run after pipeline failure")
		}
	})

	t.Run("RunCmd warns on bead not found with actionable message", func(t *testing.T) {
		// Given resolve returns a not-found error (bd available but bead not found)
		var buf bytes.Buffer
		cmd := &RunCmd{BeadID: "cap-bad", Provider: "claude", Timeout: 60}
		runner := &mockPipelineRunner{err: nil}
		wt := &mockMergeOps{mainBranch: "main"}
		bdMock := &mockBeadResolver{
			ctx:        worklog.BeadContext{TaskID: "cap-bad"},
			resolveErr: fmt.Errorf("%w: cap-bad", bead.ErrNotFound),
		}
		bridge := tui.NewBridge()
		display := tui.NewDisplay(tui.DisplayOptions{Writer: &buf, ForcePlain: true})

		// When run is called
		err := cmd.run(&buf, runner, wt, bdMock, display, bridge, context.Background())

		// Then no error is returned (pipeline still runs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// And an actionable warning was printed
		output := buf.String()
		if !strings.Contains(output, `warning: bead "cap-bad" not found (try: bd ready)`) {
			t.Errorf("output missing actionable warning, got: %q", output)
		}
		// And pipeline was called with fallback context
		if runner.input.BeadID != "cap-bad" {
			t.Errorf("BeadID = %q, want %q", runner.input.BeadID, "cap-bad")
		}
	})

	t.Run("RunCmd warns generically on other bead resolve failures", func(t *testing.T) {
		// Given resolve returns a non-not-found error
		var buf bytes.Buffer
		cmd := &RunCmd{BeadID: "cap-err", Provider: "claude", Timeout: 60}
		runner := &mockPipelineRunner{err: nil}
		wt := &mockMergeOps{mainBranch: "main"}
		bdMock := &mockBeadResolver{
			ctx:        worklog.BeadContext{TaskID: "cap-err"},
			resolveErr: fmt.Errorf("bead: parsing show output for cap-err: unexpected EOF"),
		}
		bridge := tui.NewBridge()
		display := tui.NewDisplay(tui.DisplayOptions{Writer: &buf, ForcePlain: true})

		// When run is called
		err := cmd.run(&buf, runner, wt, bdMock, display, bridge, context.Background())

		// Then no error is returned
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// And a generic warning was printed
		output := buf.String()
		if !strings.Contains(output, "warning: bead resolve failed") {
			t.Errorf("output missing generic warning, got: %q", output)
		}
	})

	t.Run("RunCmd prints merge conflict warning", func(t *testing.T) {
		// Given merge returns ErrMergeConflict
		var buf bytes.Buffer
		cmd := &RunCmd{BeadID: "cap-conflict", Provider: "claude", Timeout: 60}
		runner := &mockPipelineRunner{err: nil}
		wt := &mockMergeOps{
			mainBranch: "main",
			mergeErr:   worktree.ErrMergeConflict,
		}
		bd := &mockBeadResolver{ctx: worklog.BeadContext{TaskID: "cap-conflict"}}
		bridge := tui.NewBridge()
		display := tui.NewDisplay(tui.DisplayOptions{Writer: &buf, ForcePlain: true})

		// When run is called
		err := cmd.run(&buf, runner, wt, bd, display, bridge, context.Background())

		// Then no error is returned (best-effort)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// And the warning is printed
		output := buf.String()
		if !strings.Contains(output, "merge conflict") {
			t.Errorf("output missing merge conflict warning, got: %q", output)
		}
		if !strings.Contains(output, "capsule clean cap-conflict") {
			t.Errorf("output missing cleanup suggestion, got: %q", output)
		}
	})
}

// Compile-time interface satisfaction checks.
var (
	_ pipelineRunner = (*mockPipelineRunner)(nil)
	_ worktreeOps    = (*mockWorktreeOps)(nil)
	_ mergeOps       = (*mockMergeOps)(nil)
	_ beadResolver   = (*mockBeadResolver)(nil)
)

// mockPipelineRunner captures RunPipeline calls for testing.
type mockPipelineRunner struct {
	input orchestrator.PipelineInput
	err   error
}

func (m *mockPipelineRunner) RunPipeline(_ context.Context, input orchestrator.PipelineInput) (orchestrator.PipelineOutput, error) {
	m.input = input
	return orchestrator.PipelineOutput{Completed: m.err == nil}, m.err
}

// mockWorktreeOps stubs worktree operations for abort/clean testing.
type mockWorktreeOps struct {
	exists    bool
	removeErr error
	pruneErr  error

	removedID     string
	removedBranch bool
	pruned        bool
}

func (m *mockWorktreeOps) Exists(string) bool { return m.exists }

func (m *mockWorktreeOps) Remove(id string, deleteBranch bool) error {
	m.removedID = id
	m.removedBranch = deleteBranch
	return m.removeErr
}

func (m *mockWorktreeOps) Prune() error {
	m.pruned = true
	return m.pruneErr
}

// mockMergeOps stubs merge operations for RunCmd testing.
type mockMergeOps struct {
	mainBranch string
	mergeErr   error
	removeErr  error
	pruneErr   error

	merged bool
}

func (m *mockMergeOps) MergeToMain(string, string, string) error {
	m.merged = true
	return m.mergeErr
}

func (m *mockMergeOps) DetectMainBranch() (string, error) {
	return m.mainBranch, nil
}

func (m *mockMergeOps) Remove(_ string, _ bool) error {
	return m.removeErr
}

func (m *mockMergeOps) Prune() error { return m.pruneErr }

// mockBeadResolver stubs bead resolution for RunCmd testing.
type mockBeadResolver struct {
	ctx        worklog.BeadContext
	resolveErr error
	closeErr   error

	closed bool
}

func (m *mockBeadResolver) Resolve(string) (worklog.BeadContext, error) {
	return m.ctx, m.resolveErr
}

func (m *mockBeadResolver) Close(string) error {
	m.closed = true
	return m.closeErr
}

func TestFeature_DisplayWiring(t *testing.T) {
	t.Run("bridgeStatusCallback converts StatusUpdate to StatusUpdateMsg", func(t *testing.T) {
		// Given a bridge and a bridge status callback
		bridge := tui.NewBridge()
		cb := bridgeStatusCallback(bridge)

		// When a status update with signal data is sent
		cb(orchestrator.StatusUpdate{
			BeadID:   "cap-42",
			Phase:    "test-writer",
			Status:   orchestrator.PhasePassed,
			Progress: "1/6",
			Attempt:  2,
			MaxRetry: 3,
			Signal: &provider.Signal{
				Status:       provider.StatusPass,
				FilesChanged: []string{"foo.go", "bar.go"},
				Summary:      "All tests pass",
				Feedback:     "ok",
			},
		})

		// Then the bridge delivers a converted StatusUpdateMsg
		ev := <-bridge.Events()
		msg, ok := ev.(tui.StatusUpdateMsg)
		if !ok {
			t.Fatalf("expected StatusUpdateMsg, got %T", ev)
		}
		if msg.Phase != "test-writer" {
			t.Errorf("Phase = %q, want %q", msg.Phase, "test-writer")
		}
		if msg.Status != tui.StatusPassed {
			t.Errorf("Status = %q, want %q", msg.Status, tui.StatusPassed)
		}
		if msg.Progress != "1/6" {
			t.Errorf("Progress = %q, want %q", msg.Progress, "1/6")
		}
		if msg.Attempt != 2 {
			t.Errorf("Attempt = %d, want %d", msg.Attempt, 2)
		}
		if msg.MaxRetry != 3 {
			t.Errorf("MaxRetry = %d, want %d", msg.MaxRetry, 3)
		}
		if msg.Summary != "All tests pass" {
			t.Errorf("Summary = %q, want %q", msg.Summary, "All tests pass")
		}
		if len(msg.FilesChanged) != 2 || msg.FilesChanged[0] != "foo.go" {
			t.Errorf("FilesChanged = %v, want [foo.go bar.go]", msg.FilesChanged)
		}
		if msg.Feedback != "ok" {
			t.Errorf("Feedback = %q, want %q", msg.Feedback, "ok")
		}
	})

	t.Run("bridgeStatusCallback passes Duration to StatusUpdateMsg", func(t *testing.T) {
		// Given a bridge and a bridge status callback
		bridge := tui.NewBridge()
		cb := bridgeStatusCallback(bridge)

		// When a completed status update with Duration is sent
		cb(orchestrator.StatusUpdate{
			Phase:    "test-writer",
			Status:   orchestrator.PhasePassed,
			Progress: "1/6",
			Attempt:  1,
			MaxRetry: 3,
			Duration: 45200 * time.Millisecond,
			Signal:   &provider.Signal{Status: provider.StatusPass, Summary: "ok", Feedback: "ok"},
		})

		// Then the bridge delivers a StatusUpdateMsg with Duration set
		ev := <-bridge.Events()
		msg, ok := ev.(tui.StatusUpdateMsg)
		if !ok {
			t.Fatalf("expected StatusUpdateMsg, got %T", ev)
		}
		if msg.Duration != 45200*time.Millisecond {
			t.Errorf("Duration = %v, want %v", msg.Duration, 45200*time.Millisecond)
		}
	})

	t.Run("bridgeStatusCallback handles running status with nil signal", func(t *testing.T) {
		// Given a bridge and callback
		bridge := tui.NewBridge()
		cb := bridgeStatusCallback(bridge)

		// When a running status update (no signal) is sent
		cb(orchestrator.StatusUpdate{
			Phase:    "execute",
			Status:   orchestrator.PhaseRunning,
			Progress: "3/6",
			Attempt:  1,
			MaxRetry: 3,
		})

		// Then the bridge delivers a StatusUpdateMsg with empty signal fields
		ev := <-bridge.Events()
		msg, ok := ev.(tui.StatusUpdateMsg)
		if !ok {
			t.Fatalf("expected StatusUpdateMsg, got %T", ev)
		}
		if msg.Phase != "execute" {
			t.Errorf("Phase = %q, want %q", msg.Phase, "execute")
		}
		if msg.Summary != "" {
			t.Errorf("Summary = %q, want empty", msg.Summary)
		}
		if msg.FilesChanged != nil {
			t.Errorf("FilesChanged = %v, want nil", msg.FilesChanged)
		}
	})

	t.Run("run wires display lifecycle around pipeline", func(t *testing.T) {
		// Given a RunCmd with mocks and a plain display
		var buf bytes.Buffer
		cmd := &RunCmd{BeadID: "cap-display", Provider: "claude", Timeout: 60}
		runner := &mockPipelineRunner{err: nil}
		wt := &mockMergeOps{mainBranch: "main"}
		bd := &mockBeadResolver{ctx: worklog.BeadContext{TaskID: "cap-display", TaskTitle: "Test display"}}
		bridge := tui.NewBridge()
		display := tui.NewDisplay(tui.DisplayOptions{Writer: &buf, ForcePlain: true})

		// When run is called with display and bridge
		err := cmd.run(&buf, runner, wt, bd, display, bridge, context.Background())

		// Then no error is returned and post-pipeline ran
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !wt.merged {
			t.Error("merge was not called")
		}
		if !bd.closed {
			t.Error("bead close was not called")
		}
	})

	t.Run("run signals bridge error on pipeline failure", func(t *testing.T) {
		// Given a RunCmd where pipeline fails
		var buf bytes.Buffer
		pipeErr := &orchestrator.PipelineError{Phase: "execute", Attempt: 1, Err: fmt.Errorf("broken")}
		cmd := &RunCmd{BeadID: "cap-fail", Provider: "claude", Timeout: 60}
		runner := &mockPipelineRunner{err: pipeErr}
		wt := &mockMergeOps{mainBranch: "main"}
		bd := &mockBeadResolver{ctx: worklog.BeadContext{TaskID: "cap-fail"}}
		bridge := tui.NewBridge()
		display := tui.NewDisplay(tui.DisplayOptions{Writer: &buf, ForcePlain: true})

		// When run is called
		err := cmd.run(&buf, runner, wt, bd, display, bridge, context.Background())

		// Then pipeline error is returned
		var pe *orchestrator.PipelineError
		if !errors.As(err, &pe) {
			t.Fatalf("expected PipelineError, got %T: %v", err, err)
		}
		// And post-pipeline did NOT run
		if wt.merged {
			t.Error("merge should not run after pipeline failure")
		}
	})

	t.Run("phaseNames extracts names from PhaseDefinitions", func(t *testing.T) {
		// Given a set of phase definitions
		phases := []orchestrator.PhaseDefinition{
			{Name: "test-writer"},
			{Name: "test-review"},
			{Name: "execute"},
		}

		// When phaseNames is called
		names := phaseNames(phases)

		// Then the names are extracted in order
		if len(names) != 3 {
			t.Fatalf("got %d names, want 3", len(names))
		}
		if names[0] != "test-writer" || names[1] != "test-review" || names[2] != "execute" {
			t.Errorf("names = %v, want [test-writer test-review execute]", names)
		}
	})
}

func TestFeature_AbortCommand(t *testing.T) {
	t.Run("abort removes worktree and preserves branch", func(t *testing.T) {
		// Given an abort command and a worktree that exists
		var buf bytes.Buffer
		cmd := &AbortCmd{BeadID: "cap-test"}
		mgr := &mockWorktreeOps{exists: true}

		// When abort runs
		err := cmd.run(&buf, mgr)

		// Then no error is returned
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// And the worktree was removed without deleting the branch
		if mgr.removedID != "cap-test" {
			t.Errorf("removedID = %q, want %q", mgr.removedID, "cap-test")
		}
		if mgr.removedBranch {
			t.Error("deleteBranch = true, want false (branch should be preserved)")
		}
		// And success message is printed
		if !strings.Contains(buf.String(), "Aborted capsule cap-test") {
			t.Errorf("output = %q, want to contain abort message", buf.String())
		}
	})

	t.Run("abort returns error when worktree not found", func(t *testing.T) {
		// Given an abort command and no worktree
		var buf bytes.Buffer
		cmd := &AbortCmd{BeadID: "nonexistent"}
		mgr := &mockWorktreeOps{exists: false}

		// When abort runs
		err := cmd.run(&buf, mgr)

		// Then an error mentioning "no worktree found" is returned
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "no worktree found") {
			t.Errorf("error = %q, want to contain 'no worktree found'", err)
		}
	})

	t.Run("abort returns error when remove fails", func(t *testing.T) {
		// Given an abort command and a worktree that fails to remove
		var buf bytes.Buffer
		cmd := &AbortCmd{BeadID: "cap-fail"}
		mgr := &mockWorktreeOps{exists: true, removeErr: fmt.Errorf("lock held")}

		// When abort runs
		err := cmd.run(&buf, mgr)

		// Then the remove error is returned
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "lock held") {
			t.Errorf("error = %q, want to contain 'lock held'", err)
		}
	})
}

func TestFeature_CleanCommand(t *testing.T) {
	t.Run("clean removes worktree branch and prunes", func(t *testing.T) {
		// Given a clean command and a worktree that exists
		var buf bytes.Buffer
		cmd := &CleanCmd{BeadID: "cap-test"}
		mgr := &mockWorktreeOps{exists: true}

		// When clean runs
		err := cmd.run(&buf, mgr)

		// Then no error is returned
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// And the worktree was removed with branch deletion
		if mgr.removedID != "cap-test" {
			t.Errorf("removedID = %q, want %q", mgr.removedID, "cap-test")
		}
		if !mgr.removedBranch {
			t.Error("deleteBranch = false, want true (clean should delete branch)")
		}
		// And prune was called
		if !mgr.pruned {
			t.Error("prune was not called")
		}
		// And success message is printed
		if !strings.Contains(buf.String(), "Cleaned capsule cap-test") {
			t.Errorf("output = %q, want to contain clean message", buf.String())
		}
	})

	t.Run("clean returns error when worktree not found", func(t *testing.T) {
		// Given a clean command and no worktree
		var buf bytes.Buffer
		cmd := &CleanCmd{BeadID: "nonexistent"}
		mgr := &mockWorktreeOps{exists: false}

		// When clean runs
		err := cmd.run(&buf, mgr)

		// Then an error mentioning "no worktree found" is returned
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "no worktree found") {
			t.Errorf("error = %q, want to contain 'no worktree found'", err)
		}
	})

	t.Run("clean returns error when remove fails", func(t *testing.T) {
		// Given a clean command and a worktree that fails to remove
		var buf bytes.Buffer
		cmd := &CleanCmd{BeadID: "cap-fail"}
		mgr := &mockWorktreeOps{exists: true, removeErr: fmt.Errorf("busy")}

		// When clean runs
		err := cmd.run(&buf, mgr)

		// Then the remove error is returned
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "busy") {
			t.Errorf("error = %q, want to contain 'busy'", err)
		}
	})

	t.Run("clean returns error when prune fails", func(t *testing.T) {
		// Given a clean command where prune fails
		var buf bytes.Buffer
		cmd := &CleanCmd{BeadID: "cap-prune"}
		mgr := &mockWorktreeOps{exists: true, pruneErr: fmt.Errorf("git error")}

		// When clean runs
		err := cmd.run(&buf, mgr)

		// Then the prune error is returned
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "prune") {
			t.Errorf("error = %q, want to contain 'prune'", err)
		}
	})
}

func TestDashboardCampaignPipelineRunner_PropagatesSiblingContext(t *testing.T) {
	// Given: a dashboardCampaignPipelineRunner with a pipelineFn that captures input
	var captured dashboard.PipelineInput
	pipelineFn := func(_ context.Context, input dashboard.PipelineInput, _ func(dashboard.PhaseUpdateMsg)) (dashboard.PipelineOutput, error) {
		captured = input
		return dashboard.PipelineOutput{Success: true}, nil
	}
	runner := &dashboardCampaignPipelineRunner{pipelineFn: pipelineFn}

	// When: RunPipeline is called with orchestrator input containing SiblingContext
	input := orchestrator.PipelineInput{
		BeadID: "cap-task",
		SiblingContext: []prompt.SiblingContext{
			{BeadID: "cap-sibling", Title: "Login", Summary: "Built login", FilesChanged: []string{"auth.go"}},
		},
	}
	_, err := runner.RunPipeline(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Then: the SiblingContext is propagated to the dashboard PipelineInput
	if len(captured.SiblingContext) != 1 {
		t.Fatalf("SiblingContext len = %d, want 1", len(captured.SiblingContext))
	}
	if captured.SiblingContext[0].BeadID != "cap-sibling" {
		t.Errorf("SiblingContext[0].BeadID = %q, want %q", captured.SiblingContext[0].BeadID, "cap-sibling")
	}
	if captured.SiblingContext[0].Title != "Login" {
		t.Errorf("SiblingContext[0].Title = %q, want %q", captured.SiblingContext[0].Title, "Login")
	}
	if captured.SiblingContext[0].Summary != "Built login" {
		t.Errorf("SiblingContext[0].Summary = %q, want %q", captured.SiblingContext[0].Summary, "Built login")
	}
	if len(captured.SiblingContext[0].FilesChanged) != 1 || captured.SiblingContext[0].FilesChanged[0] != "auth.go" {
		t.Errorf("SiblingContext[0].FilesChanged = %v, want [auth.go]", captured.SiblingContext[0].FilesChanged)
	}
}

func TestDashboardCampaignPipelineRunner_ConvertsPhaseReports(t *testing.T) {
	// Given: a pipelineFn that returns PhaseReports in its output
	pipelineFn := func(_ context.Context, _ dashboard.PipelineInput, _ func(dashboard.PhaseUpdateMsg)) (dashboard.PipelineOutput, error) {
		return dashboard.PipelineOutput{
			Success: true,
			PhaseReports: []dashboard.PhaseReport{
				{
					PhaseName:    "plan",
					Status:       dashboard.PhasePassed,
					Summary:      "planned changes",
					Feedback:     "looks good",
					FilesChanged: []string{"main.go"},
					Duration:     2 * time.Second,
				},
				{
					PhaseName: "code",
					Status:    dashboard.PhasePassed,
					Summary:   "implemented",
					Duration:  5 * time.Second,
				},
			},
		}, nil
	}
	runner := &dashboardCampaignPipelineRunner{pipelineFn: pipelineFn}

	// When: RunPipeline is called
	output, err := runner.RunPipeline(context.Background(), orchestrator.PipelineInput{BeadID: "cap-conv"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Then: PhaseReports are converted to PhaseResults in the output
	if !output.Completed {
		t.Error("Completed = false, want true")
	}
	if len(output.PhaseResults) != 2 {
		t.Fatalf("PhaseResults len = %d, want 2", len(output.PhaseResults))
	}
	first := output.PhaseResults[0]
	if first.PhaseName != "plan" {
		t.Errorf("PhaseResults[0].PhaseName = %q, want %q", first.PhaseName, "plan")
	}
	if first.Signal.Status != provider.Status(dashboard.PhasePassed) {
		t.Errorf("PhaseResults[0].Signal.Status = %q, want %q", first.Signal.Status, dashboard.PhasePassed)
	}
	if first.Signal.Summary != "planned changes" {
		t.Errorf("PhaseResults[0].Signal.Summary = %q, want %q", first.Signal.Summary, "planned changes")
	}
	if first.Signal.Feedback != "looks good" {
		t.Errorf("PhaseResults[0].Signal.Feedback = %q, want %q", first.Signal.Feedback, "looks good")
	}
	if len(first.Signal.FilesChanged) != 1 || first.Signal.FilesChanged[0] != "main.go" {
		t.Errorf("PhaseResults[0].Signal.FilesChanged = %v, want [main.go]", first.Signal.FilesChanged)
	}
	if first.Duration != 2*time.Second {
		t.Errorf("PhaseResults[0].Duration = %v, want 2s", first.Duration)
	}
	second := output.PhaseResults[1]
	if second.PhaseName != "code" {
		t.Errorf("PhaseResults[1].PhaseName = %q, want %q", second.PhaseName, "code")
	}
	if second.Signal.Status != provider.Status(dashboard.PhasePassed) {
		t.Errorf("PhaseResults[1].Signal.Status = %q, want %q", second.Signal.Status, dashboard.PhasePassed)
	}
	if second.Signal.Summary != "implemented" {
		t.Errorf("PhaseResults[1].Signal.Summary = %q, want %q", second.Signal.Summary, "implemented")
	}
	if second.Duration != 5*time.Second {
		t.Errorf("PhaseResults[1].Duration = %v, want 5s", second.Duration)
	}
}

func TestPostPipeline_MergesAndClosesBead(t *testing.T) {
	// Given: mock worktree and bead resolver that succeed
	var buf bytes.Buffer
	wt := &mockMergeOps{mainBranch: "main"}
	bd := &mockBeadResolver{ctx: worklog.BeadContext{TaskID: "cap-pp"}}

	// When: postPipeline is called
	postPipeline(&buf, "cap-pp", wt, bd)

	// Then: merge and close are called
	if !wt.merged {
		t.Error("merge was not called")
	}
	if !bd.closed {
		t.Error("bead close was not called")
	}
	output := buf.String()
	if !strings.Contains(output, "Merged capsule-cap-pp") {
		t.Errorf("output missing merge message, got: %q", output)
	}
	if !strings.Contains(output, "Closed cap-pp") {
		t.Errorf("output missing close message, got: %q", output)
	}
}

func TestPostPipeline_WarnsOnMergeConflict(t *testing.T) {
	// Given: mock worktree that returns merge conflict
	var buf bytes.Buffer
	wt := &mockMergeOps{mainBranch: "main", mergeErr: worktree.ErrMergeConflict}
	bd := &mockBeadResolver{}

	// When: postPipeline is called
	postPipeline(&buf, "cap-conflict", wt, bd)

	// Then: merge conflict warning is printed
	output := buf.String()
	if !strings.Contains(output, "merge conflict") {
		t.Errorf("output missing merge conflict warning, got: %q", output)
	}
	// And: bead close is NOT called (merge failed)
	if bd.closed {
		t.Error("bead should not be closed after merge conflict")
	}
}

func TestFeature_DashboardCommand(t *testing.T) {
	t.Run("dashboard subcommand is parsed", func(t *testing.T) {
		// Given a CLI parser
		var cli CLI
		k, err := kong.New(&cli, kong.Vars{"version": "test"})
		if err != nil {
			t.Fatal(err)
		}

		// When dashboard command is invoked
		kctx, err := k.Parse([]string{"dashboard"})
		if err != nil {
			t.Fatal(err)
		}

		// Then the command is parsed correctly
		if kctx.Command() != "dashboard" {
			t.Errorf("got command %q, want %q", kctx.Command(), "dashboard")
		}
	})

	t.Run("run returns error when not a TTY", func(t *testing.T) {
		// Given a DashboardCmd
		cmd := &DashboardCmd{}

		// When run is called with isTTY=false
		err := cmd.run(false, nil)

		// Then an error mentioning "terminal" is returned
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "terminal") {
			t.Errorf("error = %q, want to contain 'terminal'", err)
		}
	})

	t.Run("run executes tea program when TTY", func(t *testing.T) {
		// Given a DashboardCmd and a mock tea program
		cmd := &DashboardCmd{}
		mock := &mockTeaRunner{}

		// When run is called with isTTY=true
		err := cmd.run(true, mock)

		// Then no error is returned
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// And the program was run
		if !mock.ran {
			t.Error("tea program was not run")
		}
	})

	t.Run("run returns tea program error", func(t *testing.T) {
		// Given a DashboardCmd and a mock that fails
		cmd := &DashboardCmd{}
		mock := &mockTeaRunner{err: fmt.Errorf("tea: terminal error")}

		// When run is called
		err := cmd.run(true, mock)

		// Then the tea error is returned
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "tea: terminal error") {
			t.Errorf("error = %q, want to contain tea error", err)
		}
	})
}

// mockTeaRunner stubs tea program execution for DashboardCmd testing.
type mockTeaRunner struct {
	ran bool
	err error
}

func (m *mockTeaRunner) Run() (tea.Model, error) {
	m.ran = true
	return nil, m.err
}

// Compile-time check: mockTeaRunner satisfies teaRunner.
var _ teaRunner = (*mockTeaRunner)(nil)
