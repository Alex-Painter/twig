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
	modeFilter
	modeHelp
)

// bgColor is the slate-black background used throughout the TUI.
const bgColor = lipgloss.Color("235")

// Styles for the TUI
var (
	baseStyle = lipgloss.NewStyle().
			Background(bgColor).
			Foreground(lipgloss.Color("252"))

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Background(bgColor).
			Foreground(lipgloss.Color("39")).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Background(bgColor).
			Foreground(lipgloss.Color("39")).
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			Background(bgColor).
			Foreground(lipgloss.Color("252"))

	mainMarkerStyle = lipgloss.NewStyle().
			Background(bgColor).
			Foreground(lipgloss.Color("214")).
			Bold(true)

	dirtyStyle = lipgloss.NewStyle().
			Background(bgColor).
			Foreground(lipgloss.Color("203"))

	pathStyle = lipgloss.NewStyle().
			Background(bgColor).
			Foreground(lipgloss.Color("243"))

	commitStyle = lipgloss.NewStyle().
			Background(bgColor).
			Foreground(lipgloss.Color("245"))

	timeStyle = lipgloss.NewStyle().
			Background(bgColor).
			Foreground(lipgloss.Color("239"))

	aheadStyle = lipgloss.NewStyle().
			Background(bgColor).
			Foreground(lipgloss.Color("114"))

	behindStyle = lipgloss.NewStyle().
			Background(bgColor).
			Foreground(lipgloss.Color("209"))

	unknownStyle = lipgloss.NewStyle().
			Background(bgColor).
			Foreground(lipgloss.Color("243"))

	sessionAttachedStyle = lipgloss.NewStyle().
				Background(bgColor).
				Foreground(lipgloss.Color("114"))

	sessionDetachedStyle = lipgloss.NewStyle().
				Background(bgColor).
				Foreground(lipgloss.Color("245"))

	helpStyle = lipgloss.NewStyle().
			Background(bgColor).
			Foreground(lipgloss.Color("241")).
			MarginTop(1)

	inputStyle = lipgloss.NewStyle().
			Background(bgColor).
			Foreground(lipgloss.Color("39"))

	promptStyle = lipgloss.NewStyle().
			Background(bgColor).
			Foreground(lipgloss.Color("252"))

	successStyle = lipgloss.NewStyle().
			Background(bgColor).
			Foreground(lipgloss.Color("114"))

	errorStyle = lipgloss.NewStyle().
			Background(bgColor).
			Foreground(lipgloss.Color("203"))

	modalStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("39")).
			Background(bgColor).
			Padding(1, 2)

	keyStyle = lipgloss.NewStyle().
			Background(bgColor).
			Foreground(lipgloss.Color("39")).
			Bold(true)
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
	filter          string
	width           int
	height          int
	// lastAction stores the last action that failed, for retry support.
	// Nil if no retryable error is present.
	lastAction retryableAction
}

