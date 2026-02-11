//go:build smoke

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestSmoke_GreenfieldNarrative exercises the capsule binary end-to-end against
// a real greenfield project (demo-greenfield template). Everything is real except
// the AI provider — a mock claude script on PATH creates real files and emits
// phase-appropriate signal JSON.
//
// This test serves as executable documentation: a reader can understand exactly
// what a user sees when they run capsule on a fresh project.
func TestSmoke_GreenfieldNarrative(t *testing.T) {
	// Prerequisites: node and bd must be on PATH.
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

	// Shared output captures for the analysis subtest.
	var happyOutput, retryOutput string
	var happyExit, retryExit int

	// ── Chapter 1: Happy Path — all phases pass ─────────────────────────

	t.Run("chapter 1: happy path", func(t *testing.T) {
		projectDir := setupGreenfieldProject(t, projectRoot)
		mockDir := createMockClaude(t, false)

		happyOutput, happyExit = runCapsuleBinary(t, binary, projectDir, mockDir, "demo-001.1.1")
		t.Log("--- capsule output ---\n" + happyOutput)

		// Exit code 0.
		if happyExit != 0 {
			t.Fatalf("exit code = %d, want 0", happyExit)
		}

		// 12 status lines: 6 phases × (running + passed).
		runningLines := countMatches(happyOutput, `\[\d/6\] \S+ running`)
		passedLines := countMatches(happyOutput, `\[\d/6\] \S+ passed`)
		if runningLines != 6 {
			t.Errorf("running lines = %d, want 6", runningLines)
		}
		if passedLines != 6 {
			t.Errorf("passed lines = %d, want 6", passedLines)
		}

		// files: lines appear for phases that changed files.
		if !strings.Contains(happyOutput, "files: src/todo.test.js") {
			t.Error("missing files line for test-writer")
		}
		if !strings.Contains(happyOutput, "files: src/todo.js") {
			t.Error("missing files line for execute")
		}

		// No retry indicators in happy path (attempt 1 is suppressed).
		if strings.Contains(happyOutput, "(attempt") {
			t.Error("unexpected retry indicator in happy path output")
		}

		// Post-pipeline messages.
		if !strings.Contains(happyOutput, "Merged capsule-demo-001.1.1") {
			t.Error("missing merge confirmation message")
		}
		if !strings.Contains(happyOutput, "Closed demo-001.1.1") {
			t.Error("missing bead close message")
		}
		if !strings.Contains(happyOutput, "Worklog: .capsule/logs/demo-001.1.1/worklog.md") {
			t.Error("missing worklog path message")
		}

		// Files on main branch.
		if _, err := os.Stat(filepath.Join(projectDir, "src", "todo.js")); err != nil {
			t.Error("src/todo.js not found on main branch")
		}
		if _, err := os.Stat(filepath.Join(projectDir, "src", "todo.test.js")); err != nil {
			t.Error("src/todo.test.js not found on main branch")
		}

		// node src/todo.test.js passes.
		nodeCmd := exec.Command("node", "src/todo.test.js")
		nodeCmd.Dir = projectDir
		nodeOut, nodeErr := nodeCmd.CombinedOutput()
		if nodeErr != nil {
			t.Errorf("node src/todo.test.js failed: %v\n%s", nodeErr, nodeOut)
		}

		// Worklog archived.
		archivePath := filepath.Join(projectDir, ".capsule", "logs", "demo-001.1.1", "worklog.md")
		if _, err := os.Stat(archivePath); err != nil {
			t.Errorf("worklog not archived at %s", archivePath)
		}

		// Worktree removed.
		wtPath := filepath.Join(projectDir, ".capsule", "worktrees", "demo-001.1.1")
		if _, err := os.Stat(wtPath); err == nil {
			t.Error("worktree still exists after pipeline")
		}

		// Bead closed.
		bdCmd := exec.Command("bd", "show", "demo-001.1.1")
		bdCmd.Dir = projectDir
		bdOut, _ := bdCmd.CombinedOutput()
		if !strings.Contains(strings.ToLower(string(bdOut)), "closed") {
			t.Errorf("bead not closed; bd show output:\n%s", bdOut)
		}
	})

	// ── Chapter 2: Retry — test-review requests changes ─────────────────

	t.Run("chapter 2: retry", func(t *testing.T) {
		projectDir := setupGreenfieldProject(t, projectRoot)
		mockDir := createMockClaude(t, true)

		retryOutput, retryExit = runCapsuleBinary(t, binary, projectDir, mockDir, "demo-001.1.1")
		t.Log("--- capsule output ---\n" + retryOutput)

		// Still exits 0 — retry recovers.
		if retryExit != 0 {
			t.Fatalf("exit code = %d, want 0", retryExit)
		}

		// Feedback line from NEEDS_WORK signal.
		if !strings.Contains(retryOutput, "feedback: Tests do not cover empty input rejection") {
			t.Error("missing feedback line from NEEDS_WORK signal")
		}

		// Retry indicator appears.
		if !strings.Contains(retryOutput, "(attempt 2/3)") {
			t.Error("missing (attempt 2/3) in retry output")
		}

		// Pipeline still completes: merge + close messages present.
		if !strings.Contains(retryOutput, "Merged capsule-demo-001.1.1") {
			t.Error("missing merge confirmation in retry scenario")
		}
		if !strings.Contains(retryOutput, "Closed demo-001.1.1") {
			t.Error("missing bead close in retry scenario")
		}
	})

	// ── Analysis: document captured outputs and UX findings ─────────────

	t.Run("output analysis", func(t *testing.T) {
		if happyOutput == "" {
			t.Skip("skipping analysis: chapter 1 did not produce output")
		}
		if retryOutput == "" {
			t.Skip("skipping analysis: chapter 2 did not produce output")
		}

		t.Log("=== HAPPY PATH OUTPUT ===")
		t.Log(happyOutput)
		t.Log("=== RETRY OUTPUT ===")
		t.Log(retryOutput)

		t.Log("=== ANALYSIS ===")

		// 1. Timestamps present but no per-phase elapsed time.
		tsPattern := regexp.MustCompile(`\[\d{2}:\d{2}:\d{2}\]`)
		if !tsPattern.MatchString(happyOutput) {
			t.Log("FINDING: no timestamps in output (unexpected)")
		} else {
			t.Log("OK: timestamps present in output")
		}
		if strings.Contains(happyOutput, "elapsed") || regexp.MustCompile(`\d+(\.\d+)?s`).MatchString(happyOutput) {
			t.Log("FINDING: per-phase elapsed time shown")
		} else {
			t.Log("OK: no per-phase elapsed time (timestamps only)")
		}

		// 2. Progress fraction during retry stays N/6.
		if strings.Contains(retryOutput, "/6]") && !strings.Contains(retryOutput, "/8]") {
			t.Log("OK: progress stays N/6 during retry (not recounted)")
		} else {
			t.Log("FINDING: progress fraction changed during retry")
		}

		// 3. Worklog path uses relative path.
		if strings.Contains(happyOutput, "Worklog: .capsule/") {
			t.Log("OK: worklog path is relative (portable)")
		} else {
			t.Log("FINDING: worklog path is absolute")
		}

		// 4. Exit codes.
		t.Logf("Happy path exit: %d, Retry exit: %d", happyExit, retryExit)
	})
}

