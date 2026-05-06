# AGENTS.md

Guidance for AI coding agents working in this repository.

## Project

`twig` is a Go TUI (built on Bubble Tea) for managing git worktrees and their associated tmux sessions. Single binary, configured via `~/.config/twig/config.toml`.

## Layout

- `main.go` — entrypoint, flag parsing, config loading, TUI bootstrap
- `config/` — TOML config loading and defaults
- `git/` — git command wrappers (status, fetch, pull, ahead/behind)
- `github/` — `gh` CLI integration for PR lookup
- `worktree/` — worktree create/list/delete logic
- `tmux/` — tmux session management
- `hooks/` — post-create hook execution
- `tui/` — Bubble Tea model, view, update loop, keybindings

## Commands

```bash
go build -o twig .   # build
go test ./...        # run all tests
go vet ./...         # static checks
```

Each package has a `_test.go` alongside it — keep that pattern when adding code.

## Conventions

- Go 1.26+; standard `gofmt` formatting.
- Errors bubble up to the TUI layer, which surfaces them inline with retry (R) / dismiss (Esc). Don't `log.Fatal` outside `main`.
- Shell out to `git`, `gh`, and `tmux` via `os/exec` rather than pulling in libraries — match the existing wrappers in `git/`, `github/`, `tmux/`.
- Keybindings live in `tui/`. Update the README keybindings table and `--help` text in `main.go` when adding or changing them.
- Version is injected at build time via `-ldflags "-X main.Version=..."` (see `.goreleaser.yml`).

## Releases

Tagged releases go through GoReleaser (`.goreleaser.yml`) and publish to the Homebrew tap `Alex-Painter/tap`. CI workflows live in `.github/workflows/`.
