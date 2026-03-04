// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package tui

import (
	"btfp/internal/ipc-shared"
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
	vizFrameMsg    string
)
