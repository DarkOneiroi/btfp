package config

import (
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

type Config struct {
	DefaultView        int    `toml:"default_view"`
	Theme              string `toml:"theme"`
	GromaPath          string `toml:"groma_path"`
	ImagePath          string `toml:"image_path"`
	AutoDownloadArt    bool   `toml:"auto_download_art"`
	AutoDownloadLyrics bool   `toml:"auto_download_lyrics"`
	UpdateMetadata     bool   `toml:"update_metadata"`
	ShowLegend         bool   `toml:"show_legend"`
	BGMode             int    `toml:"bg_mode"`
	Pattern            int    `toml:"pattern"`
	ColorMode          int    `toml:"color_mode"`
	Palette            int    `toml:"palette"`
	EQColorMode        int    `toml:"eq_color_mode"`
	EQPalette          int    `toml:"eq_palette"`
}

type Theme struct {
	Title     string `toml:"title"`
	Accent    string `toml:"accent"`
	Highlight string `toml:"highlight"`
	Text      string `toml:"text"`
	Subtext   string `toml:"subtext"`
}

func LoadConfig() (Config, Theme) {
	home, _ := os.UserHomeDir()
	c := Config{
		DefaultView:        0,
		Theme:              "omarchy",
		GromaPath:          filepath.Join(home, "go/bin/groma"),
		ImagePath:          "",
		AutoDownloadArt:    true,
		AutoDownloadLyrics: true,
		UpdateMetadata:     true,
		ShowLegend:         false,
		BGMode:             0,
		Pattern:            0,
		ColorMode:          0,
		Palette:            0,
		EQColorMode:        0,
		EQPalette:          0,
	}

	configDir := filepath.Join(home, ".config", "btfp")
	os.MkdirAll(configDir, 0755)

	configPath := filepath.Join(configDir, "config.toml")
	if data, err := os.ReadFile(configPath); err == nil {
		toml.Unmarshal(data, &c)
	} else {
		saveFile(configPath, c)
	}

	theme := LoadTheme(c.Theme)
	return c, theme
}

func LoadTheme(name string) Theme {
	t := Theme{
		Title:     "63",
		Accent:    "13",
		Highlight: "10",
		Text:      "15",
		Subtext:   "245",
	}

	home, _ := os.UserHomeDir()
	themePath := filepath.Join(home, ".config", "btfp", "themes", name+".toml")
	os.MkdirAll(filepath.Dir(themePath), 0755)

	if data, err := os.ReadFile(themePath); err == nil {
		toml.Unmarshal(data, &t)
	} else {
		saveFile(themePath, t)
	}

	return t
}

func saveFile(path string, v interface{}) {
	data, err := toml.Marshal(v)
	if err == nil {
		os.WriteFile(path, data, 0644)
	}
}
