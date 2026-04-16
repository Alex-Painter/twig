// Package hooks provides functionality for running lifecycle hooks.
package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Runner handles hook execution.
type Runner struct {
	// HooksDir is the directory containing hook scripts.
	HooksDir string
}

// NewRunner creates a new hook runner.
func NewRunner(hooksDir string) *Runner {
	return &Runner{HooksDir: hooksDir}
}

// HookResult contains the result of running a hook.
type HookResult struct {
	// Executed indicates if the hook was found and executed.
	Executed bool
	// Output contains the combined stdout/stderr of the hook.
	Output string
	// Error contains any error that occurred.
	Error error
}

// RunPostCreate runs the post-create hook if it exists.
// The hook is executed in the worktree directory with the worktree path as an argument.
func (r *Runner) RunPostCreate(worktreePath string) HookResult {
	hookPath := filepath.Join(r.HooksDir, "post-create.sh")

	// Check if hook exists
	if _, err := os.Stat(hookPath); os.IsNotExist(err) {
		return HookResult{Executed: false}
	}

	// Execute the hook
	cmd := exec.Command(hookPath, worktreePath)
	cmd.Dir = worktreePath

	output, err := cmd.CombinedOutput()

	return HookResult{
		Executed: true,
		Output:   string(output),
		Error:    err,
	}
}

// FormatResult returns a human-readable string describing the hook result.
func (r HookResult) FormatResult() string {
	if !r.Executed {
		return ""
	}

	if r.Error != nil {
		return fmt.Sprintf("Post-create hook failed: %v\nOutput:\n%s", r.Error, r.Output)
	}

	if r.Output != "" {
		return fmt.Sprintf("Post-create hook completed:\n%s", r.Output)
	}

	return "Post-create hook completed"
}
