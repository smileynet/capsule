// Package worktree manages git worktree creation, removal, and branch lifecycle.
package worktree

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Sentinel errors for caller-checkable conditions.
var (
	ErrAlreadyExists = errors.New("worktree: already exists")
	ErrNotFound      = errors.New("worktree: not found")
	ErrInvalidID     = errors.New("worktree: invalid id")
	ErrMergeConflict = errors.New("worktree: merge conflict")
)

// validateID checks that id is safe for use as a path component and git argument.
// Rejects empty, path traversal (/ \ . ..), and flag-like IDs (starting with -).
func validateID(id string) error {
	if id == "" {
		return fmt.Errorf("%w: cannot be empty", ErrInvalidID)
	}
	if strings.HasPrefix(id, "-") {
		return fmt.Errorf("%w: %q (must not start with -)", ErrInvalidID, id)
	}
	if strings.ContainsAny(id, `/\`) || id == "." || id == ".." {
		return fmt.Errorf("%w: %q", ErrInvalidID, id)
	}
	return nil
}

// Manager manages git worktrees under a base directory within a repository.
type Manager struct {
	repoRoot string
	baseDir  string
}

// NewManager creates a Manager that manages worktrees under baseDir relative to repoRoot.
func NewManager(repoRoot, baseDir string) *Manager {
	return &Manager{
		repoRoot: repoRoot,
		baseDir:  baseDir,
	}
}

// Create creates a new git worktree for the given ID, branching from baseBranch.
// The worktree is placed at <repoRoot>/<baseDir>/<id>/ on branch capsule-<id>.
func (m *Manager) Create(id, baseBranch string) error {
	if err := validateID(id); err != nil {
		return err
	}
	wtPath := m.worktreePath(id)
	if _, err := os.Stat(wtPath); err == nil {
		return fmt.Errorf("worktree %q: %w", id, ErrAlreadyExists)
	}

	parentDir := filepath.Dir(wtPath)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return fmt.Errorf("worktree: mkdir %s: %w", parentDir, err)
	}

	branchName := "capsule-" + id
	cmd := exec.Command("git", "worktree", "add", "-b", branchName, wtPath, baseBranch)
	cmd.Dir = m.repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		// Best-effort cleanup of partial directory.
		_ = os.RemoveAll(wtPath)
		return fmt.Errorf("worktree: git worktree add: %w\n%s", err, strings.TrimSpace(string(out)))
	}

	return nil
}

// Remove removes the git worktree for the given ID using --force,
// which discards any uncommitted changes in the worktree.
// If deleteBranch is true, the capsule-<id> branch is also deleted.
func (m *Manager) Remove(id string, deleteBranch bool) error {
	if err := validateID(id); err != nil {
		return err
	}
	wtPath := m.worktreePath(id)
	if _, err := os.Stat(wtPath); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("worktree %q: %w", id, ErrNotFound)
	}

	cmd := exec.Command("git", "worktree", "remove", "--force", wtPath)
	cmd.Dir = m.repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("worktree: git worktree remove: %w\n%s", err, strings.TrimSpace(string(out)))
	}

	if deleteBranch {
		branchName := "capsule-" + id
		cmd := exec.Command("git", "branch", "-D", branchName)
		cmd.Dir = m.repoRoot
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("worktree: git branch -D %s: %w\n%s", branchName, err, strings.TrimSpace(string(out)))
		}
	}

	return nil
}

// Prune removes stale git worktree tracking entries whose directories
// no longer exist. Call after bulk Remove operations or manual cleanup.
func (m *Manager) Prune() error {
	cmd := exec.Command("git", "worktree", "prune")
	cmd.Dir = m.repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("worktree: git worktree prune: %w\n%s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// List returns the IDs of all worktrees managed under the base directory.
// Only directories that are actual git worktrees are included; stale
// directories left by failed operations are excluded.
// The returned IDs are sorted alphabetically.
func (m *Manager) List() ([]string, error) {
	dir := filepath.Join(m.repoRoot, m.baseDir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("worktree: reading %s: %w", dir, err)
	}

	registered, err := m.registeredWorktrees()
	if err != nil {
		return nil, err
	}

	ids := []string{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		absPath := filepath.Join(dir, e.Name())
		if registered[absPath] {
			ids = append(ids, e.Name())
		}
	}
	sort.Strings(ids)
	return ids, nil
}

// registeredWorktrees returns a set of absolute paths that git considers
// active worktrees, parsed from "git worktree list --porcelain".
func (m *Manager) registeredWorktrees() (map[string]bool, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = m.repoRoot
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("worktree: git worktree list: %w", err)
	}

	registered := make(map[string]bool)
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "worktree ") {
			registered[strings.TrimPrefix(line, "worktree ")] = true
		}
	}
	return registered, nil
}

// Path returns the absolute path for a worktree with the given ID.
func (m *Manager) Path(id string) string {
	return m.worktreePath(id)
}

// Exists reports whether a worktree directory exists for the given ID.
func (m *Manager) Exists(id string) bool {
	if validateID(id) != nil {
		return false
	}
	_, err := os.Stat(m.worktreePath(id))
	return err == nil
}

// worktreePath returns the absolute path for a worktree with the given ID.
func (m *Manager) worktreePath(id string) string {
	return filepath.Join(m.repoRoot, m.baseDir, id)
}

// MergeToMain merges the capsule-<id> branch into mainBranch with --no-ff.
// Returns ErrMergeConflict if the merge encounters conflicts.
func (m *Manager) MergeToMain(id, mainBranch, commitMsg string) error {
	if err := validateID(id); err != nil {
		return err
	}

	// Checkout main branch.
	checkout := exec.Command("git", "checkout", mainBranch, "-q")
	checkout.Dir = m.repoRoot
	if out, err := checkout.CombinedOutput(); err != nil {
		return fmt.Errorf("worktree: git checkout %s: %w\n%s", mainBranch, err, strings.TrimSpace(string(out)))
	}

	// Merge with --no-ff.
	branchName := "capsule-" + id
	merge := exec.Command("git", "merge", "--no-ff", branchName, "-m", commitMsg)
	merge.Dir = m.repoRoot
	out, err := merge.CombinedOutput()
	if err != nil {
		outStr := string(out)
		if strings.Contains(outStr, "CONFLICT") {
			// Abort the failed merge to leave the repo clean.
			abort := exec.Command("git", "merge", "--abort")
			abort.Dir = m.repoRoot
			_ = abort.Run()
			return fmt.Errorf("%w: merging %s into %s", ErrMergeConflict, branchName, mainBranch)
		}
		return fmt.Errorf("worktree: git merge: %w\n%s", err, strings.TrimSpace(outStr))
	}
	return nil
}

// DetectMainBranch determines the main branch name.
// Checks git symbolic-ref refs/remotes/origin/HEAD first,
// then falls back to checking if "main" or "master" branches exist.
func (m *Manager) DetectMainBranch() (string, error) {
	// Try origin/HEAD.
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD")
	cmd.Dir = m.repoRoot
	if out, err := cmd.Output(); err == nil {
		ref := strings.TrimSpace(string(out))
		// refs/remotes/origin/main → main
		parts := strings.Split(ref, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1], nil
		}
	}

	// Fallback: check if "main" branch exists.
	cmd = exec.Command("git", "rev-parse", "--verify", "refs/heads/main")
	cmd.Dir = m.repoRoot
	if err := cmd.Run(); err == nil {
		return "main", nil
	}

	// Fallback: check if "master" branch exists.
	cmd = exec.Command("git", "rev-parse", "--verify", "refs/heads/master")
	cmd.Dir = m.repoRoot
	if err := cmd.Run(); err == nil {
		return "master", nil
	}

	return "", fmt.Errorf("worktree: cannot detect main branch")
}
