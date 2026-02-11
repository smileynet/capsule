package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/alecthomas/kong"

	"github.com/smileynet/capsule/internal/orchestrator"
	"github.com/smileynet/capsule/internal/provider"
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
			"--debug",
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
		if !cli.Run.Debug {
			t.Error("debug = false, want true")
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
		if cli.Run.Debug {
			t.Error("default debug = true, want false")
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

	t.Run("RunCmd wires pipeline and returns nil on success", func(t *testing.T) {
		// Given a RunCmd with a mock runner that succeeds
		var buf bytes.Buffer
		cmd := &RunCmd{BeadID: "cap-test", Provider: "claude", Timeout: 60}
		runner := &mockPipelineRunner{err: nil}

		// When run is called with the mock
		err := cmd.run(&buf, runner)

		// Then no error is returned
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// And the runner was called with the correct bead ID
		if runner.input.BeadID != "cap-test" {
			t.Errorf("BeadID = %q, want %q", runner.input.BeadID, "cap-test")
		}
	})

	t.Run("RunCmd returns pipeline error on failure", func(t *testing.T) {
		// Given a RunCmd with a mock runner that fails
		var buf bytes.Buffer
		pipeErr := &orchestrator.PipelineError{Phase: "execute", Attempt: 1, Err: fmt.Errorf("broken")}
		cmd := &RunCmd{BeadID: "cap-fail", Provider: "claude", Timeout: 60}
		runner := &mockPipelineRunner{err: pipeErr}

		// When run is called
		err := cmd.run(&buf, runner)

		// Then the pipeline error is returned (main() handles error printing)
		var pe *orchestrator.PipelineError
		if !errors.As(err, &pe) {
			t.Fatalf("expected PipelineError, got %T: %v", err, err)
		}
	})
}

// mockPipelineRunner captures RunPipeline calls for testing.
type mockPipelineRunner struct {
	input orchestrator.PipelineInput
	err   error
}

func (m *mockPipelineRunner) RunPipeline(_ context.Context, input orchestrator.PipelineInput) error {
	m.input = input
	return m.err
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
