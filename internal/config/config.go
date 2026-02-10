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
}

type rawRuntime struct {
	Provider *string        `yaml:"provider"`
	Timeout  *time.Duration `yaml:"timeout"`
}

type rawWorktree struct {
	BaseDir *string `yaml:"base_dir"`
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
}
