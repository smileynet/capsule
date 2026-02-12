package tui

import (
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"
)

// --- isTTY ---

func TestIsTTY_NonFileWriter(t *testing.T) {
	var buf bytes.Buffer
	if isTTY(&buf) {
		t.Error("non-*os.File writer should not be a TTY")
	}
}

func TestIsTTY_RegularFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	if isTTY(f) {
		t.Error("regular file should not be a TTY")
	}
}

// --- Bridge ---

func TestBridge_SendDeliversStatusUpdate(t *testing.T) {
	b := NewBridge()
	msg := StatusUpdateMsg{Phase: "test-writer", Status: StatusRunning}

	go b.Send(msg)

	got := <-b.Events()
	su, ok := got.(StatusUpdateMsg)
	if !ok {
		t.Fatalf("expected StatusUpdateMsg, got %T", got)
	}
	if su.Phase != "test-writer" {
		t.Errorf("phase = %q, want %q", su.Phase, "test-writer")
	}
}

func TestBridge_DoneSendsPipelineDoneAndCloses(t *testing.T) {
	b := NewBridge()

	go b.Done()

	got := <-b.Events()
	if _, ok := got.(PipelineDoneMsg); !ok {
		t.Fatalf("expected PipelineDoneMsg, got %T", got)
	}

	// Channel should be closed after Done.
	_, open := <-b.Events()
	if open {
		t.Error("channel should be closed after Done")
	}
}

func TestBridge_ErrorSendsPipelineErrorAndCloses(t *testing.T) {
	b := NewBridge()
	testErr := errors.New("pipeline exploded")

	go b.Error(testErr)

	got := <-b.Events()
	pe, ok := got.(PipelineErrorMsg)
	if !ok {
		t.Fatalf("expected PipelineErrorMsg, got %T", got)
	}
	if pe.Err.Error() != "pipeline exploded" {
		t.Errorf("error = %q, want %q", pe.Err, "pipeline exploded")
	}

	_, open := <-b.Events()
	if open {
		t.Error("channel should be closed after Error")
	}
}

func TestBridge_MultipleEvents(t *testing.T) {
	b := NewBridge()

	go func() {
		b.Send(StatusUpdateMsg{Phase: "phase1", Status: StatusRunning})
		b.Send(StatusUpdateMsg{Phase: "phase1", Status: StatusPassed})
		b.Done()
	}()

	var events []DisplayEvent
	for ev := range b.Events() {
		events = append(events, ev)
	}

	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}
	if _, ok := events[2].(PipelineDoneMsg); !ok {
		t.Errorf("last event should be PipelineDoneMsg, got %T", events[2])
	}
}

// --- PlainDisplay ---

func TestPlainDisplay_RendersStatusUpdate(t *testing.T) {
	var buf bytes.Buffer
	d := &PlainDisplay{w: &buf}
	ctx := context.Background()

	ch := make(chan DisplayEvent, 2)
	ch <- StatusUpdateMsg{
		Phase:    "test-writer",
		Status:   StatusRunning,
		Progress: "1/3",
		Attempt:  1,
		MaxRetry: 3,
	}
	ch <- PipelineDoneMsg{}
	close(ch)

	err := d.Run(ctx, ch)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "test-writer") {
		t.Error("output should contain phase name")
	}
	if !strings.Contains(out, "1/3") {
		t.Error("output should contain progress")
	}
	if !strings.Contains(out, "running") {
		t.Error("output should contain status")
	}
}

func TestPlainDisplay_RendersRetryInfo(t *testing.T) {
	var buf bytes.Buffer
	d := &PlainDisplay{w: &buf}
	ctx := context.Background()

	ch := make(chan DisplayEvent, 2)
	ch <- StatusUpdateMsg{
		Phase:    "test-review",
		Status:   StatusRunning,
		Progress: "2/3",
		Attempt:  2,
		MaxRetry: 3,
	}
	ch <- PipelineDoneMsg{}
	close(ch)

	if err := d.Run(ctx, ch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "attempt 2/3") {
		t.Errorf("output should show retry info, got:\n%s", out)
	}
}