// ── Helpers ─────────────────────────────────────────────────────────────────

// setupGreenfieldProject creates a fresh demo-greenfield project via
// setup-template.sh and copies prompts + templates into it.
func setupGreenfieldProject(t *testing.T, projectRoot string) string {
	t.Helper()

	cmd := exec.Command(filepath.Join(projectRoot, "scripts", "setup-template.sh"),
		"--template=demo-greenfield")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("setup-template.sh failed: %v\n%s", err, out)
	}
	projectDir := strings.TrimSpace(string(out))
	t.Cleanup(func() { os.RemoveAll(projectDir) })

	// Copy prompts/ into project.
	copyDir(t, filepath.Join(projectRoot, "prompts"), filepath.Join(projectDir, "prompts"))

	// Copy templates/worklog.md.template into project.
	dstTemplates := filepath.Join(projectDir, "templates")
	if err := os.MkdirAll(dstTemplates, 0o755); err != nil {
		t.Fatalf("mkdir templates: %v", err)
	}
	copyFile(t,
		filepath.Join(projectRoot, "templates", "worklog.md.template"),
		filepath.Join(projectDir, "templates", "worklog.md.template"))

	// Create .capsule/ config directory (config defaults used).
	if err := os.MkdirAll(filepath.Join(projectDir, ".capsule"), 0o755); err != nil {
		t.Fatalf("mkdir .capsule: %v", err)
	}

	return projectDir
}

