package dashboard

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func samplePhaseNames() []string {
	return []string{"plan", "code", "test", "review"}
}

func TestPipeline_InitialState(t *testing.T) {
	// Given: a fresh pipeline state with sample phase names
	ps := newPipelineState(samplePhaseNames())

	// Then: cursor is at 0, autoFollow is on, not running, all phases pending
	if ps.cursor != 0 {
		t.Errorf("cursor = %d, want 0", ps.cursor)
	}
	if !ps.autoFollow {
		t.Error("autoFollow should be true initially")
	}
	if ps.running {
		t.Error("running should be false initially")
	}
	if len(ps.phases) != 4 {
		t.Errorf("len(phases) = %d, want 4", len(ps.phases))
	}
	for _, p := range ps.phases {
		if p.Status != PhasePending {
			t.Errorf("phase %q status = %q, want pending", p.Name, p.Status)
		}
	}
}

func TestPipeline_ViewPendingPhases(t *testing.T) {
	// Given: a pipeline state with all phases pending
	ps := newPipelineState(samplePhaseNames())

	// When: the view is rendered
	view := ps.View(60, 20)
	plain := stripANSI(view)

	// Then: all phase names and pending indicators are shown
	for _, name := range samplePhaseNames() {
		if !strings.Contains(plain, name) {
			t.Errorf("view should contain phase %q, got:\n%s", name, plain)
		}
	}
	if !strings.Contains(plain, "○") {
		t.Errorf("pending phases should show ○ indicator, got:\n%s", plain)
	}
}

func TestPipeline_ViewPassedPhase(t *testing.T) {
	// Given: a pipeline state with "plan" marked as passed
	ps := newPipelineState(samplePhaseNames())
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "plan", Status: PhasePassed, Duration: 2 * time.Second})

	// When: the view is rendered
	view := ps.View(60, 20)
	plain := stripANSI(view)

	// Then: a checkmark and duration are shown
	if !strings.Contains(plain, "✓") {
		t.Errorf("passed phase should show ✓ indicator, got:\n%s", plain)
	}
	if !strings.Contains(plain, "2.0s") {
		t.Errorf("passed phase should show duration, got:\n%s", plain)
	}
}

func TestPipeline_ViewFailedPhase(t *testing.T) {
	// Given: a pipeline state with "code" marked as failed
	ps := newPipelineState(samplePhaseNames())
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "code", Status: PhaseFailed})

	// When: the view is rendered
	view := ps.View(60, 20)
	plain := stripANSI(view)

	// Then: a cross indicator is shown
	if !strings.Contains(plain, "✗") {
		t.Errorf("failed phase should show ✗ indicator, got:\n%s", plain)
	}
}

func TestPipeline_ViewSkippedPhase(t *testing.T) {
	// Given: a pipeline state with "review" marked as skipped
	ps := newPipelineState(samplePhaseNames())
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "review", Status: PhaseSkipped})

	// When: the view is rendered
	view := ps.View(60, 20)
	plain := stripANSI(view)

	// Then: a dash indicator is shown
	if !strings.Contains(plain, "–") {
		t.Errorf("skipped phase should show – indicator, got:\n%s", plain)
	}
}

func TestPipeline_RetryCounter(t *testing.T) {
	// Given: a pipeline state with "code" on attempt 2 of 3
	ps := newPipelineState(samplePhaseNames())
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "code", Status: PhaseRunning, Attempt: 2, MaxRetry: 3})

	// When: the view is rendered
	view := ps.View(60, 20)
	plain := stripANSI(view)

	// Then: the retry counter (2/3) is shown
	if !strings.Contains(plain, "(2/3)") {
		t.Errorf("retry counter should show (2/3), got:\n%s", plain)
	}
}

func TestPipeline_DurationNotShownForZero(t *testing.T) {
	// Given: a passed phase with zero duration
	ps := newPipelineState(samplePhaseNames())
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "plan", Status: PhasePassed})

	// When: the view is rendered
	view := ps.View(60, 20)
	plain := stripANSI(view)

	// Then: "0.0s" is not displayed
	if strings.Contains(plain, "0.0s") {
		t.Errorf("zero duration should not be displayed, got:\n%s", plain)
	}
}

