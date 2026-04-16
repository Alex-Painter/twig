// Package tui provides the terminal user interface for twig.
package tui

import (
	"fmt"
	"strings"

	"github.com/Alex-Painter/twig/config"
	"github.com/Alex-Painter/twig/git"
	"github.com/Alex-Painter/twig/tmux"
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

	dirtyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("203"))

	pathStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	commitStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	timeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("239"))

	aheadStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("114"))

	behindStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("209"))

	unknownStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	sessionAttachedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("114"))

	sessionDetachedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("245"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)
)

// Model represents the TUI state.
type Model struct {
	config       *config.Config
	worktrees    []git.Worktree
	tmuxSessions map[string]tmux.SessionStatus
	tmuxClient   *tmux.Client
	cursor       int
	err          error
}

// worktreesMsg is sent when worktrees are loaded.
type worktreesMsg struct {
	worktrees    []git.Worktree
	tmuxSessions map[string]tmux.SessionStatus
	err          error
}

// New creates a new TUI model.
func New(cfg *config.Config) Model {
	return Model{
		config:     cfg,
		tmuxClient: tmux.NewClient(),
	}
}

// Init initializes the model and returns an initial command.
func (m Model) Init() tea.Cmd {
	return m.loadWorktrees
}

// loadWorktrees fetches the list of worktrees and tmux session status.
func (m Model) loadWorktrees() tea.Msg {
	worktrees, err := git.ListWorktrees(m.config.Repo)
	tmuxSessions := m.tmuxClient.ListSessions()
	return worktreesMsg{worktrees: worktrees, tmuxSessions: tmuxSessions, err: err}
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
		m.tmuxSessions = msg.tmuxSessions
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

	// Dirty indicator
	dirty := " "
	if wt.IsDirty {
		dirty = dirtyStyle.Render("*")
	}

	// Ahead/behind indicator
	aheadBehind := m.formatAheadBehind(wt, selected)

	// Last commit info - truncate message if too long
	commitMsg := wt.LastCommit.Message
	if len(commitMsg) > 30 {
		commitMsg = commitMsg[:27] + "..."
	}
	commitTime := wt.LastCommit.RelativeTime

	// Apply styles based on selection
	if selected {
		if !wt.IsMain {
			branch = selectedStyle.Render(branch)
		}
		commitMsg = selectedStyle.Render(commitMsg)
		commitTime = selectedStyle.Render(commitTime)
	} else {
		if !wt.IsMain {
			branch = normalStyle.Render(branch)
		}
		commitMsg = commitStyle.Render(commitMsg)
		commitTime = timeStyle.Render(commitTime)
	}

	// Tmux session status
	sessionStatus := m.formatSessionStatus(wt, selected)

	return fmt.Sprintf("%-30s %s %-8s %-10s %-32s %s", branch, dirty, aheadBehind, sessionStatus, commitMsg, commitTime)
}

// formatSessionStatus formats the tmux session status indicator.
func (m Model) formatSessionStatus(wt git.Worktree, selected bool) string {
	sessionName := m.config.SessionName(wt.Branch)
	status, exists := m.tmuxSessions[sessionName]

	if !exists {
		return ""
	}

	statusStr := status.String()
	if selected {
		return selectedStyle.Render(statusStr)
	}

	switch status {
	case tmux.SessionAttached:
		return sessionAttachedStyle.Render(statusStr)
	case tmux.SessionDetached:
		return sessionDetachedStyle.Render(statusStr)
	default:
		return ""
	}
}

// formatAheadBehind formats the ahead/behind indicator.
func (m Model) formatAheadBehind(wt git.Worktree, selected bool) string {
	if wt.Ahead == -1 || wt.Behind == -1 {
		if selected {
			return selectedStyle.Render("?")
		}
		return unknownStyle.Render("?")
	}

	var parts []string
	if wt.Ahead > 0 {
		s := fmt.Sprintf("↑%d", wt.Ahead)
		if selected {
			s = selectedStyle.Render(s)
		} else {
			s = aheadStyle.Render(s)
		}
		parts = append(parts, s)
	}
	if wt.Behind > 0 {
		s := fmt.Sprintf("↓%d", wt.Behind)
		if selected {
			s = selectedStyle.Render(s)
		} else {
			s = behindStyle.Render(s)
		}
		parts = append(parts, s)
	}

	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " ")
}

// Run starts the TUI application.
func Run(cfg *config.Config) error {
	p := tea.NewProgram(New(cfg))
	_, err := p.Run()
	return err
}
