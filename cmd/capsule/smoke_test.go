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

// TestSmoke_OrchestratorWiring exercises the run/abort/clean commands at the binary level,
// validating exit code mapping and error formatting from cap-9qv.5.
func TestSmoke_OrchestratorWiring(t *testing.T) {
	projectRoot := findProjectRoot(t)
	binary := filepath.Join(projectRoot, "capsule")

	// Ensure binary exists (built by GoProjectSkeleton suite or build here).
	if _, err := os.Stat(binary); err != nil {
		cmd := exec.Command("go", "build",
			"-ldflags", "-X main.version=smoke-test -X main.commit=abc1234 -X main.date=2026-01-01",
			"-o", binary, "./cmd/capsule")
		cmd.Dir = projectRoot
		out, buildErr := cmd.CombinedOutput()
		if buildErr != nil {
			t.Fatalf("go build failed: %v\n%s", buildErr, out)
		}
		t.Cleanup(func() { os.Remove(binary) })
	}

	t.Run("capsule run without bead-id exits non-zero", func(t *testing.T) {
		// Given the binary
		// When run is invoked without a required bead-id argument
		cmd := exec.Command(binary, "run")
		out, err := cmd.CombinedOutput()

		// Then it exits non-zero
		if err == nil {
			t.Fatal("expected non-zero exit code when bead-id missing")
		}
		// And the error mentions the missing argument
		output := string(out)
		if !strings.Contains(output, "bead-id") && !strings.Contains(output, "expected") {
			t.Errorf("expected error about missing bead-id, got: %q", output)
		}
	})

	t.Run("capsule run with unknown provider exits with setup error", func(t *testing.T) {
		// Given the binary
		// When run is invoked with an unregistered provider
		cmd := exec.Command(binary, "run", "test-bead", "--provider", "nonexistent")
		cmd.Dir = projectRoot
		out, err := cmd.CombinedOutput()

		// Then it exits non-zero
		if err == nil {
			t.Fatal("expected non-zero exit code for unknown provider")
		}
		// And the error mentions the unknown provider
		output := string(out)
		if !strings.Contains(output, "unknown provider") && !strings.Contains(output, "nonexistent") {
			t.Errorf("expected error about unknown provider, got: %q", output)
		}
		// And exit code is 2 (setup error, not pipeline error)
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() != 2 {
				t.Errorf("exit code = %d, want 2 (setup error)", exitErr.ExitCode())
			}
		}
	})

	t.Run("capsule abort with nonexistent worktree exits with setup error", func(t *testing.T) {
		// Given the binary
		// When abort is invoked for a nonexistent worktree
		cmd := exec.Command(binary, "abort", "nonexistent-bead")
		cmd.Dir = projectRoot
		out, err := cmd.CombinedOutput()

		// Then it exits non-zero
		if err == nil {
			t.Fatal("expected non-zero exit code for nonexistent worktree")
		}
		// And the error mentions no worktree found
		output := string(out)
		if !strings.Contains(output, "no worktree found") {
			t.Errorf("expected error about missing worktree, got: %q", output)
		}
		// And exit code is 2 (setup error)
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() != 2 {
				t.Errorf("exit code = %d, want 2 (setup error)", exitErr.ExitCode())
			}
		}
	})

	t.Run("capsule run with --no-tui flag is accepted", func(t *testing.T) {
		// Given the binary
		// When run is invoked with --no-tui and an unknown provider (to trigger a clean setup error)
		cmd := exec.Command(binary, "run", "test-bead", "--no-tui", "--provider", "nonexistent")
		cmd.Dir = projectRoot
		out, err := cmd.CombinedOutput()

		// Then it exits non-zero
		if err == nil {
			t.Fatal("expected non-zero exit code for unknown provider")
		}
		// And --no-tui is accepted (no parse error about the flag)
		output := string(out)
		if strings.Contains(output, "unknown flag") || strings.Contains(output, "--no-tui") {
			t.Errorf("--no-tui should be accepted, but got parse error: %q", output)
		}
		// And the error mentions the unknown provider
		if !strings.Contains(output, "unknown provider") && !strings.Contains(output, "nonexistent") {
			t.Errorf("expected error about unknown provider, got: %q", output)
		}
		// And exit code is 2 (setup error, not parse error)
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() != 2 {
				t.Errorf("exit code = %d, want 2 (setup error)", exitErr.ExitCode())
			}
		}
	})

	t.Run("capsule clean with nonexistent worktree exits with setup error", func(t *testing.T) {
		// Given the binary
		// When clean is invoked for a nonexistent worktree
		cmd := exec.Command(binary, "clean", "nonexistent-bead")
		cmd.Dir = projectRoot
		out, err := cmd.CombinedOutput()

		// Then it exits non-zero
		if err == nil {
			t.Fatal("expected non-zero exit code for nonexistent worktree")
		}
		// And the error mentions no worktree found
		output := string(out)
		if !strings.Contains(output, "no worktree found") {
			t.Errorf("expected error about missing worktree, got: %q", output)
		}
		// And exit code is 2 (setup error)
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() != 2 {
				t.Errorf("exit code = %d, want 2 (setup error)", exitErr.ExitCode())
			}
		}
	})
}

// TestSmoke_DashboardTTY exercises the dashboard command at the binary level,
// validating TTY detection from cap-kxw.
func TestSmoke_DashboardTTY(t *testing.T) {
	projectRoot := findProjectRoot(t)
	binary := filepath.Join(projectRoot, "capsule")

	// Ensure binary exists.
	if _, err := os.Stat(binary); err != nil {
		cmd := exec.Command("go", "build",
			"-ldflags", "-X main.version=smoke-test -X main.commit=abc1234 -X main.date=2026-01-01",
			"-o", binary, "./cmd/capsule")
		cmd.Dir = projectRoot
		out, buildErr := cmd.CombinedOutput()
		if buildErr != nil {
			t.Fatalf("go build failed: %v\n%s", buildErr, out)
		}
		t.Cleanup(func() { os.Remove(binary) })
	}

	t.Run("capsule dashboard without TTY exits with error", func(t *testing.T) {
		// Given: the binary running without a TTY (test subprocess has no terminal)
		// When: capsule dashboard is invoked
		cmd := exec.Command(binary, "dashboard")
		cmd.Dir = projectRoot
		out, err := cmd.CombinedOutput()

		// Then: it exits non-zero
		if err == nil {
			t.Fatal("expected non-zero exit code without TTY")
		}

		// And: the error mentions TTY requirement
		output := string(out)
		if !strings.Contains(output, "terminal") && !strings.Contains(output, "TTY") {
			t.Errorf("expected error about TTY requirement, got: %q", output)
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
