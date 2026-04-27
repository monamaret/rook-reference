package main

import (
	"fmt"
	"os"
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
			fmt.Printf("rook-server-cli version %s\n", v)
			os.Exit(0)
		case "--help", "-h":
			fmt.Printf("rook-server-cli version %s\n\nUsage: rook-server-cli [flags]\n\nFlags:\n  -v, --version   Print version and exit\n  -h, --help      Print this help and exit\n\nAdmin subcommands (user, space) are available from v0.3+.\n", v)
			os.Exit(0)
		}
	}

	// Default: print version and exit (v0.1 stub behaviour).
	fmt.Printf("rook-server-cli version %s\n", v)
}
