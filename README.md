# twig

A TUI application to manage git worktrees and tmux sessions.

## Overview

`twig` provides a unified interface for managing git worktrees alongside their associated tmux sessions. It displays all worktrees in a scrollable list with status indicators, and provides keybindings for common operations.

## Features

### Implemented

- [x] Configuration file (`~/.config/twig/config.toml`)
- [x] List all worktrees with branch names and paths
- [x] Visual indicator for main repository clone (★)
- [x] Show dirty/clean status for each worktree (*)
- [x] Show last commit message and relative time
- [x] Keyboard navigation (↑/↓ or j/k)
- [x] Refresh list (r)
- [x] Quit (q)

### Planned
- [ ] Show commits ahead/behind remote
- [ ] Show tmux session status (attached/detached)
- [ ] Create worktree from branch name (n)
- [ ] Create worktree from GitHub PR number (#123)
- [ ] Post-create hooks for dependency installation
- [ ] Delete worktree with safety checks (d)
- [ ] Fetch all remotes (f)
- [ ] Pull current branch (p)
- [ ] Filter/search worktrees (/)
- [ ] Help modal (?)
- [ ] Inline error handling with retry/dismiss

## Installation

```bash
# Clone the repository
git clone https://github.com/Alex-Painter/twig.git
cd twig

# Build
go build -o twig .

# Optionally, move to your PATH
mv twig ~/bin/
```

## Configuration

Create a config file at `~/.config/twig/config.toml`:

```toml
# Required: path to your main git repository
repo = "~/workspace/myproject"

# Required: directory where worktrees will be created
worktree_dir = "~/workspace/worktrees/myproject"

# Optional: tmux session naming pattern (default: "{repo}-{branch}")
# Supports {repo} and {branch} placeholders
session_pattern = "{repo}-{branch}"

# Optional: tmux windows to create (default: ["editor", "dev", "shell"])
windows = ["editor", "dev", "shell"]

# Optional: directory for hook scripts (default: ~/.config/twig/hooks)
hooks_dir = "~/.config/twig/hooks"
```

## Usage

```bash
# Launch the TUI
twig
```

### Keybindings

| Key | Action |
|-----|--------|
| `↑` / `k` | Move cursor up |
| `↓` / `j` | Move cursor down |
| `r` | Refresh worktree list |
| `q` | Quit |

## Development

```bash
# Run tests
go test ./...

# Build
go build -o twig .
```

## License

MIT
