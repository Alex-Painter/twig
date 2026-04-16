// Package tui provides the terminal user interface for twig.
package tui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Alex-Painter/twig/config"
	"github.com/Alex-Painter/twig/git"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Styles for the TUI
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	mainMarkerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)

	pathStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)
)

// Model represents the TUI state.
type Model struct {
	config    *config.Config
	worktrees []git.Worktree
	cursor    int
	err       error
}

// worktreesMsg is sent when worktrees are loaded.
type worktreesMsg struct {
	worktrees []git.Worktree
	err       error
}

// New creates a new TUI model.
func New(cfg *config.Config) Model {
	return Model{
		config: cfg,
	}
}

// Init initializes the model and returns an initial command.
func (m Model) Init() tea.Cmd {
	return m.loadWorktrees
}

// loadWorktrees fetches the list of worktrees.
func (m Model) loadWorktrees() tea.Msg {
	worktrees, err := git.ListWorktrees(m.config.Repo)
	return worktreesMsg{worktrees: worktrees, err: err}
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.worktrees)-1 {
				m.cursor++
			}
		case "r":
			return m, m.loadWorktrees
		}

	case worktreesMsg:
		m.worktrees = msg.worktrees
		m.err = msg.err
		// Reset cursor if out of bounds
		if m.cursor >= len(m.worktrees) {
			m.cursor = max(0, len(m.worktrees)-1)
		}
	}

	return m, nil
}

// View renders the TUI.
func (m Model) View() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("twig - Git Worktree Manager"))
	b.WriteString("\n")

	// Error display
	if m.err != nil {
		b.WriteString(fmt.Sprintf("Error: %v\n\n", m.err))
	}

	// Worktree list
	if len(m.worktrees) == 0 && m.err == nil {
		b.WriteString("No worktrees found.\n")
	} else {
		for i, wt := range m.worktrees {
			cursor := "  "
			if i == m.cursor {
				cursor = "> "
			}

			// Format the row
			row := m.formatWorktreeRow(wt, i == m.cursor)
			b.WriteString(cursor + row + "\n")
		}
	}

	// Help
	b.WriteString(helpStyle.Render("↑/↓: navigate • r: refresh • q: quit"))

	return b.String()
}

// formatWorktreeRow formats a single worktree row for display.
func (m Model) formatWorktreeRow(wt git.Worktree, selected bool) string {
	// Branch name with main marker
	branch := wt.Branch
	if wt.IsMain {
		branch = mainMarkerStyle.Render("★ " + branch)
	}

	// Shorten path for display
	path := wt.Path
	home := filepath.Dir(filepath.Dir(m.config.Repo)) // Go up two levels for shorter paths
	if strings.HasPrefix(path, home) {
		path = "~" + strings.TrimPrefix(path, home)
	}

	// Apply styles based on selection
	if selected {
		if !wt.IsMain {
			branch = selectedStyle.Render(branch)
		}
		path = selectedStyle.Render(path)
	} else {
		if !wt.IsMain {
			branch = normalStyle.Render(branch)
		}
		path = pathStyle.Render(path)
	}

	return fmt.Sprintf("%-30s %s", branch, path)
}

// Run starts the TUI application.
func Run(cfg *config.Config) error {
	p := tea.NewProgram(New(cfg))
	_, err := p.Run()
	return err
}
