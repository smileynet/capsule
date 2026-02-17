// Package config handles layered YAML configuration with environment overrides.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config holds all capsule configuration.
type Config struct {
	Runtime  Runtime  `yaml:"runtime"`
	Worktree Worktree `yaml:"worktree"`
	Pipeline Pipeline `yaml:"pipeline"`
	Campaign Campaign `yaml:"campaign"`
}

// Runtime holds provider and execution settings.
type Runtime struct {
	Provider string        `yaml:"provider"`
	Timeout  time.Duration `yaml:"timeout"`
}

// Worktree holds worktree directory settings.
type Worktree struct {
	BaseDir string `yaml:"base_dir"`
}

// Pipeline holds pipeline execution settings.
type Pipeline struct {
	Phases     string      `yaml:"phases"`     // "default" | "minimal" | path to YAML
	Checkpoint bool        `yaml:"checkpoint"` // Enable state checkpointing
	Retry      RetryConfig `yaml:"retry"`      // Pipeline-wide retry defaults
}

// RetryConfig holds retry strategy settings.
type RetryConfig struct {
	MaxAttempts      int     `yaml:"max_attempts"`
	BackoffFactor    float64 `yaml:"backoff_factor"`
	EscalateProvider string  `yaml:"escalate_provider"`
	EscalateAfter    int     `yaml:"escalate_after"`
}

// Campaign holds campaign orchestration settings.
type Campaign struct {
	FailureMode      string `yaml:"failure_mode"`      // "abort" | "continue"
	CircuitBreaker   int    `yaml:"circuit_breaker"`   // Consecutive failures before stopping
	DiscoveryFiling  bool   `yaml:"discovery_filing"`  // File findings as new beads
	CrossRunContext  bool   `yaml:"cross_run_context"` // Include sibling context in prompts
	ValidationPhases string `yaml:"validation_phases"` // Phase set for feature validation
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Runtime: Runtime{
			Provider: "claude",
			Timeout:  5 * time.Minute,
		},
		Worktree: Worktree{
			BaseDir: ".capsule/worktrees",
		},
		Pipeline: Pipeline{
			Phases:     "default",
			Checkpoint: false,
			Retry: RetryConfig{
				MaxAttempts:   3,
				BackoffFactor: 1.0,
			},
		},
		Campaign: Campaign{
			FailureMode:    "abort",
			CircuitBreaker: 3,
		},
	}
}

// Load reads a single YAML config file at path and returns a Config.
// For merging multiple config sources, use LoadLayered instead.
// If the file does not exist, defaults are returned without error.
// If the file contains invalid YAML or unknown fields, an error is returned.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &cfg, nil
		}
		return nil, fmt.Errorf("config: reading %s: %w", path, err)
	}

	if len(data) == 0 {
		return &cfg, nil
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		// Comment-only YAML files produce EOF with no decoded content.
		if errors.Is(err, io.EOF) {
			return &cfg, nil
		}
		return nil, fmt.Errorf("config: parsing %s: %w", path, err)
	}

	return &cfg, nil
}

// LoadLayered loads config from multiple paths with increasing priority.
// Later paths override earlier ones. Missing files are skipped.
func LoadLayered(paths ...string) (*Config, error) {
	cfg := DefaultConfig()

	for _, path := range paths {
		layer, err := loadLayer(path)
		if err != nil {
			return nil, err
		}
		if layer == nil {
			continue
		}
		cfg.merge(layer)
	}

	return &cfg, nil
}

// Validate checks that config values are usable.
func (c *Config) Validate() error {
	if c.Runtime.Provider == "" {
		return errors.New("config: runtime.provider cannot be empty")
	}
	if c.Runtime.Timeout <= 0 {
		return fmt.Errorf("config: runtime.timeout must be positive, got %v", c.Runtime.Timeout)
	}
	if c.Worktree.BaseDir == "" {
		return errors.New("config: worktree.base_dir cannot be empty")
	}
	if c.Pipeline.Retry.MaxAttempts < 0 {
		return fmt.Errorf("config: pipeline.retry.max_attempts must be non-negative, got %d", c.Pipeline.Retry.MaxAttempts)
	}
	if c.Pipeline.Retry.BackoffFactor < 0 {
		return fmt.Errorf("config: pipeline.retry.backoff_factor must be non-negative, got %v", c.Pipeline.Retry.BackoffFactor)
	}
	// BackoffFactor in (0, 1.0) would shrink timeouts on retry; reject.
	if c.Pipeline.Retry.BackoffFactor > 0 && c.Pipeline.Retry.BackoffFactor < 1.0 {
		return fmt.Errorf("config: pipeline.retry.backoff_factor must be 0 (disabled) or >= 1.0, got %v", c.Pipeline.Retry.BackoffFactor)
	}
	switch c.Campaign.FailureMode {
	case "", "abort", "continue":
		// valid
	default:
		return fmt.Errorf("config: campaign.failure_mode must be \"abort\" or \"continue\", got %q", c.Campaign.FailureMode)
	}
	if c.Campaign.CircuitBreaker < 0 {
		return fmt.Errorf("config: campaign.circuit_breaker must be non-negative, got %d", c.Campaign.CircuitBreaker)
	}
	return nil
}

