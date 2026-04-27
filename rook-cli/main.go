package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/rook-project/rook-reference/rook-cli/internal/config"
)

// version is injected at build time via -ldflags "-X main.version=<ver>"
var version string

func main() {
	v := version
	if v == "" {
		v = "dev"
	}

	for _, arg := range os.Args[1:] {
		switch arg {
		case "--version", "-v":
			fmt.Printf("rook version %s\n", v)
			os.Exit(0)
		case "--help", "-h":
			printHelp(v)
			os.Exit(0)
		}
	}

	// Load config from resolved XDG path.
	cfgPath := config.ConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		if errors.Is(err, config.ErrNotFound) {
			fmt.Printf("rook version %s\n", v)
			fmt.Fprintln(os.Stderr, "No config found. Run rook after setup to configure.")
			os.Exit(0)
		}
		fmt.Fprintf(os.Stderr, "rook: error loading config: %v\n", err)
		os.Exit(1)
	}
	if err := config.Validate(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "rook: invalid config at %s: %v\n", cfgPath, err)
		os.Exit(1)
	}

	// v0.1 skeleton: print version and exit. TUI launcher is implemented in v0.2+.
	fmt.Printf("rook version %s\n", v)
}

func printHelp(v string) {
	fmt.Printf("rook version %s\n\nUsage: rook [flags]\n\nFlags:\n  -v, --version   Print version and exit\n  -h, --help      Print this help and exit\n\nConfig is read from %s\n", v, config.ConfigPath())
}
