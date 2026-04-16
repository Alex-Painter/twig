// Package github provides functions for interacting with GitHub via the gh CLI.
package github

import (
	"fmt"
	"os/exec"
	"strings"
)

// CommandRunner is an interface for running commands.
// This allows for mocking in tests.
type CommandRunner interface {
	Run(name string, args ...string) (string, error)
}

// DefaultRunner is the default command runner that uses exec.Command.
type DefaultRunner struct {
	// WorkDir is the working directory for commands.
	WorkDir string
}

// Run executes a command and returns its output.
func (r DefaultRunner) Run(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	if r.WorkDir != "" {
		cmd.Dir = r.WorkDir
	}
	output, err := cmd.Output()
	return string(output), err
}

// Client provides GitHub operations via the gh CLI.
type Client struct {
	runner CommandRunner
}

// NewClient creates a new GitHub client with the default command runner.
func NewClient(repoPath string) *Client {
	return &Client{runner: DefaultRunner{WorkDir: repoPath}}
}

// NewClientWithRunner creates a new GitHub client with a custom command runner.
// This is useful for testing.
func NewClientWithRunner(runner CommandRunner) *Client {
	return &Client{runner: runner}
}

// GetPRBranch returns the branch name for a pull request.
func (c *Client) GetPRBranch(prNumber int) (string, error) {
	output, err := c.runner.Run("gh", "pr", "view", fmt.Sprintf("%d", prNumber), "--json", "headRefName", "-q", ".headRefName")
	if err != nil {
		return "", fmt.Errorf("failed to get PR branch: %w", err)
	}

	branch := strings.TrimSpace(output)
	if branch == "" {
		return "", fmt.Errorf("PR #%d not found or has no branch", prNumber)
	}

	return branch, nil
}

// IsAuthenticated checks if the gh CLI is authenticated.
func (c *Client) IsAuthenticated() bool {
	_, err := c.runner.Run("gh", "auth", "status")
	return err == nil
}

// IsInstalled checks if the gh CLI is installed.
func (c *Client) IsInstalled() bool {
	_, err := c.runner.Run("gh", "--version")
	return err == nil
}
