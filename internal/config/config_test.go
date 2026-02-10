package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

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

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
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
	cfg, err := Load("/nonexistent/capsule.yaml")
	if err != nil {
		t.Fatalf("Load() should return defaults for missing file, got error: %v", err)
	}
	want := DefaultConfig()
	if *cfg != want {
		t.Errorf("Load(missing) = %+v, want defaults %+v", *cfg, want)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "capsule.yaml")
	if err := os.WriteFile(cfgPath, []byte("{{invalid yaml"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatal("Load(invalid YAML) should return error")
	}
}

func TestLoad_PartialConfig(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "capsule.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
runtime:
  provider: gemini
`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Runtime.Provider != "gemini" {
		t.Errorf("provider = %q, want %q", cfg.Runtime.Provider, "gemini")
	}
	// Unset fields should retain defaults.
	if cfg.Runtime.Timeout != 5*time.Minute {
		t.Errorf("timeout = %v, want default %v", cfg.Runtime.Timeout, 5*time.Minute)
	}
	if cfg.Worktree.BaseDir != ".capsule/worktrees" {
		t.Errorf("base dir = %q, want default %q", cfg.Worktree.BaseDir, ".capsule/worktrees")
	}
}

func TestLoad_LayeredPriority(t *testing.T) {
	// Setup: user config sets provider, project config overrides timeout.
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

	cfg, err := LoadLayered(userCfg, projectCfg)
	if err != nil {
		t.Fatalf("LoadLayered() error = %v", err)
	}
	// Provider from user config (project doesn't set it).
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
			for k, v := range tt.envs {
				t.Setenv(k, v)
			}
			cfg := DefaultConfig()
			err := cfg.ApplyEnv()

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
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "capsule.yaml")
	if err := os.WriteFile(cfgPath, []byte(`
runtime:
  provder: openai
`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(cfgPath)
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
			cfg := DefaultConfig()
			tt.modify(&cfg)
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoad_CommentOnlyFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "capsule.yaml")
	if err := os.WriteFile(cfgPath, []byte("# just a comment\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load(comment-only) error = %v", err)
	}
	want := DefaultConfig()
	if *cfg != want {
		t.Errorf("Load(comment-only) = %+v, want defaults %+v", *cfg, want)
	}
}

func TestLoadLayered_AllMissing(t *testing.T) {
	cfg, err := LoadLayered("/no/user.yaml", "/no/project.yaml")
	if err != nil {
		t.Fatalf("LoadLayered(all missing) error = %v", err)
	}
	want := DefaultConfig()
	if *cfg != want {
		t.Errorf("got %+v, want defaults %+v", *cfg, want)
	}
}

func TestLoad_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "capsule.yaml")
	if err := os.WriteFile(cfgPath, []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("Load(empty) error = %v", err)
	}
	want := DefaultConfig()
	if *cfg != want {
		t.Errorf("Load(empty) = %+v, want defaults %+v", *cfg, want)
	}
}
