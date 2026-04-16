package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Alex-Painter/twig/config"
	"github.com/Alex-Painter/twig/tui"
)

// Version is set at build time via -ldflags "-X main.Version=<version>".
// Defaults to "dev" for local builds.
var Version = "dev"

const usage = `twig - TUI to manage git worktrees and tmux sessions

Usage:
  twig [flags]

Flags:
  --version    Print version and exit
  --help       Print this help message and exit

Configuration:
  twig reads its config from ~/.config/twig/config.toml

See https://github.com/Alex-Painter/twig for more information.
`

func main() {
	showVersion := flag.Bool("version", false, "print version and exit")
	showHelp := flag.Bool("help", false, "print help and exit")

	flag.Usage = func() {
		fmt.Fprint(os.Stderr, usage)
	}
	flag.Parse()

	if *showVersion {
		fmt.Printf("twig %s\n", Version)
		return
	}

	if *showHelp {
		fmt.Print(usage)
		return
	}

	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if err := tui.Run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
