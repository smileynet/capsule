//go:build smoke

package main

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
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

// TestSmoke_PipelinePause exercises the SIGUSR1 pause trigger end-to-end.
// It starts a capsule pipeline with a slow mock provider, sends SIGUSR1 mid-flight,
// and verifies that the process exits with code 3 and prints the pause message.
func TestSmoke_PipelinePause(t *testing.T) {
	// Prerequisites: node, bd, and git must be on PATH.
	for _, cmd := range []string{"node", "bd", "git"} {
		if _, err := exec.LookPath(cmd); err != nil {
			t.Skipf("skipping: %s not on PATH", cmd)
		}
	}

	projectRoot := findProjectRoot(t)
	binary := filepath.Join(projectRoot, "capsule")

	// Build binary if not already present.
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

	projectDir := setupGreenfieldProject(t, projectRoot)
	stateDir := t.TempDir()
	mockDir := createSlowMockClaude(t, stateDir)

	// Start capsule in background.
	cmd := exec.Command(binary, "run", "demo-001.1.1", "--timeout", "60")
	cmd.Dir = projectDir
	cmd.Env = append(os.Environ(), "PATH="+mockDir+":"+os.Getenv("PATH"))

	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	cmd.Stdout = outW
	cmd.Stderr = outW

	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start capsule: %v", err)
	}
	outW.Close() // Close write end in parent so reads see EOF when child exits.

	// Read output in background.
	outputCh := make(chan string, 1)
	go func() {
		data, _ := io.ReadAll(outR)
		outputCh <- string(data)
	}()

	// Wait for the first phase to complete (mock writes sentinel after output).
	donePath := filepath.Join(stateDir, "phase_done")
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(donePath); err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if _, err := os.Stat(donePath); err != nil {
		_ = cmd.Process.Kill()
		t.Fatalf("timed out waiting for first phase to complete")
	}

	// Send SIGUSR1 — the atomic bool is set and checked before the next phase.
	if err := cmd.Process.Signal(syscall.SIGUSR1); err != nil {
		t.Fatalf("failed to send SIGUSR1: %v", err)
	}

	// Wait for capsule to exit with a timeout.
	waitDone := make(chan error, 1)
	go func() { waitDone <- cmd.Wait() }()
	select {
	case err = <-waitDone:
	case <-time.After(30 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("capsule did not exit within 30s after SIGUSR1")
	}

	output := <-outputCh
	t.Log("--- capsule output ---\n" + output)

	// Assert exit code 3 (paused).
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected ExitError, got: %v\nOutput:\n%s", err, output)
	}
	if exitErr.ExitCode() != 3 {
		t.Fatalf("exit code = %d, want 3\nOutput:\n%s", exitErr.ExitCode(), output)
	}

	// Assert pause message.
	if !strings.Contains(output, "Pipeline paused") {
		t.Errorf("output missing 'Pipeline paused':\n%s", output)
	}
	if !strings.Contains(output, "capsule run demo-001.1.1") {
		t.Errorf("output missing resume hint:\n%s", output)
	}
}

// createSlowMockClaude writes a mock claude that sleeps 2s per invocation,
// giving the test time to send SIGUSR1 between phases.
// It writes a sentinel file on the first invocation so the test knows a phase has started.
func createSlowMockClaude(t *testing.T, stateDir string) string {
	t.Helper()
	mockDir := t.TempDir()
	mockPath := filepath.Join(mockDir, "claude")

	script := `#!/usr/bin/env bash
set -euo pipefail

STATE_DIR="` + stateDir + `"

# Parse args: extract -p <prompt>.
PROMPT=""
while [ $# -gt 0 ]; do
    case "$1" in
        -p) PROMPT="$2"; shift 2 ;;
        --dangerously-skip-permissions) shift ;;
        *) shift ;;
    esac
done

FIRST_LINE=$(printf '%s\n' "$PROMPT" | head -1)

# Track call count so sentinel is written only once (after first phase completes).
COUNTER="$STATE_DIR/call_count"
COUNT=$(cat "$COUNTER" 2>/dev/null || echo 0)
COUNT=$((COUNT + 1))
echo "$COUNT" > "$COUNTER"

# Signal first phase completion on exit (for test synchronization).
signal_done() { if [ "$COUNT" -eq 1 ]; then touch "$STATE_DIR/phase_done"; fi; }
trap signal_done EXIT

# Sleep to give the test time to send SIGUSR1 between phases.
sleep 2

# Test-writer phase: create test file.
if printf '%s\n' "$FIRST_LINE" | grep -qi 'test-writer'; then
    mkdir -p src
    cat > src/todo.test.js << 'TESTEOF'
const assert = require('assert');
const { TodoApp } = require('./todo.js');
const app = new TodoApp();
const todo = app.addTodo('Buy milk');
assert.strictEqual(todo.text, 'Buy milk');
console.log('PASS');
TESTEOF
    git add src/todo.test.js
    git diff --cached --quiet || git commit -q -m "Add tests"
    printf '{"status":"PASS","feedback":"Tests created","files_changed":["src/todo.test.js"],"summary":"tests"}\n'
    exit 0
fi

# Execute phase: create implementation.
if printf '%s\n' "$FIRST_LINE" | grep -qi '^# Execute Phase'; then
    mkdir -p src
    cat > src/todo.js << 'IMPLEOF'
class TodoApp {
  constructor() { this.todos = []; }
  addTodo(text) {
    const todo = { id: '1', text, completed: false, createdAt: new Date().toISOString() };
    this.todos.push(todo);
    return todo;
  }
}
module.exports = { TodoApp };
IMPLEOF
    git add src/todo.js
    git diff --cached --quiet || git commit -q -m "Implement TodoApp"
    printf '{"status":"PASS","feedback":"Done","files_changed":["src/todo.js"],"summary":"impl"}\n'
    exit 0
fi

# Merge phase.
if printf '%s\n' "$FIRST_LINE" | grep -qi 'merge'; then
    git add src/ 2>/dev/null || true
    git diff --cached --quiet || git commit -q -m "merge"
    HASH=$(git rev-parse --short HEAD)
    printf '{"status":"PASS","feedback":"Merged","files_changed":[],"summary":"merge","commit_hash":"%s"}\n' "$HASH"
    exit 0
fi

# All other phases: just pass.
printf '{"status":"PASS","feedback":"OK","files_changed":[],"summary":"passed"}\n'
`
	if err := os.WriteFile(mockPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write slow mock claude: %v", err)
	}
	return mockDir
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