func TestPipeline_RetryCounterNotShownForFirstAttempt(t *testing.T) {
	// Given: a running phase on first attempt
	ps := newPipelineState(samplePhaseNames())
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "code", Status: PhaseRunning, Attempt: 1, MaxRetry: 3})

	// When: the view is rendered
	view := ps.View(60, 20)
	plain := stripANSI(view)

	// Then: no retry counter is shown for the first attempt
	if strings.Contains(plain, "(1/3)") {
		t.Errorf("retry counter should not show for first attempt, got:\n%s", plain)
	}
}

func TestPipeline_CursorDown(t *testing.T) {
	// Given: a pipeline state with cursor at 0
	ps := newPipelineState(samplePhaseNames())

	// When: down key is pressed
	ps, _ = ps.Update(tea.KeyMsg{Type: tea.KeyDown})

	// Then: the cursor moves to position 1
	if ps.cursor != 1 {
		t.Errorf("after down: cursor = %d, want 1", ps.cursor)
	}
}

func TestPipeline_CursorUp(t *testing.T) {
	// Given: a pipeline state with cursor moved to position 1
	ps := newPipelineState(samplePhaseNames())
	ps, _ = ps.Update(tea.KeyMsg{Type: tea.KeyDown})

	// When: up key is pressed
	ps, _ = ps.Update(tea.KeyMsg{Type: tea.KeyUp})

	// Then: the cursor returns to position 0
	if ps.cursor != 0 {
		t.Errorf("after down+up: cursor = %d, want 0", ps.cursor)
	}
}

func TestPipeline_CursorWrapsDown(t *testing.T) {
	// Given: a pipeline state with cursor at 0
	ps := newPipelineState(samplePhaseNames())

	// When: down is pressed past the last phase
	for range len(samplePhaseNames()) {
		ps, _ = ps.Update(tea.KeyMsg{Type: tea.KeyDown})
	}

	// Then: the cursor wraps to position 0
	if ps.cursor != 0 {
		t.Errorf("after wrapping down: cursor = %d, want 0", ps.cursor)
	}
}

func TestPipeline_CursorWrapsUp(t *testing.T) {
	// Given: a pipeline state with cursor at 0
	ps := newPipelineState(samplePhaseNames())

	// When: up is pressed from position 0
	ps, _ = ps.Update(tea.KeyMsg{Type: tea.KeyUp})

	// Then: the cursor wraps to the last phase
	want := len(samplePhaseNames()) - 1
	if ps.cursor != want {
		t.Errorf("after wrapping up: cursor = %d, want %d", ps.cursor, want)
	}
}

func TestPipeline_CursorDisablesAutoFollow(t *testing.T) {
	// Given: a pipeline state with autoFollow enabled
	ps := newPipelineState(samplePhaseNames())
	if !ps.autoFollow {
		t.Fatal("autoFollow should be true initially")
	}

	// When: cursor is moved manually
	ps, _ = ps.Update(tea.KeyMsg{Type: tea.KeyDown})

	// Then: autoFollow is disabled
	if ps.autoFollow {
		t.Error("cursor movement should disable autoFollow")
	}
}

func TestPipeline_AutoFollowTracksRunningPhase(t *testing.T) {
	// Given: a pipeline state with autoFollow enabled
	ps := newPipelineState(samplePhaseNames())

	// When: phases start running sequentially
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "plan", Status: PhaseRunning})

	// Then: cursor follows the running phase
	if ps.cursor != 0 {
		t.Errorf("cursor should follow running phase 'plan' at 0, got %d", ps.cursor)
	}

	// When: the next phase starts running
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "code", Status: PhaseRunning})

	// Then: cursor follows to the new running phase
	if ps.cursor != 1 {
		t.Errorf("cursor should follow running phase 'code' at 1, got %d", ps.cursor)
	}

	// When: the third phase starts running
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "test", Status: PhaseRunning})

	// Then: cursor follows to the third phase
	if ps.cursor != 2 {
		t.Errorf("cursor should follow running phase 'test' at 2, got %d", ps.cursor)
	}
}

