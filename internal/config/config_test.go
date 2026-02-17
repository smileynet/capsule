package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	// Given no configuration loaded
	// When DefaultConfig is called
	cfg := DefaultConfig()

	// Then sensible defaults are returned
	if cfg.Runtime.Provider != "claude" {
		t.Errorf("default provider = %q, want %q", cfg.Runtime.Provider, "claude")
	}
	if cfg.Runtime.Timeout != 5*time.Minute {
		t.Errorf("default timeout = %v, want %v", cfg.Runtime.Timeout, 5*time.Minute)
	}
	if cfg.Worktree.BaseDir != ".capsule/worktrees" {
		t.Errorf("default base dir = %q, want %q", cfg.Worktree.BaseDir, ".capsule/worktrees")
	}
}

func TestLoad_ValidFile(t *testing.T) {
	// Given a config.yaml with provider, timeout, and base_dir set
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "capsule.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
runtime:
  provider: openai
  timeout: 10m
worktree:
  base_dir: /tmp/worktrees
`), 0o644); err != nil {
		t.Fatal(err)
	}

	// When config is loaded from the file
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Then Runtime.Provider, Runtime.Timeout, and Worktree.BaseDir are set
	if cfg.Runtime.Provider != "openai" {
		t.Errorf("provider = %q, want %q", cfg.Runtime.Provider, "openai")
	}
	if cfg.Runtime.Timeout != 10*time.Minute {
		t.Errorf("timeout = %v, want %v", cfg.Runtime.Timeout, 10*time.Minute)
	}
	if cfg.Worktree.BaseDir != "/tmp/worktrees" {
		t.Errorf("base dir = %q, want %q", cfg.Worktree.BaseDir, "/tmp/worktrees")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	// Given no config file exists
	// When Load is called with a nonexistent path
	cfg, err := Load("/nonexistent/capsule.yaml")
	if err != nil {
		t.Fatalf("Load() should return defaults for missing file, got error: %v", err)
	}

	// Then sensible defaults are used
	want := DefaultConfig()
	if *cfg != want {
		t.Errorf("Load(missing) = %+v, want defaults %+v", *cfg, want)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	// Given a config file with invalid YAML syntax
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "capsule.yaml")
	if err := os.WriteFile(cfgPath, []byte("{{invalid yaml"), 0o644); err != nil {
		t.Fatal(err)
	}

	// When Load is called
	_, err := Load(cfgPath)

	// Then an error is returned
	if err == nil {
		t.Fatal("Load(invalid YAML) should return error")
	}
}

func TestLoad_PartialConfig(t *testing.T) {
	// Given a config file that only sets provider (timeout and base_dir omitted)
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "capsule.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
runtime:
  provider: gemini
`), 0o644); err != nil {
		t.Fatal(err)
	}

	// When config is loaded
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Then provider is set and unset fields retain defaults
	if cfg.Runtime.Provider != "gemini" {
		t.Errorf("provider = %q, want %q", cfg.Runtime.Provider, "gemini")
	}
	if cfg.Runtime.Timeout != 5*time.Minute {
		t.Errorf("timeout = %v, want default %v", cfg.Runtime.Timeout, 5*time.Minute)
	}
	if cfg.Worktree.BaseDir != ".capsule/worktrees" {
		t.Errorf("base dir = %q, want default %q", cfg.Worktree.BaseDir, ".capsule/worktrees")
	}
}

func TestLoad_LayeredPriority(t *testing.T) {
	// Given a user config with provider+timeout and a project config overriding timeout
	userDir := t.TempDir()
	projectDir := t.TempDir()

	userCfg := filepath.Join(userDir, "capsule.yaml")
	if err := os.WriteFile(userCfg, []byte(`
runtime:
  provider: openai
  timeout: 2m
`), 0o644); err != nil {
		t.Fatal(err)
	}

	projectCfg := filepath.Join(projectDir, "capsule.yaml")
	if err := os.WriteFile(projectCfg, []byte(`
runtime:
  timeout: 8m
`), 0o644); err != nil {
		t.Fatal(err)
	}

	// When configs are loaded with layered priority (user < project)
	cfg, err := LoadLayered(userCfg, projectCfg)
	if err != nil {
		t.Fatalf("LoadLayered() error = %v", err)
	}

	// Then project overrides user, unset fields fall through
	if cfg.Runtime.Provider != "openai" {
		t.Errorf("provider = %q, want %q", cfg.Runtime.Provider, "openai")
	}
	// Timeout from project config (overrides user).
	if cfg.Runtime.Timeout != 8*time.Minute {
		t.Errorf("timeout = %v, want %v", cfg.Runtime.Timeout, 8*time.Minute)
	}
	// BaseDir retains default when neither layer sets it.
	if cfg.Worktree.BaseDir != ".capsule/worktrees" {
		t.Errorf("base dir = %q, want default %q", cfg.Worktree.BaseDir, ".capsule/worktrees")
	}
}

