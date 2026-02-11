package worktree

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// initGitRepo creates a bare-minimum git repo in dir with one commit.
func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	for _, args := range [][]string{
		{"init", "--initial-branch=main"},
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
		{"commit", "--allow-empty", "-m", "init"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "HOME="+dir)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %s\n%s", args, err, out)
		}
	}
}

func TestCreate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git worktree test in short mode")
	}

	tests := []struct {
		name       string
		id         string
		baseBranch string
		setup      func(t *testing.T, m *Manager)
		wantErr    error
	}{
		{
			name:       "creates worktree and branch",
			id:         "task-1",
			baseBranch: "HEAD",
		},
		{
			name:       "already exists error",
			id:         "task-1",
			baseBranch: "HEAD",
			setup: func(t *testing.T, m *Manager) {
				t.Helper()
				if err := m.Create("task-1", "HEAD"); err != nil {
					t.Fatalf("setup Create: %v", err)
				}
			},
			wantErr: ErrAlreadyExists,
		},
		{
			name:       "rejects empty id",
			id:         "",
			baseBranch: "HEAD",
			wantErr:    ErrInvalidID,
		},
		{
			name:       "rejects path traversal",
			id:         "../escape",
			baseBranch: "HEAD",
			wantErr:    ErrInvalidID,
		},
		{
			name:       "rejects flag-like id",
			id:         "--version",
			baseBranch: "HEAD",
			wantErr:    ErrInvalidID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given a git repo with optional pre-existing worktree state
			repoDir := t.TempDir()
			initGitRepo(t, repoDir)
			baseDir := ".capsule/worktrees"
			m := NewManager(repoDir, baseDir)

			if tt.setup != nil {
				tt.setup(t, m)
			}

			// When Create is called
			err := m.Create(tt.id, tt.baseBranch)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error wrapping %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then directory exists at expected path
			wtPath := filepath.Join(repoDir, baseDir, tt.id)
			if _, err := os.Stat(wtPath); errors.Is(err, os.ErrNotExist) {
				t.Errorf("worktree dir does not exist: %s", wtPath)
			}

			// Then git branch capsule-<id> exists
			branchName := "capsule-" + tt.id
			cmd := exec.Command("git", "branch", "--list", branchName)
			cmd.Dir = repoDir
			out, err := cmd.Output()
			if err != nil {
				t.Fatalf("git branch --list: %v", err)
			}
			if len(out) == 0 {
				t.Errorf("branch %q was not created", branchName)
			}
		})
	}
}

func TestRemove(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git worktree test in short mode")
	}

	tests := []struct {
		name         string
		id           string
		deleteBranch bool
		setup        func(t *testing.T, m *Manager)
		wantErr      error
	}{
		{
			name:         "removes worktree and branch",
			id:           "task-1",
			deleteBranch: true,
			setup: func(t *testing.T, m *Manager) {
				t.Helper()
				if err := m.Create("task-1", "HEAD"); err != nil {
					t.Fatalf("setup Create: %v", err)
				}
			},
		},
		{
			name:         "removes worktree keeps branch",
			id:           "task-1",
			deleteBranch: false,
			setup: func(t *testing.T, m *Manager) {
				t.Helper()
				if err := m.Create("task-1", "HEAD"); err != nil {
					t.Fatalf("setup Create: %v", err)
				}
			},
		},
		{
			name:    "not found error",
			id:      "nonexistent",
			wantErr: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given a git repo with optional pre-existing worktree
			repoDir := t.TempDir()
			initGitRepo(t, repoDir)
			baseDir := ".capsule/worktrees"
			m := NewManager(repoDir, baseDir)

			if tt.setup != nil {
				tt.setup(t, m)
			}

			// When Remove is called
			err := m.Remove(tt.id, tt.deleteBranch)

			if tt.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error %v, got nil", tt.wantErr)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("expected error wrapping %v, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then directory is gone
			wtPath := filepath.Join(repoDir, baseDir, tt.id)
			if _, err := os.Stat(wtPath); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("worktree dir still exists: %s", wtPath)
			}

			// Then branch state matches deleteBranch flag
			branchName := "capsule-" + tt.id
			cmd := exec.Command("git", "branch", "--list", branchName)
			cmd.Dir = repoDir
			out, err := cmd.Output()
			if err != nil {
				t.Fatalf("git branch --list: %v", err)
			}
			branchExists := len(out) > 0
			if tt.deleteBranch && branchExists {
				t.Errorf("branch %q should have been deleted", branchName)
			}
			if !tt.deleteBranch && !branchExists {
				t.Errorf("branch %q should have been preserved", branchName)
			}
		})
	}
}