func TestPipeline_AutoFollowDisabledDoesNotTrack(t *testing.T) {
	// Given: a pipeline state with autoFollow disabled by manual cursor movement
	ps := newPipelineState(samplePhaseNames())
	ps, _ = ps.Update(tea.KeyMsg{Type: tea.KeyDown})
	ps, _ = ps.Update(tea.KeyMsg{Type: tea.KeyDown})
	if ps.cursor != 2 {
		t.Fatalf("cursor = %d, want 2", ps.cursor)
	}

	// When: a phase starts running
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "plan", Status: PhaseRunning})

	// Then: cursor does not follow the running phase
	if ps.cursor != 2 {
		t.Errorf("cursor should stay at 2 when autoFollow disabled, got %d", ps.cursor)
	}
}

func TestPipeline_CursorMarker(t *testing.T) {
	// Given: a pipeline state with phases
	ps := newPipelineState(samplePhaseNames())

	// When: the view is rendered
	view := ps.View(60, 20)
	plain := stripANSI(view)

	// Then: the cursor marker is visible
	if !strings.Contains(plain, CursorMarker) {
		t.Errorf("view should contain cursor marker %q, got:\n%s", CursorMarker, plain)
	}
}

func TestPipeline_VimKeys(t *testing.T) {
	// Given: a pipeline state with cursor at 0
	ps := newPipelineState(samplePhaseNames())

	// When: j is pressed
	ps, _ = ps.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	// Then: cursor moves down
	if ps.cursor != 1 {
		t.Errorf("after j: cursor = %d, want 1", ps.cursor)
	}

	// When: k is pressed
	ps, _ = ps.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})

	// Then: cursor moves up
	if ps.cursor != 0 {
		t.Errorf("after k: cursor = %d, want 0", ps.cursor)
	}
}

