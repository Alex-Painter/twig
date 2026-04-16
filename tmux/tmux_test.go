package tmux

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

func TestSessionExists_True(t *testing.T) {
	mock := NewMockRunner()
	mock.SetResponse("tmux has-session -t test-session", "", nil)

	client := NewClientWithRunner(mock)
	if !client.SessionExists("test-session") {
		t.Error("expected SessionExists to return true")
	}
}

func TestSessionExists_False(t *testing.T) {
	mock := NewMockRunner()
	mock.SetResponse("tmux has-session -t nonexistent", "", errors.New("session not found"))

	client := NewClientWithRunner(mock)
	if client.SessionExists("nonexistent") {
		t.Error("expected SessionExists to return false")
	}
}

func TestIsAttached_True(t *testing.T) {
	mock := NewMockRunner()
	mock.SetResponse("tmux list-sessions -F #{session_name}:#{session_attached}",
		"test-session:1\nother-session:0\n", nil)

	client := NewClientWithRunner(mock)
	if !client.IsAttached("test-session") {
		t.Error("expected IsAttached to return true")
	}
}

func TestIsAttached_False(t *testing.T) {
	mock := NewMockRunner()
	mock.SetResponse("tmux list-sessions -F #{session_name}:#{session_attached}",
		"test-session:0\nother-session:1\n", nil)

	client := NewClientWithRunner(mock)
	if client.IsAttached("test-session") {
		t.Error("expected IsAttached to return false")
	}
}

func TestIsAttached_SessionNotFound(t *testing.T) {
	mock := NewMockRunner()
	mock.SetResponse("tmux list-sessions -F #{session_name}:#{session_attached}",
		"other-session:1\n", nil)

	client := NewClientWithRunner(mock)
	if client.IsAttached("nonexistent") {
		t.Error("expected IsAttached to return false for nonexistent session")
	}
}

func TestGetSessionStatus_None(t *testing.T) {
	mock := NewMockRunner()
	mock.SetResponse("tmux has-session -t test-session", "", errors.New("no session"))

	client := NewClientWithRunner(mock)
	status := client.GetSessionStatus("test-session")
	if status != SessionNone {
		t.Errorf("expected SessionNone, got %v", status)
	}
}

func TestGetSessionStatus_Detached(t *testing.T) {
	mock := NewMockRunner()
	mock.SetResponse("tmux has-session -t test-session", "", nil)
	mock.SetResponse("tmux list-sessions -F #{session_name}:#{session_attached}",
		"test-session:0\n", nil)

	client := NewClientWithRunner(mock)
	status := client.GetSessionStatus("test-session")
	if status != SessionDetached {
		t.Errorf("expected SessionDetached, got %v", status)
	}
}

func TestGetSessionStatus_Attached(t *testing.T) {
	mock := NewMockRunner()
	mock.SetResponse("tmux has-session -t test-session", "", nil)
	mock.SetResponse("tmux list-sessions -F #{session_name}:#{session_attached}",
		"test-session:1\n", nil)

	client := NewClientWithRunner(mock)
	status := client.GetSessionStatus("test-session")
	if status != SessionAttached {
		t.Errorf("expected SessionAttached, got %v", status)
	}
}

func TestListSessions(t *testing.T) {
	mock := NewMockRunner()
	mock.SetResponse("tmux list-sessions -F #{session_name}:#{session_attached}",
		"session-a:1\nsession-b:0\nsession-c:0\n", nil)

	client := NewClientWithRunner(mock)
	sessions := client.ListSessions()

	if len(sessions) != 3 {
		t.Fatalf("expected 3 sessions, got %d", len(sessions))
	}

	if sessions["session-a"] != SessionAttached {
		t.Errorf("expected session-a to be attached")
	}
	if sessions["session-b"] != SessionDetached {
		t.Errorf("expected session-b to be detached")
	}
	if sessions["session-c"] != SessionDetached {
		t.Errorf("expected session-c to be detached")
	}
}

func TestListSessions_NoSessions(t *testing.T) {
	mock := NewMockRunner()
	mock.SetResponse("tmux list-sessions -F #{session_name}:#{session_attached}",
		"", errors.New("no server running"))

	client := NewClientWithRunner(mock)
	sessions := client.ListSessions()

	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestSessionStatus_String(t *testing.T) {
	tests := []struct {
		status SessionStatus
		want   string
	}{
		{SessionNone, ""},
		{SessionDetached, "detached"},
		{SessionAttached, "attached"},
	}

	for _, tt := range tests {
		if got := tt.status.String(); got != tt.want {
			t.Errorf("SessionStatus(%d).String() = %q, want %q", tt.status, got, tt.want)
		}
	}
}