// ApplyEnv applies environment variable overrides to the config.
// Supported variables: CAPSULE_PROVIDER, CAPSULE_TIMEOUT, CAPSULE_WORKTREE_BASE_DIR.
func (c *Config) ApplyEnv() error {
	if v := os.Getenv("CAPSULE_PROVIDER"); v != "" {
		c.Runtime.Provider = v
	}
	if v := os.Getenv("CAPSULE_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("config: invalid CAPSULE_TIMEOUT %q: %w", v, err)
		}
		c.Runtime.Timeout = d
	}
	if v := os.Getenv("CAPSULE_WORKTREE_BASE_DIR"); v != "" {
		c.Worktree.BaseDir = v
	}
	return nil
}

// rawConfig mirrors Config but uses pointers to distinguish set vs unset fields.
type rawConfig struct {
	Runtime  *rawRuntime  `yaml:"runtime"`
	Worktree *rawWorktree `yaml:"worktree"`
	Pipeline *rawPipeline `yaml:"pipeline"`
	Campaign *rawCampaign `yaml:"campaign"`
}

type rawRuntime struct {
	Provider *string        `yaml:"provider"`
	Timeout  *time.Duration `yaml:"timeout"`
}

type rawWorktree struct {
	BaseDir *string `yaml:"base_dir"`
}

type rawPipeline struct {
	Phases     *string         `yaml:"phases"`
	Checkpoint *bool           `yaml:"checkpoint"`
	Retry      *rawRetryConfig `yaml:"retry"`
}

type rawRetryConfig struct {
	MaxAttempts      *int     `yaml:"max_attempts"`
	BackoffFactor    *float64 `yaml:"backoff_factor"`
	EscalateProvider *string  `yaml:"escalate_provider"`
	EscalateAfter    *int     `yaml:"escalate_after"`
}

type rawCampaign struct {
	FailureMode      *string `yaml:"failure_mode"`
	CircuitBreaker   *int    `yaml:"circuit_breaker"`
	DiscoveryFiling  *bool   `yaml:"discovery_filing"`
	CrossRunContext  *bool   `yaml:"cross_run_context"`
	ValidationPhases *string `yaml:"validation_phases"`
}

// loadLayer reads a single config file into a rawConfig for selective merging.
// Returns nil if the file does not exist. Rejects unknown fields.
func loadLayer(path string) (*rawConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("config: reading %s: %w", path, err)
	}

	if len(data) == 0 {
		return nil, nil
	}

	var raw rawConfig
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&raw); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, nil
		}
		return nil, fmt.Errorf("config: parsing %s: %w", path, err)
	}

	return &raw, nil
}

// merge applies non-nil fields from a rawConfig layer onto this Config.
func (c *Config) merge(layer *rawConfig) {
	if layer.Runtime != nil {
		if layer.Runtime.Provider != nil {
			c.Runtime.Provider = *layer.Runtime.Provider
		}
		if layer.Runtime.Timeout != nil {
			c.Runtime.Timeout = *layer.Runtime.Timeout
		}
	}
	if layer.Worktree != nil {
		if layer.Worktree.BaseDir != nil {
			c.Worktree.BaseDir = *layer.Worktree.BaseDir
		}
	}
	if layer.Pipeline != nil {
		if layer.Pipeline.Phases != nil {
			c.Pipeline.Phases = *layer.Pipeline.Phases
		}
		if layer.Pipeline.Checkpoint != nil {
			c.Pipeline.Checkpoint = *layer.Pipeline.Checkpoint
		}
		if layer.Pipeline.Retry != nil {
			if layer.Pipeline.Retry.MaxAttempts != nil {
				c.Pipeline.Retry.MaxAttempts = *layer.Pipeline.Retry.MaxAttempts
			}
			if layer.Pipeline.Retry.BackoffFactor != nil {
				c.Pipeline.Retry.BackoffFactor = *layer.Pipeline.Retry.BackoffFactor
			}
			if layer.Pipeline.Retry.EscalateProvider != nil {
				c.Pipeline.Retry.EscalateProvider = *layer.Pipeline.Retry.EscalateProvider
			}
			if layer.Pipeline.Retry.EscalateAfter != nil {
				c.Pipeline.Retry.EscalateAfter = *layer.Pipeline.Retry.EscalateAfter
			}
		}
	}
	if layer.Campaign != nil {
		if layer.Campaign.FailureMode != nil {
			c.Campaign.FailureMode = *layer.Campaign.FailureMode
		}
		if layer.Campaign.CircuitBreaker != nil {
			c.Campaign.CircuitBreaker = *layer.Campaign.CircuitBreaker
		}
		if layer.Campaign.DiscoveryFiling != nil {
			c.Campaign.DiscoveryFiling = *layer.Campaign.DiscoveryFiling
		}
		if layer.Campaign.CrossRunContext != nil {
			c.Campaign.CrossRunContext = *layer.Campaign.CrossRunContext
		}
		if layer.Campaign.ValidationPhases != nil {
			c.Campaign.ValidationPhases = *layer.Campaign.ValidationPhases
		}
	}
}