func TestApplyEnv(t *testing.T) {
	tests := []struct {
		name    string
		envs    map[string]string
		wantErr bool
		check   func(*testing.T, Config)
	}{
		{
			name: "CAPSULE_PROVIDER overrides provider",
			envs: map[string]string{"CAPSULE_PROVIDER": "gemini"},
			check: func(t *testing.T, c Config) {
				if c.Runtime.Provider != "gemini" {
					t.Errorf("provider = %q, want %q", c.Runtime.Provider, "gemini")
				}
			},
		},
		{
			name: "CAPSULE_TIMEOUT overrides timeout",
			envs: map[string]string{"CAPSULE_TIMEOUT": "30s"},
			check: func(t *testing.T, c Config) {
				if c.Runtime.Timeout != 30*time.Second {
					t.Errorf("timeout = %v, want %v", c.Runtime.Timeout, 30*time.Second)
				}
			},
		},
		{
			name: "CAPSULE_WORKTREE_BASE_DIR overrides base dir",
			envs: map[string]string{"CAPSULE_WORKTREE_BASE_DIR": "/custom/dir"},
			check: func(t *testing.T, c Config) {
				if c.Worktree.BaseDir != "/custom/dir" {
					t.Errorf("base dir = %q, want %q", c.Worktree.BaseDir, "/custom/dir")
				}
			},
		},
		{
			name:    "invalid CAPSULE_TIMEOUT returns error",
			envs:    map[string]string{"CAPSULE_TIMEOUT": "notaduration"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given a default config and environment variable per test case
			for k, v := range tt.envs {
				t.Setenv(k, v)
			}
			cfg := DefaultConfig()

			// When ApplyEnv is called
			err := cfg.ApplyEnv()

			// Then the expected override or error is observed
			if tt.wantErr {
				if err == nil {
					t.Fatal("ApplyEnv() should return error")
				}
				return
			}
			if err != nil {
				t.Fatalf("ApplyEnv() error = %v", err)
			}
			tt.check(t, cfg)
		})
	}
}

