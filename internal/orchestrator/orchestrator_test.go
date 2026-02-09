package orchestrator

import (
	"testing"

	"github.com/smileynet/capsule/internal/provider"
)

// Compile-time check: MockProvider satisfies the orchestrator's Provider interface.
var _ Provider = (*provider.MockProvider)(nil)

func TestPackageCompiles(t *testing.T) {
	// Placeholder: verifies the package compiles and is testable.
}
