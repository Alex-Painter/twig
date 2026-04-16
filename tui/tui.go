// Package tui provides the terminal user interface for twig.
package tui

import (
	"fmt"
	"strings"

	"github.com/Alex-Painter/twig/config"
	"github.com/Alex-Painter/twig/git"
	"github.com/Alex-Painter/twig/tmux"
	"github.com/Alex-Painter/twig/worktree"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// viewMode represents the current mode of the TUI.
type viewMode int

const (
	modeList viewMode = iota
	modeCreate
	modeDeleteConfirm
	modeDeleteForceConfirm
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

	inputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212"))

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("114"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("203"))
)

// Model represents the TUI state.
type Model struct {
	config          *config.Config
	worktrees       []git.Worktree
	tmuxSessions    map[string]tmux.SessionStatus
	tmuxClient      *tmux.Client
	worktreeManager *worktree.Manager
	cursor          int
	err             error
	mode            viewMode
	input           string
	statusMessage   string
	statusIsError   bool
	busy            bool
	busyMessage     string
	spinner         spinner.Model
}

// worktreesMsg is sent when worktrees are loaded.
type worktreesMsg struct {
	worktrees    []git.Worktree
	tmuxSessions map[string]tmux.SessionStatus
	err          error
}

// createResultMsg is sent when worktree creation completes.
type createResultMsg struct {
	branchName string
	result     worktree.CreateResult
	err        error
}

// deleteResultMsg is sent when worktree deletion completes.
type deleteResultMsg struct {
	branch string
	err    error
}

// New creates a new TUI model.
func New(cfg *config.Config) Model {
	tmuxClient := tmux.NewClient()

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

	return Model{
		config:          cfg,
		tmuxClient:      tmuxClient,
		worktreeManager: worktree.NewManager(cfg, tmuxClient),
		mode:            modeList,
		spinner:         s,
	}
}

// Init initializes the model and returns an initial command.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.loadWorktrees, m.spinner.Tick)
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
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		// Busy state: ignore all input except ctrl+c
		if m.busy {
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			return m, nil
		}

		// Handle input mode
		if m.mode == modeCreate {
			return m.handleCreateInput(msg)
		}

		// Handle delete confirmation modes
		if m.mode == modeDeleteConfirm || m.mode == modeDeleteForceConfirm {
			return m.handleDeleteConfirm(msg)
		}

		// Handle list mode
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
			m.statusMessage = ""
			m.busy = true
			m.busyMessage = "Refreshing..."
			return m, m.loadWorktrees
		case "n":
			m.mode = modeCreate
			m.input = ""
			m.statusMessage = ""
		case "d":
			if len(m.worktrees) == 0 {
				return m, nil
			}
			wt := m.worktrees[m.cursor]
			if wt.IsMain {
				m.statusMessage = "Cannot delete main clone"
				m.statusIsError = true
				return m, nil
			}
			if wt.IsDirty {
				m.mode = modeDeleteForceConfirm
			} else {
				m.mode = modeDeleteConfirm
			}
			m.statusMessage = ""
		}

	case worktreesMsg:
		m.worktrees = msg.worktrees
		m.tmuxSessions = msg.tmuxSessions
		m.err = msg.err
		m.busy = false
		m.busyMessage = ""
		// Reset cursor if out of bounds
		if m.cursor >= len(m.worktrees) {
			m.cursor = max(0, len(m.worktrees)-1)
		}

	case createResultMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Failed to create worktree: %v", msg.err)
			m.statusIsError = true
		} else {
			// Build success message
			m.statusMessage = fmt.Sprintf("Created worktree for '%s'", msg.branchName)

			// Add hook result if any
			hookMsg := msg.result.HookResult.FormatResult()
			if hookMsg != "" {
				m.statusMessage += "\n" + hookMsg
			}

			// Check if hook failed
			if msg.result.HookResult.Error != nil {
				m.statusIsError = true
			} else {
				m.statusIsError = false
			}
		}
		m.mode = modeList
		m.busyMessage = "Refreshing..."
		return m, m.loadWorktrees

	case deleteResultMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Failed to delete worktree: %v", msg.err)
			m.statusIsError = true
		} else {
			m.statusMessage = fmt.Sprintf("Deleted worktree '%s'", msg.branch)
			m.statusIsError = false
		}
		m.mode = modeList
		m.busyMessage = "Refreshing..."
		return m, m.loadWorktrees
	}

	return m, nil
}

