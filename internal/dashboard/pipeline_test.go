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
	ps := newPipelineState(samplePhaseNames())
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
	ps := newPipelineState(samplePhaseNames())
	view := ps.View(60, 20)
	plain := stripANSI(view)

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
	ps := newPipelineState(samplePhaseNames())
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "plan", Status: PhasePassed, Duration: 2 * time.Second})

	view := ps.View(60, 20)
	plain := stripANSI(view)
	if !strings.Contains(plain, "✓") {
		t.Errorf("passed phase should show ✓ indicator, got:\n%s", plain)
	}
	if !strings.Contains(plain, "2.0s") {
		t.Errorf("passed phase should show duration, got:\n%s", plain)
	}
}

func TestPipeline_ViewFailedPhase(t *testing.T) {
	ps := newPipelineState(samplePhaseNames())
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "code", Status: PhaseFailed})

	view := ps.View(60, 20)
	plain := stripANSI(view)
	if !strings.Contains(plain, "✗") {
		t.Errorf("failed phase should show ✗ indicator, got:\n%s", plain)
	}
}

func TestPipeline_ViewSkippedPhase(t *testing.T) {
	ps := newPipelineState(samplePhaseNames())
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "review", Status: PhaseSkipped})

	view := ps.View(60, 20)
	plain := stripANSI(view)
	if !strings.Contains(plain, "–") {
		t.Errorf("skipped phase should show – indicator, got:\n%s", plain)
	}
}

func TestPipeline_RetryCounter(t *testing.T) {
	ps := newPipelineState(samplePhaseNames())
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "code", Status: PhaseRunning, Attempt: 2, MaxRetry: 3})

	view := ps.View(60, 20)
	plain := stripANSI(view)
	if !strings.Contains(plain, "(2/3)") {
		t.Errorf("retry counter should show (2/3), got:\n%s", plain)
	}
}

func TestPipeline_DurationNotShownForZero(t *testing.T) {
	ps := newPipelineState(samplePhaseNames())
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "plan", Status: PhasePassed})

	view := ps.View(60, 20)
	plain := stripANSI(view)
	if strings.Contains(plain, "0.0s") {
		t.Errorf("zero duration should not be displayed, got:\n%s", plain)
	}
}

func TestPipeline_RetryCounterNotShownForFirstAttempt(t *testing.T) {
	ps := newPipelineState(samplePhaseNames())
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "code", Status: PhaseRunning, Attempt: 1, MaxRetry: 3})

	view := ps.View(60, 20)
	plain := stripANSI(view)
	if strings.Contains(plain, "(1/3)") {
		t.Errorf("retry counter should not show for first attempt, got:\n%s", plain)
	}
}

func TestPipeline_CursorDown(t *testing.T) {
	ps := newPipelineState(samplePhaseNames())
	ps, _ = ps.Update(tea.KeyMsg{Type: tea.KeyDown})

	if ps.cursor != 1 {
		t.Errorf("after down: cursor = %d, want 1", ps.cursor)
	}
}

func TestPipeline_CursorUp(t *testing.T) {
	ps := newPipelineState(samplePhaseNames())
	ps, _ = ps.Update(tea.KeyMsg{Type: tea.KeyDown})
	ps, _ = ps.Update(tea.KeyMsg{Type: tea.KeyUp})

	if ps.cursor != 0 {
		t.Errorf("after down+up: cursor = %d, want 0", ps.cursor)
	}
}

func TestPipeline_CursorWrapsDown(t *testing.T) {
	ps := newPipelineState(samplePhaseNames())
	for range len(samplePhaseNames()) {
		ps, _ = ps.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	if ps.cursor != 0 {
		t.Errorf("after wrapping down: cursor = %d, want 0", ps.cursor)
	}
}

func TestPipeline_CursorWrapsUp(t *testing.T) {
	ps := newPipelineState(samplePhaseNames())
	ps, _ = ps.Update(tea.KeyMsg{Type: tea.KeyUp})
	want := len(samplePhaseNames()) - 1
	if ps.cursor != want {
		t.Errorf("after wrapping up: cursor = %d, want %d", ps.cursor, want)
	}
}

func TestPipeline_CursorDisablesAutoFollow(t *testing.T) {
	ps := newPipelineState(samplePhaseNames())
	if !ps.autoFollow {
		t.Fatal("autoFollow should be true initially")
	}

	ps, _ = ps.Update(tea.KeyMsg{Type: tea.KeyDown})
	if ps.autoFollow {
		t.Error("cursor movement should disable autoFollow")
	}
}

func TestPipeline_AutoFollowTracksRunningPhase(t *testing.T) {
	ps := newPipelineState(samplePhaseNames())

	// First phase starts running.
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "plan", Status: PhaseRunning})
	if ps.cursor != 0 {
		t.Errorf("cursor should follow running phase 'plan' at 0, got %d", ps.cursor)
	}

	// Second phase starts running.
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "code", Status: PhaseRunning})
	if ps.cursor != 1 {
		t.Errorf("cursor should follow running phase 'code' at 1, got %d", ps.cursor)
	}

	// Third phase starts running.
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "test", Status: PhaseRunning})
	if ps.cursor != 2 {
		t.Errorf("cursor should follow running phase 'test' at 2, got %d", ps.cursor)
	}
}

