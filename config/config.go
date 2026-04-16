// Package config handles loading and validation of twig configuration.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// Config represents the twig configuration.
type Config struct {
	// Repo is the path to the main git repository.
	Repo string `toml:"repo"`

	// WorktreeDir is the directory where worktrees are created.
	WorktreeDir string `toml:"worktree_dir"`

	// SessionPattern is the tmux session naming pattern.
	// Supports {repo} and {branch} placeholders.
	SessionPattern string `toml:"session_pattern"`

	// Windows is the list of tmux window names to create.
	Windows []string `toml:"windows"`

	// HooksDir is the directory containing hook scripts.
	HooksDir string `toml:"hooks_dir"`
}

// DefaultConfigPath returns the default config file path.
func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".config", "twig", "config.toml"), nil
}

// Load reads and parses the config file at the given path.
// If path is empty, the default config path is used.
// Returns an error if required fields are missing.
func Load(path string) (*Config, error) {
	if path == "" {
		defaultPath, err := DefaultConfigPath()
		if err != nil {
			return nil, err
		}
		path = defaultPath
	}

	// Expand ~ in the provided path
	path = expandHome(path)

	cfg := &Config{}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("config file not found: %s", path)
	}

	// Parse TOML
	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults
	cfg.applyDefaults()

	// Expand ~ in paths
	cfg.expandPaths()

	// Validate required fields
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// applyDefaults sets default values for optional fields.
func (c *Config) applyDefaults() {
	if c.SessionPattern == "" {
		c.SessionPattern = "{repo}-{branch}"
	}

	if len(c.Windows) == 0 {
		c.Windows = []string{"editor", "dev", "shell"}
	}

	if c.HooksDir == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			c.HooksDir = filepath.Join(home, ".config", "twig", "hooks")
		}
	}
}

// expandPaths expands ~ to the home directory in all path fields.
func (c *Config) expandPaths() {
	c.Repo = expandHome(c.Repo)
	c.WorktreeDir = expandHome(c.WorktreeDir)
	c.HooksDir = expandHome(c.HooksDir)
}

// expandHome replaces ~ with the user's home directory.
func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return home
	}
	return path
}

// validate checks that required fields are present.
func (c *Config) validate() error {
	var errs []string

	if c.Repo == "" {
		errs = append(errs, "repo is required")
	}

	if c.WorktreeDir == "" {
		errs = append(errs, "worktree_dir is required")
	}

	if len(errs) > 0 {
		return errors.New("config validation failed: " + strings.Join(errs, ", "))
	}

	return nil
}

// RepoName returns the base name of the repo path.
// This is used for the {repo} placeholder in session patterns.
func (c *Config) RepoName() string {
	return filepath.Base(c.Repo)
}

// SessionName generates a tmux session name for the given branch.
func (c *Config) SessionName(branch string) string {
	name := c.SessionPattern
	name = strings.ReplaceAll(name, "{repo}", c.RepoName())
	name = strings.ReplaceAll(name, "{branch}", branch)
	return name
}
