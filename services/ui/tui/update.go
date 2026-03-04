// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package tui

import (
	"btfp/internal/models"
	"path/filepath"

	"github.com/charmbracelet/bubbles/list"
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
		newTrackStarted := false
		if msg.CurrentTrack != nil && (m.currTrack == nil || m.currTrack.Path != msg.CurrentTrack.Path) {
			newTrackStarted = true
		}

		m.handleServerState(msg)

		if newTrackStarted {
			cmds = append(cmds, m.loadLyrics(m.currTrack.Path), m.syncMetadataAndArt(m.currTrack.Path))
		}
		cmds = append(cmds, m.listenToServer())

	case vizFrameMsg:
		m.vizData = string(msg)
		m.vizPending = false

	case vizTickMsg:
		if m.bgMode != bgVisualization {
			m.vizData = ""
			m.vizPending = false
		} else if !m.vizPending {
			m.vizPending = true
			cmds = append(cmds, m.requestVizFrame(m.width, m.height, m.isPlaying, m.isMuted, m.volume, m.preset, m.colorMode, m.palette, m.bgMode, m.startTime))
		}
		cmds = append(cmds, vizTick())

	case libraryMsg:
		oldIdx := m.libList.Index()
		m.libList.SetItems(msg)
		if oldIdx < len(m.libList.Items()) {
			m.libList.Select(oldIdx)
		}

	case artDownloadedMsg:
		delete(m.artCache, filepath.Dir(string(msg)))
		if m.view == viewLibrary {
			cmds = append(cmds, m.loadDirectory(m.currentDir))
		}

	case lyricsDownloadedMsg:
		m.currentLyrics = msg.lyrics

	case tea.KeyMsg:
		if m.libList.FilterState() == list.Filtering {
			var listCmd tea.Cmd
			m.libList, listCmd = m.libList.Update(msg)
			return m, listCmd
		}
		if m.playList.FilterState() == list.Filtering {
			var listCmd tea.Cmd
			m.playList, listCmd = m.playList.Update(msg)
			return m, listCmd
		}

		modelCmd, handled := m.processKey(msg)
		if handled {
			return m, modelCmd
		}

		var listCmd tea.Cmd
		if m.view == viewLibrary {
			m.libList, listCmd = m.libList.Update(msg)
			if sel, ok := m.libList.SelectedItem().(item); ok {
				cmds = append(cmds, m.syncMetadataAndArt(sel.path))
			}
		} else if m.view == viewPlaylist {
			m.playList, listCmd = m.playList.Update(msg)
			if sel, ok := m.playList.SelectedItem().(item); ok {
				cmds = append(cmds, m.syncMetadataAndArt(sel.path))
			}
		}
		return m, tea.Batch(append(cmds, listCmd)...)

	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.libList.SetSize(m.width/3-2, m.height-2)
		m.playList.SetSize(m.width/2-2, m.height-2)

	case tickMsg:
		cmds = append(cmds, tick())

	case errMsg:
		m.vizPending = false
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleServerState(msg serverStateMsg) {
	m.playingIdx = msg.PlayingIdx
	m.playlist = make([]models.Track, len(msg.Playlist))
	for i, t := range msg.Playlist {
		m.playlist[i] = models.Track{Title: t.Title, Artist: t.Artist, Path: t.Path, Length: t.Length}
	}

	if msg.CurrentTrack != nil {
		m.currTrack = &models.Track{
			Title:  msg.CurrentTrack.Title,
			Artist: msg.CurrentTrack.Artist,
			Path:   msg.CurrentTrack.Path,
			Length: msg.CurrentTrack.Length,
		}
	} else {
		m.currTrack = nil
	}

	m.isPlaying = msg.IsPlaying
	m.isMuted = msg.IsMuted
	m.volume = msg.Volume
	m.elapsed = msg.Elapsed

	m.syncPlaylist()
}