func TestPipeline_SelectedPhase(t *testing.T) {
	tests := []struct {
		name  string
		setup func() pipelineState
		want  string
	}{
		{
			// Given: a fresh pipeline state
			// Then: the first phase is selected
			name: "first phase selected by default",
			setup: func() pipelineState {
				return newPipelineState(samplePhaseNames())
			},
			want: "plan",
		},
		{
			// Given: a pipeline state with cursor moved down
			// Then: the second phase is selected
			name: "second phase after cursor down",
			setup: func() pipelineState {
				ps := newPipelineState(samplePhaseNames())
				ps, _ = ps.Update(tea.KeyMsg{Type: tea.KeyDown})
				return ps
			},
			want: "code",
		},
		{
			// Given: a pipeline state with no phases
			// Then: empty string is returned
			name: "empty phases returns empty",
			setup: func() pipelineState {
				return newPipelineState(nil)
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := tt.setup()
			if got := ps.SelectedPhase(); got != tt.want {
				t.Errorf("SelectedPhase() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPipeline_EmptyPhases(t *testing.T) {
	// Given: a pipeline state with no phases
	ps := newPipelineState(nil)

	// When: the view is rendered
	view := ps.View(60, 20)
	plain := stripANSI(view)

	// Then: a "No phases" message is shown
	if !strings.Contains(plain, "No phases") {
		t.Errorf("empty phases should show 'No phases', got:\n%s", plain)
	}
}

func TestPipeline_SpinnerTick(t *testing.T) {
	// Given: a pipeline state with a spinner
	ps := newPipelineState(samplePhaseNames())

	// When: a spinner tick message is processed
	tickMsg := ps.spinner.Tick()
	_, cmd := ps.Update(tickMsg)

	// Then: a follow-up tick command is produced
	if cmd == nil {
		t.Error("spinner tick should produce a follow-up command")
	}
}

func TestPipeline_PhaseUpdateSetsRunning(t *testing.T) {
	// Given: a pipeline state with no running phases
	ps := newPipelineState(samplePhaseNames())

	// When: a phase starts running
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "plan", Status: PhaseRunning})

	// Then: the running flag is set
	if !ps.running {
		t.Error("running should be true after a phase starts")
	}
}

func TestPipeline_ReportStoredOnPass(t *testing.T) {
	// Given: a pipeline state with phases
	ps := newPipelineState(samplePhaseNames())

	// When: a phase passes with summary and files
	ps, _ = ps.Update(PhaseUpdateMsg{
		Phase:        "plan",
		Status:       PhasePassed,
		Duration:     2 * time.Second,
		Summary:      "All checks passed",
		FilesChanged: []string{"main.go", "util.go"},
	})

	// Then: a report is stored with all fields populated
	report := ps.reports["plan"]
	if report == nil {
		t.Fatal("expected report for passed phase")
	}
	if report.PhaseName != "plan" {
		t.Errorf("PhaseName = %q, want %q", report.PhaseName, "plan")
	}
	if report.Status != PhasePassed {
		t.Errorf("Status = %q, want %q", report.Status, PhasePassed)
	}
	if report.Summary != "All checks passed" {
		t.Errorf("Summary = %q, want %q", report.Summary, "All checks passed")
	}
	if len(report.FilesChanged) != 2 {
		t.Errorf("FilesChanged len = %d, want 2", len(report.FilesChanged))
	}
	if report.Duration != 2*time.Second {
		t.Errorf("Duration = %v, want 2s", report.Duration)
	}
}

func TestPipeline_ReportStoredOnFail(t *testing.T) {
	// Given: a pipeline state with phases
	ps := newPipelineState(samplePhaseNames())

	// When: a phase fails with feedback
	ps, _ = ps.Update(PhaseUpdateMsg{
		Phase:    "code",
		Status:   PhaseFailed,
		Duration: 5 * time.Second,
		Summary:  "Compilation failed",
		Feedback: "error in main.go:42",
	})

	// Then: a report is stored with failure details and feedback
	report := ps.reports["code"]
	if report == nil {
		t.Fatal("expected report for failed phase")
	}
	if report.Status != PhaseFailed {
		t.Errorf("Status = %q, want %q", report.Status, PhaseFailed)
	}
	if report.Feedback != "error in main.go:42" {
		t.Errorf("Feedback = %q, want %q", report.Feedback, "error in main.go:42")
	}
}

func TestPipeline_NoReportForRunningPhase(t *testing.T) {
	// Given: a pipeline state with phases
	ps := newPipelineState(samplePhaseNames())

	// When: a phase starts running (no completion)
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "plan", Status: PhaseRunning})

	// Then: no report is stored for the running phase
	if ps.reports["plan"] != nil {
		t.Error("running phase should not store a report")
	}
}

func TestPipeline_NoReportForPendingPhase(t *testing.T) {
	// Given: a fresh pipeline state (all phases pending)
	ps := newPipelineState(samplePhaseNames())

	// Then: no report exists for any pending phase
	if ps.reports["plan"] != nil {
		t.Error("pending phase should not have a report")
	}
}

func TestPipeline_ViewReportPending(t *testing.T) {
	// Given: a pipeline state with cursor on a pending phase
	ps := newPipelineState(samplePhaseNames())

	// When: the report view is rendered
	view := ps.ViewReport(60, 20)
	plain := stripANSI(view)

	// Then: "Waiting" is shown
	if !strings.Contains(plain, "Waiting") {
		t.Errorf("pending phase report should show 'Waiting', got:\n%s", plain)
	}
}

func TestPipeline_ViewReportRunning(t *testing.T) {
	// Given: a pipeline state with "plan" running
	ps := newPipelineState(samplePhaseNames())
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "plan", Status: PhaseRunning})

	// When: the report view is rendered
	view := ps.ViewReport(60, 20)
	plain := stripANSI(view)

	// Then: the phase name and "Running" are shown
	if !strings.Contains(plain, "plan") {
		t.Errorf("running phase report should show phase name, got:\n%s", plain)
	}
	if !strings.Contains(plain, "Running") {
		t.Errorf("running phase report should show 'Running', got:\n%s", plain)
	}
}

func TestPipeline_ViewReportPassed(t *testing.T) {
	// Given: a pipeline state with "plan" passed with summary and files
	ps := newPipelineState(samplePhaseNames())
	ps, _ = ps.Update(PhaseUpdateMsg{
		Phase:        "plan",
		Status:       PhasePassed,
		Duration:     3 * time.Second,
		Summary:      "All checks passed",
		FilesChanged: []string{"main.go", "util.go"},
	})

	// When: the report view is rendered
	view := ps.ViewReport(60, 20)
	plain := stripANSI(view)

	// Then: phase name, status, summary, files, and duration are shown
	if !strings.Contains(plain, "plan") {
		t.Errorf("report should show phase name, got:\n%s", plain)
	}
	if !strings.Contains(plain, "Passed") {
		t.Errorf("report should show 'Passed', got:\n%s", plain)
	}
	if !strings.Contains(plain, "All checks passed") {
		t.Errorf("report should show summary, got:\n%s", plain)
	}
	if !strings.Contains(plain, "main.go") {
		t.Errorf("report should show files changed, got:\n%s", plain)
	}
	if !strings.Contains(plain, "3.0s") {
		t.Errorf("report should show duration, got:\n%s", plain)
	}
}