func TestLoad_UnknownField(t *testing.T) {
	// Given a config file with a typo ("provder" instead of "provider")
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "capsule.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
runtime:
  provder: openai
`), 0o644); err != nil {
		t.Fatal(err)
	}

	// When Load is called
	_, err := Load(cfgPath)

	// Then an error is returned (strict parsing rejects unknown fields)
	if err == nil {
		t.Fatal("Load() should return error for unknown field 'provder'")
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		{
			name:   "defaults are valid",
			modify: func(*Config) {},
		},
		{
			name:    "empty provider",
			modify:  func(c *Config) { c.Runtime.Provider = "" },
			wantErr: true,
		},
		{
			name:    "negative timeout",
			modify:  func(c *Config) { c.Runtime.Timeout = -1 * time.Second },
			wantErr: true,
		},
		{
			name:    "zero timeout",
			modify:  func(c *Config) { c.Runtime.Timeout = 0 },
			wantErr: true,
		},
		{
			name:    "empty base dir",
			modify:  func(c *Config) { c.Worktree.BaseDir = "" },
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given a config modified per test case
			cfg := DefaultConfig()
			tt.modify(&cfg)

			// When Validate is called
			err := cfg.Validate()

			// Then the expected validation result is returned
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoad_CommentOnlyFile(t *testing.T) {
	// Given a config file containing only YAML comments
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "capsule.yaml")
	if err := os.WriteFile(cfgPath, []byte("# just a comment\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// When Load is called
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load(comment-only) error = %v", err)
	}

	// Then defaults are returned (comment-only is treated as empty)
	want := DefaultConfig()
	if *cfg != want {
		t.Errorf("Load(comment-only) = %+v, want defaults %+v", *cfg, want)
	}
}

func TestLoadLayered_AllMissing(t *testing.T) {
	// Given no config files exist at any layer path
	// When LoadLayered is called with nonexistent paths
	cfg, err := LoadLayered("/no/user.yaml", "/no/project.yaml")
	if err != nil {
		t.Fatalf("LoadLayered(all missing) error = %v", err)
	}

	// Then defaults are returned
	want := DefaultConfig()
	if *cfg != want {
		t.Errorf("got %+v, want defaults %+v", *cfg, want)
	}
}

func TestDefaultConfig_PipelineDefaults(t *testing.T) {
	// Given no configuration loaded
	// When DefaultConfig is called
	cfg := DefaultConfig()

	// Then pipeline defaults are set
	if cfg.Pipeline.Phases != "default" {
		t.Errorf("pipeline.phases = %q, want %q", cfg.Pipeline.Phases, "default")
	}
	if cfg.Pipeline.Checkpoint {
		t.Error("pipeline.checkpoint should default to false")
	}
	if cfg.Pipeline.Retry.MaxAttempts != 3 {
		t.Errorf("pipeline.retry.max_attempts = %d, want 3", cfg.Pipeline.Retry.MaxAttempts)
	}
	if cfg.Pipeline.Retry.BackoffFactor != 1.0 {
		t.Errorf("pipeline.retry.backoff_factor = %v, want 1.0", cfg.Pipeline.Retry.BackoffFactor)
	}
}

func TestDefaultConfig_CampaignDefaults(t *testing.T) {
	// Given no configuration loaded
	// When DefaultConfig is called
	cfg := DefaultConfig()

	// Then campaign defaults are set
	if cfg.Campaign.FailureMode != "abort" {
		t.Errorf("campaign.failure_mode = %q, want %q", cfg.Campaign.FailureMode, "abort")
	}
	if cfg.Campaign.CircuitBreaker != 3 {
		t.Errorf("campaign.circuit_breaker = %d, want 3", cfg.Campaign.CircuitBreaker)
	}
	if cfg.Campaign.DiscoveryFiling {
		t.Error("campaign.discovery_filing should default to false")
	}
	if cfg.Campaign.CrossRunContext {
		t.Error("campaign.cross_run_context should default to false")
	}
}

func TestLoad_PipelineConfig(t *testing.T) {
	// Given a config file with pipeline settings
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "capsule.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
pipeline:
  phases: minimal
  checkpoint: true
  retry:
    max_attempts: 5
    backoff_factor: 1.5
    escalate_provider: openai
    escalate_after: 3
`), 0o644); err != nil {
		t.Fatal(err)
	}

	// When config is loaded
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Then pipeline settings are applied
	if cfg.Pipeline.Phases != "minimal" {
		t.Errorf("phases = %q, want %q", cfg.Pipeline.Phases, "minimal")
	}
	if !cfg.Pipeline.Checkpoint {
		t.Error("checkpoint should be true")
	}
	if cfg.Pipeline.Retry.MaxAttempts != 5 {
		t.Errorf("max_attempts = %d, want 5", cfg.Pipeline.Retry.MaxAttempts)
	}
	if cfg.Pipeline.Retry.BackoffFactor != 1.5 {
		t.Errorf("backoff_factor = %v, want 1.5", cfg.Pipeline.Retry.BackoffFactor)
	}
	if cfg.Pipeline.Retry.EscalateProvider != "openai" {
		t.Errorf("escalate_provider = %q, want %q", cfg.Pipeline.Retry.EscalateProvider, "openai")
	}
	if cfg.Pipeline.Retry.EscalateAfter != 3 {
		t.Errorf("escalate_after = %d, want 3", cfg.Pipeline.Retry.EscalateAfter)
	}
}

