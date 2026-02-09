//go:build smoke

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestSmoke_GoProjectSkeleton exercises the built binary end-to-end,
// validating all feature scenarios from f-2.1-go-project-skeleton.feature.
//
// Subtests run sequentially and depend on the first subtest building the binary.
// The "make test" subtest invokes go test ./... which is safe because smoke tests
// are behind a build tag and won't be re-invoked.
func TestSmoke_GoProjectSkeleton(t *testing.T) {
	// Find project root (where go.mod lives)
	projectRoot := findProjectRoot(t)
	binary := filepath.Join(projectRoot, "capsule")
	t.Cleanup(func() { os.Remove(binary) })

	t.Run("go build produces a capsule binary", func(t *testing.T) {
		// Given: the project
		// When: go build runs
		cmd := exec.Command("go", "build",
			"-ldflags", "-X main.version=smoke-test -X main.commit=abc1234 -X main.date=2026-01-01",
			"-o", binary, "./cmd/capsule")
		cmd.Dir = projectRoot
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("go build failed: %v\n%s", err, out)
		}

		// Then: a capsule binary is produced
		info, err := os.Stat(binary)
		if err != nil {
			t.Fatalf("binary not found: %v", err)
		}
		if info.Size() == 0 {
			t.Fatal("binary is empty")
		}
	})

	t.Run("capsule version prints version commit and date", func(t *testing.T) {
		// Given: the binary
		if _, err := os.Stat(binary); err != nil {
			t.Fatal("binary not available -- the build subtest must run first and succeed")
		}

		// When: capsule --version runs
		cmd := exec.Command(binary, "--version")
		out, err := cmd.CombinedOutput()
		output := string(out)

		// Then: version, commit, and date are printed
		if err != nil {
			// Kong may exit non-zero on --version in some configurations
			if !strings.Contains(output, "smoke-test") {
				t.Fatalf("--version failed: %v\n%s", err, output)
			}
		}
		for _, want := range []string{"smoke-test", "abc1234", "2026-01-01"} {
			if !strings.Contains(output, want) {
				t.Errorf("version output = %q, want to contain %q", output, want)
			}
		}
	})

	t.Run("capsule without args exits non-zero with usage", func(t *testing.T) {
		// Given: the binary
		if _, err := os.Stat(binary); err != nil {
			t.Fatal("binary not available -- the build subtest must run first and succeed")
		}

		// When: capsule runs without arguments
		cmd := exec.Command(binary)
		out, err := cmd.CombinedOutput()

		// Then: exit code is non-zero
		if err == nil {
			t.Fatal("expected non-zero exit code when no command provided")
		}

		// And: usage or error message is printed
		output := string(out)
		if !strings.Contains(output, "run") && !strings.Contains(output, "expected") {
			t.Errorf("expected usage or error output, got: %q", output)
		}
	})

	t.Run("make test-full passes", func(t *testing.T) {
		// Given: the Makefile
		makefile := filepath.Join(projectRoot, "Makefile")
		if _, err := os.Stat(makefile); err != nil {
			t.Fatalf("Makefile not found: %v", err)
		}

		// When: make test-full runs (all tests, no -short)
		cmd := exec.Command("make", "test-full")
		cmd.Dir = projectRoot
		out, err := cmd.CombinedOutput()

		// Then: all tests pass
		if err != nil {
			t.Fatalf("make test-full failed: %v\n%s", err, out)
		}
	})
}

// findProjectRoot walks up from the test file to find the directory containing go.mod.
func findProjectRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find project root (go.mod)")
		}
		dir = parent
	}
}
