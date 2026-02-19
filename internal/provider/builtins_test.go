package provider

import (
	"testing"
	"time"
)

func TestRegisterBuiltins(t *testing.T) {
	// Given an empty registry
	reg := NewRegistry()

	// When RegisterBuiltins is called
	RegisterBuiltins(reg, 5*time.Minute)

	// Then both claude and kiro are available
	available := reg.AvailableProviders()
	if len(available) != 2 {
		t.Fatalf("AvailableProviders() len = %d, want 2", len(available))
	}

	// And both create valid providers with correct names
	for _, name := range []string{"claude", "kiro"} {
		p, err := reg.NewProvider(name)
		if err != nil {
			t.Fatalf("NewProvider(%q) error: %v", name, err)
		}
		if p.Name() != name {
			t.Errorf("NewProvider(%q).Name() = %q", name, p.Name())
		}
	}
}
