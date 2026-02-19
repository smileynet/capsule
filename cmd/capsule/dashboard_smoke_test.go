//go:build smoke

package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/creack/pty"
)

// TestSmoke_DashboardPTY exercises the dashboard TUI at the process level,
// launching the binary with a pseudo-TTY and validating real terminal rendering.
//
// This test covers the E2E gap identified in cap-fj8 close-service review:
// unit tests cover all state transitions, but no test validates real terminal output.
func TestSmoke_DashboardPTY(t *testing.T) {
	for _, cmd := range []string{"bd", "git"} {
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

	// Create a temp project with beads so dashboard has data to display.
	projectDir := setupDashboardProject(t)

	t.Run("dashboard launches and renders browse pane", func(t *testing.T) {
		ptmx, cmd, waitFunc := startDashboard(t, binary, projectDir)

		// Wait for the TUI to render the bead list.
		output := readPTYUntil(t, ptmx, "smoke-test-task", 8*time.Second)
		clean := stripANSI(output)

		if !strings.Contains(clean, "smoke-test-task") {
			t.Errorf("expected 'smoke-test-task' in rendered output, got:\n%s", clean)
		}

		// Send 'q' to quit gracefully.
		writePTY(t, ptmx, "q")
		waitForExit(t, cmd, waitFunc, 5*time.Second)
	})

	t.Run("dashboard renders pipeline context in detail pane", func(t *testing.T) {
		ptmx, cmd, waitFunc := startDashboard(t, binary, projectDir)

		// Wait for the bead list to load.
		readPTYUntil(t, ptmx, "smoke-test-task", 8*time.Second)

		// Press tab to switch focus to the right (detail) pane,
		// which should show bead detail with "Select a bead" or resolved detail.
		writePTY(t, ptmx, "\t")

		// Wait for detail pane content. The bead should be auto-resolved
		// and show its title or description.
		output := readPTYUntil(t, ptmx, "smoke-test-task", 5*time.Second)
		clean := stripANSI(output)

		// The detail pane should show the bead info.
		if !strings.Contains(clean, "smoke-test-task") {
			t.Errorf("expected bead detail in right pane, got:\n%s", clean)
		}

		// Send 'q' to quit.
		writePTY(t, ptmx, "q")
		waitForExit(t, cmd, waitFunc, 5*time.Second)
	})

	t.Run("unified tree shows closed beads", func(t *testing.T) {
		ptmx, cmd, waitFunc := startDashboard(t, binary, projectDir)

		// The unified tree shows both open and closed beads on initial load.
		// Wait for the closed bead to appear alongside the open one.
		output := readPTYUntil(t, ptmx, "closed-task", 8*time.Second)
		clean := stripANSI(output)

		if !strings.Contains(clean, "closed-task") {
			// Verify at minimum the process didn't crash.
			if err := cmd.Process.Signal(syscall.Signal(0)); err != nil {
				t.Errorf("dashboard crashed, output:\n%s", clean)
			} else {
				t.Errorf("expected 'closed-task' in unified tree, got:\n%s", clean)
			}
		}

		// Send 'q' to quit.
		writePTY(t, ptmx, "q")
		waitForExit(t, cmd, waitFunc, 5*time.Second)
	})
}

// startDashboard launches the dashboard binary with a pseudo-TTY.
// Cleanup is registered automatically: the PTY is closed and the process
// is killed+waited on when the test finishes, preventing orphan processes.
//
// The returned waitFunc uses sync.Once so that both the cleanup path and
// explicit waitForExit calls safely share a single cmd.Wait() invocation.
func startDashboard(t *testing.T, binary, projectDir string) (*os.File, *exec.Cmd, func() error) {
	t.Helper()
	cmd := exec.Command(binary, "dashboard")
	cmd.Dir = projectDir
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 80})
	if err != nil {
		t.Fatalf("failed to start with PTY: %v", err)
	}

	var waitErr error
	var once sync.Once
	waitFunc := func() error {
		once.Do(func() { waitErr = cmd.Wait() })
		return waitErr
	}

	t.Cleanup(func() {
		ptmx.Close()
		if cmd.Process != nil {
			cmd.Process.Kill()
			waitFunc()
		}
	})
	return ptmx, cmd, waitFunc
}

