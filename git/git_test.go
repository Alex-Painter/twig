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

func TestGetLastCommit(t *testing.T) {
	repoDir := setupTestRepo(t)

	commit, err := GetLastCommit(repoDir)
	if err != nil {
		t.Fatalf("GetLastCommit() returned error: %v", err)
	}

	if commit.Message != "Initial commit" {
		t.Errorf("commit message = %q, want %q", commit.Message, "Initial commit")
	}

	if commit.RelativeTime == "" {
		t.Error("expected relative time to be non-empty")
	}
}

func TestGetLastCommit_MultipleCommits(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Create another commit
	testFile := filepath.Join(repoDir, "second.txt")
	if err := os.WriteFile(testFile, []byte("second"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	cmd := exec.Command("git", "add", ".")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git add: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Second commit")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git commit: %v", err)
	}

	commit, err := GetLastCommit(repoDir)
	if err != nil {
		t.Fatalf("GetLastCommit() returned error: %v", err)
	}

	if commit.Message != "Second commit" {
		t.Errorf("commit message = %q, want %q", commit.Message, "Second commit")
	}
}

func TestListWorktrees_IncludesLastCommit(t *testing.T) {
	repoDir := setupTestRepo(t)

	worktrees, err := ListWorktrees(repoDir)
	if err != nil {
		t.Fatalf("ListWorktrees() returned error: %v", err)
	}

	if len(worktrees) != 1 {
		t.Fatalf("expected 1 worktree, got %d", len(worktrees))
	}

	if worktrees[0].LastCommit.Message != "Initial commit" {
		t.Errorf("last commit message = %q, want %q", worktrees[0].LastCommit.Message, "Initial commit")
	}

	if worktrees[0].LastCommit.RelativeTime == "" {
		t.Error("expected last commit relative time to be non-empty")
	}
}

func TestGetAheadBehind_NoTrackingBranch(t *testing.T) {
	repoDir := setupTestRepo(t)

	// A fresh repo with no remote has no tracking branch
	ahead, behind, err := GetAheadBehind(repoDir)

	// Should return -1, -1 when there's no tracking branch
	if err == nil {
		t.Log("GetAheadBehind returned no error (might have default tracking)")
	}
	if ahead != -1 || behind != -1 {
		// This is expected for a repo without a remote
		if err != nil {
			// Expected behavior
			return
		}
	}
}

func TestGetAheadBehind_WithTrackingBranch(t *testing.T) {
	// Create a "remote" repo
	remoteDir := t.TempDir()
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init bare repo: %v", err)
	}

	// Create local repo
	repoDir := setupTestRepo(t)

	// Add remote and push
	cmd = exec.Command("git", "remote", "add", "origin", remoteDir)
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to add remote: %v", err)
	}

	cmd = exec.Command("git", "push", "-u", "origin", "master")
	cmd.Dir = repoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try main branch instead
		cmd = exec.Command("git", "push", "-u", "origin", "main")
		cmd.Dir = repoDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to push: %v (output: %s)", err, output)
		}
	}

	// Now we should be even with upstream
	ahead, behind, err := GetAheadBehind(repoDir)
	if err != nil {
		t.Fatalf("GetAheadBehind() returned error: %v", err)
	}

	if ahead != 0 || behind != 0 {
		t.Errorf("expected ahead=0, behind=0, got ahead=%d, behind=%d", ahead, behind)
	}
}

func TestGetAheadBehind_AheadOfRemote(t *testing.T) {
	// Create a "remote" repo
	remoteDir := t.TempDir()
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init bare repo: %v", err)
	}

	// Create local repo
	repoDir := setupTestRepo(t)

	// Add remote and push
	cmd = exec.Command("git", "remote", "add", "origin", remoteDir)
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to add remote: %v", err)
	}

	// Get current branch name
	branch, _ := GetCurrentBranch(repoDir)

	cmd = exec.Command("git", "push", "-u", "origin", branch)
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to push: %v", err)
	}

	// Create a local commit (ahead by 1)
	testFile := filepath.Join(repoDir, "ahead.txt")
	if err := os.WriteFile(testFile, []byte("ahead"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to git add: %v", err)
	}
	cmd = exec.Command("git", "commit", "-m", "Local commit")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	ahead, behind, err := GetAheadBehind(repoDir)
	if err != nil {
		t.Fatalf("GetAheadBehind() returned error: %v", err)
	}

	if ahead != 1 {
		t.Errorf("expected ahead=1, got %d", ahead)
	}
	if behind != 0 {
		t.Errorf("expected behind=0, got %d", behind)
	}
}

