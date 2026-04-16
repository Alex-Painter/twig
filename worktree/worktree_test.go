package worktree

import (
	"errors"
	"testing"

	"github.com/Alex-Painter/twig/config"
	"github.com/Alex-Painter/twig/git"
	"github.com/Alex-Painter/twig/tmux"
)

func TestIsPRReference(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"#123", true},
		{"#1", true},
		{"#99999", true},
		{"feature-branch", false},
		{"123", false},
		{"#", false},
		{"#abc", false},
		{"", false},
		{"fix-#123", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := IsPRReference(tt.input); got != tt.want {
				t.Errorf("IsPRReference(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// mockTmuxRunner is a simple runner for tmux client tests in the worktree package.
type mockTmuxRunner struct{}

func (m mockTmuxRunner) Run(name string, args ...string) (string, error) {
	// Always succeed - minimal stub
	return "", errors.New("no session")
}

func TestDelete_MainClone(t *testing.T) {
	cfg := &config.Config{
		Repo:           "/path/to/repo",
		WorktreeDir:    "/path/to/worktrees",
		SessionPattern: "{repo}:{branch}",
		Windows:        []string{"shell"},
	}

	tmuxClient := tmux.NewClientWithRunner(mockTmuxRunner{})
	mgr := NewManager(cfg, tmuxClient)

	mainWorktree := git.Worktree{
		Path:   "/path/to/repo",
		Branch: "main",
		IsMain: true,
	}

	err := mgr.Delete(mainWorktree, false)
	if err != ErrCannotDeleteMain {
		t.Errorf("expected ErrCannotDeleteMain, got %v", err)
	}
}

func TestDelete_DirtyWithoutForce(t *testing.T) {
	cfg := &config.Config{
		Repo:           "/path/to/repo",
		WorktreeDir:    "/path/to/worktrees",
		SessionPattern: "{repo}:{branch}",
		Windows:        []string{"shell"},
	}

	tmuxClient := tmux.NewClientWithRunner(mockTmuxRunner{})
	mgr := NewManager(cfg, tmuxClient)

	dirtyWorktree := git.Worktree{
		Path:    "/path/to/worktrees/feature",
		Branch:  "feature",
		IsMain:  false,
		IsDirty: true,
	}

	err := mgr.Delete(dirtyWorktree, false)
	if err != ErrDirtyWorktree {
		t.Errorf("expected ErrDirtyWorktree, got %v", err)
	}
}
