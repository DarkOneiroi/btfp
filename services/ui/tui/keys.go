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

	tea "github.com/charmbracelet/bubbletea"
)

// processKey handles model-level shortcuts and returns if the key was consumed
func (m *Model) processKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	key := msg.String()
	var cmds []tea.Cmd

	switch key {
	case "q", "ctrl+c":
		m.sendCommand(ipc.Command{Type: ipc.CmdQuit})
		return tea.Quit, true

	case "tab":
		m.handleTab()
		return nil, true

	case "backspace":
		if m.view == viewLibrary {
			m.handleBackspace(&cmds)
			return tea.Batch(cmds...), true
		}

	case " ":
		if m.view == viewLibrary {
			m.handleSpace()
			return nil, true
		}
		m.sendCommand(ipc.Command{Type: ipc.CmdPause})
		return nil, true

	case "a":
		if m.view == viewLibrary {
			m.handleAddTrack(&cmds)
			return tea.Batch(cmds...), true
		}

	case "enter":
		m.handleEnter(&cmds)
		return tea.Batch(cmds...), true

	case "up", "down":
		if m.view == viewPlayer {
			if key == "up" {
				m.adjustVolume(0.1)
			} else {
				m.adjustVolume(-0.1)
			}
			return nil, true
		}

	case "c", "i", "p":
		m.handleVizKeys(key)
		return nil, true

	case "v":
		m.cycleBGMode()
		if m.bgMode != bgVisualization && m.bgMode != bgBars {
			m.vizData = ""
			m.vizPending = false
		}
		return nil, true

	case "h", "?":
		m.showLegend = !m.showLegend
		return nil, true

	case "+", "=":
		m.adjustVolume(0.1)
		return nil, true

	case "-", "_":
		m.adjustVolume(-0.1)
		return nil, true

	case "m":
		m.toggleMute()
		return nil, true

	case "l", "right":
		if m.view == viewPlayer {
			m.seek(5 * time.Second)
			return nil, true
		}

	case "left":
		if m.view == viewPlayer {
			m.seek(-5 * time.Second)
			return nil, true
		}

	case "n":
		m.nextTrack(&cmds)
		return tea.Batch(cmds...), true

	case "b":
		m.prevTrack(&cmds)
		return tea.Batch(cmds...), true
	}

	return nil, false
}

func (m *Model) handleBackspace(cmds *[]tea.Cmd) {
	home, _ := os.UserHomeDir()
	if m.currentDir != filepath.Join(home, "Music") {
		m.currentDir = filepath.Dir(m.currentDir)
		*cmds = append(*cmds, m.loadDirectory(m.currentDir))
	}
}

func (m *Model) handleTab() {
	if !m.lockedView {
		m.view = (m.view + 1) % 4
		if m.view == viewPlaylist {
			m.updatePlaylistItems()
		}
	}
}

func (m *Model) handleSpace() {
	if sel, ok := m.libList.SelectedItem().(item); ok && sel.title != ".." {
		m.selectedPaths[sel.path] = !m.selectedPaths[sel.path]
		idx := m.libList.Index()
		itm := sel
		itm.selected = m.selectedPaths[sel.path]
		m.libList.SetItem(idx, itm)
	}
}

func (m *Model) handleAddTrack(cmds *[]tea.Cmd) {
	for p, s := range m.selectedPaths {
		if s {
			m.AddPathToPlaylist(p)
		}
	}
	m.selectedPaths = make(map[string]bool)
	m.syncPlaylist()
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
				if !m.multiWindow {
					m.view = viewPlayer
				}
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
	// Sync preset if it was forced to EQ by background mode
	if m.preset >= int(visualizations.PatternTypeCount) {
		m.preset = 0
	}
	switch key {
	case "c":
		m.preset = (m.preset + 1) % int(visualizations.PatternTypeCount)
	case "i":
		m.colorMode = (m.colorMode + 1) % int(visualizations.ColorModeCount)
	case "p":
		m.palette = (m.palette + 1) % int(visualizations.PaletteTypeCount)
	}
	m.vizPending = false
}

func (m *Model) adjustVolume(delta float64) {
	m.volume += delta
	if m.volume > 1.0 { m.volume = 1.0 }
	if m.volume < 0.0 { m.volume = 0.0 }
	m.sendCommand(ipc.Command{Type: ipc.CmdVolume, Payload: m.volume})
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

