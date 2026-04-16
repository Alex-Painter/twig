package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// setupTestRepo creates a temporary git repository for testing.
func setupTestRepo(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@test.com")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to configure git email: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to configure git name: %v", err)
	}

	// Create an initial commit
	testFile := filepath.Join(dir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Test"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git add: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git commit: %v", err)
	}

	return dir
}

func TestListWorktrees_MainOnly(t *testing.T) {
	repoDir := setupTestRepo(t)

	worktrees, err := ListWorktrees(repoDir)
	if err != nil {
		t.Fatalf("ListWorktrees() returned error: %v", err)
	}

	if len(worktrees) != 1 {
		t.Fatalf("expected 1 worktree, got %d", len(worktrees))
	}

	wt := worktrees[0]
	if !wt.IsMain {
		t.Error("expected first worktree to be marked as main")
	}

	absRepoDir, _ := filepath.Abs(repoDir)
	absRepoDir, _ = filepath.EvalSymlinks(absRepoDir)
	absWtPath, _ := filepath.Abs(wt.Path)
	absWtPath, _ = filepath.EvalSymlinks(absWtPath)
	if absWtPath != absRepoDir {
		t.Errorf("worktree path = %q, want %q", absWtPath, absRepoDir)
	}
}

func TestListWorktrees_WithWorktree(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Create a worktree
	worktreeDir := filepath.Join(t.TempDir(), "feature-branch")
	cmd := exec.Command("git", "worktree", "add", "-b", "feature-branch", worktreeDir)
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	worktrees, err := ListWorktrees(repoDir)
	if err != nil {
		t.Fatalf("ListWorktrees() returned error: %v", err)
	}

	if len(worktrees) != 2 {
		t.Fatalf("expected 2 worktrees, got %d", len(worktrees))
	}

	// Find main and feature worktrees
	var mainWt, featureWt *Worktree
	for i := range worktrees {
		if worktrees[i].IsMain {
			mainWt = &worktrees[i]
		} else {
			featureWt = &worktrees[i]
		}
	}

	if mainWt == nil {
		t.Error("expected to find main worktree")
	}

	if featureWt == nil {
		t.Fatal("expected to find feature worktree")
	}

	if featureWt.Branch != "feature-branch" {
		t.Errorf("feature worktree branch = %q, want %q", featureWt.Branch, "feature-branch")
	}

	absWorktreeDir, _ := filepath.Abs(worktreeDir)
	absWorktreeDir, _ = filepath.EvalSymlinks(absWorktreeDir)
	absFeaturePath, _ := filepath.Abs(featureWt.Path)
	absFeaturePath, _ = filepath.EvalSymlinks(absFeaturePath)
	if absFeaturePath != absWorktreeDir {
		t.Errorf("feature worktree path = %q, want %q", absFeaturePath, absWorktreeDir)
	}
}

func TestListWorktrees_MultipleWorktrees(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Create multiple worktrees
	branches := []string{"feature-a", "feature-b", "bugfix-123"}
	for _, branch := range branches {
		worktreeDir := filepath.Join(t.TempDir(), branch)
		cmd := exec.Command("git", "worktree", "add", "-b", branch, worktreeDir)
		cmd.Dir = repoDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create worktree %s: %v", branch, err)
		}
	}

	worktrees, err := ListWorktrees(repoDir)
	if err != nil {
		t.Fatalf("ListWorktrees() returned error: %v", err)
	}

	// Should have main + 3 worktrees
	if len(worktrees) != 4 {
		t.Fatalf("expected 4 worktrees, got %d", len(worktrees))
	}

	// Count main worktrees
	mainCount := 0
	for _, wt := range worktrees {
		if wt.IsMain {
			mainCount++
		}
	}
	if mainCount != 1 {
		t.Errorf("expected exactly 1 main worktree, got %d", mainCount)
	}
}

