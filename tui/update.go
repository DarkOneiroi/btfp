package tui

import (
	"btfp/player"
	"btfp/utils"
	"btfp/visualizations"
	"math"
	"math/rand"
	"path/filepath"
	"strings"
	"time"

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
			// If handleKeyPress returns a non-nil cmd, it might be tea.Quit
			// We check if it's tea.Quit by running it? No, we check the type if possible
			// or just append it. But for Quit, we want to return immediately.
			cmds = append(cmds, cmd)
		}

	case tea.WindowSizeMsg:
		m.handleWindowResize(msg)

	case tickMsg:
		m.handlePlaybackTick(&cmds)

	case errMsg:
		if m.conn != nil {
			return m, tea.Quit
		}
		m.err = msg
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

	currentStatus := m.player.GetStatus()
	if msg.CurrentTrack != nil {
		if currentStatus.CurrentTrack == nil || currentStatus.CurrentTrack.Path != msg.CurrentTrack.Path {
			newTrack := &player.Track{
				Title:  msg.CurrentTrack.Title,
				Artist: msg.CurrentTrack.Artist,
				Path:   msg.CurrentTrack.Path,
				Length: msg.CurrentTrack.Length,
			}
			currentStatus.CurrentTrack = newTrack
			*cmds = append(*cmds, m.syncMetadataAndArt(newTrack.Path))
		}
	} else {
		currentStatus.CurrentTrack = nil
	}

	currentStatus.IsPlaying = msg.IsPlaying
	currentStatus.IsMuted = msg.IsMuted
	currentStatus.Volume = msg.Volume
	currentStatus.Elapsed = msg.Elapsed
	m.player.SetStatus(currentStatus)
	m.syncPlaylist()

	*cmds = append(*cmds, m.listenToServer())
}

func (m *Model) handleLyricsDownloaded(msg lyricsDownloadedMsg, cmds *[]tea.Cmd) {
	status := m.player.GetStatus()
	if status.CurrentTrack != nil {
		lrcPath := strings.TrimSuffix(status.CurrentTrack.Path, filepath.Ext(status.CurrentTrack.Path)) + ".lrc"
		if msg.path == lrcPath {
			m.currentLyrics = msg.lyrics
			if m.view == viewPlayer && (m.bgMode == bgEmpty || m.bgMode == bgVisualization) {
				m.bgMode = bgKaraoke
			}
		}
	}

	base := strings.TrimSuffix(msg.path, ".lrc")
	for _, ext := range utils.SupportedExtensions {
		delete(m.metadataCache, base+ext)
	}

	if m.view == viewLibrary {
		*cmds = append(*cmds, m.loadDirectory(m.currentDir))
	}
}

func (m *Model) handleVizTick(cmds *[]tea.Cmd) {
	if (m.view == viewPlayer || m.view == viewViz) && (m.bgMode == bgVisualization || m.bgMode == bgEQBars) {
		if m.vizFrame == nil {
			m.vizFrame = visualizations.NewFrame(m.width, m.height, visualizations.PatternType(m.preset))
			m.vizFrame.ColorMode = visualizations.ColorMode(m.colorMode)
			m.vizFrame.PaletteType = visualizations.PaletteType(m.palette)
		}

		m.vizFrame.Time = time.Since(m.startTime).Seconds()
		levels := make([]float64, 32)

		status := m.player.GetStatus()
		if status.IsPlaying && !status.IsMuted {
			for i := range levels {
				levels[i] = (math.Sin(m.vizFrame.Time*float64(i+1)*0.5)+1.0)*0.3 + rand.Float64()*0.2
				levels[i] *= status.Volume
				if levels[i] > 1.0 {
					levels[i] = 1.0
				}
			}
		}

		m.vizFrame.AudioLevels = levels
		if m.bgMode == bgEQBars {
			m.vizFrame.PatternType = visualizations.PatternEQ
		} else {
			m.vizFrame.PatternType = visualizations.PatternType(m.preset)
		}
		m.vizFrame.GeneratePattern(levels[0])
	}
	*cmds = append(*cmds, vizTick())
}

func (m *Model) handleWindowResize(msg tea.WindowSizeMsg) {
	m.width, m.height = msg.Width, msg.Height
	m.libList.SetSize(m.width/3-2, m.height-2)
	m.playList.SetSize(m.width/2-2, m.height-2)
	if m.vizFrame != nil {
		m.vizFrame.Width, m.vizFrame.Height = m.width, m.height
		m.vizFrame.Data = make([]float64, m.width*m.height)
	}
}

func (m *Model) handlePlaybackTick(cmds *[]tea.Cmd) {
	if m.conn == nil {
		m.player.Update()
		if m.player.GetStatus().IsDone {
			m.skipTrack(1)
			*cmds = append(*cmds, m.syncMetadataAndArt(m.playlist[m.playingIdx].Path))
		}
	}
	*cmds = append(*cmds, tick())
}
