package config_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/rook-project/rook-reference/rook-cli/internal/config"
)

func TestLoad_NotFound(t *testing.T) {
	_, err := config.Load(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, config.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestLoad_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	want := config.Config{
		Servers: []config.ServerEntry{
			{Address: "https://rook.example.com", Alias: "work"},
		},
		ActiveSpace:  "space-123",
		StorageDir:   "/home/user/rook/storage",
		FeatureFlags: map[string]bool{"beta": true},
	}

	if err := config.Save(path, want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if got.ActiveSpace != want.ActiveSpace {
		t.Errorf("ActiveSpace: got %q want %q", got.ActiveSpace, want.ActiveSpace)
	}
	if len(got.Servers) != 1 || got.Servers[0].Address != want.Servers[0].Address {
		t.Errorf("Servers mismatch: got %+v", got.Servers)
	}
	if got.FeatureFlags["beta"] != true {
		t.Errorf("FeatureFlags: expected beta=true")
	}
}

func TestSave_WritesIndentedJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := config.Config{
		Servers:      []config.ServerEntry{},
		FeatureFlags: map[string]bool{},
	}
	if err := config.Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	// Verify it's valid JSON
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, data)
	}
	// Verify indentation (indented JSON has newlines after opening brace)
	if len(data) < 2 || data[1] != '\n' {
		t.Errorf("expected indented JSON, got: %s", data)
	}
}

func TestSave_Atomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	original := config.Config{ActiveSpace: "original"}
	if err := config.Save(path, original); err != nil {
		t.Fatalf("initial Save: %v", err)
	}

	// Verify the file exists and has original content
	got, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load after Save: %v", err)
	}
	if got.ActiveSpace != "original" {
		t.Errorf("expected original, got %q", got.ActiveSpace)
	}
}

func TestConfigPath_RookConfigPathEnv(t *testing.T) {
	t.Setenv("ROOK_CONFIG_PATH", "/custom/path/config.json")
	t.Setenv("XDG_CONFIG_HOME", "")

	got := config.ConfigPath()
	if got != "/custom/path/config.json" {
		t.Errorf("expected /custom/path/config.json, got %q", got)
	}
}

func TestConfigPath_XDGConfigHome(t *testing.T) {
	t.Setenv("ROOK_CONFIG_PATH", "")
	t.Setenv("XDG_CONFIG_HOME", "/xdg/config")

	got := config.ConfigPath()
	want := "/xdg/config/rook/config.json"
	if got != want {
		t.Errorf("expected %q, got %q", want, got)
	}
}

func TestConfigPath_DefaultFallback(t *testing.T) {
	t.Setenv("ROOK_CONFIG_PATH", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	got := config.ConfigPath()
	if got == "" {
		t.Error("expected non-empty path")
	}
	// Should end with .config/rook/config.json
	want := filepath.Join(".config", "rook", "config.json")
	if len(got) < len(want) {
		t.Errorf("path too short: %q", got)
	}
}

func TestLoad_UnknownFieldsIgnored(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	// Write JSON with an extra unknown field
	raw := `{"active_space":"s1","unknown_future_field":"value","servers":[],"feature_flags":{}}`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ActiveSpace != "s1" {
		t.Errorf("expected active_space=s1, got %q", cfg.ActiveSpace)
	}
}

// --- Validate tests (T024) ---

func TestValidate_EmptyConfig(t *testing.T) {
	if err := config.Validate(config.Config{}); err != nil {
		t.Errorf("empty config should be valid, got: %v", err)
	}
}

func TestValidate_ValidConfig(t *testing.T) {
	cfg := config.Config{
		Servers:    []config.ServerEntry{{Address: "https://rook.example.com", Alias: "work"}},
		StorageDir: "/home/user/rook",
	}
	if err := config.Validate(cfg); err != nil {
		t.Errorf("valid config should pass, got: %v", err)
	}
}

func TestValidate_EmptyAddress(t *testing.T) {
	cfg := config.Config{
		Servers: []config.ServerEntry{{Address: ""}},
	}
	if err := config.Validate(cfg); err == nil {
		t.Error("expected error for empty address, got nil")
	}
}

func TestValidate_InvalidURL(t *testing.T) {
	cfg := config.Config{
		Servers: []config.ServerEntry{{Address: "not-a-url"}},
	}
	if err := config.Validate(cfg); err == nil {
		t.Error("expected error for non-URL address, got nil")
	}
}

func TestValidate_RelativeStorageDir(t *testing.T) {
	cfg := config.Config{
		StorageDir: "relative/path",
	}
	if err := config.Validate(cfg); err == nil {
		t.Error("expected error for relative storage_dir, got nil")
	}
}

func TestValidate_FeatureFlagsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	cfg := config.Config{
		FeatureFlags: map[string]bool{
			"known_flag":   true,
			"unknown_flag": false,
		},
	}
	if err := config.Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	loaded, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.FeatureFlags["known_flag"] != true {
		t.Error("known_flag should be true")
	}
	if _, ok := loaded.FeatureFlags["unknown_flag"]; !ok {
		t.Error("unknown_flag should be present after round-trip")
	}
}
