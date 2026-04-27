package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrNotFound is returned by Load when the config file does not exist.
var ErrNotFound = errors.New("config: file not found")

// Config holds the rook-cli configuration loaded from disk.
type Config struct {
	Servers      []ServerEntry   `json:"servers"`
	ActiveSpace  string          `json:"active_space"`
	StorageDir   string          `json:"storage_dir"`
	FeatureFlags map[string]bool `json:"feature_flags"`
}

// ServerEntry represents a configured rook-server endpoint.
type ServerEntry struct {
	Address string `json:"address"`
	Alias   string `json:"alias,omitempty"`
}

// Load reads and decodes the config file at path.
// Returns ErrNotFound if the file does not exist.
// Unknown JSON fields are silently ignored for forward-compatibility.
func Load(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, ErrNotFound
		}
		return Config{}, fmt.Errorf("config: open %s: %w", path, err)
	}
	defer f.Close()

	var cfg Config
	dec := json.NewDecoder(f)
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("config: decode %s: %w", path, err)
	}
	return cfg, nil
}

// Save atomically writes cfg as indented JSON to path.
// It writes to a temporary file in the same directory, then renames it
// to path so the operation is atomic on POSIX systems (rename(2)).
func Save(path string, cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("config: marshal: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("config: mkdir %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, "config-*.json")
	if err != nil {
		return fmt.Errorf("config: create temp: %w", err)
	}
	tmpName := tmp.Name()

	// Clean up temp file on any error path.
	defer func() {
		if err != nil {
			os.Remove(tmpName)
		}
	}()

	if _, err = tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("config: write temp: %w", err)
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("config: close temp: %w", err)
	}
	if err = os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("config: rename to %s: %w", path, err)
	}
	return nil
}

// Validate checks that cfg contains only well-formed values.
// It returns nil for a zero-value Config (forward-compatible: missing fields are valid).
func Validate(cfg Config) error {
	for i, s := range cfg.Servers {
		if s.Address == "" {
			return fmt.Errorf("config: servers[%d].address is empty", i)
		}
		if !strings.HasPrefix(s.Address, "http://") && !strings.HasPrefix(s.Address, "https://") {
			return fmt.Errorf("config: servers[%d].address %q is not a valid absolute URL", i, s.Address)
		}
	}
	if cfg.StorageDir != "" && !filepath.IsAbs(cfg.StorageDir) {
		return fmt.Errorf("config: storage_dir %q is not an absolute path", cfg.StorageDir)
	}
	return nil
}
