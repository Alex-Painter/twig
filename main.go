package main

import (
	"fmt"
	"os"

	"github.com/Alex-Painter/twig/config"
)

func main() {
	cfg, err := config.Load("")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Twig Configuration")
	fmt.Println("==================")
	fmt.Printf("Repo:            %s\n", cfg.Repo)
	fmt.Printf("Worktree Dir:    %s\n", cfg.WorktreeDir)
	fmt.Printf("Session Pattern: %s\n", cfg.SessionPattern)
	fmt.Printf("Windows:         %v\n", cfg.Windows)
	fmt.Printf("Hooks Dir:       %s\n", cfg.HooksDir)
	fmt.Println()
	fmt.Printf("Repo Name:       %s\n", cfg.RepoName())
	fmt.Printf("Example Session: %s\n", cfg.SessionName("feature-auth"))
}
