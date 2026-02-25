package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"
)

// View renders the current state of the application
func (m *Model) View() string {
	if m.err != nil {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center,
			fmt.Sprintf("Error: %v", m.err))
	}

	var res string
	switch m.view {
	case viewLibrary:
		res = m.renderLibraryView()
	case viewPlaylist:
		res = m.renderPlaylistView()
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

	return res
}

func (m *Model) getBackground() string {
	switch m.bgMode {
	case bgVisualization, bgEQBars:
		if m.vizFrame == nil {
			return ""
		}
		return m.vizFrame.Render(false)
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
