package tui

import (
	"btfp/ipc"
	"btfp/visualizations"
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
		if m.conn == nil {
			return tea.Quit
		}

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
		if m.conn == nil {
			m.player.TogglePause()
		}
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

				if m.conn == nil {
					m.playlist = append(m.playlist, track)
					m.playingIdx = len(m.playlist) - 1
					_ = m.player.PlayTrack(&m.playlist[m.playingIdx])
					m.syncPlaylist()
					*cmds = append(*cmds, m.syncMetadataAndArt(sel.path))
				}
				m.view = viewPlayer
			}
		}
	} else if m.view == viewPlaylist {
		if m.playList.Index() >= 0 && m.playList.Index() < len(m.playlist) {
			idx := m.playList.Index()
			m.sendCommand(ipc.Command{Type: ipc.CmdPlay, Payload: idx})

			if m.conn == nil {
				m.playingIdx = idx
				_ = m.player.PlayTrack(&m.playlist[m.playingIdx])
				m.syncPlaylist()
				*cmds = append(*cmds, m.syncMetadataAndArt(m.playlist[m.playingIdx].Path))
			}
		}
	}
}

func (m *Model) handleVizKeys(key string) {
	if (m.view == viewPlayer || m.view == viewViz) && m.vizFrame != nil {
		switch key {
		case "c":
			m.preset = (m.preset + 1) % int(visualizations.PatternTypeCount)
			m.vizFrame.PatternType = visualizations.PatternType(m.preset)
		case "i":
			m.colorMode = (m.colorMode + 1) % int(visualizations.ColorModeCount)
			m.vizFrame.ColorMode = visualizations.ColorMode(m.colorMode)
		case "p":
			m.palette = (m.palette + 1) % int(visualizations.PaletteTypeCount)
			m.vizFrame.PaletteType = visualizations.PaletteType(m.palette)
		}
	}
}

func (m *Model) adjustVolume(delta float64) {
	vol := m.player.GetStatus().Volume + delta
	if vol < 0 {
		vol = 0
	}
	if vol > 1 {
		vol = 1
	}
	m.sendCommand(ipc.Command{Type: ipc.CmdVolume, Payload: vol})
	if m.conn == nil {
		m.player.SetVolume(vol)
	}
}

func (m *Model) toggleMute() {
	m.sendCommand(ipc.Command{Type: ipc.CmdMute})
	if m.conn == nil {
		m.player.ToggleMute()
	}
}

func (m *Model) seek(d time.Duration) {
	m.sendCommand(ipc.Command{Type: ipc.CmdSeek, Payload: d})
	if m.conn == nil {
		m.player.Seek(d)
	}
}

func (m *Model) nextTrack(cmds *[]tea.Cmd) {
	m.sendCommand(ipc.Command{Type: ipc.CmdNext})
	if m.conn == nil {
		m.skipTrack(1)
		if m.playingIdx >= 0 && m.playingIdx < len(m.playlist) {
			*cmds = append(*cmds, m.syncMetadataAndArt(m.playlist[m.playingIdx].Path))
		}
	}
}

func (m *Model) prevTrack(cmds *[]tea.Cmd) {
	m.sendCommand(ipc.Command{Type: ipc.CmdPrev})
	if m.conn == nil {
		m.skipTrack(-1)
		if m.playingIdx >= 0 && m.playingIdx < len(m.playlist) {
			*cmds = append(*cmds, m.syncMetadataAndArt(m.playlist[m.playingIdx].Path))
		}
	}
}
