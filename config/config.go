package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config defines the application settings
type Config struct {
	MusicPath          string `toml:"music_path"`
	DefaultView        int    `toml:"default_view"`
	BGMode             int    `toml:"bg_mode"`
	Pattern            int    `toml:"pattern"`
	ColorMode          int    `toml:"color_mode"`
	Palette            int    `toml:"palette"`
	ShowLegend         bool   `toml:"show_legend"`
	AutoDownloadArt    bool   `toml:"auto_download_art"`
	AutoDownloadLyrics bool   `toml:"auto_download_lyrics"`
	UpdateMetadata     bool   `toml:"update_metadata"`
	ImagePath          string `toml:"image_path"`
	Theme              string `toml:"theme"`
}

// Theme defines the ANSI color codes for UI components
type Theme struct {
	Accent    string `toml:"accent"`
	Title     string `toml:"title"`
	Text      string `toml:"text"`
	Subtext   string `toml:"subtext"`
	Highlight string `toml:"highlight"`
}

// LoadConfig reads the application configuration from Disk
func LoadConfig() (Config, Theme) {
	home, _ := os.UserHomeDir()
	configDir := filepath.Join(home, ".config", "btfp")
	configPath := filepath.Join(configDir, "config.toml")

	c := Config{
		MusicPath:          filepath.Join(home, "Music"),
		DefaultView:        0,
		BGMode:             0,
		Pattern:            0,
		ColorMode:          0,
		Palette:            0,
		ShowLegend:         true,
		AutoDownloadLyrics: true,
		AutoDownloadArt:    true,
		UpdateMetadata:     true,
		Theme:              "default",
	}

	_ = os.MkdirAll(configDir, 0755)
	if data, err := os.ReadFile(configPath); err == nil {
		_, _ = toml.Decode(string(data), &c)
	}

	return c, LoadTheme(c.Theme)
}

// LoadTheme reads a theme file from disk or returns the default theme
func LoadTheme(name string) Theme {
	home, _ := os.UserHomeDir()
	themePath := filepath.Join(home, ".config", "btfp", "themes", name+".toml")

	t := Theme{
		Accent:    "63",
		Title:     "255",
		Text:      "252",
		Subtext:   "245",
		Highlight: "214",
	}

	_ = os.MkdirAll(filepath.Dir(themePath), 0755)
	if data, err := os.ReadFile(themePath); err == nil {
		_, _ = toml.Decode(string(data), &t)
	} else if name == "default" {
		// Save default theme if it doesn't exist
		// Note: error ignored intentionally during config init
		writeDefaultTheme(themePath, t)
	}

	return t
}

func writeDefaultTheme(path string, t Theme) {
	data, err := toml.Marshal(t)
	if err == nil {
		_ = os.WriteFile(path, data, 0644)
	}
}