func TestList(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git worktree test in short mode")
	}

	tests := []struct {
		name  string
		setup func(t *testing.T, m *Manager)
		want  []string
	}{
		{
			name: "empty when no worktrees",
			want: []string{},
		},
		{
			name: "returns created worktree IDs sorted",
			setup: func(t *testing.T, m *Manager) {
				t.Helper()
				for _, id := range []string{"task-b", "task-a"} {
					if err := m.Create(id, "HEAD"); err != nil {
						t.Fatalf("setup Create %s: %v", id, err)
					}
				}
			},
			want: []string{"task-a", "task-b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given a git repo with optional pre-created worktrees
			repoDir := t.TempDir()
			initGitRepo(t, repoDir)
			m := NewManager(repoDir, ".capsule/worktrees")

			if tt.setup != nil {
				tt.setup(t, m)
			}

			// When List is called
			got, err := m.List()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Then returned IDs match expected (List guarantees sorted order)
			if !slices.Equal(got, tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPrune(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git worktree test in short mode")
	}

	repoDir := t.TempDir()
	initGitRepo(t, repoDir)
	m := NewManager(repoDir, ".capsule/worktrees")

	// Given: a worktree was created then its directory manually removed
	if err := m.Create("orphan-1", "HEAD"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	wtPath := filepath.Join(repoDir, ".capsule/worktrees", "orphan-1")
	if err := os.RemoveAll(wtPath); err != nil {
		t.Fatalf("manual remove: %v", err)
	}

	// When Prune is called
	if err := m.Prune(); err != nil {
		t.Fatalf("Prune: %v", err)
	}

	// Then git no longer tracks the orphaned worktree
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git worktree list: %v", err)
	}
	if strings.Contains(string(out), "orphan-1") {
		t.Error("Prune did not clean orphaned worktree from git tracking")
	}
}

func TestListExcludesStaleDirectories(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git worktree test in short mode")
	}

	repoDir := t.TempDir()
	initGitRepo(t, repoDir)
	m := NewManager(repoDir, ".capsule/worktrees")

	// Given: a real worktree and a stale directory (not a git worktree)
	if err := m.Create("real-wt", "HEAD"); err != nil {
		t.Fatalf("Create: %v", err)
	}
	staleDir := filepath.Join(repoDir, ".capsule/worktrees", "stale-dir")
	if err := os.MkdirAll(staleDir, 0o755); err != nil {
		t.Fatalf("mkdir stale: %v", err)
	}

	// When List is called
	got, err := m.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}

	// Then only the real worktree is returned, not the stale directory
	want := []string{"real-wt"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestPath(t *testing.T) {
	// Given a manager with repo root and base directory
	m := NewManager("/repo", ".capsule/worktrees")

	// When Path is called with an ID
	got := m.Path("task-1")

	// Then it returns the absolute path to the worktree
	want := filepath.Join("/repo", ".capsule/worktrees", "task-1")
	if got != want {
		t.Errorf("Path(%q) = %q, want %q", "task-1", got, want)
	}
}

func TestExists(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping git worktree test in short mode")
	}

	tests := []struct {
		name  string
		setup func(t *testing.T, m *Manager)
		id    string
		want  bool
	}{
		{
			name: "false for nonexistent worktree",
			id:   "nope",
			want: false,
		},
		{
			name: "true for existing worktree",
			id:   "task-1",
			setup: func(t *testing.T, m *Manager) {
				t.Helper()
				if err := m.Create("task-1", "HEAD"); err != nil {
					t.Fatalf("setup Create: %v", err)
				}
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given a git repo with optional pre-created worktree
			repoDir := t.TempDir()
			initGitRepo(t, repoDir)
			m := NewManager(repoDir, ".capsule/worktrees")

			if tt.setup != nil {
				tt.setup(t, m)
			}

			// When Exists is called
			got := m.Exists(tt.id)

			// Then result matches expected
			if got != tt.want {
				t.Errorf("Exists(%q) = %v, want %v", tt.id, got, tt.want)
			}
		})
	}
}