// handleDeleteConfirm handles keyboard input in delete confirmation mode.
func (m Model) handleDeleteConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		if m.cursor >= len(m.worktrees) {
			m.mode = modeList
			return m, nil
		}
		wt := m.worktrees[m.cursor]
		force := m.mode == modeDeleteForceConfirm
		m.mode = modeList
		m.busy = true
		m.busyMessage = fmt.Sprintf("Deleting worktree '%s'...", wt.Branch)
		return m, m.deleteWorktree(wt, force)

	case "n", "N", "esc", "q":
		m.mode = modeList
		return m, nil
	}
	return m, nil
}

// deleteWorktree returns a command that deletes a worktree.
func (m Model) deleteWorktree(wt git.Worktree, force bool) tea.Cmd {
	return func() tea.Msg {
		err := m.worktreeManager.Delete(wt, force)
		return deleteResultMsg{branch: wt.Branch, err: err}
	}
}

// handleCreateInput handles keyboard input in create mode.
func (m Model) handleCreateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		if m.input == "" {
			m.mode = modeList
			return m, nil
		}
		branchName := m.input
		m.input = ""
		m.mode = modeList
		m.busy = true
		m.busyMessage = fmt.Sprintf("Creating worktree for '%s'...", branchName)
		return m, m.createWorktree(branchName)

	case tea.KeyEsc:
		m.mode = modeList
		m.input = ""
		return m, nil

	case tea.KeyBackspace:
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}
		return m, nil

	case tea.KeyRunes:
		m.input += string(msg.Runes)
		return m, nil

	case tea.KeySpace:
		m.input += " "
		return m, nil
	}

	return m, nil
}

// createWorktree returns a command that creates a worktree.
func (m Model) createWorktree(branchName string) tea.Cmd {
	return func() tea.Msg {
		result, err := m.worktreeManager.Create(branchName)
		return createResultMsg{branchName: branchName, result: result, err: err}
	}
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

	// Busy indicator (spinner + message)
	if m.busyMessage != "" {
		b.WriteString(m.spinner.View())
		b.WriteString(" ")
		b.WriteString(promptStyle.Render(m.busyMessage))
		b.WriteString("\n\n")
	}

	// Status message
	if m.statusMessage != "" {
		if m.statusIsError {
			b.WriteString(errorStyle.Render(m.statusMessage))
		} else {
			b.WriteString(successStyle.Render(m.statusMessage))
		}
		b.WriteString("\n\n")
	}

	// Create mode input
	if m.mode == modeCreate {
		b.WriteString(promptStyle.Render("Branch name (or #PR): "))
		b.WriteString(inputStyle.Render(m.input))
		b.WriteString(inputStyle.Render("█"))
		b.WriteString("\n\n")
		b.WriteString(helpStyle.Render("Enter: create • Esc: cancel"))
		return b.String()
	}

	// Delete confirmation
	if m.mode == modeDeleteConfirm || m.mode == modeDeleteForceConfirm {
		if m.cursor < len(m.worktrees) {
			wt := m.worktrees[m.cursor]
			if m.mode == modeDeleteForceConfirm {
				b.WriteString(errorStyle.Render(fmt.Sprintf("WARNING: '%s' has uncommitted changes!", wt.Branch)))
				b.WriteString("\n")
				b.WriteString(promptStyle.Render(fmt.Sprintf("Force delete worktree '%s'? [y/N]", wt.Branch)))
			} else {
				b.WriteString(promptStyle.Render(fmt.Sprintf("Delete worktree '%s'? [y/N]", wt.Branch)))
			}
			b.WriteString("\n\n")
			b.WriteString(helpStyle.Render("y: confirm • n/Esc: cancel"))
		}
		return b.String()
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
	b.WriteString(helpStyle.Render("↑/↓: navigate • n: new • d: delete • r: refresh • q: quit"))

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
