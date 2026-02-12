package provider

import (
	"errors"
	"sort"
	"testing"
)

func TestRegistry(t *testing.T) {
	t.Run("register and create provider", func(t *testing.T) {
		// Given a registry with a "mock" factory registered
		r := NewRegistry()
		r.Register("mock", func() (Executor, error) {
			return &MockProvider{NameVal: "mock"}, nil
		})

		// When NewProvider is called with "mock"
		p, err := r.NewProvider("mock")

		// Then a provider with the correct name is returned
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.Name() != "mock" {
			t.Errorf("Name() = %q, want %q", p.Name(), "mock")
		}
	})

	t.Run("unknown provider returns UnknownProviderError", func(t *testing.T) {
		// Given a registry with only "claude" registered
		r := NewRegistry()
		r.Register("claude", func() (Executor, error) {
			return &MockProvider{NameVal: "claude"}, nil
		})

		// When NewProvider is called with "nonexistent"
		_, err := r.NewProvider("nonexistent")

		// Then an UnknownProviderError is returned with available providers
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		var upe *UnknownProviderError
		if !errors.As(err, &upe) {
			t.Fatalf("expected *UnknownProviderError, got %T", err)
		}
		if upe.Name != "nonexistent" {
			t.Errorf("Name = %q, want %q", upe.Name, "nonexistent")
		}
		if len(upe.Available) != 1 || upe.Available[0] != "claude" {
			t.Errorf("Available = %v, want [claude]", upe.Available)
		}
	})

	t.Run("available providers returns sorted names", func(t *testing.T) {
		// Given a registry with "zebra" and "alpha" registered
		r := NewRegistry()
		r.Register("zebra", func() (Executor, error) {
			return &MockProvider{NameVal: "zebra"}, nil
		})
		r.Register("alpha", func() (Executor, error) {
			return &MockProvider{NameVal: "alpha"}, nil
		})

		// When AvailableProviders is called
		got := r.AvailableProviders()

		// Then names are returned in sorted order
		want := []string{"alpha", "zebra"}
		if len(got) != len(want) {
			t.Fatalf("len = %d, want %d", len(got), len(want))
		}
		if !sort.StringsAreSorted(got) {
			t.Errorf("not sorted: %v", got)
		}
		for i, name := range got {
			if name != want[i] {
				t.Errorf("AvailableProviders()[%d] = %q, want %q", i, name, want[i])
			}
		}
	})

	t.Run("duplicate registration overwrites", func(t *testing.T) {
		// Given a registry where "mock" is registered twice
		r := NewRegistry()
		r.Register("mock", func() (Executor, error) {
			return &MockProvider{NameVal: "first"}, nil
		})
		r.Register("mock", func() (Executor, error) {
			return &MockProvider{NameVal: "second"}, nil
		})

		// When NewProvider is called
		p, err := r.NewProvider("mock")

		// Then the second registration wins
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if p.Name() != "second" {
			t.Errorf("Name() = %q, want %q (second factory should win)", p.Name(), "second")
		}
	})

	t.Run("factory error propagated", func(t *testing.T) {
		// Given a registry with a factory that returns an error
		r := NewRegistry()
		factoryErr := errors.New("config missing")
		r.Register("broken", func() (Executor, error) {
			return nil, factoryErr
		})

		// When NewProvider is called
		_, err := r.NewProvider("broken")

		// Then the factory error is propagated
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, factoryErr) {
			t.Errorf("expected wrapped factoryErr, got %v", err)
		}
	})

	t.Run("empty registry returns empty slice", func(t *testing.T) {
		// Given an empty registry
		r := NewRegistry()

		// When AvailableProviders is called
		got := r.AvailableProviders()

		// Then a non-nil empty slice is returned
		if got == nil {
			t.Fatal("AvailableProviders() returned nil, want empty slice")
		}
		if len(got) != 0 {
			t.Errorf("len = %d, want 0", len(got))
		}
	})

	t.Run("factory creates fresh instance each call", func(t *testing.T) {
		// Given a registry with a factory that tracks call count
		r := NewRegistry()
		callCount := 0
		r.Register("mock", func() (Executor, error) {
			callCount++
			return &MockProvider{NameVal: "mock"}, nil
		})

		// When NewProvider is called twice
		p1, err := r.NewProvider("mock")
		if err != nil {
			t.Fatalf("unexpected error creating p1: %v", err)
		}
		p2, err := r.NewProvider("mock")
		if err != nil {
			t.Fatalf("unexpected error creating p2: %v", err)
		}
		// Then the factory is called twice and returns different instances
		if callCount != 2 {
			t.Errorf("factory called %d times, want 2", callCount)
		}
		if p1 == p2 {
			t.Error("expected different instances, got same pointer")
		}
	})
}

func TestUnknownProviderError(t *testing.T) {
	// Given an UnknownProviderError with name and available providers
	err := &UnknownProviderError{
		Name:      "gemini",
		Available: []string{"claude", "mock"},
	}

	// When Error() is called
	// Then it returns a formatted message with the unknown name and available options
	want := `unknown provider "gemini" (available: claude, mock)`
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
}

func TestRegisterPanics(t *testing.T) {
	t.Run("empty name panics", func(t *testing.T) {
		// Given a new registry
		// When Register is called with an empty name
		// Then it panics
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic for empty name, got none")
			}
		}()
		r := NewRegistry()
		r.Register("", func() (Executor, error) {
			return &MockProvider{NameVal: "x"}, nil
		})
	})

	t.Run("nil factory panics", func(t *testing.T) {
		// Given a new registry
		// When Register is called with a nil factory
		// Then it panics
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic for nil factory, got none")
			}
		}()
		r := NewRegistry()
		r.Register("mock", nil)
	})
}
