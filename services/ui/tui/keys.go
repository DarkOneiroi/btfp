// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package tui

import (
	"btfp/internal/ipc-shared"
	"btfp/services/visualization/visualizations"
	"os"
	"path/filepath"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// handleKeyPress processes keyboard input and returns an optional command
func (m *Model) handleKeyPress(msg tea.KeyMsg) tea.Cmd {
	// If any list is filtering, let the list handle all keys
	if m.libList.FilterState() == list.Filtering || m.playList.FilterState() == list.Filtering {
		return nil
	}

	key := msg.String()
	var cmds []tea.Cmd

	switch key {
	case "q", "ctrl+c":
		m.sendCommand(ipc.Command{Type: ipc.CmdQuit})
		return tea.Quit

	case "backspace":
		m.handleBackspace(&cmds)

	case "tab":
		m.handleTab()

	case " ":
		m.handleSpace()

	case "a":
		m.handleAddTrack(&cmds)

	case "enter":
		m.handleEnter(&cmds)

	case "c", "i", "p":
		m.handleVizKeys(key)

	case "v":
		if m.view == viewPlayer || m.view == viewViz {
			m.cycleBGMode()
		}

	case "h", "?":
		m.showLegend = !m.showLegend

	case "+", "=":
		m.adjustVolume(0.1)

	case "-", "_":
		m.adjustVolume(-0.1)

	case "m":
		m.toggleMute()

	case "l", "right":
		m.seek(5 * time.Second)

	case "left":
		m.seek(-5 * time.Second)

	case "n":
		m.nextTrack(&cmds)

	case "b":
		m.prevTrack(&cmds)

	case "t":
		m.toggleTTSLanguage()

	case "s":
		m.cycleTTSSpeaker()
	}

	if len(cmds) > 0 {
		return tea.Batch(cmds...)
	}
	return nil
}

func (m *Model) handleBackspace(cmds *[]tea.Cmd) {
	if m.view == viewLibrary {
		home, _ := os.UserHomeDir()
		if m.currentDir != filepath.Join(home, "Music") {
			m.currentDir = filepath.Dir(m.currentDir)
			*cmds = append(*cmds, m.loadDirectory(m.currentDir))
		}
	}
}

func (m *Model) handleTab() {
	if !m.lockedView {
		m.view = (m.view + 1) % 4 // Cycle through all 4 views
		if m.view == viewPlaylist {
			m.updatePlaylistItems()
		}
	}
}

func (m *Model) handleSpace() {
	if m.view == viewLibrary {
		if sel, ok := m.libList.SelectedItem().(item); ok && sel.title != ".." {
			m.selectedPaths[sel.path] = !m.selectedPaths[sel.path]
			its := m.libList.Items()
			for i, it := range its {
				itm := it.(item)
				if itm.path == sel.path {
					itm.selected = m.selectedPaths[sel.path]
					its[i] = itm
				}
			}
			m.libList.SetItems(its)
		}
	} else {
		m.sendCommand(ipc.Command{Type: ipc.CmdPause})
	}
}

func (m *Model) handleAddTrack(cmds *[]tea.Cmd) {
	if m.view == viewLibrary {
		for p, s := range m.selectedPaths {
			if s {
				m.addPathToPlaylist(p)
			}
		}
		m.selectedPaths = make(map[string]bool)
		m.syncPlaylist()
		*cmds = append(*cmds, m.loadDirectory(m.currentDir))
	}
}

func (m *Model) handleEnter(cmds *[]tea.Cmd) {
	if m.view == viewLibrary {
		if sel, ok := m.libList.SelectedItem().(item); ok {
			if sel.isDir {
				m.currentDir = sel.path
				*cmds = append(*cmds, m.loadDirectory(m.currentDir))
			} else {
				track := m.getTrackMetadata(sel.path)
				m.sendCommand(ipc.Command{Type: ipc.CmdPlayTrack, Payload: track})
				m.view = viewPlayer
			}
		}
	} else if m.view == viewPlaylist {
		if m.playList.Index() >= 0 && m.playList.Index() < len(m.playlist) {
			idx := m.playList.Index()
			m.sendCommand(ipc.Command{Type: ipc.CmdPlay, Payload: idx})
		}
	}
}

func (m *Model) handleVizKeys(key string) {
	if (m.view == viewPlayer || m.view == viewViz) && m.vizFrame != nil {
		switch key {
		case "c":
			m.preset = (m.preset + 1) % int(visualizations.PatternTypeCount)
		case "i":
			m.colorMode = (m.colorMode + 1) % int(visualizations.ColorModeCount)
		case "p":
			m.palette = (m.palette + 1) % int(visualizations.PaletteTypeCount)
		}
	}
}

func (m *Model) adjustVolume(delta float64) {
	// We don't have current status locally easily, so we just send relative?
	// The server handles absolute volume. TUI should probably just send CmdVolumeUp/Down
	// or track the last known volume from server state.
	m.sendCommand(ipc.Command{Type: ipc.CmdVolume, Payload: delta})
}

func (m *Model) toggleMute() {
	m.sendCommand(ipc.Command{Type: ipc.CmdMute})
}

func (m *Model) seek(d time.Duration) {
	m.sendCommand(ipc.Command{Type: ipc.CmdSeek, Payload: d})
}

func (m *Model) nextTrack(cmds *[]tea.Cmd) {
	m.sendCommand(ipc.Command{Type: ipc.CmdNext})
}

func (m *Model) prevTrack(cmds *[]tea.Cmd) {
	m.sendCommand(ipc.Command{Type: ipc.CmdPrev})
}

func (m *Model) toggleTTSLanguage() {
	lang := "en"
	if m.cfg.TTSLanguage == "en" {
		lang = "cs"
	}
	m.sendCommand(ipc.Command{Type: ipc.CmdTTSLanguage, Payload: lang})
}

func (m *Model) cycleTTSSpeaker() {
	speaker := (int(m.cfg.TTSSpeed) + 1) % 2 // Assume 2 speakers for now
	m.sendCommand(ipc.Command{Type: ipc.CmdTTSSpeaker, Payload: speaker})
}