func TestPipeline_AutoFollowDisabledDoesNotTrack(t *testing.T) {
	ps := newPipelineState(samplePhaseNames())

	// Manually move cursor to disable autoFollow.
	ps, _ = ps.Update(tea.KeyMsg{Type: tea.KeyDown})
	ps, _ = ps.Update(tea.KeyMsg{Type: tea.KeyDown})
	if ps.cursor != 2 {
		t.Fatalf("cursor = %d, want 2", ps.cursor)
	}

	// Phase starts running — cursor should NOT follow.
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "plan", Status: PhaseRunning})
	if ps.cursor != 2 {
		t.Errorf("cursor should stay at 2 when autoFollow disabled, got %d", ps.cursor)
	}
}

func TestPipeline_CursorMarker(t *testing.T) {
	ps := newPipelineState(samplePhaseNames())
	view := ps.View(60, 20)
	plain := stripANSI(view)
	if !strings.Contains(plain, CursorMarker) {
		t.Errorf("view should contain cursor marker %q, got:\n%s", CursorMarker, plain)
	}
}

func TestPipeline_VimKeys(t *testing.T) {
	ps := newPipelineState(samplePhaseNames())

	// j moves down.
	ps, _ = ps.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if ps.cursor != 1 {
		t.Errorf("after j: cursor = %d, want 1", ps.cursor)
	}

	// k moves up.
	ps, _ = ps.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
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
			name: "first phase selected by default",
			setup: func() pipelineState {
				return newPipelineState(samplePhaseNames())
			},
			want: "plan",
		},
		{
			name: "second phase after cursor down",
			setup: func() pipelineState {
				ps := newPipelineState(samplePhaseNames())
				ps, _ = ps.Update(tea.KeyMsg{Type: tea.KeyDown})
				return ps
			},
			want: "code",
		},
		{
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
	ps := newPipelineState(nil)
	view := ps.View(60, 20)
	plain := stripANSI(view)
	if !strings.Contains(plain, "No phases") {
		t.Errorf("empty phases should show 'No phases', got:\n%s", plain)
	}
}

func TestPipeline_SpinnerTick(t *testing.T) {
	ps := newPipelineState(samplePhaseNames())
	tickMsg := ps.spinner.Tick()
	_, cmd := ps.Update(tickMsg)
	if cmd == nil {
		t.Error("spinner tick should produce a follow-up command")
	}
}

func TestPipeline_PhaseUpdateSetsRunning(t *testing.T) {
	ps := newPipelineState(samplePhaseNames())

	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "plan", Status: PhaseRunning})
	if !ps.running {
		t.Error("running should be true after a phase starts")
	}
}

func TestPipeline_PhaseUpdateIgnoresUnknownPhase(t *testing.T) {
	ps := newPipelineState(samplePhaseNames())
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "nonexistent", Status: PhaseRunning})

	// Cursor should not move, running should not change.
	if ps.cursor != 0 {
		t.Errorf("cursor = %d, want 0", ps.cursor)
	}
	if ps.running {
		t.Error("running should remain false for unknown phase")
	}
}

func TestPipeline_MultiplePhaseProgression(t *testing.T) {
	ps := newPipelineState(samplePhaseNames())

	// plan: running → passed
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "plan", Status: PhaseRunning})
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "plan", Status: PhasePassed, Duration: time.Second})

	// code: running
	ps, _ = ps.Update(PhaseUpdateMsg{Phase: "code", Status: PhaseRunning})

	view := ps.View(60, 20)
	plain := stripANSI(view)

	if !strings.Contains(plain, "✓") {
		t.Errorf("view should show ✓ for passed plan phase, got:\n%s", plain)
	}
	if !strings.Contains(plain, "1.0s") {
		t.Errorf("view should show duration for plan phase, got:\n%s", plain)
	}
	// Auto-follow should have cursor at code (index 1).
	if ps.cursor != 1 {
		t.Errorf("cursor should auto-follow to code at 1, got %d", ps.cursor)
	}
}
