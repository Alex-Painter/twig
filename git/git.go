// Package git provides functions for interacting with git repositories and worktrees.
package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Worktree represents a git worktree or the main repository clone.
type Worktree struct {
	// Path is the absolute path to the worktree directory.
	Path string

	// Branch is the currently checked out branch.
	Branch string

	// IsMain indicates if this is the main repository clone (not a worktree).
	IsMain bool

	// IsDirty indicates if the worktree has uncommitted changes.
	IsDirty bool

	// LastCommit contains info about the most recent commit.
	LastCommit CommitInfo

	// Ahead is the number of commits ahead of the remote tracking branch.
	// -1 indicates unknown (no tracking branch or error).
	Ahead int

	// Behind is the number of commits behind the remote tracking branch.
	// -1 indicates unknown (no tracking branch or error).
	Behind int
}

// CommitInfo contains information about a commit.
type CommitInfo struct {
	// Message is the commit message (first line only).
	Message string

	// RelativeTime is the human-readable relative time (e.g., "2 hours ago").
	RelativeTime string
}

// ListWorktrees returns all worktrees for the repository at repoPath.
// The main repository is included in the list with IsMain set to true.
// Each worktree's dirty status is also populated.
func ListWorktrees(repoPath string) ([]Worktree, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	worktrees, err := parseWorktreeList(output, repoPath)
	if err != nil {
		return nil, err
	}

	// Populate dirty status, last commit, and ahead/behind for each worktree
	for i := range worktrees {
		dirty, err := IsDirty(worktrees[i].Path)
		if err == nil {
			worktrees[i].IsDirty = dirty
		}

		commit, err := GetLastCommit(worktrees[i].Path)
		if err == nil {
			worktrees[i].LastCommit = commit
		}

		ahead, behind, err := GetAheadBehind(worktrees[i].Path)
		if err == nil {
			worktrees[i].Ahead = ahead
			worktrees[i].Behind = behind
		} else {
			worktrees[i].Ahead = -1
			worktrees[i].Behind = -1
		}
	}

	return worktrees, nil
}

// parseWorktreeList parses the output of `git worktree list --porcelain`.
// The porcelain format looks like:
//
//	worktree /path/to/main
//	HEAD abc123
//	branch refs/heads/main
//
//	worktree /path/to/worktree
//	HEAD def456
//	branch refs/heads/feature
func parseWorktreeList(output []byte, mainRepoPath string) ([]Worktree, error) {
	var worktrees []Worktree
	var current *Worktree

	// Normalize the main repo path for comparison, resolving symlinks
	mainRepoPath, _ = filepath.Abs(mainRepoPath)
	mainRepoPath, _ = filepath.EvalSymlinks(mainRepoPath)

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "worktree ") {
			if current != nil {
				worktrees = append(worktrees, *current)
			}
			path := strings.TrimPrefix(line, "worktree ")
			absPath, _ := filepath.Abs(path)
			absPath, _ = filepath.EvalSymlinks(absPath)
			current = &Worktree{
				Path:   path,
				IsMain: absPath == mainRepoPath,
			}
		} else if strings.HasPrefix(line, "branch ") {
			if current != nil {
				branch := strings.TrimPrefix(line, "branch ")
				// Strip refs/heads/ prefix
				branch = strings.TrimPrefix(branch, "refs/heads/")
				current.Branch = branch
			}
		} else if strings.HasPrefix(line, "detached") {
			if current != nil {
				current.Branch = "(detached)"
			}
		}
	}

	// Don't forget the last worktree
	if current != nil {
		worktrees = append(worktrees, *current)
	}

	return worktrees, scanner.Err()
}

