package provider

import (
	"context"
	"errors"
	"testing"
	"time"
)

// --- Signal parsing tests ---

func TestParseSignal(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    Signal
		wantErr bool
	}{
		{
			name: "valid signal last line",
			output: `Reading worklog.md...
Found 3 acceptance criteria.
{"status":"PASS","feedback":"All good","files_changed":["src/foo.go"],"summary":"Tests created"}`,
			want: Signal{
				Status:       StatusPass,
				Feedback:     "All good",
				FilesChanged: []string{"src/foo.go"},
				Summary:      "Tests created",
			},
		},
		{
			name:   "valid signal only line",
			output: `{"status":"NEEDS_WORK","feedback":"Fix tests","files_changed":[],"summary":"Issues found"}`,
			want: Signal{
				Status:       StatusNeedsWork,
				Feedback:     "Fix tests",
				FilesChanged: []string{},
				Summary:      "Issues found",
			},
		},
		{
			name:   "valid signal with markdown fences",
			output: "Some output\n```json\n" + `{"status":"ERROR","feedback":"Failed","files_changed":[],"summary":"Phase error"}` + "\n```",
			want: Signal{
				Status:       StatusError,
				Feedback:     "Failed",
				FilesChanged: []string{},
				Summary:      "Phase error",
			},
		},
		{
			name:   "signal with commit_hash",
			output: `{"status":"PASS","feedback":"Committed","files_changed":["main.go"],"summary":"Merged","commit_hash":"abc1234"}`,
			want: Signal{
				Status:       StatusPass,
				Feedback:     "Committed",
				FilesChanged: []string{"main.go"},
				Summary:      "Merged",
				CommitHash:   "abc1234",
			},
		},
		{
			name:    "no JSON in output",
			output:  "just some text\nwith no json",
			wantErr: true,
		},
		{
			name:    "empty output",
			output:  "",
			wantErr: true,
		},
		{
			name:    "missing status field",
			output:  `{"feedback":"ok","files_changed":[],"summary":"done"}`,
			wantErr: true,
		},
		{
			name:    "invalid status value",
			output:  `{"status":"INVALID","feedback":"ok","files_changed":[],"summary":"done"}`,
			wantErr: true,
		},
		{
			name:    "missing feedback field",
			output:  `{"status":"PASS","files_changed":[],"summary":"done"}`,
			wantErr: true,
		},
		{
			name:    "missing summary field",
			output:  `{"status":"PASS","feedback":"ok","files_changed":[]}`,
			wantErr: true,
		},
		{
			name:    "malformed JSON",
			output:  `{"status":"PASS", broken json`,
			wantErr: true,
		},
		{
			name: "multiple JSON objects picks last",
			output: `{"status":"ERROR","feedback":"first","files_changed":[],"summary":"first"}
some log output
{"status":"PASS","feedback":"second","files_changed":["a.go"],"summary":"second"}`,
			want: Signal{
				Status:       StatusPass,
				Feedback:     "second",
				FilesChanged: []string{"a.go"},
				Summary:      "second",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: provider output per test case
			// When: ParseSignal is called
			got, err := ParseSignal(tt.output)

			// Then: the expected signal or error is returned
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				var spe *SignalParseError
				if !errors.As(err, &spe) {
					t.Errorf("expected *SignalParseError, got %T", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Status != tt.want.Status {
				t.Errorf("Status = %q, want %q", got.Status, tt.want.Status)
			}
			if got.Feedback != tt.want.Feedback {
				t.Errorf("Feedback = %q, want %q", got.Feedback, tt.want.Feedback)
			}
			if got.Summary != tt.want.Summary {
				t.Errorf("Summary = %q, want %q", got.Summary, tt.want.Summary)
			}
			if got.CommitHash != tt.want.CommitHash {
				t.Errorf("CommitHash = %q, want %q", got.CommitHash, tt.want.CommitHash)
			}
			if len(got.FilesChanged) != len(tt.want.FilesChanged) {
				t.Fatalf("FilesChanged len = %d, want %d", len(got.FilesChanged), len(tt.want.FilesChanged))
			}
			for i, f := range got.FilesChanged {
				if f != tt.want.FilesChanged[i] {
					t.Errorf("FilesChanged[%d] = %q, want %q", i, f, tt.want.FilesChanged[i])
				}
			}
		})
	}
}

// --- Error type tests ---

func TestErrorTypes(t *testing.T) {
	t.Run("SignalParseError", func(t *testing.T) {
		// Given: a SignalParseError with a reason
		err := &SignalParseError{Reason: "missing field: status"}
		want := "provider: signal parse: missing field: status"

		// When/Then: Error() returns the formatted message and errors.As succeeds
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
		var target *SignalParseError
		if !errors.As(err, &target) {
			t.Error("errors.As failed for *SignalParseError")
		}
	})

	t.Run("ProviderError", func(t *testing.T) {
		// Given: a ProviderError wrapping a cause
		cause := errors.New("connection refused")
		err := &ProviderError{Provider: "claude", Err: cause}
		want := "provider: claude: connection refused"

		// When/Then: Error() returns the formatted message and Unwrap works
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
		if !errors.Is(err, cause) {
			t.Error("errors.Is failed to unwrap cause")
		}
		var target *ProviderError
		if !errors.As(err, &target) {
			t.Error("errors.As failed for *ProviderError")
		}
	})

	t.Run("TimeoutError", func(t *testing.T) {
		// Given: a TimeoutError with provider and duration
		err := &TimeoutError{Provider: "claude", Duration: 30 * time.Second}
		want := "provider: claude: timed out after 30s"

		// When/Then: Error() returns the formatted message and errors.As succeeds
		if err.Error() != want {
			t.Errorf("Error() = %q, want %q", err.Error(), want)
		}
		var target *TimeoutError
		if !errors.As(err, &target) {
			t.Error("errors.As failed for *TimeoutError")
		}
	})
}

// --- Provider interface and MockProvider tests ---

func TestMockProvider(t *testing.T) {
	t.Run("returns configured result", func(t *testing.T) {
		// Given: a MockProvider with a configured ExecuteFunc
		mock := &MockProvider{
			NameVal: "mock",
			ExecuteFunc: func(ctx context.Context, prompt, workDir string) (Result, error) {
				return Result{
					Output:   `{"status":"PASS","feedback":"ok","files_changed":[],"summary":"done"}`,
					ExitCode: 0,
					Duration: 100 * time.Millisecond,
				}, nil
			},
		}

		// When: Name and Execute are called
		// Then: the configured name and result are returned
		if mock.Name() != "mock" {
			t.Errorf("Name() = %q, want %q", mock.Name(), "mock")
		}

		result, err := mock.Execute(context.Background(), "test prompt", "/tmp/work")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("ExitCode = %d, want 0", result.ExitCode)
		}
		if result.Duration != 100*time.Millisecond {
			t.Errorf("Duration = %v, want 100ms", result.Duration)
		}
	})

	t.Run("returns configured error", func(t *testing.T) {
		// Given: a MockProvider whose ExecuteFunc returns a ProviderError
		mock := &MockProvider{
			NameVal: "mock",
			ExecuteFunc: func(ctx context.Context, prompt, workDir string) (Result, error) {
				return Result{}, &ProviderError{Provider: "mock", Err: errors.New("failed")}
			},
		}

		// When: Execute is called
		_, err := mock.Execute(context.Background(), "test prompt", "/tmp/work")

		// Then: the configured error is returned
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var pe *ProviderError
		if !errors.As(err, &pe) {
			t.Errorf("expected *ProviderError, got %T", err)
		}
	})

	t.Run("nil ExecuteFunc returns zero result", func(t *testing.T) {
		// Given: a MockProvider with no ExecuteFunc set
		mock := &MockProvider{NameVal: "mock"}

		// When: Execute is called
		result, err := mock.Execute(context.Background(), "prompt", "/dir")

		// Then: a zero Result is returned with no error
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Output != "" {
			t.Errorf("Output = %q, want empty", result.Output)
		}
		if result.ExitCode != 0 {
			t.Errorf("ExitCode = %d, want 0", result.ExitCode)
		}
	})

	t.Run("captures call args", func(t *testing.T) {
		// Given: a MockProvider that captures call arguments
		var capturedPrompt, capturedWorkDir string
		mock := &MockProvider{
			NameVal: "mock",
			ExecuteFunc: func(ctx context.Context, prompt, workDir string) (Result, error) {
				capturedPrompt = prompt
				capturedWorkDir = workDir
				return Result{}, nil
			},
		}

		// When: Execute is called with specific arguments
		_, _ = mock.Execute(context.Background(), "my prompt", "/my/dir")

		// Then: the arguments are captured correctly
		if capturedPrompt != "my prompt" {
			t.Errorf("prompt = %q, want %q", capturedPrompt, "my prompt")
		}
		if capturedWorkDir != "/my/dir" {
			t.Errorf("workDir = %q, want %q", capturedWorkDir, "/my/dir")
		}
	})
}

