package provider

import "time"

// ClaudePreset is the built-in CommandConfig for Claude Code.
var ClaudePreset = CommandConfig{
	Name:            "claude",
	Binary:          "claude",
	PromptFlag:      "-p",
	PermissionFlags: []string{"--dangerously-skip-permissions"},
}

// KiroPreset is the built-in CommandConfig for Kiro CLI.
var KiroPreset = CommandConfig{
	Name:            "kiro",
	Binary:          "kiro-cli",
	Subcommand:      "chat",
	PermissionFlags: []string{"--trust-all-tools"},
	ExtraFlags:      []string{"--no-interactive", "--wrap", "never"},
	StripANSI:       true,
}

// RegisterBuiltins registers the built-in provider presets on the given registry.
func RegisterBuiltins(reg *Registry, timeout time.Duration) {
	reg.Register("claude", func() (Executor, error) {
		return NewGenericProvider(ClaudePreset, WithTimeout(timeout)), nil
	})
	reg.Register("kiro", func() (Executor, error) {
		return NewGenericProvider(KiroPreset, WithTimeout(timeout)), nil
	})
}