// createMockClaude writes a mock claude shell script to a temp directory.
// If withRetry is true, the mock returns NEEDS_WORK on the first test-review call.
func createMockClaude(t *testing.T, withRetry bool) string {
	t.Helper()
	mockDir := t.TempDir()
	stateDir := t.TempDir()
	mockPath := filepath.Join(mockDir, "claude")

	retryFlag := "false"
	if withRetry {
		retryFlag = "true"
	}

	script := `#!/usr/bin/env bash
set -euo pipefail

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
MOCK_RETRY="` + retryFlag + `"
MOCK_STATE_DIR="` + stateDir + `"

# Detect phase name for per-phase counters.
PHASE_NAME="unknown"
case "$FIRST_LINE" in
    *[Tt]est-[Ww]riter*) PHASE_NAME="test-writer" ;;
    *[Tt]est-[Rr]eview*) PHASE_NAME="test-review" ;;
    *[Ee]xecute-[Rr]eview*) PHASE_NAME="execute-review" ;;
    *[Ee]xecute*) PHASE_NAME="execute" ;;
    *[Ss]ign-[Oo]ff*) PHASE_NAME="sign-off" ;;
    *[Mm]erge*) PHASE_NAME="merge" ;;
esac

# Track per-phase call count for retry scenarios.
PHASE_COUNTER="$MOCK_STATE_DIR/count_${PHASE_NAME}"
PHASE_COUNT=$(cat "$PHASE_COUNTER" 2>/dev/null || echo 0)
PHASE_COUNT=$((PHASE_COUNT + 1))
echo "$PHASE_COUNT" > "$PHASE_COUNTER"

# Retry: test-review returns NEEDS_WORK on its first invocation.
if [ "$MOCK_RETRY" = "true" ] && [ "$PHASE_NAME" = "test-review" ] && [ "$PHASE_COUNT" -eq 1 ]; then
    printf '{"status":"NEEDS_WORK","feedback":"Tests do not cover empty input rejection","files_changed":[],"summary":"needs work"}\n'
    exit 0
fi

# Test-writer phase: create failing tests.
if printf '%s\n' "$FIRST_LINE" | grep -qi 'test-writer'; then
    mkdir -p src
    cat > src/todo.test.js << 'TESTEOF'
const assert = require('assert');
const { TodoApp } = require('./todo.js');

// Test: add todo creates item with correct structure
const app1 = new TodoApp();
const todo = app1.addTodo('Buy milk');
assert.strictEqual(todo.text, 'Buy milk');
assert.strictEqual(todo.completed, false);
assert.ok(todo.id);
assert.ok(todo.createdAt);
console.log('PASS: add todo creates item with correct structure');

// Test: add todo with empty input rejected
const app2 = new TodoApp();
assert.throws(() => app2.addTodo(''), /empty/i);
assert.throws(() => app2.addTodo('   '), /empty/i);
console.log('PASS: empty input rejected');

// Test: todos persist via toJSON/fromJSON
const app3 = new TodoApp();
app3.addTodo('Test persistence');
const data = app3.toJSON();
const app4 = TodoApp.fromJSON(data);
assert.strictEqual(app4.todos.length, 1);
assert.strictEqual(app4.todos[0].text, 'Test persistence');
console.log('PASS: todos persist via toJSON/fromJSON');

console.log('\nAll tests passed!');
TESTEOF
    git add src/todo.test.js
    git diff --cached --quiet || git commit -q -m "Add todo tests"
    printf '{"status":"PASS","feedback":"Tests created","files_changed":["src/todo.test.js"],"summary":"3 tests for TodoApp.addTodo"}\n'
    exit 0
fi

# Execute phase (NOT execute-review): create implementation.
if printf '%s\n' "$FIRST_LINE" | grep -qi '^# Execute Phase'; then
    mkdir -p src
    cat > src/todo.js << 'IMPLEOF'
class TodoApp {
  constructor() { this.todos = []; }
  addTodo(text) {
    if (!text || !text.trim()) throw new Error('Todo text cannot be empty');
    const todo = {
      id: Date.now().toString(36) + Math.random().toString(36).slice(2, 6),
      text: text.trim(),
      completed: false,
      createdAt: new Date().toISOString()
    };
    this.todos.push(todo);
    return todo;
  }
  toJSON() { return JSON.stringify(this.todos); }
  static fromJSON(json) {
    const app = new TodoApp();
    app.todos = JSON.parse(json);
    return app;
  }
}
module.exports = { TodoApp };
IMPLEOF
    git add src/todo.js
    git diff --cached --quiet || git commit -q -m "Implement TodoApp.addTodo"
    printf '{"status":"PASS","feedback":"Implementation complete","files_changed":["src/todo.js"],"summary":"TodoApp.addTodo implemented"}\n'
    exit 0
fi

# Merge phase: ensure all implementation files are committed.
if printf '%s\n' "$FIRST_LINE" | grep -qi 'merge'; then
    # Stage any unstaged implementation files (safety net).
    git add src/ 2>/dev/null || true
    git diff --cached --quiet || git commit -q -m "merge: stage remaining files"
    HASH=$(git rev-parse --short HEAD)
    printf '{"status":"PASS","feedback":"Merged","files_changed":[],"summary":"merge complete","commit_hash":"%s"}\n' "$HASH"
    exit 0
fi

# All other phases (test-review, execute-review, sign-off): just pass.
printf '{"status":"PASS","feedback":"Approved","files_changed":[],"summary":"phase passed"}\n'
`
	if err := os.WriteFile(mockPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write mock claude: %v", err)
	}
	return mockDir
}

// runCapsuleBinary runs the capsule binary and returns its output and exit code.
func runCapsuleBinary(t *testing.T, binary, projectDir, mockDir, beadID string) (string, int) {
	t.Helper()
	cmd := exec.Command(binary, "run", beadID, "--timeout", "60")
	cmd.Dir = projectDir
	cmd.Env = append(os.Environ(), "PATH="+mockDir+":"+os.Getenv("PATH"))

	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("capsule run failed unexpectedly: %v\n%s", err, out)
		}
	}
	return string(out), exitCode
}

// countMatches returns the number of lines matching the given regex pattern.
func countMatches(output, pattern string) int {
	re := regexp.MustCompile(pattern)
	return len(re.FindAllString(output, -1))
}

// copyDir recursively copies src directory to dst.
func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatalf("readdir %s: %v", src, err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dst, err)
	}
	for _, e := range entries {
		srcPath := filepath.Join(src, e.Name())
		dstPath := filepath.Join(dst, e.Name())
		if e.IsDir() {
			copyDir(t, srcPath, dstPath)
			continue
		}
		copyFile(t, srcPath, dstPath)
	}
}

// copyFile copies a single file from src to dst.
func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("write %s: %v", dst, err)
	}
}
