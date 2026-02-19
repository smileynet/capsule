package provider

import "time"

// ClaudePreset returns the built-in CommandConfig for Claude Code.
func ClaudePreset() CommandConfig {
	return CommandConfig{
		Name:            "claude",
		Binary:          "claude",
		PromptFlag:      "-p",
		PermissionFlags: []string{"--dangerously-skip-permissions"},
	}
}

// KiroPreset returns the built-in CommandConfig for Kiro CLI.
func KiroPreset() CommandConfig {
	return CommandConfig{
		Name:            "kiro",
		Binary:          "kiro-cli",
		Subcommand:      "chat",
		PermissionFlags: []string{"--trust-all-tools"},
		ExtraFlags:      []string{"--no-interactive", "--wrap", "never"},
		StripANSI:       true,
	}
}

// RegisterBuiltins registers the built-in provider presets on the given registry.
func RegisterBuiltins(reg *Registry, timeout time.Duration) {
	reg.Register("claude", func() (Executor, error) {
		return NewGenericProvider(ClaudePreset(), WithTimeout(timeout)), nil
	})
	reg.Register("kiro", func() (Executor, error) {
		return NewGenericProvider(KiroPreset(), WithTimeout(timeout)), nil
	})
}