func TestLoad_CampaignConfig(t *testing.T) {
	// Given a config file with campaign settings
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "capsule.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
campaign:
  failure_mode: continue
  circuit_breaker: 5
  discovery_filing: true
  cross_run_context: true
  validation_phases: thorough
`), 0o644); err != nil {
		t.Fatal(err)
	}

	// When config is loaded
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	// Then campaign settings are applied
	if cfg.Campaign.FailureMode != "continue" {
		t.Errorf("failure_mode = %q, want %q", cfg.Campaign.FailureMode, "continue")
	}
	if cfg.Campaign.CircuitBreaker != 5 {
		t.Errorf("circuit_breaker = %d, want 5", cfg.Campaign.CircuitBreaker)
	}
	if !cfg.Campaign.DiscoveryFiling {
		t.Error("discovery_filing should be true")
	}
	if !cfg.Campaign.CrossRunContext {
		t.Error("cross_run_context should be true")
	}
	if cfg.Campaign.ValidationPhases != "thorough" {
		t.Errorf("validation_phases = %q, want %q", cfg.Campaign.ValidationPhases, "thorough")
	}
}

func TestLoadLayered_PipelineMerge(t *testing.T) {
	// Given user config sets pipeline phases, project overrides retry
	userDir := t.TempDir()
	projectDir := t.TempDir()

	userCfg := filepath.Join(userDir, "capsule.yaml")
	if err := os.WriteFile(userCfg, []byte(`
pipeline:
  phases: minimal
  retry:
    max_attempts: 2
`), 0o644); err != nil {
		t.Fatal(err)
	}

	projectCfg := filepath.Join(projectDir, "capsule.yaml")
	if err := os.WriteFile(projectCfg, []byte(`
pipeline:
  retry:
    max_attempts: 5
`), 0o644); err != nil {
		t.Fatal(err)
	}

	// When configs are loaded with layered priority
	cfg, err := LoadLayered(userCfg, projectCfg)
	if err != nil {
		t.Fatalf("LoadLayered() error = %v", err)
	}

	// Then project overrides user for retry, phases falls through from user
	if cfg.Pipeline.Phases != "minimal" {
		t.Errorf("phases = %q, want %q", cfg.Pipeline.Phases, "minimal")
	}
	if cfg.Pipeline.Retry.MaxAttempts != 5 {
		t.Errorf("max_attempts = %d, want 5", cfg.Pipeline.Retry.MaxAttempts)
	}
}

func TestValidate_PipelineFields(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*Config)
		wantErr bool
	}{
		{
			name:    "negative max_attempts",
			modify:  func(c *Config) { c.Pipeline.Retry.MaxAttempts = -1 },
			wantErr: true,
		},
		{
			name:    "negative backoff_factor",
			modify:  func(c *Config) { c.Pipeline.Retry.BackoffFactor = -1.0 },
			wantErr: true,
		},
		{
			name:    "backoff_factor between 0 and 1 is invalid",
			modify:  func(c *Config) { c.Pipeline.Retry.BackoffFactor = 0.5 },
			wantErr: true,
		},
		{
			name:   "backoff_factor 0 is valid (disabled)",
			modify: func(c *Config) { c.Pipeline.Retry.BackoffFactor = 0 },
		},
		{
			name:   "backoff_factor 1.0 is valid",
			modify: func(c *Config) { c.Pipeline.Retry.BackoffFactor = 1.0 },
		},
		{
			name:   "backoff_factor 2.0 is valid",
			modify: func(c *Config) { c.Pipeline.Retry.BackoffFactor = 2.0 },
		},
		{
			name:    "invalid failure_mode",
			modify:  func(c *Config) { c.Campaign.FailureMode = "invalid" },
			wantErr: true,
		},
		{
			name:    "negative circuit_breaker",
			modify:  func(c *Config) { c.Campaign.CircuitBreaker = -1 },
			wantErr: true,
		},
		{
			name:   "continue failure_mode is valid",
			modify: func(c *Config) { c.Campaign.FailureMode = "continue" },
		},
		{
			name:   "zero max_attempts is valid",
			modify: func(c *Config) { c.Pipeline.Retry.MaxAttempts = 0 },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modify(&cfg)
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	// Given an empty config file
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "capsule.yaml")
	if err := os.WriteFile(cfgPath, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	// When Load is called
	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load(empty) error = %v", err)
	}

	// Then defaults are returned
	want := DefaultConfig()
	if *cfg != want {
		t.Errorf("Load(empty) = %+v, want defaults %+v", *cfg, want)
	}
}
