package github

import (
	"errors"
	"testing"
)

// MockRunner is a mock implementation of CommandRunner for testing.
type MockRunner struct {
	responses map[string]mockResponse
}

type mockResponse struct {
	output string
	err    error
}

func NewMockRunner() *MockRunner {
	return &MockRunner{
		responses: make(map[string]mockResponse),
	}
}

func (m *MockRunner) SetResponse(cmd string, output string, err error) {
	m.responses[cmd] = mockResponse{output: output, err: err}
}

func (m *MockRunner) Run(name string, args ...string) (string, error) {
	// Build a key from the command
	key := name
	for _, arg := range args {
		key += " " + arg
	}

	if resp, ok := m.responses[key]; ok {
		return resp.output, resp.err
	}
	return "", errors.New("command not mocked: " + key)
}

func TestGetPRBranch_Success(t *testing.T) {
	mock := NewMockRunner()
	mock.SetResponse("gh pr view 123 --json headRefName -q .headRefName", "feature-branch\n", nil)

	client := NewClientWithRunner(mock)
	branch, err := client.GetPRBranch(123)
	if err != nil {
		t.Fatalf("GetPRBranch() returned error: %v", err)
	}

	if branch != "feature-branch" {
		t.Errorf("branch = %q, want %q", branch, "feature-branch")
	}
}

func TestGetPRBranch_NotFound(t *testing.T) {
	mock := NewMockRunner()
	mock.SetResponse("gh pr view 999 --json headRefName -q .headRefName",
		"", errors.New("Could not resolve to a PullRequest"))

	client := NewClientWithRunner(mock)
	_, err := client.GetPRBranch(999)
	if err == nil {
		t.Error("expected GetPRBranch() to return error for non-existent PR")
	}
}

func TestGetPRBranch_EmptyBranch(t *testing.T) {
	mock := NewMockRunner()
	mock.SetResponse("gh pr view 123 --json headRefName -q .headRefName", "\n", nil)

	client := NewClientWithRunner(mock)
	_, err := client.GetPRBranch(123)
	if err == nil {
		t.Error("expected GetPRBranch() to return error for empty branch")
	}
}

func TestIsAuthenticated_True(t *testing.T) {
	mock := NewMockRunner()
	mock.SetResponse("gh auth status", "Logged in to github.com\n", nil)

	client := NewClientWithRunner(mock)
	if !client.IsAuthenticated() {
		t.Error("expected IsAuthenticated() to return true")
	}
}

func TestIsAuthenticated_False(t *testing.T) {
	mock := NewMockRunner()
	mock.SetResponse("gh auth status", "", errors.New("not logged in"))

	client := NewClientWithRunner(mock)
	if client.IsAuthenticated() {
		t.Error("expected IsAuthenticated() to return false")
	}
}

func TestIsInstalled_True(t *testing.T) {
	mock := NewMockRunner()
	mock.SetResponse("gh --version", "gh version 2.0.0\n", nil)

	client := NewClientWithRunner(mock)
	if !client.IsInstalled() {
		t.Error("expected IsInstalled() to return true")
	}
}

func TestIsInstalled_False(t *testing.T) {
	mock := NewMockRunner()
	mock.SetResponse("gh --version", "", errors.New("command not found"))

	client := NewClientWithRunner(mock)
	if client.IsInstalled() {
		t.Error("expected IsInstalled() to return false")
	}
}