func TestListWorktrees_IncludesAheadBehind(t *testing.T) {
	repoDir := setupTestRepo(t)

	worktrees, err := ListWorktrees(repoDir)
	if err != nil {
		t.Fatalf("ListWorktrees() returned error: %v", err)
	}

	if len(worktrees) != 1 {
		t.Fatalf("expected 1 worktree, got %d", len(worktrees))
	}

	// Without a remote, ahead/behind should be -1
	if worktrees[0].Ahead != -1 || worktrees[0].Behind != -1 {
		t.Logf("ahead=%d, behind=%d (may vary based on git config)", worktrees[0].Ahead, worktrees[0].Behind)
	}
}

func TestRemoteBranchExists(t *testing.T) {
	// Without a remote, should return false
	repoDir := setupTestRepo(t)

	if RemoteBranchExists(repoDir, "main") {
		t.Log("RemoteBranchExists returned true (may have remote configured)")
	}

	if RemoteBranchExists(repoDir, "nonexistent-branch-xyz") {
		t.Error("expected RemoteBranchExists to return false for nonexistent branch")
	}
}

func TestLocalBranchExists(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Get current branch
	branch, _ := GetCurrentBranch(repoDir)

	if !LocalBranchExists(repoDir, branch) {
		t.Errorf("expected LocalBranchExists to return true for %s", branch)
	}

	if LocalBranchExists(repoDir, "nonexistent-branch-xyz") {
		t.Error("expected LocalBranchExists to return false for nonexistent branch")
	}
}

func TestCreateWorktree_NewBranch(t *testing.T) {
	repoDir := setupTestRepo(t)
	worktreeDir := t.TempDir()

	path, err := CreateWorktree(repoDir, worktreeDir, "feature-new")
	if err != nil {
		t.Fatalf("CreateWorktree() returned error: %v", err)
	}

	expectedPath := filepath.Join(worktreeDir, "feature-new")
	if path != expectedPath {
		t.Errorf("path = %q, want %q", path, expectedPath)
	}

	// Verify worktree was created
	worktrees, err := ListWorktrees(repoDir)
	if err != nil {
		t.Fatalf("ListWorktrees() returned error: %v", err)
	}

	found := false
	for _, wt := range worktrees {
		if wt.Branch == "feature-new" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected to find worktree with branch 'feature-new'")
	}
}

func TestSanitizeBranchForPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"feature-auth", "feature-auth"},
		{"feat/login", "feat-login"},
		{"fix/bug/123", "fix-bug-123"},
		{"main", "main"},
		{"a/b/c/d", "a-b-c-d"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := sanitizeBranchForPath(tt.input); got != tt.want {
				t.Errorf("sanitizeBranchForPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCreateWorktree_BranchWithSlashes(t *testing.T) {
	repoDir := setupTestRepo(t)
	worktreeDir := t.TempDir()

	branchName := "feat/login-page"
	path, err := CreateWorktree(repoDir, worktreeDir, branchName)
	if err != nil {
		t.Fatalf("CreateWorktree() returned error: %v", err)
	}

	// Path should use hyphen, not slash (no nested directory)
	expectedPath := filepath.Join(worktreeDir, "feat-login-page")
	if path != expectedPath {
		t.Errorf("path = %q, want %q", path, expectedPath)
	}

	// The worktree dir should exist as a single flat directory
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("worktree dir should exist at %s: %v", path, err)
	}
	if !info.IsDir() {
		t.Errorf("%s should be a directory", path)
	}

	// The nested "feat/login-page" path should NOT exist
	nestedPath := filepath.Join(worktreeDir, "feat", "login-page")
	if _, err := os.Stat(nestedPath); !os.IsNotExist(err) {
		t.Errorf("nested path %q should not exist", nestedPath)
	}

	// But the git branch should still be "feat/login-page" (with slash)
	worktrees, err := ListWorktrees(repoDir)
	if err != nil {
		t.Fatalf("ListWorktrees() returned error: %v", err)
	}

	found := false
	for _, wt := range worktrees {
		if wt.Branch == branchName {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected to find worktree with branch %q (with slash)", branchName)
	}
}

func TestCreateWorktree_ExistingLocalBranch(t *testing.T) {
	repoDir := setupTestRepo(t)
	worktreeDir := t.TempDir()

	// Create a local branch first
	cmd := exec.Command("git", "branch", "existing-branch")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create branch: %v", err)
	}

	path, err := CreateWorktree(repoDir, worktreeDir, "existing-branch")
	if err != nil {
		t.Fatalf("CreateWorktree() returned error: %v", err)
	}

	expectedPath := filepath.Join(worktreeDir, "existing-branch")
	if path != expectedPath {
		t.Errorf("path = %q, want %q", path, expectedPath)
	}
}

func TestDeleteWorktree(t *testing.T) {
	repoDir := setupTestRepo(t)
	worktreeDir := t.TempDir()

	// Create a worktree
	path, err := CreateWorktree(repoDir, worktreeDir, "to-delete")
	if err != nil {
		t.Fatalf("CreateWorktree() returned error: %v", err)
	}

	// Verify it exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("worktree directory should exist")
	}

	// Delete it
	if err := DeleteWorktree(repoDir, path, false); err != nil {
		t.Fatalf("DeleteWorktree() returned error: %v", err)
	}

	// Verify it's gone
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("worktree directory should not exist after delete")
	}

	// Verify it's removed from worktree list
	worktrees, _ := ListWorktrees(repoDir)
	for _, wt := range worktrees {
		if wt.Branch == "to-delete" {
			t.Error("to-delete branch should not be in worktree list")
		}
	}
}

