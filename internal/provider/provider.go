// Package provider abstracts AI CLI execution behind a common interface.
package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Status represents the outcome of a pipeline phase.
type Status string

const (
	StatusPass      Status = "PASS"
	StatusNeedsWork Status = "NEEDS_WORK"
	StatusError     Status = "ERROR"
)

// Signal is the structured output produced by a pipeline phase.
type Signal struct {
	Status       Status   `json:"status"`
	Feedback     string   `json:"feedback"`
	FilesChanged []string `json:"files_changed"`
	Summary      string   `json:"summary"`
	CommitHash   string   `json:"commit_hash,omitempty"`
}

// Result holds the raw output from a provider execution.
type Result struct {
	Output   string
	ExitCode int
	Duration time.Duration
}

// ParseSignal extracts the Signal from this result's output.
func (r Result) ParseSignal() (Signal, error) {
	return ParseSignal(r.Output)
}

// Verify MockProvider satisfies Executor at compile time.
var _ Executor = (*MockProvider)(nil)

// MockProvider is a test double that satisfies any Provider-shaped interface.
type MockProvider struct {
	NameVal     string
	ExecuteFunc func(ctx context.Context, prompt, workDir string) (Result, error)
}

// Name returns the configured provider name.
func (m *MockProvider) Name() string { return m.NameVal }

// Execute delegates to ExecuteFunc, returning a zero Result if ExecuteFunc is nil.
func (m *MockProvider) Execute(ctx context.Context, prompt, workDir string) (Result, error) {
	if m.ExecuteFunc == nil {
		return Result{}, nil
	}
	return m.ExecuteFunc(ctx, prompt, workDir)
}

// ParseSignal extracts the last valid Signal JSON from phase output.
// It strips markdown code fences before scanning for JSON objects.
func ParseSignal(output string) (Signal, error) {
	// Strip markdown code fence lines.
	var cleaned []string
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			continue
		}
		cleaned = append(cleaned, line)
	}

	// Scan for the last valid JSON object containing a signal.
	var lastSignal *Signal
	for _, line := range cleaned {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var s Signal
		if err := json.Unmarshal([]byte(line), &s); err != nil {
			continue
		}
		// Must have all required fields to be considered a signal.
		if s.Status != "" && s.Feedback != "" && s.Summary != "" {
			lastSignal = &s
		}
	}

	if lastSignal == nil {
		return Signal{}, &SignalParseError{Reason: "no valid signal JSON found in output"}
	}

	// Validate status value.
	switch lastSignal.Status {
	case StatusPass, StatusNeedsWork, StatusError:
		// valid
	default:
		return Signal{}, &SignalParseError{Reason: fmt.Sprintf("invalid status value: %q", lastSignal.Status)}
	}

	// Ensure files_changed is never nil (normalize to empty slice).
	if lastSignal.FilesChanged == nil {
		lastSignal.FilesChanged = []string{}
	}

	return *lastSignal, nil
}

// SignalParseError indicates the phase output could not be parsed into a Signal.
type SignalParseError struct {
	Reason string
}

func (e *SignalParseError) Error() string {
	return "provider: signal parse: " + e.Reason
}

// ProviderError wraps an error from a specific provider.
type ProviderError struct {
	Provider string
	Err      error
}

func (e *ProviderError) Error() string {
	return fmt.Sprintf("provider: %s: %s", e.Provider, e.Err)
}

func (e *ProviderError) Unwrap() error {
	return e.Err
}

// TimeoutError indicates a provider execution exceeded its time limit.
type TimeoutError struct {
	Provider string
	Duration time.Duration
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("provider: %s: timed out after %s", e.Provider, e.Duration)
}
