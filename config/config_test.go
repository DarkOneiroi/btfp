package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Use a temporary home for testing
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)

	cfg, theme := LoadConfig()

	if cfg.Theme != "omarchy" {
		t.Errorf("expected default theme omarchy, got %s", cfg.Theme)
	}

	if theme.Title != "63" {
		t.Errorf("expected default theme title color 63, got %s", theme.Title)
	}

	// Verify config file was created
	configPath := filepath.Join(tmpHome, ".config", "btfp", "config.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file was not created on first load")
	}
}

func TestLoadTheme(t *testing.T) {
	tmpHome := t.TempDir()
	os.Setenv("HOME", tmpHome)

	theme := LoadTheme("test-theme")
	if theme.Accent != "13" {
		t.Errorf("expected default accent 13, got %s", theme.Accent)
	}
}