// retryableAction represents an action that can be retried after failure.
type retryableAction struct {
	// description is shown in the UI (e.g., "create worktree 'foo'").
	description string
	// cmd is the command to re-run.
	cmd tea.Cmd
	// busyMessage is the message to show while retrying.
	busyMessage string
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

// fetchResultMsg is sent when fetch completes.
type fetchResultMsg struct {
	err error
}

// pullResultMsg is sent when pull completes.
type pullResultMsg struct {
	branch string
	err    error
}

// New creates a new TUI model.
func New(cfg *config.Config) Model {
	tmuxClient := tmux.NewClient()

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Background(bgColor).Foreground(lipgloss.Color("39"))

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

// filteredWorktrees returns the worktrees filtered by the current filter string.
// If filter is empty, returns all worktrees.
func (m Model) filteredWorktrees() []git.Worktree {
	if m.filter == "" {
		return m.worktrees
	}
	filter := strings.ToLower(m.filter)
	var result []git.Worktree
	for _, wt := range m.worktrees {
		if strings.Contains(strings.ToLower(wt.Branch), filter) {
			result = append(result, wt)
		}
	}
	return result
}

// Update handles messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

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

		// Handle filter mode
		if m.mode == modeFilter {
			return m.handleFilterInput(msg)
		}

		// Handle help mode - any key dismisses
		if m.mode == modeHelp {
			// ctrl+c should still quit
			if msg.String() == "ctrl+c" {
				return m, tea.Quit
			}
			m.mode = modeList
			return m, nil
		}

		// Handle retry/dismiss when there's a failed action
		if m.lastAction.cmd != nil {
			switch msg.String() {
			case "R":
				// Retry the last action
				action := m.lastAction
				m.lastAction = retryableAction{}
				m.statusMessage = ""
				m.busy = true
				m.busyMessage = action.busyMessage
				return m, action.cmd
			case "esc", "x":
				// Dismiss error
				m.lastAction = retryableAction{}
				m.statusMessage = ""
				return m, nil
			}
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
			if m.cursor < len(m.filteredWorktrees())-1 {
				m.cursor++
			}
		case "r":
			m.statusMessage = ""
			m.lastAction = retryableAction{}
			m.busy = true
			m.busyMessage = "Refreshing..."
			return m, m.loadWorktrees
		case "f":
			m.statusMessage = ""
			m.busy = true
			m.busyMessage = "Fetching all remotes..."
			return m, m.fetchAll
		case "p":
			wts := m.filteredWorktrees()
			if len(wts) == 0 {
				return m, nil
			}
			wt := wts[m.cursor]
			m.statusMessage = ""
			m.busy = true
			m.busyMessage = fmt.Sprintf("Pulling '%s'...", wt.Branch)
			return m, m.pullWorktree(wt)
		case "/":
			m.mode = modeFilter
			m.statusMessage = ""
		case "?":
			m.mode = modeHelp
			m.statusMessage = ""
		case "n":
			m.mode = modeCreate
			m.input = ""
			m.statusMessage = ""
		case "d":
			wts := m.filteredWorktrees()
			if len(wts) == 0 {
				return m, nil
			}
			wt := wts[m.cursor]
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
		// Reset cursor if out of bounds (based on filtered results)
		filtered := m.filteredWorktrees()
		if m.cursor >= len(filtered) {
			m.cursor = max(0, len(filtered)-1)
		}

	case createResultMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Failed to create worktree: %v", msg.err)
			m.statusIsError = true
			m.lastAction = retryableAction{
				description: fmt.Sprintf("create '%s'", msg.branchName),
				cmd:         m.createWorktree(msg.branchName),
				busyMessage: fmt.Sprintf("Creating worktree for '%s'...", msg.branchName),
			}
			m.busy = false
			m.busyMessage = ""
			m.mode = modeList
			return m, nil
		}

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
		m.mode = modeList
		m.busyMessage = "Refreshing..."
		return m, m.loadWorktrees

	case deleteResultMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Failed to delete worktree: %v", msg.err)
			m.statusIsError = true
			// Find the worktree by branch to build retry command
			for _, wt := range m.worktrees {
				if wt.Branch == msg.branch {
					m.lastAction = retryableAction{
						description: fmt.Sprintf("delete '%s'", msg.branch),
						cmd:         m.deleteWorktree(wt, true), // force on retry since first attempt failed
						busyMessage: fmt.Sprintf("Deleting worktree '%s'...", msg.branch),
					}
					break
				}
			}
			m.busy = false
			m.busyMessage = ""
			m.mode = modeList
			return m, nil
		}
		m.statusMessage = fmt.Sprintf("Deleted worktree '%s'", msg.branch)
		m.statusIsError = false
		m.mode = modeList
		m.busyMessage = "Refreshing..."
		return m, m.loadWorktrees

	case fetchResultMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Failed to fetch: %v", msg.err)
			m.statusIsError = true
			m.lastAction = retryableAction{
				description: "fetch all remotes",
				cmd:         m.fetchAll,
				busyMessage: "Fetching all remotes...",
			}
			m.busy = false
			m.busyMessage = ""
			return m, nil
		}
		m.statusMessage = "Fetched all remotes"
		m.statusIsError = false
		m.busyMessage = "Refreshing..."
		return m, m.loadWorktrees

	case pullResultMsg:
		if msg.err != nil {
			m.statusMessage = fmt.Sprintf("Failed to pull: %v", msg.err)
			m.statusIsError = true
			// Find worktree to retry
			for _, wt := range m.worktrees {
				if wt.Branch == msg.branch {
					m.lastAction = retryableAction{
						description: fmt.Sprintf("pull '%s'", msg.branch),
						cmd:         m.pullWorktree(wt),
						busyMessage: fmt.Sprintf("Pulling '%s'...", msg.branch),
					}
					break
				}
			}
			m.busy = false
			m.busyMessage = ""
			return m, nil
		}
		m.statusMessage = fmt.Sprintf("Pulled '%s'", msg.branch)
		m.statusIsError = false
		m.busyMessage = "Refreshing..."
		return m, m.loadWorktrees
	}

	return m, nil
}

// handleDeleteConfirm handles keyboard input in delete confirmation mode.
func (m Model) handleDeleteConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		wts := m.filteredWorktrees()
		if m.cursor >= len(wts) {
			m.mode = modeList
			return m, nil
		}
		wt := wts[m.cursor]
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

