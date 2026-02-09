// Package orchestrator sequences pipeline phases with retry logic.
package orchestrator

import (
	"context"

	"github.com/smileynet/capsule/internal/provider"
)

// Provider executes AI completions against a configured backend.
// Defined here (the consumer) per Go convention: accept interfaces, return structs.
type Provider interface {
	// Name returns the provider identifier (e.g. "claude").
	Name() string
	// Execute runs a prompt in the given working directory and returns the raw result.
	Execute(ctx context.Context, prompt, workDir string) (provider.Result, error)
}