func TestPipeline_ViewReportFailed(t *testing.T) {
	// Given: a pipeline state with "code" failed and cursor moved to it
	ps := newPipelineState(samplePhaseNames())
	ps, _ = ps.Update(PhaseUpdateMsg{
		Phase:    "code",
		Status:   PhaseFailed,
		Duration: 5 * time.Second,
		Summary:  "Build failed",
		Feedback: "error in main.go:42",
	})
	ps, _ = ps.Update(tea.KeyMsg{Type: tea.KeyDown})

	// When: the report view is rendered
	view := ps.ViewReport(60, 20)
	plain := stripANSI(view)

	// Then: "Failed" status and feedback are shown
	if !strings.Contains(plain, "Failed") {
		t.Errorf("report should show 'Failed', got:\n%s", plain)
	}
	if !strings.Contains(plain, "error in main.go:42") {
		t.Errorf("report should show feedback, got:\n%s", plain)
	}
}

func TestPipeline_ViewReportNoFeedbackForPassedPhase(t *testing.T) {
	// Given: a pipeline state with "plan" passed (no feedback)
	ps := newPipelineState(samplePhaseNames())
	ps, _ = ps.Update(PhaseUpdateMsg{
		Phase:    "plan",
		Status:   PhasePassed,
		Duration: 2 * time.Second,
		Summary:  "Done",
	})

	// When: the report view is rendered
	view := ps.ViewReport(60, 20)
	plain := stripANSI(view)

	// Then: no "Feedback" header is shown for passed phases
	if strings.Contains(plain, "Feedback") {
		t.Errorf("passed phase report should not show feedback header, got:\n%s", plain)
	}
}

func TestPipeline_ViewReportSkipped(t *testing.T) {
	// Given: a pipeline state with "plan" skipped
	ps := newPipelineState(samplePhaseNames())
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "plan", Status: PhaseSkipped})

	// When: the report view is rendered
	view := ps.ViewReport(60, 20)
	plain := stripANSI(view)

	// Then: "Skipped" is shown
	if !strings.Contains(plain, "Skipped") {
		t.Errorf("skipped phase report should show 'Skipped', got:\n%s", plain)
	}
}

func TestPipeline_ViewReportEmptyPhases(t *testing.T) {
	// Given: a pipeline state with no phases
	ps := newPipelineState(nil)

	// When: the report view is rendered
	view := ps.ViewReport(60, 20)

	// Then: empty string is returned
	if view != "" {
		t.Errorf("empty phases ViewReport should return empty, got: %q", view)
	}
}

func TestPipeline_PhaseUpdateIgnoresUnknownPhase(t *testing.T) {
	// Given: a pipeline state with known phases
	ps := newPipelineState(samplePhaseNames())

	// When: an update for an unknown phase is received
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "nonexistent", Status: PhaseRunning})

	// Then: cursor and running state are unchanged
	if ps.cursor != 0 {
		t.Errorf("cursor = %d, want 0", ps.cursor)
	}
	if ps.running {
		t.Error("running should remain false for unknown phase")
	}
}

func TestPipeline_MultiplePhaseProgression(t *testing.T) {
	// Given: a pipeline state with autoFollow enabled
	ps := newPipelineState(samplePhaseNames())

	// When: plan runs and passes, then code starts running
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "plan", Status: PhaseRunning})
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "plan", Status: PhasePassed, Duration: time.Second})
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "code", Status: PhaseRunning})

	// Then: view shows passed plan with duration and cursor auto-follows to code
	view := ps.View(60, 20)
	plain := stripANSI(view)
	if !strings.Contains(plain, "✓") {
		t.Errorf("view should show ✓ for passed plan phase, got:\n%s", plain)
	}
	if !strings.Contains(plain, "1.0s") {
		t.Errorf("view should show duration for plan phase, got:\n%s", plain)
	}
	if ps.cursor != 1 {
		t.Errorf("cursor should auto-follow to code at 1, got %d", ps.cursor)
	}
}
