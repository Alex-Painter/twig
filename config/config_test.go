package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ValidConfig(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	configContent := `
repo = "/path/to/repo"
worktree_dir = "/path/to/worktrees"
session_pattern = "custom-{repo}-{branch}"
windows = ["code", "server", "term"]
hooks_dir = "/path/to/hooks"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Repo != "/path/to/repo" {
		t.Errorf("Repo = %q, want %q", cfg.Repo, "/path/to/repo")
	}
	if cfg.WorktreeDir != "/path/to/worktrees" {
		t.Errorf("WorktreeDir = %q, want %q", cfg.WorktreeDir, "/path/to/worktrees")
	}
	if cfg.SessionPattern != "custom-{repo}-{branch}" {
		t.Errorf("SessionPattern = %q, want %q", cfg.SessionPattern, "custom-{repo}-{branch}")
	}
	if len(cfg.Windows) != 3 || cfg.Windows[0] != "code" {
		t.Errorf("Windows = %v, want [code server term]", cfg.Windows)
	}
	if cfg.HooksDir != "/path/to/hooks" {
		t.Errorf("HooksDir = %q, want %q", cfg.HooksDir, "/path/to/hooks")
	}
}

func TestLoad_Defaults(t *testing.T) {
	// Create a minimal config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	configContent := `
repo = "/path/to/repo"
worktree_dir = "/path/to/worktrees"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	// Check defaults were applied
	if cfg.SessionPattern != "{repo}-{branch}" {
		t.Errorf("SessionPattern = %q, want default %q", cfg.SessionPattern, "{repo}-{branch}")
	}
	if len(cfg.Windows) != 3 || cfg.Windows[0] != "editor" || cfg.Windows[1] != "dev" || cfg.Windows[2] != "shell" {
		t.Errorf("Windows = %v, want default [editor dev shell]", cfg.Windows)
	}
	// HooksDir should default to ~/.config/twig/hooks
	home, _ := os.UserHomeDir()
	expectedHooksDir := filepath.Join(home, ".config", "twig", "hooks")
	if cfg.HooksDir != expectedHooksDir {
		t.Errorf("HooksDir = %q, want default %q", cfg.HooksDir, expectedHooksDir)
	}
}

func TestLoad_MissingRepo(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	configContent := `
worktree_dir = "/path/to/worktrees"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Load() should return error when repo is missing")
	}
}

func TestLoad_MissingWorktreeDir(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	configContent := `
repo = "/path/to/repo"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Load() should return error when worktree_dir is missing")
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.toml")
	if err == nil {
		t.Fatal("Load() should return error when config file doesn't exist")
	}
}

func TestLoad_InvalidTOML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	configContent := `this is not valid toml {{{`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	_, err := Load(configPath)
	if err == nil {
		t.Fatal("Load() should return error for invalid TOML")
	}
}

func TestLoad_ExpandsHomePath(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.toml")

	configContent := `
repo = "~/workspace/myrepo"
worktree_dir = "~/workspace/worktrees"
hooks_dir = "~/myhooks"
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	home, _ := os.UserHomeDir()

	expectedRepo := filepath.Join(home, "workspace/myrepo")
	if cfg.Repo != expectedRepo {
		t.Errorf("Repo = %q, want %q", cfg.Repo, expectedRepo)
	}

	expectedWorktreeDir := filepath.Join(home, "workspace/worktrees")
	if cfg.WorktreeDir != expectedWorktreeDir {
		t.Errorf("WorktreeDir = %q, want %q", cfg.WorktreeDir, expectedWorktreeDir)
	}

	expectedHooksDir := filepath.Join(home, "myhooks")
	if cfg.HooksDir != expectedHooksDir {
		t.Errorf("HooksDir = %q, want %q", cfg.HooksDir, expectedHooksDir)
	}
}

func TestRepoName(t *testing.T) {
	cfg := &Config{
		Repo: "/Users/alex/workspace/maverick",
	}

	if got := cfg.RepoName(); got != "maverick" {
		t.Errorf("RepoName() = %q, want %q", got, "maverick")
	}
}

func TestSessionName(t *testing.T) {
	tests := []struct {
		name           string
		sessionPattern string
		repo           string
		branch         string
		want           string
	}{
		{
			name:           "default pattern",
			sessionPattern: "{repo}-{branch}",
			repo:           "/path/to/maverick",
			branch:         "feature-auth",
			want:           "maverick-feature-auth",
		},
		{
			name:           "custom pattern",
			sessionPattern: "work_{branch}",
			repo:           "/path/to/maverick",
			branch:         "fix-bug",
			want:           "work_fix-bug",
		},
		{
			name:           "repo only",
			sessionPattern: "{repo}",
			repo:           "/path/to/myproject",
			branch:         "main",
			want:           "myproject",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{
				Repo:           tt.repo,
				SessionPattern: tt.sessionPattern,
			}
			if got := cfg.SessionName(tt.branch); got != tt.want {
				t.Errorf("SessionName(%q) = %q, want %q", tt.branch, got, tt.want)
			}
		})
	}
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}

	tests := []struct {
		input string
		want  string
	}{
		{"~/foo/bar", filepath.Join(home, "foo/bar")},
		{"~", home},
		{"/absolute/path", "/absolute/path"},
		{"relative/path", "relative/path"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := expandHome(tt.input); got != tt.want {
				t.Errorf("expandHome(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
