// Package tmux provides functions for interacting with tmux sessions.
package tmux

import (
	"os/exec"
	"strings"
)

// SessionStatus represents the status of a tmux session.
type SessionStatus int

const (
	// SessionNone indicates no session exists.
	SessionNone SessionStatus = iota
	// SessionDetached indicates a session exists but is not attached.
	SessionDetached
	// SessionAttached indicates a session exists and is attached.
	SessionAttached
)

// String returns a human-readable string for the session status.
func (s SessionStatus) String() string {
	switch s {
	case SessionAttached:
		return "attached"
	case SessionDetached:
		return "detached"
	default:
		return ""
	}
}

// CommandRunner is an interface for running commands.
// This allows for mocking in tests.
type CommandRunner interface {
	Run(name string, args ...string) (string, error)
}

// DefaultRunner is the default command runner that uses exec.Command.
type DefaultRunner struct{}

// Run executes a command and returns its output.
func (r DefaultRunner) Run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.Output()
	return string(output), err
}

// Client provides tmux operations.
type Client struct {
	runner CommandRunner
}

// NewClient creates a new tmux client with the default command runner.
func NewClient() *Client {
	return &Client{runner: DefaultRunner{}}
}

// NewClientWithRunner creates a new tmux client with a custom command runner.
// This is useful for testing.
func NewClientWithRunner(runner CommandRunner) *Client {
	return &Client{runner: runner}
}

// SessionExists returns true if a tmux session with the given name exists.
func (c *Client) SessionExists(sessionName string) bool {
	_, err := c.runner.Run("tmux", "has-session", "-t", sessionName)
	return err == nil
}

// IsAttached returns true if the session is currently attached.
func (c *Client) IsAttached(sessionName string) bool {
	// List sessions with format that shows attached status
	output, err := c.runner.Run("tmux", "list-sessions", "-F", "#{session_name}:#{session_attached}")
	if err != nil {
		return false
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 && parts[0] == sessionName {
			return parts[1] == "1"
		}
	}
	return false
}

// GetSessionStatus returns the status of a tmux session.
func (c *Client) GetSessionStatus(sessionName string) SessionStatus {
	if !c.SessionExists(sessionName) {
		return SessionNone
	}
	if c.IsAttached(sessionName) {
		return SessionAttached
	}
	return SessionDetached
}

// ListSessions returns a map of session names to their attached status.
// This is more efficient than calling GetSessionStatus for each session.
func (c *Client) ListSessions() map[string]SessionStatus {
	result := make(map[string]SessionStatus)

	output, err := c.runner.Run("tmux", "list-sessions", "-F", "#{session_name}:#{session_attached}")
	if err != nil {
		return result
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			if parts[1] == "1" {
				result[parts[0]] = SessionAttached
			} else {
				result[parts[0]] = SessionDetached
			}
		}
	}
	return result
}

// CreateSession creates a new tmux session with the given name and windows.
// The session starts in the specified working directory.
// The first window is created with the session, additional windows are added after.
func (c *Client) CreateSession(sessionName string, windows []string, workdir string) error {
	if len(windows) == 0 {
		windows = []string{"shell"}
	}

	// Create session with first window
	_, err := c.runner.Run("tmux", "new-session", "-d", "-s", sessionName, "-n", windows[0], "-c", workdir)
	if err != nil {
		return err
	}

	// Create additional windows
	for _, windowName := range windows[1:] {
		_, err := c.runner.Run("tmux", "new-window", "-t", sessionName, "-n", windowName, "-c", workdir)
		if err != nil {
			return err
		}
	}

	// Select first window
	_, err = c.runner.Run("tmux", "select-window", "-t", sessionName+":"+windows[0])
	return err
}
