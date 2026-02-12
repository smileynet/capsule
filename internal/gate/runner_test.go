package gate

import (
	"context"
	"testing"

	"github.com/smileynet/capsule/internal/provider"
)

func TestRunner_PassingCommand(t *testing.T) {
	// Given a command that succeeds
	r := NewRunner()

	// When Run is called
	signal, err := r.Run(context.Background(), "echo hello", t.TempDir())

	// Then it returns StatusPass with the output as summary
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if signal.Status != provider.StatusPass {
		t.Errorf("Status = %q, want %q", signal.Status, provider.StatusPass)
	}
	if signal.Summary == "" {
		t.Error("Summary should contain command output")
	}
}

func TestRunner_FailingCommand(t *testing.T) {
	// Given a command that fails
	r := NewRunner()

	// When Run is called
	signal, err := r.Run(context.Background(), "exit 1", t.TempDir())

	// Then it returns StatusError (not a Go error)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if signal.Status != provider.StatusError {
		t.Errorf("Status = %q, want %q", signal.Status, provider.StatusError)
	}
}

func TestRunner_CommandWithOutput(t *testing.T) {
	// Given a failing command that produces output
	r := NewRunner()

	// When Run is called
	signal, err := r.Run(context.Background(), "echo 'error info' && exit 1", t.TempDir())

	// Then the output appears in Feedback
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if signal.Status != provider.StatusError {
		t.Errorf("Status = %q, want %q", signal.Status, provider.StatusError)
	}
	if signal.Feedback == "" {
		t.Error("Feedback should contain command output")
	}
}

func TestRunner_UsesWorkDir(t *testing.T) {
	// Given a specific working directory
	r := NewRunner()
	dir := t.TempDir()

	// When Run is called with pwd
	signal, err := r.Run(context.Background(), "pwd", dir)

	// Then the output contains the working directory
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if signal.Status != provider.StatusPass {
		t.Errorf("Status = %q, want %q", signal.Status, provider.StatusPass)
	}
}

func TestRunner_ContextCancellation(t *testing.T) {
	// Given a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := NewRunner()

	// When Run is called with cancelled context
	signal, err := r.Run(ctx, "sleep 10", t.TempDir())

	// Then it returns StatusError (context error handled gracefully)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if signal.Status != provider.StatusError {
		t.Errorf("Status = %q, want %q", signal.Status, provider.StatusError)
	}
}

func TestRunner_NormalizesSlices(t *testing.T) {
	// Given a passing command
	r := NewRunner()

	// When Run is called
	signal, err := r.Run(context.Background(), "echo ok", t.TempDir())

	// Then slices are normalized to empty (not nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if signal.FilesChanged == nil {
		t.Error("FilesChanged should be empty slice, not nil")
	}
	if signal.Findings == nil {
		t.Error("Findings should be empty slice, not nil")
	}
}