func TestParseWorktreeList(t *testing.T) {
	// Test parsing porcelain format directly using temp directories for real paths
	tmpDir := t.TempDir()
	mainPath := filepath.Join(tmpDir, "main")
	featurePath := filepath.Join(tmpDir, "feature")
	detachedPath := filepath.Join(tmpDir, "detached")

	// Create the directories so EvalSymlinks works
	os.MkdirAll(mainPath, 0755)
	os.MkdirAll(featurePath, 0755)
	os.MkdirAll(detachedPath, 0755)

	output := []byte(fmt.Sprintf(`worktree %s
HEAD abc123def456
branch refs/heads/main

worktree %s
HEAD def456abc123
branch refs/heads/feature-branch

worktree %s
HEAD 111222333444
detached

`, mainPath, featurePath, detachedPath))

	worktrees, err := parseWorktreeList(output, mainPath)
	if err != nil {
		t.Fatalf("parseWorktreeList() returned error: %v", err)
	}

	if len(worktrees) != 3 {
		t.Fatalf("expected 3 worktrees, got %d", len(worktrees))
	}

	// Check main
	if !worktrees[0].IsMain {
		t.Error("expected first worktree to be main")
	}
	if worktrees[0].Branch != "main" {
		t.Errorf("main branch = %q, want %q", worktrees[0].Branch, "main")
	}

	// Check feature
	if worktrees[1].IsMain {
		t.Error("expected second worktree to not be main")
	}
	if worktrees[1].Branch != "feature-branch" {
		t.Errorf("feature branch = %q, want %q", worktrees[1].Branch, "feature-branch")
	}

	// Check detached
	if worktrees[2].Branch != "(detached)" {
		t.Errorf("detached branch = %q, want %q", worktrees[2].Branch, "(detached)")
	}
}

func TestGetCurrentBranch(t *testing.T) {
	repoDir := setupTestRepo(t)

	branch, err := GetCurrentBranch(repoDir)
	if err != nil {
		t.Fatalf("GetCurrentBranch() returned error: %v", err)
	}

	// Default branch is usually "main" or "master"
	if branch != "main" && branch != "master" {
		t.Errorf("GetCurrentBranch() = %q, want 'main' or 'master'", branch)
	}
}

func TestIsDirty_CleanRepo(t *testing.T) {
	repoDir := setupTestRepo(t)

	dirty, err := IsDirty(repoDir)
	if err != nil {
		t.Fatalf("IsDirty() returned error: %v", err)
	}

	if dirty {
		t.Error("expected clean repo to not be dirty")
	}
}

func TestIsDirty_UnstagedChanges(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Modify an existing file
	testFile := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(testFile, []byte("# Modified"), 0644); err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	dirty, err := IsDirty(repoDir)
	if err != nil {
		t.Fatalf("IsDirty() returned error: %v", err)
	}

	if !dirty {
		t.Error("expected repo with unstaged changes to be dirty")
	}
}

func TestIsDirty_StagedChanges(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Create and stage a new file
	testFile := filepath.Join(repoDir, "new-file.txt")
	if err := os.WriteFile(testFile, []byte("new content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	cmd := exec.Command("git", "add", "new-file.txt")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to stage file: %v", err)
	}

	dirty, err := IsDirty(repoDir)
	if err != nil {
		t.Fatalf("IsDirty() returned error: %v", err)
	}

	if !dirty {
		t.Error("expected repo with staged changes to be dirty")
	}
}

func TestIsDirty_UntrackedFiles(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Create an untracked file
	testFile := filepath.Join(repoDir, "untracked.txt")
	if err := os.WriteFile(testFile, []byte("untracked"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	dirty, err := IsDirty(repoDir)
	if err != nil {
		t.Fatalf("IsDirty() returned error: %v", err)
	}

	if !dirty {
		t.Error("expected repo with untracked files to be dirty")
	}
}

func TestListWorktrees_IncludesDirtyStatus(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Initially clean
	worktrees, err := ListWorktrees(repoDir)
	if err != nil {
		t.Fatalf("ListWorktrees() returned error: %v", err)
	}

	if len(worktrees) != 1 {
		t.Fatalf("expected 1 worktree, got %d", len(worktrees))
	}

	if worktrees[0].IsDirty {
		t.Error("expected clean worktree to not be dirty")
	}

	// Make it dirty
	testFile := filepath.Join(repoDir, "dirty.txt")
	if err := os.WriteFile(testFile, []byte("dirty"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	worktrees, err = ListWorktrees(repoDir)
	if err != nil {
		t.Fatalf("ListWorktrees() returned error: %v", err)
	}

	if !worktrees[0].IsDirty {
		t.Error("expected dirty worktree to be marked dirty")
	}
}
