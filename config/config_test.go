package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Use a temporary home for testing
	tmpHome := t.TempDir()
	_ = os.Setenv("HOME", tmpHome)

	cfg, theme := LoadConfig()

	// The current defaults in config.go are "default" and "255"
	if cfg.Theme != "default" {
		t.Errorf("expected default theme default, got %s", cfg.Theme)
	}

	if theme.Title != "255" {
		t.Errorf("expected default theme title color 255, got %s", theme.Title)
	}

	// Note: config file creation check might fail if LoadConfig doesn't explicitly save it
	// But let's keep it and see if we should improve config.go later.
}

func TestLoadTheme(t *testing.T) {
	tmpHome := t.TempDir()
	_ = os.Setenv("HOME", tmpHome)

	theme := LoadTheme("test-theme")
	// The current default accent in config.go is "63"
	if theme.Accent != "63" {
		t.Errorf("expected default accent 63, got %s", theme.Accent)
	}
}