// GetCurrentBranch returns the current branch name for the repository at the given path.
func GetCurrentBranch(repoPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

// IsDirty returns true if the worktree has uncommitted changes.
// This includes staged changes, unstaged changes, and untracked files.
func IsDirty(worktreePath string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = worktreePath

	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	// If there's any output, the worktree is dirty
	return len(strings.TrimSpace(string(output))) > 0, nil
}

// GetLastCommit returns information about the most recent commit in the worktree.
func GetLastCommit(worktreePath string) (CommitInfo, error) {
	// Get commit message (first line) and relative time in one call
	// Format: %s = subject, %cr = committer date relative
	cmd := exec.Command("git", "log", "-1", "--format=%s|%cr")
	cmd.Dir = worktreePath

	output, err := cmd.Output()
	if err != nil {
		return CommitInfo{}, err
	}

	parts := strings.SplitN(strings.TrimSpace(string(output)), "|", 2)
	info := CommitInfo{}

	if len(parts) >= 1 {
		info.Message = parts[0]
	}
	if len(parts) >= 2 {
		info.RelativeTime = parts[1]
	}

	return info, nil
}

// AheadBehind contains the number of commits ahead and behind the remote.
type AheadBehind struct {
	Ahead  int
	Behind int
}

// GetAheadBehind returns the number of commits ahead and behind the remote tracking branch.
// Returns (-1, -1) if there's no tracking branch or an error occurs.
func GetAheadBehind(worktreePath string) (int, int, error) {
	// Use rev-list to count commits ahead and behind
	// This requires a tracking branch to be set
	cmd := exec.Command("git", "rev-list", "--left-right", "--count", "@{upstream}...HEAD")
	cmd.Dir = worktreePath

	output, err := cmd.Output()
	if err != nil {
		// No tracking branch or other error
		return -1, -1, err
	}

	// Output format: "behind\tahead\n"
	parts := strings.Fields(strings.TrimSpace(string(output)))
	if len(parts) != 2 {
		return -1, -1, nil
	}

	var behind, ahead int
	_, err = fmt.Sscanf(parts[0], "%d", &behind)
	if err != nil {
		return -1, -1, err
	}
	_, err = fmt.Sscanf(parts[1], "%d", &ahead)
	if err != nil {
		return -1, -1, err
	}

	return ahead, behind, nil
}

// RemoteBranchExists checks if a branch exists on the remote.
func RemoteBranchExists(repoPath, branchName string) bool {
	cmd := exec.Command("git", "ls-remote", "--exit-code", "--heads", "origin", branchName)
	cmd.Dir = repoPath
	err := cmd.Run()
	return err == nil
}

// LocalBranchExists checks if a branch exists locally.
func LocalBranchExists(repoPath, branchName string) bool {
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branchName)
	cmd.Dir = repoPath
	err := cmd.Run()
	return err == nil
}

// CreateWorktree creates a new git worktree for the given branch.
// It follows this logic:
//  1. If remote branch exists, fetch it and create worktree tracking remote
//  2. If local branch exists, create worktree using local branch
//  3. Otherwise, create new branch from HEAD
//
// Returns the path to the created worktree.
func CreateWorktree(repoPath, worktreeDir, branchName string) (string, error) {
	worktreePath := filepath.Join(worktreeDir, branchName)

	// Check if remote branch exists
	if RemoteBranchExists(repoPath, branchName) {
		// Fetch the remote branch
		cmd := exec.Command("git", "fetch", "origin", branchName)
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("failed to fetch remote branch: %w", err)
		}

		// Check if local branch already exists
		if LocalBranchExists(repoPath, branchName) {
			// Use existing local branch
			cmd = exec.Command("git", "worktree", "add", worktreePath, branchName)
			cmd.Dir = repoPath
			if output, err := cmd.CombinedOutput(); err != nil {
				return "", fmt.Errorf("failed to create worktree: %s", output)
			}
		} else {
			// Create local branch tracking remote
			cmd = exec.Command("git", "worktree", "add", "--track", "-b", branchName, worktreePath, "origin/"+branchName)
			cmd.Dir = repoPath
			if output, err := cmd.CombinedOutput(); err != nil {
				return "", fmt.Errorf("failed to create worktree: %s", output)
			}
		}

		// Set upstream tracking
		cmd = exec.Command("git", "branch", "--set-upstream-to=origin/"+branchName, branchName)
		cmd.Dir = worktreePath
		cmd.Run() // Ignore error, might already be set

		return worktreePath, nil
	}

	// Check if local branch exists
	if LocalBranchExists(repoPath, branchName) {
		cmd := exec.Command("git", "worktree", "add", worktreePath, branchName)
		cmd.Dir = repoPath
		if output, err := cmd.CombinedOutput(); err != nil {
			return "", fmt.Errorf("failed to create worktree: %s", output)
		}
		return worktreePath, nil
	}

	// Create new branch from HEAD
	cmd := exec.Command("git", "worktree", "add", "-b", branchName, worktreePath)
	cmd.Dir = repoPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("failed to create worktree: %s", output)
	}

	return worktreePath, nil
}

// DeleteWorktree removes a git worktree at the given path.
// If force is true, it will delete even with uncommitted changes.
func DeleteWorktree(repoPath, worktreePath string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, worktreePath)

	cmd := exec.Command("git", args...)
	cmd.Dir = repoPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to delete worktree: %s", output)
	}
	return nil
}
