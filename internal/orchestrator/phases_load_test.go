package orchestrator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPresetPhases(t *testing.T) {
	tests := []struct {
		name       string
		preset     string
		wantLen    int
		wantNil    bool
		firstPhase string
	}{
		{name: "default", preset: "default", wantLen: 6, firstPhase: "test-writer"},
		{name: "empty string", preset: "", wantLen: 6, firstPhase: "test-writer"},
		{name: "minimal", preset: "minimal", wantLen: 3, firstPhase: "test-writer"},
		{name: "thorough", preset: "thorough", wantLen: 7, firstPhase: "test-writer"},
		{name: "unknown returns nil", preset: "unknown", wantNil: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			phases := PresetPhases(tt.preset)
			if tt.wantNil {
				if phases != nil {
					t.Errorf("expected nil for unknown preset, got %d phases", len(phases))
				}
				return
			}
			if len(phases) != tt.wantLen {
				t.Errorf("len(phases) = %d, want %d", len(phases), tt.wantLen)
			}
			if phases[0].Name != tt.firstPhase {
				t.Errorf("first phase = %q, want %q", phases[0].Name, tt.firstPhase)
			}
		})
	}
}

func TestThoroughPhases_HasGate(t *testing.T) {
	// Given the thorough preset
	phases := ThoroughPhases()

	// Then it contains a gate phase
	var found bool
	for _, p := range phases {
		if p.Kind == Gate {
			found = true
			if p.Command == "" {
				t.Error("gate phase should have a command")
			}
			if !p.Optional {
				t.Error("lint gate should be optional")
			}
		}
	}
	if !found {
		t.Error("thorough phases should contain a gate phase")
	}
}

func TestParsePhasesYAML_Valid(t *testing.T) {
	yaml := `
phases:
  - name: test-writer
    kind: worker
  - name: test-quality
    kind: reviewer
    prompt: test-quality
    retry_target: test-writer
    max_retries: 2
  - name: execute
    kind: worker
  - name: lint
    kind: gate
    command: "make lint"
    optional: true
  - name: code-review
    kind: reviewer
    retry_target: execute
    max_retries: 3
  - name: merge
    kind: worker
`
	phases, err := ParsePhasesYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(phases) != 6 {
		t.Fatalf("len(phases) = %d, want 6", len(phases))
	}

	// Check test-quality reviewer
	if phases[1].Kind != Reviewer {
		t.Errorf("phases[1].Kind = %v, want Reviewer", phases[1].Kind)
	}
	if phases[1].Prompt != "test-quality" {
		t.Errorf("phases[1].Prompt = %q, want %q", phases[1].Prompt, "test-quality")
	}
	if phases[1].RetryTarget != "test-writer" {
		t.Errorf("phases[1].RetryTarget = %q, want %q", phases[1].RetryTarget, "test-writer")
	}

	// Check gate
	if phases[3].Kind != Gate {
		t.Errorf("phases[3].Kind = %v, want Gate", phases[3].Kind)
	}
	if phases[3].Command != "make lint" {
		t.Errorf("phases[3].Command = %q, want %q", phases[3].Command, "make lint")
	}
	if !phases[3].Optional {
		t.Error("phases[3].Optional should be true")
	}
}

func TestParsePhasesYAML_WithTimeout(t *testing.T) {
	yaml := `
phases:
  - name: slow-worker
    kind: worker
    timeout: 10m
`
	phases, err := ParsePhasesYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if phases[0].Timeout.Minutes() != 10 {
		t.Errorf("Timeout = %v, want 10m", phases[0].Timeout)
	}
}

