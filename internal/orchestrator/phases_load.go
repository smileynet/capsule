package orchestrator

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// phaseYAML is the YAML representation of a PhaseDefinition.
type phaseYAML struct {
	Name        string `yaml:"name"`
	Kind        string `yaml:"kind"`                   // "worker" | "reviewer" | "gate"
	Prompt      string `yaml:"prompt,omitempty"`       // Template name override
	Command     string `yaml:"command,omitempty"`      // Shell command for gate
	MaxRetries  int    `yaml:"max_retries,omitempty"`  // 0 means use pipeline default
	RetryTarget string `yaml:"retry_target,omitempty"` // Phase to retry on NEEDS_WORK
	Optional    bool   `yaml:"optional,omitempty"`     // Continue pipeline on failure
	Condition   string `yaml:"condition,omitempty"`    // "files_match:<glob>" or empty
	Provider    string `yaml:"provider,omitempty"`     // Per-phase provider override
	Timeout     string `yaml:"timeout,omitempty"`      // Duration string (e.g. "5m")
}

// phasesFile is the top-level YAML structure for a phases file.
type phasesFile struct {
	Phases []phaseYAML `yaml:"phases"`
}

// LoadPhases resolves a phases specifier to a slice of PhaseDefinitions.
// The specifier can be a preset name ("default", "minimal", "thorough")
// or a path to a YAML file.
func LoadPhases(specifier string) ([]PhaseDefinition, error) {
	if phases := PresetPhases(specifier); phases != nil {
		return phases, nil
	}

	return LoadPhasesFile(specifier)
}

// LoadPhasesFile loads phase definitions from a YAML file.
func LoadPhasesFile(path string) ([]PhaseDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("phases: reading %s: %w", path, err)
	}
	return ParsePhasesYAML(data)
}

// ParsePhasesYAML parses phase definitions from YAML bytes.
func ParsePhasesYAML(data []byte) ([]PhaseDefinition, error) {
	var file phasesFile
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&file); err != nil {
		return nil, fmt.Errorf("phases: parsing YAML: %w", err)
	}

	if len(file.Phases) == 0 {
		return nil, errors.New("phases: no phases defined")
	}

	phases := make([]PhaseDefinition, len(file.Phases))
	for i, py := range file.Phases {
		pd, err := convertPhaseYAML(py)
		if err != nil {
			return nil, fmt.Errorf("phases[%d] %q: %w", i, py.Name, err)
		}
		phases[i] = pd
	}

	if err := ValidatePhases(phases); err != nil {
		return nil, err
	}

	return phases, nil
}

// convertPhaseYAML converts a phaseYAML to a PhaseDefinition.
func convertPhaseYAML(py phaseYAML) (PhaseDefinition, error) {
	if py.Name == "" {
		return PhaseDefinition{}, errors.New("name is required")
	}

	pd := PhaseDefinition{
		Name:        py.Name,
		Prompt:      py.Prompt,
		Command:     py.Command,
		MaxRetries:  py.MaxRetries,
		RetryTarget: py.RetryTarget,
		Optional:    py.Optional,
		Condition:   py.Condition,
		Provider:    py.Provider,
	}

	switch py.Kind {
	case "worker", "":
		pd.Kind = Worker
	case "reviewer":
		pd.Kind = Reviewer
	case "gate":
		pd.Kind = Gate
	default:
		return PhaseDefinition{}, fmt.Errorf("invalid kind %q (must be worker, reviewer, or gate)", py.Kind)
	}

	if py.Timeout != "" {
		d, err := time.ParseDuration(py.Timeout)
		if err != nil {
			return PhaseDefinition{}, fmt.Errorf("invalid timeout %q: %w", py.Timeout, err)
		}
		pd.Timeout = d
	}

	return pd, nil
}

// ValidatePhases checks phase definitions for consistency errors.
func ValidatePhases(phases []PhaseDefinition) error {
	names := make(map[string]int, len(phases))
	for i, p := range phases {
		if _, exists := names[p.Name]; exists {
			return fmt.Errorf("phases: duplicate phase name %q", p.Name)
		}
		names[p.Name] = i
	}

	for _, p := range phases {
		// Gates must have a Command.
		if p.Kind == Gate {
			if p.Command == "" {
				return fmt.Errorf("phases: gate %q must have a command", p.Name)
			}
		}

		// Workers can't have RetryTarget.
		if p.Kind == Worker && p.RetryTarget != "" {
			return fmt.Errorf("phases: worker %q cannot have retry_target", p.Name)
		}

		// RetryTarget must reference an existing phase.
		if p.RetryTarget != "" {
			if _, exists := names[p.RetryTarget]; !exists {
				return fmt.Errorf("phases: %q retry_target %q not found", p.Name, p.RetryTarget)
			}
		}

		// Condition syntax validation.
		if p.Condition != "" {
			if err := validateCondition(p.Condition); err != nil {
				return fmt.Errorf("phases: %q condition: %w", p.Name, err)
			}
		}
	}

	// Check for cycles in retry target graph.
	return detectRetryCycles(phases, names)
}

// validateCondition checks that a condition string has valid syntax.
func validateCondition(cond string) error {
	if !strings.HasPrefix(cond, "files_match:") {
		return fmt.Errorf("unknown condition syntax %q (expected files_match:<glob>)", cond)
	}
	glob := strings.TrimPrefix(cond, "files_match:")
	if glob == "" {
		return errors.New("files_match condition requires a glob pattern")
	}
	if _, err := filepath.Match(glob, "test"); err != nil {
		return fmt.Errorf("invalid glob pattern %q: %w", glob, err)
	}
	return nil
}

// detectRetryCycles checks for cycles in the retry target graph.
func detectRetryCycles(phases []PhaseDefinition, names map[string]int) error {
	for _, p := range phases {
		if p.RetryTarget == "" {
			continue
		}
		visited := map[string]bool{p.Name: true}
		current := p.RetryTarget
		for current != "" {
			if visited[current] {
				return fmt.Errorf("phases: cycle in retry targets involving %q", p.Name)
			}
			visited[current] = true
			idx, ok := names[current]
			if !ok {
				break
			}
			current = phases[idx].RetryTarget
		}
	}
	return nil
}
