// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package tui

import (
	"btfp/internal/ipc-shared"
	"btfp/internal/utils"
	"btfp/services/core/player"
	"btfp/services/visualization/visualizations"
	"encoding/gob"
	"net"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Update handles all state transitions based on messages
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case serverStateMsg:
		if msg.ShouldQuit {
			return m, tea.Quit
		}
		m.handleServerState(msg, &cmds)

	case libraryMsg:
		m.libList.SetItems(msg)

	case artDownloadedMsg:
		delete(m.artCache, filepath.Dir(string(msg)))
		if m.view == viewLibrary {
			cmds = append(cmds, m.loadDirectory(m.currentDir))
		}

	case lyricsDownloadedMsg:
		m.handleLyricsDownloaded(msg, &cmds)

	case vizTickMsg:
		m.handleVizTick(&cmds)

	case tea.KeyMsg:
		cmd := m.handleKeyPress(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

	case tea.WindowSizeMsg:
		m.handleWindowResize(msg)

	case tickMsg:
		m.handlePlaybackTick(&cmds)

	case errMsg:
		return m, tea.Quit
	}

	// ALWAYS update active lists so they handle cursor movement, filtering, etc.
	var cmdLib, cmdPlay tea.Cmd
	m.libList, cmdLib = m.libList.Update(msg)
	m.playList, cmdPlay = m.playList.Update(msg)
	cmds = append(cmds, cmdLib, cmdPlay)

	return m, tea.Batch(cmds...)
}

func (m *Model) handleServerState(msg serverStateMsg, cmds *[]tea.Cmd) {
	m.playingIdx = msg.PlayingIdx
	m.playlist = make([]player.Track, len(msg.Playlist))
	for i, t := range msg.Playlist {
		m.playlist[i] = player.Track{Title: t.Title, Artist: t.Artist, Path: t.Path, Length: t.Length}
	}

	m.syncPlaylist()

	m.cfg.TTSLanguage = msg.TTSLanguage
	m.cfg.TTSSpeed = float64(msg.TTSSpeaker) // Temp mapping back

	*cmds = append(*cmds, m.listenToServer())
}

func (m *Model) handleLyricsDownloaded(msg lyricsDownloadedMsg, _ *[]tea.Cmd) {
	m.currentLyrics = msg.lyrics
	if m.view == viewPlayer && (m.bgMode == bgEmpty || m.bgMode == bgVisualization) {
		m.bgMode = bgKaraoke
	}

	base := strings.TrimSuffix(msg.path, ".lrc")
	for _, ext := range utils.SupportedExtensions {
		delete(m.metadataCache, base+ext)
	}
}

func (m *Model) handleVizTick(cmds *[]tea.Cmd) {
	if (m.view == viewPlayer || m.view == viewViz) && (m.bgMode == bgVisualization || m.bgMode == bgEQBars) {
		if m.vizConn == nil {
			conn, err := net.Dial("unix", ipc.VizSocketPath)
			if err == nil {
				m.vizConn = conn
			}
		}

		if m.vizConn != nil {
			enc := gob.NewEncoder(m.vizConn)
			dec := gob.NewDecoder(m.vizConn)

			levels := make([]float64, 32)
			pattern := m.preset
			if m.bgMode == bgEQBars {
				pattern = int(visualizations.PatternEQ)
			}

			payload := map[string]interface{}{
				"width":   m.width,
				"height":  m.height,
				"levels":  levels,
				"pattern": pattern,
			}

			_ = enc.Encode(ipc.Command{Type: ipc.CmdVizGenerate, Payload: payload})
			var rendered string
			_ = dec.Decode(&rendered)
		}
	}
	*cmds = append(*cmds, vizTick())
}

func (m *Model) handleWindowResize(msg tea.WindowSizeMsg) {
	m.width, m.height = msg.Width, msg.Height
	m.libList.SetSize(m.width/3-2, m.height-2)
	m.playList.SetSize(m.width/2-2, m.height-2)
}

func (m *Model) handlePlaybackTick(cmds *[]tea.Cmd) {
	*cmds = append(*cmds, tick())
}