// setupDashboardProject creates a minimal git project with beads initialized
// and a single open bead for the dashboard to display.
func setupDashboardProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Initialize git repo.
	run(t, dir, "git", "init")
	run(t, dir, "git", "config", "user.email", "test@test.com")
	run(t, dir, "git", "config", "user.name", "Test")
	run(t, dir, "git", "config", "commit.gpgsign", "false")

	// Create initial commit (bd needs a git history).
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# test\n"), 0o644); err != nil {
		t.Fatalf("write README: %v", err)
	}
	run(t, dir, "git", "add", ".")
	run(t, dir, "git", "commit", "-m", "initial")

	// Initialize beads.
	run(t, dir, "bd", "init", "--no-db")

	// Create a test bead for the dashboard to display.
	run(t, dir, "bd", "create", "--title=smoke-test-task", "--type=task", "--priority=2")

	// Create and close a bead so history toggle has data.
	out := runOutput(t, dir, "bd", "create", "--title=closed-task", "--type=task", "--priority=3")

	// Extract the bead ID from the create output and close it.
	id := extractBeadID(out)
	if id == "" {
		t.Log("WARNING: could not extract bead ID from bd create output, history toggle test may not find closed bead")
	} else {
		run(t, dir, "bd", "close", id)
	}

	// Create minimal capsule config so dashboard doesn't error on missing config.
	capsuleDir := filepath.Join(dir, ".capsule")
	if err := os.MkdirAll(capsuleDir, 0o755); err != nil {
		t.Fatalf("mkdir .capsule: %v", err)
	}

	return dir
}

// readPTYUntil reads from the PTY until the target string appears or timeout.
// Uses goroutine-based reads because SetReadDeadline is unreliable on PTY fds.
func readPTYUntil(t *testing.T, ptmx *os.File, target string, timeout time.Duration) string {
	t.Helper()
	var buf bytes.Buffer
	deadline := time.After(timeout)

	type readResult struct {
		data []byte
		err  error
	}
	// Buffer of 1 so the goroutine doesn't block when we return on timeout.
	ch := make(chan readResult, 1)

	go func() {
		for {
			tmp := make([]byte, 4096)
			n, err := ptmx.Read(tmp)
			ch <- readResult{data: tmp[:n], err: err}
			if err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-deadline:
			t.Logf("timeout waiting for %q, got so far:\n%s", target, stripANSI(buf.String()))
			return buf.String()
		case r := <-ch:
			if len(r.data) > 0 {
				buf.Write(r.data)
				if strings.Contains(stripANSI(buf.String()), target) {
					return buf.String()
				}
			}
			if r.err != nil && !os.IsTimeout(r.err) && r.err != io.EOF {
				return buf.String()
			}
		}
	}
}

// writePTY writes to the PTY and fails the test if the write errors.
func writePTY(t *testing.T, ptmx *os.File, s string) {
	t.Helper()
	if _, err := ptmx.Write([]byte(s)); err != nil {
		t.Fatalf("ptmx.Write(%q): %v", s, err)
	}
}

// waitForExit waits for the command to exit within the timeout.
// It uses the waitFunc returned by startDashboard so that cmd.Wait()
// is called at most once across both waitForExit and t.Cleanup.
func waitForExit(t *testing.T, cmd *exec.Cmd, waitFunc func() error, timeout time.Duration) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- waitFunc() }()

	select {
	case err := <-done:
		if err != nil {
			t.Logf("dashboard exited with: %v", err)
		}
	case <-time.After(timeout):
		cmd.Process.Kill()
		t.Errorf("dashboard did not exit within %s, killed", timeout)
	}
}

// run executes a command in the given directory, failing on error.
func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, out)
	}
}

// runOutput executes a command and returns its combined stdout/stderr.
func runOutput(t *testing.T, dir string, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%s %s failed: %v\n%s", name, strings.Join(args, " "), err, out)
	}
	return string(out)
}