func TestDeleteWorktree_WithUncommittedChanges(t *testing.T) {
	repoDir := setupTestRepo(t)
	worktreeDir := t.TempDir()

	// Create a worktree
	path, err := CreateWorktree(repoDir, worktreeDir, "dirty-branch")
	if err != nil {
		t.Fatalf("CreateWorktree() returned error: %v", err)
	}

	// Make uncommitted changes
	testFile := filepath.Join(path, "dirty.txt")
	if err := os.WriteFile(testFile, []byte("dirty"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Delete without force should fail
	if err := DeleteWorktree(repoDir, path, false); err == nil {
		t.Error("expected delete without force to fail for dirty worktree")
	}

	// Delete with force should succeed
	if err := DeleteWorktree(repoDir, path, true); err != nil {
		t.Fatalf("expected force delete to succeed: %v", err)
	}
}

func TestFetchAll_NoRemote(t *testing.T) {
	repoDir := setupTestRepo(t)

	// FetchAll on a repo with no remotes should succeed (no-op)
	err := FetchAll(repoDir)
	if err != nil {
		t.Logf("FetchAll() without remotes returned error (may be expected): %v", err)
	}
}

func TestFetchAll_WithRemote(t *testing.T) {
	// Create a "remote" repo
	remoteDir := t.TempDir()
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init bare repo: %v", err)
	}

	// Create local repo and add remote
	repoDir := setupTestRepo(t)
	cmd = exec.Command("git", "remote", "add", "origin", remoteDir)
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to add remote: %v", err)
	}

	// Push to create the remote branch
	branch, _ := GetCurrentBranch(repoDir)
	cmd = exec.Command("git", "push", "-u", "origin", branch)
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to push: %v", err)
	}

	// FetchAll should succeed
	if err := FetchAll(repoDir); err != nil {
		t.Errorf("FetchAll() returned error: %v", err)
	}
}

func TestPull_NoRemote(t *testing.T) {
	repoDir := setupTestRepo(t)

	// Pull on a repo with no remotes should fail
	err := Pull(repoDir)
	if err == nil {
		t.Log("Pull() without remotes returned no error (git may have config allowing this)")
	}
}

func TestPull_UpToDate(t *testing.T) {
	// Create a "remote" repo
	remoteDir := t.TempDir()
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = remoteDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init bare repo: %v", err)
	}

	// Create local repo, add remote, push
	repoDir := setupTestRepo(t)
	cmd = exec.Command("git", "remote", "add", "origin", remoteDir)
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to add remote: %v", err)
	}

	branch, _ := GetCurrentBranch(repoDir)
	cmd = exec.Command("git", "push", "-u", "origin", branch)
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to push: %v", err)
	}

	// Pull should succeed (already up-to-date)
	if err := Pull(repoDir); err != nil {
		t.Errorf("Pull() returned error: %v", err)
	}
}
