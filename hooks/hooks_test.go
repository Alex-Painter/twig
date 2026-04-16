package hooks

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRunPostCreate_NoHook(t *testing.T) {
	runner := NewRunner(t.TempDir())
	result := runner.RunPostCreate("/some/path")

	if result.Executed {
		t.Error("expected Executed to be false when hook doesn't exist")
	}
	if result.Error != nil {
		t.Errorf("expected no error, got %v", result.Error)
	}
}

func TestRunPostCreate_HookExists(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	hooksDir := t.TempDir()
	worktreeDir := t.TempDir()

	// Create a simple hook script
	hookPath := filepath.Join(hooksDir, "post-create.sh")
	hookContent := `#!/bin/bash
echo "Hook ran with arg: $1"
echo "In directory: $(pwd)"
`
	if err := os.WriteFile(hookPath, []byte(hookContent), 0755); err != nil {
		t.Fatalf("failed to create hook: %v", err)
	}

	runner := NewRunner(hooksDir)
	result := runner.RunPostCreate(worktreeDir)

	if !result.Executed {
		t.Error("expected Executed to be true")
	}
	if result.Error != nil {
		t.Errorf("expected no error, got %v", result.Error)
	}
	if result.Output == "" {
		t.Error("expected output from hook")
	}
}

func TestRunPostCreate_HookFails(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping on windows")
	}

	hooksDir := t.TempDir()
	worktreeDir := t.TempDir()

	// Create a hook that fails
	hookPath := filepath.Join(hooksDir, "post-create.sh")
	hookContent := `#!/bin/bash
echo "About to fail"
exit 1
`
	if err := os.WriteFile(hookPath, []byte(hookContent), 0755); err != nil {
		t.Fatalf("failed to create hook: %v", err)
	}

	runner := NewRunner(hooksDir)
	result := runner.RunPostCreate(worktreeDir)

	if !result.Executed {
		t.Error("expected Executed to be true")
	}
	if result.Error == nil {
		t.Error("expected error from failed hook")
	}
	if result.Output == "" {
		t.Error("expected output from hook even on failure")
	}
}

func TestHookResult_FormatResult(t *testing.T) {
	tests := []struct {
		name   string
		result HookResult
		want   string
	}{
		{
			name:   "not executed",
			result: HookResult{Executed: false},
			want:   "",
		},
		{
			name:   "success with output",
			result: HookResult{Executed: true, Output: "done\n"},
			want:   "Post-create hook completed:\ndone\n",
		},
		{
			name:   "success no output",
			result: HookResult{Executed: true, Output: ""},
			want:   "Post-create hook completed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.FormatResult()
			if got != tt.want {
				t.Errorf("FormatResult() = %q, want %q", got, tt.want)
			}
		})
	}
}
