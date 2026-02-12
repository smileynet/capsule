package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// Executor is the minimal interface a registered provider must satisfy.
type Executor interface {
	Name() string
	Execute(ctx context.Context, prompt, workDir string) (Result, error)
}

// Factory creates a provider instance.
type Factory func() (Executor, error)

// Registry maps provider names to factory functions.
// It is not safe for concurrent use; registration should happen at startup.
type Registry struct {
	factories map[string]Factory
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{factories: make(map[string]Factory)}
}

// Register adds a named provider factory. Overwrites if name already exists.
// Panics if name is empty or f is nil (programmer error).
func (r *Registry) Register(name string, f Factory) {
	if name == "" {
		panic("provider: Register called with empty name")
	}
	if f == nil {
		panic("provider: Register called with nil factory")
	}
	r.factories[name] = f
}

// NewProvider instantiates a provider by name.
// Returns an error if the name is not registered or the factory fails.
func (r *Registry) NewProvider(name string) (Executor, error) {
	f, ok := r.factories[name]
	if !ok {
		return nil, &UnknownProviderError{
			Name:      name,
			Available: r.AvailableProviders(),
		}
	}
	p, err := f()
	if err != nil {
		return nil, fmt.Errorf("provider factory %q: %w", name, err)
	}
	return p, nil
}

// AvailableProviders returns registered provider names in sorted order.
func (r *Registry) AvailableProviders() []string {
	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// UnknownProviderError indicates a provider name is not registered.
type UnknownProviderError struct {
	Name      string
	Available []string
}

func (e *UnknownProviderError) Error() string {
	return fmt.Sprintf("unknown provider %q (available: %s)", e.Name, strings.Join(e.Available, ", "))
}
