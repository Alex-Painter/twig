package main

import (
	"fmt"
	"os"

	"github.com/Alex-Painter/twig/config"
	"github.com/Alex-Painter/twig/tui"
)

func main() {
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
