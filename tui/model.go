// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package tui

import (
	"encoding/gob"
	"net"
	"os"
	"path/filepath"
	"time"

	"btfp/config"
	"btfp/ipc"
	"btfp/player"
	"btfp/visualizations"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// View states for navigation
type viewState int

const (
	viewLibrary viewState = iota
	viewPlaylist
	viewPlayer
	viewViz
)

// Background display modes
type backgroundMode int

const (
	bgVisualization backgroundMode = iota
	bgEQBars
	bgKaraoke
	bgEmpty
	bgImage
)

// lrcLine represents a single line of lyrics with its start time
type lrcLine struct {
	time time.Duration
	text string
}

// item represents a file or directory in the library list
type item struct {
	title, desc, path string
	isDir             bool
	selected          bool // Staged for adding
	inPlaylist        bool // Already in playlist
}

// Title returns the formatted title with an icon indicating state
func (i item) Title() string {
	var prefix string
	switch {
	case i.selected:
		prefix = "󰄲 " // Staged icon
	case i.inPlaylist:
		prefix = "󰄵 " // In playlist icon
	case i.isDir:
		prefix = "󰉋 " // Folder icon
	default:
		prefix = "󰈣 " // File icon
	}
	return prefix + i.title
}

// Description returns the item's metadata description
func (i item) Description() string { return i.desc }

// FilterValue returns the string used for list filtering
func (i item) FilterValue() string { return i.title }

// Model represents the main application state
type Model struct {
	// Terminal dimensions
	width, height int

	// Navigation and state
	view       viewState
	lockedView bool
	bgMode     backgroundMode
	err        error

	// Lists
	libList  list.Model
	playList list.Model

	// Data
	playlist      []player.Track
	playingIdx    int
	currentDir    string
	selectedPaths map[string]bool
	currentLyrics []lrcLine
	startTime     time.Time

	// Configuration
	cfg   config.Config
	theme config.Theme

	// Visualizations
	vizFrame   *visualizations.Frame
	preset     int
	colorMode  int
	palette    int
	showLegend bool
	LastResize time.Time

	// Infrastructure
	player player.Player
	conn   net.Conn
	enc    *gob.Encoder
	dec    *gob.Decoder

	// Caches
	metadataCache  map[string]string
	artCache       map[string]string
	playlistCounts map[string]int
}

// NewModel initializes a new application model
func NewModel(initialView string) *Model {
	cfg, theme := config.LoadConfig()
	home, _ := os.UserHomeDir()
	musicDir := filepath.Join(home, "Music")

	startView := viewState(cfg.DefaultView)
	if startView > viewViz {
		startView = viewLibrary
	}

	locked := false
	if initialView != "" {
		locked = true
		switch initialView {
		case "library":
			startView = viewLibrary
		case "playlist":
			startView = viewPlaylist
		case "player":
			startView = viewPlayer
		case "viz":
			startView = viewViz
		}
	}

	m := &Model{
		view:           startView,
		lockedView:     locked,
		bgMode:         backgroundMode(cfg.BGMode),
		currentDir:     musicDir,
		selectedPaths:  make(map[string]bool),
		preset:         cfg.Pattern,
		colorMode:      cfg.ColorMode,
		palette:        cfg.Palette,
		showLegend:     cfg.ShowLegend,
		player:         player.NewMusicPlayer(cfg),
		playingIdx:     -1,
		cfg:            cfg,
		theme:          theme,
		startTime:      time.Now(),
		metadataCache:  make(map[string]string),
		artCache:       make(map[string]string),
		playlistCounts: make(map[string]int),
	}

	// Initialize lists with default styling
	m.libList = list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	m.libList.Title = "LIBRARY"
	m.playList = list.New([]list.Item{}, list.NewDefaultDelegate(), 0, 0)
	m.playList.Title = "PLAYLIST"

	return m
}

// Init initializes the Bubble Tea loop
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadDirectory(m.currentDir),
		tick(),
		vizTick(),
		m.listenToServer(),
	)
}

// SetConn configures the IPC connection for the model
func (m *Model) SetConn(conn net.Conn, enc *gob.Encoder, dec *gob.Decoder) {
	m.conn = conn
	m.enc = enc
	m.dec = dec
}

func (m *Model) sendCommand(cmd ipc.Command) {
	if m.enc != nil {
		if err := m.enc.Encode(cmd); err != nil {
			m.err = err
		}
	}
}

func tick() tea.Cmd {
	return tea.Tick(time.Millisecond*500, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func vizTick() tea.Cmd {
	return tea.Tick(time.Millisecond*33, func(t time.Time) tea.Msg {
		return vizTickMsg(t)
	})
}
