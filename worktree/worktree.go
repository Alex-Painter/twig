// Package worktree provides high-level operations for managing git worktrees
// and their associated tmux sessions.
package worktree

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/Alex-Painter/twig/config"
	"github.com/Alex-Painter/twig/git"
	"github.com/Alex-Painter/twig/github"
	"github.com/Alex-Painter/twig/tmux"
)

// prPattern matches PR references like #123
var prPattern = regexp.MustCompile(`^#(\d+)$`)

// Manager handles worktree operations.
type Manager struct {
	config       *config.Config
	tmuxClient   *tmux.Client
	githubClient *github.Client
}

// NewManager creates a new worktree manager.
func NewManager(cfg *config.Config, tmuxClient *tmux.Client) *Manager {
	return &Manager{
		config:       cfg,
		tmuxClient:   tmuxClient,
		githubClient: github.NewClient(cfg.Repo),
	}
}

// Create creates a new worktree and tmux session for the given branch or PR.
// If input matches #123 pattern, it fetches the branch name from the PR.
// Returns the path to the created worktree.
func (m *Manager) Create(input string) (string, error) {
	branchName, err := m.resolveBranchName(input)
	if err != nil {
		return "", err
	}

	// Create the git worktree
	worktreePath, err := git.CreateWorktree(m.config.Repo, m.config.WorktreeDir, branchName)
	if err != nil {
		return "", err
	}

	// Create the tmux session
	sessionName := m.config.SessionName(branchName)
	err = m.tmuxClient.CreateSession(sessionName, m.config.Windows, worktreePath)
	if err != nil {
		// Worktree created but tmux session failed - still return the path
		// The user can create the session manually
		return worktreePath, err
	}

	return worktreePath, nil
}

// resolveBranchName resolves the input to a branch name.
// If input is a PR reference (#123), it fetches the branch from GitHub.
// Otherwise, it returns the input as-is.
func (m *Manager) resolveBranchName(input string) (string, error) {
	matches := prPattern.FindStringSubmatch(input)
	if matches == nil {
		// Not a PR reference, use input as branch name
		return input, nil
	}

	// Parse PR number
	prNumber, err := strconv.Atoi(matches[1])
	if err != nil {
		return "", fmt.Errorf("invalid PR number: %s", input)
	}

	// Check if gh is installed
	if !m.githubClient.IsInstalled() {
		return "", fmt.Errorf("gh CLI is not installed (required for PR checkout)")
	}

	// Check if authenticated
	if !m.githubClient.IsAuthenticated() {
		return "", fmt.Errorf("gh CLI is not authenticated (run 'gh auth login')")
	}

	// Fetch branch name from PR
	branch, err := m.githubClient.GetPRBranch(prNumber)
	if err != nil {
		return "", fmt.Errorf("failed to get branch for PR #%d: %w", prNumber, err)
	}

	return branch, nil
}

// IsPRReference returns true if the input looks like a PR reference (#123).
func IsPRReference(input string) bool {
	return prPattern.MatchString(input)
}