// --- Executor interface test ---

func TestExecutorInterface(t *testing.T) {
	// Given: a MockProvider assigned to the Executor interface
	var e Executor = &MockProvider{
		NameVal: "test",
		ExecuteFunc: func(ctx context.Context, prompt, workDir string) (Result, error) {
			return Result{Output: "ok"}, nil
		},
	}
	// When: Name and Execute are called via the interface
	// Then: the expected values are returned
	if e.Name() != "test" {
		t.Errorf("Name() = %q, want %q", e.Name(), "test")
	}
	r, err := e.Execute(context.Background(), "p", "/d")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Output != "ok" {
		t.Errorf("Output = %q, want %q", r.Output, "ok")
	}
}

// --- Result.ParseSignal convenience method test ---

func TestResultParseSignal(t *testing.T) {
	// Given: a Result with a valid JSON signal in its output
	r := Result{
		Output:   `{"status":"PASS","feedback":"ok","files_changed":["a.go"],"summary":"done"}`,
		ExitCode: 0,
		Duration: time.Second,
	}
	// When: ParseSignal is called on the Result
	sig, err := r.ParseSignal()

	// Then: the parsed signal has the expected status
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sig.Status != StatusPass {
		t.Errorf("Status = %q, want %q", sig.Status, StatusPass)
	}
}