// handleFilterInput handles keyboard input in filter mode.
func (m Model) handleFilterInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter, tea.KeyEsc:
		m.mode = modeList
		// Reset cursor if now out of bounds
		filtered := m.filteredWorktrees()
		if m.cursor >= len(filtered) {
			m.cursor = max(0, len(filtered)-1)
		}
		return m, nil

	case tea.KeyBackspace:
		if len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
			// Reset cursor since filtered list may change
			m.cursor = 0
		}
		return m, nil

	case tea.KeyRunes:
		m.filter += string(msg.Runes)
		m.cursor = 0
		return m, nil

	case tea.KeySpace:
		m.filter += " "
		m.cursor = 0
		return m, nil
	}

	// ctrl+u to clear filter
	if msg.String() == "ctrl+u" {
		m.filter = ""
		m.cursor = 0
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

// fetchAll fetches all remotes.
func (m Model) fetchAll() tea.Msg {
	err := git.FetchAll(m.config.Repo)
	return fetchResultMsg{err: err}
}

// pullWorktree pulls the given worktree.
func (m Model) pullWorktree(wt git.Worktree) tea.Cmd {
	return func() tea.Msg {
		err := git.Pull(wt.Path)
		return pullResultMsg{branch: wt.Branch, err: err}
	}
}

// renderHelp renders the help modal.
func (m Model) renderHelp() string {
	keys := []struct {
		key  string
		desc string
	}{
		{"↑/k", "Move cursor up"},
		{"↓/j", "Move cursor down"},
		{"n", "Create new worktree (branch name or #PR)"},
		{"d", "Delete selected worktree"},
		{"f", "Fetch all remotes"},
		{"p", "Pull selected worktree"},
		{"/", "Filter worktrees by branch name"},
		{"r", "Refresh list"},
		{"?", "Show this help"},
		{"q", "Quit"},
	}

	var content strings.Builder
	content.WriteString(titleStyle.Render("Keybindings"))
	content.WriteString("\n")
	for _, k := range keys {
		content.WriteString(fmt.Sprintf("  %s  %s\n", keyStyle.Render(fmt.Sprintf("%-5s", k.key)), k.desc))
	}
	content.WriteString("\n")
	content.WriteString(helpStyle.Render("Press any key to close"))

	return modalStyle.Render(content.String())
}

// View renders the TUI.
func (m Model) View() string {
	return m.wrapBackground(m.renderContent())
}

// wrapBackground wraps content in the base background style, padding each line
// to the terminal width so the background extends across the full screen.
func (m Model) wrapBackground(content string) string {
	if m.width == 0 {
		// Before WindowSizeMsg arrives, just render without padding
		return baseStyle.Render(content)
	}

	lines := strings.Split(content, "\n")
	var out strings.Builder
	for i, line := range lines {
		// Pad each line to the full width with background
		padded := baseStyle.Width(m.width).Render(line)
		out.WriteString(padded)
		if i < len(lines)-1 {
			out.WriteString("\n")
		}
	}
	return out.String()
}

// renderContent builds the unstyled view body.
func (m Model) renderContent() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("twig - Git Worktree Manager"))
	b.WriteString("\n")

	// Error display
	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %v", m.err)))
		b.WriteString("\n\n")
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
		b.WriteString("\n")

		// Retry/dismiss hint
		if m.lastAction.cmd != nil {
			b.WriteString(helpStyle.Render("R: retry • Esc/x: dismiss"))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Help modal
	if m.mode == modeHelp {
		b.WriteString(m.renderHelp())
		return b.String()
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

	// Filter input line (above list when in filter mode or when filter is active)
	if m.mode == modeFilter {
		b.WriteString(promptStyle.Render("Filter: "))
		b.WriteString(inputStyle.Render(m.filter))
		b.WriteString(inputStyle.Render("█"))
		b.WriteString("\n")
	} else if m.filter != "" {
		b.WriteString(promptStyle.Render("Filter: "))
		b.WriteString(inputStyle.Render(m.filter))
		b.WriteString(helpStyle.Render("  (press / to edit)"))
		b.WriteString("\n")
	}

	// Worktree list (filtered)
	filtered := m.filteredWorktrees()
	if len(m.worktrees) == 0 && m.err == nil {
		b.WriteString(normalStyle.Render("No worktrees found."))
		b.WriteString("\n")
	} else if len(filtered) == 0 {
		b.WriteString(helpStyle.Render("No worktrees match filter."))
		b.WriteString("\n")
	} else {
		for i, wt := range filtered {
			cursor := "  "
			if i == m.cursor {
				cursor = "> "
			}

			// Format the row
			row := m.formatWorktreeRow(wt, i == m.cursor)
			b.WriteString(normalStyle.Render(cursor))
			b.WriteString(row)
			b.WriteString("\n")
		}
	}

	// Help
	if m.mode == modeFilter {
		b.WriteString(helpStyle.Render("type to filter • ctrl+u: clear • Enter/Esc: done"))
	} else {
		b.WriteString(helpStyle.Render("?: help • /: filter • n: new • d: delete • f: fetch • p: pull • r: refresh • q: quit"))
	}

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