func TestParsePhasesYAML_DefaultKind(t *testing.T) {
	// Given YAML without kind (defaults to worker)
	yaml := `
phases:
  - name: execute
`
	phases, err := ParsePhasesYAML([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if phases[0].Kind != Worker {
		t.Errorf("Kind = %v, want Worker (default)", phases[0].Kind)
	}
}

func TestParsePhasesYAML_Errors(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		wantErr string
	}{
		{
			name:    "no phases",
			yaml:    "phases: []",
			wantErr: "no phases defined",
		},
		{
			name:    "missing name",
			yaml:    "phases:\n  - kind: worker",
			wantErr: "name is required",
		},
		{
			name:    "invalid kind",
			yaml:    "phases:\n  - name: x\n    kind: invalid",
			wantErr: "invalid kind",
		},
		{
			name:    "gate without command",
			yaml:    "phases:\n  - name: lint\n    kind: gate",
			wantErr: "must have a command",
		},
		{
			name:    "worker with retry_target",
			yaml:    "phases:\n  - name: w\n    kind: worker\n    retry_target: w",
			wantErr: "cannot have retry_target",
		},
		{
			name:    "retry_target not found",
			yaml:    "phases:\n  - name: r\n    kind: reviewer\n    retry_target: missing",
			wantErr: "not found",
		},
		{
			name:    "duplicate name",
			yaml:    "phases:\n  - name: x\n  - name: x",
			wantErr: "duplicate phase name",
		},
		{
			name:    "invalid timeout",
			yaml:    "phases:\n  - name: x\n    timeout: notaduration",
			wantErr: "invalid timeout",
		},
		{
			name:    "unknown field",
			yaml:    "phases:\n  - name: x\n    bogus: true",
			wantErr: "parsing YAML",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParsePhasesYAML([]byte(tt.yaml))
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestValidatePhases_Condition(t *testing.T) {
	tests := []struct {
		name      string
		condition string
		wantErr   bool
	}{
		{name: "empty is valid", condition: ""},
		{name: "files_match glob", condition: "files_match:*.go"},
		{name: "unknown prefix", condition: "env_match:FOO", wantErr: true},
		{name: "empty glob", condition: "files_match:", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			phases := []PhaseDefinition{{Name: "test", Kind: Worker, Condition: tt.condition}}
			err := ValidatePhases(phases)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePhases() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePhases_RetryCycle(t *testing.T) {
	// Given phases with a cycle: a retries b, b retries a
	phases := []PhaseDefinition{
		{Name: "a", Kind: Reviewer, RetryTarget: "b"},
		{Name: "b", Kind: Reviewer, RetryTarget: "a"},
	}
	err := ValidatePhases(phases)
	if err == nil {
		t.Fatal("expected cycle detection error")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("error = %q, want containing 'cycle'", err.Error())
	}
}

func TestLoadPhasesFile(t *testing.T) {
	// Given a phases YAML file on disk
	dir := t.TempDir()
	path := filepath.Join(dir, "phases.yaml")
	if err := os.WriteFile(path, []byte(`
phases:
  - name: worker
  - name: reviewer
    kind: reviewer
    retry_target: worker
    max_retries: 2
`), 0o644); err != nil {
		t.Fatal(err)
	}

	// When loaded
	phases, err := LoadPhasesFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Then phases are parsed correctly
	if len(phases) != 2 {
		t.Errorf("len(phases) = %d, want 2", len(phases))
	}
}

func TestLoadPhases_Preset(t *testing.T) {
	// Given a preset name
	phases, err := LoadPhases("minimal")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(phases) != 3 {
		t.Errorf("len(phases) = %d, want 3", len(phases))
	}
}

func TestLoadPhases_FileNotFound(t *testing.T) {
	// Given a nonexistent file path
	_, err := LoadPhases("/nonexistent/phases.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestPromptName(t *testing.T) {
	tests := []struct {
		name  string
		phase PhaseDefinition
		want  string
	}{
		{name: "uses Prompt field", phase: PhaseDefinition{Name: "test", Prompt: "custom"}, want: "custom"},
		{name: "falls back to Name", phase: PhaseDefinition{Name: "test"}, want: "test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.phase.PromptName(); got != tt.want {
				t.Errorf("PromptName() = %q, want %q", got, tt.want)
			}
		})
	}
}
