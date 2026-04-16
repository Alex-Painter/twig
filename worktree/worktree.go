// Package worktree provides high-level operations for managing git worktrees
// and their associated tmux sessions.
package worktree

import (
	"github.com/Alex-Painter/twig/config"
	"github.com/Alex-Painter/twig/git"
	"github.com/Alex-Painter/twig/tmux"
)

// Manager handles worktree operations.
type Manager struct {
	config     *config.Config
	tmuxClient *tmux.Client
}

// NewManager creates a new worktree manager.
func NewManager(cfg *config.Config, tmuxClient *tmux.Client) *Manager {
	return &Manager{
		config:     cfg,
		tmuxClient: tmuxClient,
	}
}

// Create creates a new worktree and tmux session for the given branch.
// Returns the path to the created worktree.
func (m *Manager) Create(branchName string) (string, error) {
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
