package provider

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
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
	case "ansi_output":
		fmt.Println("\x1b[32mThinking...\x1b[0m")
		fmt.Println(`{"status":"PASS","feedback":"All good","files_changed":[],"summary":"Done"}`)
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

func TestGenericProvider_Name(t *testing.T) {
	tests := []struct {
		name   string
		preset CommandConfig
		want   string
	}{
		{"claude preset", ClaudePreset, "claude"},
		{"kiro preset", KiroPreset, "kiro"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given a GenericProvider with the preset
			p := NewGenericProvider(tt.preset)

			// When Name is called
			// Then it returns the preset name
			if got := p.Name(); got != tt.want {
				t.Errorf("Name() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGenericProvider_Execute(t *testing.T) {
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
			// Given a provider configured per test case
			p := NewGenericProvider(ClaudePreset, WithTimeout(tt.timeout))
			// Override command builder to use re-exec helper.
			p.cmdBuilder = func(ctx context.Context, prompt, workDir string) *exec.Cmd {
				return helperCommand(ctx, tt.mode)
			}

			// When Execute is called
			result, err := p.Execute(context.Background(), "test prompt", t.TempDir())

			// Then the expected outcome is observed
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

func TestGenericProvider_ExecuteRespectsContext(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in short mode")
	}

	// Given an already-cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Already cancelled.

	p := NewGenericProvider(ClaudePreset)
	p.cmdBuilder = func(ctx context.Context, prompt, workDir string) *exec.Cmd {
		return helperCommand(ctx, "slow")
	}

	// When Execute is called with the cancelled context
	_, err := p.Execute(ctx, "prompt", t.TempDir())

	// Then a ProviderError is returned
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
	var pe *ProviderError
	if !errors.As(err, &pe) {
		t.Errorf("expected *ProviderError for cancelled context, got %T: %v", err, err)
	}
}

func TestGenericProvider_StripANSI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in short mode")
	}

	// Given a provider with StripANSI enabled
	cfg := ClaudePreset
	cfg.StripANSI = true
	p := NewGenericProvider(cfg, WithTimeout(5*time.Second))
	p.cmdBuilder = func(ctx context.Context, prompt, workDir string) *exec.Cmd {
		return helperCommand(ctx, "ansi_output")
	}

	// When Execute is called
	result, err := p.Execute(context.Background(), "prompt", t.TempDir())

	// Then ANSI codes are stripped from output
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result.Output, "\x1b[") {
		t.Errorf("output still contains ANSI escape codes: %q", result.Output)
	}

	// And the signal can still be parsed
	sig, err := result.ParseSignal()
	if err != nil {
		t.Fatalf("ParseSignal error: %v", err)
	}
	if sig.Status != StatusPass {
		t.Errorf("Status = %q, want %q", sig.Status, StatusPass)
	}
}

func TestGenericProvider_ErrorIncludesProviderName(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping subprocess test in short mode")
	}

	// Given a kiro provider that fails
	p := NewGenericProvider(KiroPreset, WithTimeout(5*time.Second))
	p.cmdBuilder = func(ctx context.Context, prompt, workDir string) *exec.Cmd {
		return helperCommand(ctx, "error_exit")
	}

	// When Execute is called
	_, err := p.Execute(context.Background(), "prompt", t.TempDir())

	// Then the error includes the provider name "kiro"
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var pe *ProviderError
	if !errors.As(err, &pe) {
		t.Fatalf("expected *ProviderError, got %T", err)
	}
	if pe.Provider != "kiro" {
		t.Errorf("Provider = %q, want %q", pe.Provider, "kiro")
	}
}

func TestBuildArgs(t *testing.T) {
	tests := []struct {
		name   string
		config CommandConfig
		prompt string
		want   []string
	}{
		{
			name:   "claude preset uses prompt flag",
			config: ClaudePreset,
			prompt: "test prompt",
			want:   []string{"--dangerously-skip-permissions", "-p", "test prompt"},
		},
		{
			name:   "kiro preset uses subcommand and positional prompt",
			config: KiroPreset,
			prompt: "test prompt",
			want:   []string{"chat", "--trust-all-tools", "--no-interactive", "--wrap", "never", "test prompt"},
		},
		{
			name: "minimal config with only binary and positional prompt",
			config: CommandConfig{
				Name:   "minimal",
				Binary: "ai-tool",
			},
			prompt: "do stuff",
			want:   []string{"do stuff"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When buildArgs is called
			got := buildArgs(tt.config, tt.prompt)

			// Then the argument list matches
			if !slices.Equal(got, tt.want) {
				t.Errorf("buildArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no escape codes", "hello world", "hello world"},
		{"color codes", "\x1b[32mgreen\x1b[0m", "green"},
		{"bold", "\x1b[1mbold\x1b[0m text", "bold text"},
		{"mixed content", "line1\n\x1b[31merror\x1b[0m\nline3", "line1\nerror\nline3"},
		{"empty string", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// When stripANSI is called
			got := stripANSI(tt.input)

			// Then escape codes are removed
			if got != tt.want {
				t.Errorf("stripANSI() = %q, want %q", got, tt.want)
			}
		})
	}
}
