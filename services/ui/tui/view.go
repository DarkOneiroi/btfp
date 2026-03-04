// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package tui

import (
	"btfp/internal/utils"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

// View renders the current state of the application
func (m *Model) View() string {
	var res string
	switch m.view {
	case viewLibrary:
		bg := m.getBackground()
		fg := m.renderLibraryView()
		if bg != "" {
			res = m.overlayUI(bg, fg)
		} else {
			res = fg
		}
	case viewPlaylist:
		bg := m.getBackground()
		fg := m.renderPlaylistView()
		if bg != "" {
			res = m.overlayUI(bg, fg)
		} else {
			res = fg
		}
	case viewPlayer:
		fg, bg := m.renderPlayerView(), m.getBackground()
		if m.lockedView {
			bg = ""
		}
		res = m.overlayUI(bg, fg)
	case viewViz:
		res = m.getBackground()
	}

	if m.showLegend {
		res = m.overlayUI(res, m.renderLegend())
	}

	if m.err != nil {
		errBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF0000")).
			Padding(0, 1).
			Render(fmt.Sprintf("Error: %v", m.err))
		res = m.overlayUI(res, errBox)
	}

	// OVERLAY IMAGE PROTOCOL (CACHED)
	artPath := m.GetArtPath()
	var panelX int
	showArt := false
	if m.view == viewLibrary {
		panelX = (m.width / 3) * 2
		showArt = true
	} else if m.view == viewPlaylist {
		panelX = m.width / 2
		showArt = true
	}

	if showArt && artPath != "" {
		if artPath != m.lastArtPath || m.width != m.lastArtWidth || m.view != m.lastArtView {
			m.lastArtPath = artPath
			m.lastArtWidth = m.width
			m.lastArtView = m.view

			// Move cursor to row 3 (exactly after title). Col is panelX+2 (inside padding).
			move := fmt.Sprintf("\033[%d;%dH", 3, panelX+2)
			art := utils.ImageToASCII(artPath, (m.width - panelX - 4), true)
			if strings.HasPrefix(art, "\033]") {
				m.artProtocol = move + art
			} else {
				m.artProtocol = ""
			}
		}
		// Always append the protocol to the final output to keep it drawn in the AltScreen.
		// Since we're using absolute coordinates, the terminal handles the placement.
		if m.artProtocol != "" {
			res += m.artProtocol
		}
	} else {
		m.lastArtPath = ""
		m.artProtocol = ""
	}

	return res
}

func (m *Model) getBackground() string {
	switch m.bgMode {
	case bgVisualization, bgBars:
		return m.vizData
	case bgKaraoke:
		return m.renderKaraoke()
	case bgImage:
		return m.renderImage()
	default:
		return ""
	}
}

func (m *Model) renderListView(l list.Model) string {
	view := l.View()
	lines := strings.Split(view, "\n")
	h := l.Height()
	if h <= 0 {
		return view
	}

	totalItems := len(l.Items())
	if totalItems <= h {
		return view
	}

	cursor := l.Index()
	scrollPos := int(float64(cursor) / float64(totalItems) * float64(h))

	var sb strings.Builder
	for i, line := range lines {
		if i >= h {
			break
		}
		indicator := " "
		if i == scrollPos {
			indicator = "┃"
		} else if i > 0 && i < h-1 {
			indicator = "│"
		}
		sb.WriteString(line + indicator + "\n")
	}
	return sb.String()
}