func TestPlainDisplay_RendersSignalData(t *testing.T) {
	var buf bytes.Buffer
	d := &PlainDisplay{w: &buf}
	ctx := context.Background()

	ch := make(chan DisplayEvent, 2)
	ch <- StatusUpdateMsg{
		Phase:        "test-writer",
		Status:       StatusPassed,
		Progress:     "1/3",
		Summary:      "All tests pass",
		FilesChanged: []string{"foo.go", "bar.go"},
	}
	ch <- PipelineDoneMsg{}
	close(ch)

	if err := d.Run(ctx, ch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "foo.go") {
		t.Error("output should contain files changed")
	}
	if !strings.Contains(out, "All tests pass") {
		t.Error("output should contain summary")
	}
}

func TestPlainDisplay_RendersFeedbackOnFailure(t *testing.T) {
	var buf bytes.Buffer
	d := &PlainDisplay{w: &buf}
	ctx := context.Background()

	ch := make(chan DisplayEvent, 2)
	ch <- StatusUpdateMsg{
		Phase:    "test-review",
		Status:   StatusFailed,
		Progress: "2/3",
		Feedback: "Tests need more coverage",
	}
	ch <- PipelineDoneMsg{}
	close(ch)

	if err := d.Run(ctx, ch); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Tests need more coverage") {
		t.Errorf("output should show feedback on failure, got:\n%s", out)
	}
}

func TestPlainDisplay_HandlesContextCancellation(t *testing.T) {
	var buf bytes.Buffer
	d := &PlainDisplay{w: &buf}
	ctx, cancel := context.WithCancel(context.Background())

	ch := make(chan DisplayEvent) // Unbuffered, will block.

	done := make(chan error, 1)
	go func() {
		done <- d.Run(ctx, ch)
	}()

	cancel()

	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not return after context cancellation")
	}
}

func TestPlainDisplay_ReturnsErrorFromPipelineError(t *testing.T) {
	var buf bytes.Buffer
	d := &PlainDisplay{w: &buf}
	ctx := context.Background()

	ch := make(chan DisplayEvent, 1)
	ch <- PipelineErrorMsg{Err: errors.New("provider crashed")}
	close(ch)

	err := d.Run(ctx, ch)
	if err == nil || !strings.Contains(err.Error(), "provider crashed") {
		t.Errorf("expected pipeline error, got %v", err)
	}
}

// --- NewDisplay factory ---

func TestNewDisplay_ForcePlainReturnsPlainDisplay(t *testing.T) {
	d := NewDisplay(DisplayOptions{
		Writer:     os.Stdout,
		ForcePlain: true,
		Phases:     []string{"phase1"},
	})

	if _, ok := d.(*PlainDisplay); !ok {
		t.Errorf("ForcePlain should return *PlainDisplay, got %T", d)
	}
}

func TestNewDisplay_NonTTYReturnsPlainDisplay(t *testing.T) {
	var buf bytes.Buffer
	d := NewDisplay(DisplayOptions{
		Writer: &buf,
		Phases: []string{"phase1"},
	})

	if _, ok := d.(*PlainDisplay); !ok {
		t.Errorf("non-TTY writer should return *PlainDisplay, got %T", d)
	}
}

func TestNewDisplay_DefaultsWriterToStdout(t *testing.T) {
	d := NewDisplay(DisplayOptions{
		ForcePlain: true,
		Phases:     []string{"phase1"},
	})

	pd, ok := d.(*PlainDisplay)
	if !ok {
		t.Fatalf("expected *PlainDisplay, got %T", d)
	}
	if pd.w != os.Stdout {
		t.Error("default Writer should be os.Stdout")
	}
}
