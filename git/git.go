// Package git provides functions for interacting with git repositories and worktrees.
package git

import (
	"bufio"
	"bytes"
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

	// Populate dirty status for each worktree
	for i := range worktrees {
		dirty, err := IsDirty(worktrees[i].Path)
		if err == nil {
			worktrees[i].IsDirty = dirty
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
