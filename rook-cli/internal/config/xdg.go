package config

import (
	"os"
	"path/filepath"
)

// ConfigPath returns the resolved path to the rook-cli config file.
// Resolution order:
//  1. ROOK_CONFIG_PATH environment variable (if non-empty)
//  2. $XDG_CONFIG_HOME/rook/config.json
//  3. ~/.config/rook/config.json
func ConfigPath() string {
	if p := os.Getenv("ROOK_CONFIG_PATH"); p != "" {
		return p
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "rook", "config.json")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback: relative path in current directory (should not happen in practice)
		return filepath.Join(".config", "rook", "config.json")
	}
	return filepath.Join(home, ".config", "rook", "config.json")
}
