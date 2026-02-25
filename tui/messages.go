package tui

import (
	"btfp/ipc"
	"time"

	"github.com/charmbracelet/bubbles/list"
)

// Message types for the Bubble Tea loop
type (
	vizTickMsg          time.Time
	errMsg              error
	artDownloadedMsg    string
	lyricsDownloadedMsg struct {
		path   string
		lyrics []lrcLine
	}
	tickMsg        time.Time
	libraryMsg     []list.Item
	serverStateMsg ipc.PlayerState
)
