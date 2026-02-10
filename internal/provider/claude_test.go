package provider

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

// TestHelperProcess is the re-exec helper. It is not a real test —
// it is invoked by exec.Command pointing at the test binary itself.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_TEST_HELPER_PROCESS") != "1" {
		return
	}

	switch os.Getenv("GO_TEST_HELPER_MODE") {
	case "success":
		fmt.Println("Thinking...")
		fmt.Println(`{"status":"PASS","feedback":"All good","files_changed":["src/main.go"],"summary":"Done"}`)
		os.Exit(0)
	case "error_exit":
		fmt.Fprintln(os.Stderr, "claude: API key invalid")
		os.Exit(1)
	case "signal_in_stderr":
		fmt.Fprintln(os.Stderr, "warning: something happened")
		fmt.Println(`{"status":"NEEDS_WORK","feedback":"Fix tests","files_changed":[],"summary":"Issues"}`)
		os.Exit(0)
	case "slow":
		time.Sleep(5 * time.Second)
		fmt.Println(`{"status":"PASS","feedback":"ok","files_changed":[],"summary":"ok"}`)
		os.Exit(0)
	default:
		fmt.Fprintln(os.Stderr, "unknown test helper mode")
		os.Exit(2)
	}
}

// helperCommand builds an exec.Cmd that re-invokes the test binary
// in helper mode, respecting context cancellation.
func helperCommand(ctx context.Context, mode string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestHelperProcess$")
	cmd.Env = append(os.Environ(),
		"GO_TEST_HELPER_PROCESS=1",
		"GO_TEST_HELPER_MODE="+mode,
	)
	return cmd
}

func TestClaudeProvider_Name(t *testing.T) {
	// Given: a default ClaudeProvider
	p := NewClaudeProvider()

	// When: Name is called
	// Then: it returns "claude"
	if got := p.Name(); got != "claude" {
		t.Errorf("Name() = %q, want %q", got, "claude")
	}
}

func TestClaudeProvider_Execute(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess tests in short mode")
	}

	tests := []struct {
		name       string
		mode       string
		timeout    time.Duration
		wantErr    bool
		errType    string // "provider", "timeout"
		wantExit   int
		wantStatus Status
	}{
		{
			name:       "successful execution with signal",
			mode:       "success",
			timeout:    5 * time.Second,
			wantExit:   0,
			wantStatus: StatusPass,
		},
		{
			name:    "non-zero exit returns ProviderError",
			mode:    "error_exit",
			timeout: 5 * time.Second,
			wantErr: true,
			errType: "provider",
		},
		{
			name:       "stderr noise does not affect stdout parsing",
			mode:       "signal_in_stderr",
			timeout:    5 * time.Second,
			wantExit:   0,
			wantStatus: StatusNeedsWork,
		},
		{
			name:    "timeout kills process",
			mode:    "slow",
			timeout: 100 * time.Millisecond,
			wantErr: true,
			errType: "timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: a provider configured per test case
			p := NewClaudeProvider(WithTimeout(tt.timeout))
			// Override command builder to use re-exec helper.
			p.cmdBuilder = func(ctx context.Context, prompt, workDir string) *exec.Cmd {
				return helperCommand(ctx, tt.mode)
			}

			// When: Execute is called
			result, err := p.Execute(context.Background(), "test prompt", t.TempDir())

			// Then: the expected outcome is observed
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				switch tt.errType {
				case "provider":
					var pe *ProviderError
					if !errors.As(err, &pe) {
						t.Errorf("expected *ProviderError, got %T: %v", err, err)
					}
				case "timeout":
					var te *TimeoutError
					if !errors.As(err, &te) {
						t.Errorf("expected *TimeoutError, got %T: %v", err, err)
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.ExitCode != tt.wantExit {
				t.Errorf("ExitCode = %d, want %d", result.ExitCode, tt.wantExit)
			}
			if result.Duration <= 0 {
				t.Error("Duration should be positive")
			}

			sig, err := result.ParseSignal()
			if err != nil {
				t.Fatalf("ParseSignal error: %v", err)
			}
			if sig.Status != tt.wantStatus {
				t.Errorf("Status = %q, want %q", sig.Status, tt.wantStatus)
			}
		})
	}
}

func TestClaudeProvider_ExecuteRespectsContext(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in short mode")
	}

	// Given: an already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Already cancelled.

	p := NewClaudeProvider()
	p.cmdBuilder = func(ctx context.Context, prompt, workDir string) *exec.Cmd {
		return helperCommand(ctx, "slow")
	}

	// When: Execute is called with the cancelled context
	_, err := p.Execute(ctx, "prompt", t.TempDir())

	// Then: a ProviderError is returned
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	var pe *ProviderError
	if !errors.As(err, &pe) {
		t.Errorf("expected *ProviderError for cancelled context, got %T: %v", err, err)
	}
}
