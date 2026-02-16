package tui

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mattn/go-isatty"
)

// DisplayEvent is an event sent to a Display via the update channel.
// Implemented by StatusUpdateMsg, PipelineDoneMsg, and PipelineErrorMsg.
type DisplayEvent interface {
	isDisplayEvent()
}

// Verify at compile time that message types implement DisplayEvent.
var (
	_ DisplayEvent = StatusUpdateMsg{}
	_ DisplayEvent = PipelineDoneMsg{}
	_ DisplayEvent = PipelineErrorMsg{}
	_ DisplayEvent = OutputMsg{}
)

// Display renders pipeline status updates.
type Display interface {
	Run(ctx context.Context, events <-chan DisplayEvent) error
}

// DisplayOptions configures display creation.
type DisplayOptions struct {
	Writer     io.Writer          // Output destination (default: os.Stdout).
	ForcePlain bool               // Force plain text even if TTY.
	Phases     []string           // Phase names for TUI initialization.
	CancelFunc context.CancelFunc // Called by TUI on abort keypress (ignored by PlainDisplay).
}

// NewDisplay returns a TUI display when stdout is a TTY, or a plain text
// display otherwise. ForcePlain overrides TTY detection.
func NewDisplay(opts DisplayOptions) Display {
	if opts.Writer == nil {
		opts.Writer = os.Stdout
	}

	if opts.ForcePlain || !isTTY(opts.Writer) {
		return &PlainDisplay{w: opts.Writer}
	}

	return &TUIDisplay{phases: opts.Phases, w: opts.Writer, cancelFunc: opts.CancelFunc}
}

// isTTY reports whether w is connected to a terminal.
func isTTY(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}

// Bridge manages the channel between a status producer and a Display consumer.
type Bridge struct {
	ch chan DisplayEvent
}

// NewBridge creates a Bridge with a buffered event channel.
func NewBridge() *Bridge {
	return &Bridge{ch: make(chan DisplayEvent, 16)}
}

// Events returns the read-only channel for Display.Run() to consume.
func (b *Bridge) Events() <-chan DisplayEvent {
	return b.ch
}

// Send delivers a StatusUpdateMsg to the display.
// It blocks if the channel buffer (16) is full.
func (b *Bridge) Send(msg StatusUpdateMsg) {
	b.ch <- msg
}

// Done signals successful pipeline completion and closes the channel.
func (b *Bridge) Done() {
	b.ch <- PipelineDoneMsg{}
	close(b.ch)
}

// Error signals pipeline failure and closes the channel.
func (b *Bridge) Error(err error) {
	b.ch <- PipelineErrorMsg{Err: err}
	close(b.ch)
}

// PlainDisplay renders status updates as timestamped text lines.
type PlainDisplay struct {
	w io.Writer
}

// Run loops over events, printing each status update as a text line.
// Returns the pipeline error if the pipeline failed, or context error if cancelled.
func (d *PlainDisplay) Run(ctx context.Context, events <-chan DisplayEvent) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case ev, ok := <-events:
			if !ok {
				return nil
			}
			switch msg := ev.(type) {
			case StatusUpdateMsg:
				d.renderUpdate(msg)
			case OutputMsg:
				// Detail output is TUI-only; ignored in plain text mode.
			case PipelineDoneMsg:
				return nil
			case PipelineErrorMsg:
				return msg.Err
			}
		}
	}
}

func (d *PlainDisplay) renderUpdate(su StatusUpdateMsg) {
	ts := time.Now().Format("15:04:05")
	retry := ""
	if su.Attempt > 1 {
		retry = fmt.Sprintf(" (attempt %d/%d)", su.Attempt, su.MaxRetry)
	}
	_, _ = fmt.Fprintf(d.w, "[%s] [%s] %s %s%s\n", ts, su.Progress, su.Phase, su.Status, retry)

	if su.Status == StatusRunning {
		return
	}

	if len(su.FilesChanged) > 0 {
		_, _ = fmt.Fprintf(d.w, "         files: %s\n", strings.Join(su.FilesChanged, ", "))
	}
	if su.Summary != "" {
		_, _ = fmt.Fprintf(d.w, "         summary: %s\n", su.Summary)
	}
	// Feedback is only meaningful for failed/error phases (NEEDS_WORK from orchestrator).
	if su.Feedback != "" && (su.Status == StatusFailed || su.Status == StatusError) {
		_, _ = fmt.Fprintf(d.w, "         feedback: %s\n", su.Feedback)
	}
}

// TUIDisplay renders status updates using a Bubble Tea terminal UI.
// Falls back to PlainDisplay if the TUI program fails to start.
type TUIDisplay struct {
	phases     []string
	w          io.Writer
	cancelFunc context.CancelFunc
}

// Run starts the Bubble Tea program and feeds events from the channel.
// If the TUI fails to initialize, it falls back to plain text output.
func (d *TUIDisplay) Run(ctx context.Context, events <-chan DisplayEvent) error {
	var opts []ModelOption
	if d.cancelFunc != nil {
		opts = append(opts, WithCancelFunc(d.cancelFunc))
	}
	model := NewModel(d.phases, opts...)
	p := tea.NewProgram(model, tea.WithOutput(d.w))

	// Forward events through an intermediate channel so we can stop
	// the goroutine cleanly on TUI failure before falling back.
	fwd := make(chan DisplayEvent, 16)
	stop := make(chan struct{})

	go func() {
		defer close(fwd)
		for ev := range events {
			select {
			case fwd <- ev:
			case <-stop:
				return
			}
		}
	}()

	go func() {
		for ev := range fwd {
			p.Send(ev)
		}
	}()

	_, err := p.Run()
	if err != nil {
		close(stop)
		// Fall back to plain text for remaining events from the original channel.
		plain := &PlainDisplay{w: d.w}
		return plain.Run(ctx, events)
	}

	return nil
}
