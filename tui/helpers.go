// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package tui

import (
	"btfp/ipc"
	"btfp/player"
	"btfp/utils"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bogem/id3v2/v2"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// --- Library & Navigation Helpers ---

func (m *Model) loadDirectory(path string) tea.Cmd {
	return func() tea.Msg {
		entries, err := os.ReadDir(path)
		if err != nil {
			return errMsg(err)
		}
		var items []list.Item
		home, _ := os.UserHomeDir()
		if path != filepath.Join(home, "Music") && path != "/" {
			items = append(items, item{title: "..", desc: "Go up", path: filepath.Dir(path), isDir: true})
		}
		for _, entry := range entries {
			info, _ := entry.Info()
			fullPath := filepath.Join(path, entry.Name())
			if entry.IsDir() || utils.IsSupportedAudioFile(entry.Name()) {
				desc := "Dir"
				inPlaylist := false
				if !entry.IsDir() {
					desc = fmt.Sprintf("%.1f MB", float64(info.Size())/1024/1024)
					if m.playlistCounts[fullPath] > 0 {
						inPlaylist = true
					}
				} else {
					if cnt := m.playlistCounts[fullPath]; cnt > 0 {
						desc = fmt.Sprintf("Dir (%d tracks)", cnt)
						inPlaylist = true
					}
				}
				items = append(items, item{
					title:      entry.Name(),
					desc:       desc,
					path:       fullPath,
					isDir:      entry.IsDir(),
					selected:   m.selectedPaths[fullPath],
					inPlaylist: inPlaylist,
				})
			}
		}
		return libraryMsg(items)
	}
}

func (m *Model) syncPlaylist() {
	m.updatePlaylistCounts()
	if m.view == viewPlaylist {
		m.updatePlaylistItems()
	}
}

func (m *Model) updatePlaylistItems() {
	items := make([]list.Item, len(m.playlist))
	for i, t := range m.playlist {
		title := t.Title
		if i == m.playingIdx {
			title = "▶ " + title
		}
		items[i] = item{title: title, desc: t.Artist, path: t.Path}
	}
	m.playList.SetItems(items)
}

func (m *Model) updatePlaylistCounts() {
	m.playlistCounts = make(map[string]int)
	for _, t := range m.playlist {
		m.playlistCounts[t.Path]++
		dir := filepath.Dir(t.Path)
		for {
			m.playlistCounts[dir]++
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
}

// --- Metadata & Lyrics Helpers ---

func (m *Model) getTrackMetadata(path string) player.Track {
	fileName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	dir := filepath.Dir(path)
	artistName := filepath.Base(filepath.Dir(dir))

	track := player.Track{
		Title: fileName,
		Path:  path,
	}

	tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err == nil {
		defer func() { _ = tag.Close() }()
		// Using direct frame IDs to avoid any potential high-level method misinterpretations
		// TPE1 = Lead artist, TIT2 = Title
		artist := tag.GetTextFrame("TPE1").Text
		title := tag.GetTextFrame("TIT2").Text

		if artist != "" {
			track.Artist = artist
		}
		if title != "" {
			track.Title = title
		}
	}

	if track.Artist == "" && artistName != "Music" && artistName != "." {
		track.Artist = artistName
	}

	if track.Artist == "" {
		track.Artist = "Unknown Artist"
	}

	return track
}

func (m *Model) syncMetadataAndArt(songPath string) tea.Cmd {
	dir := filepath.Dir(songPath)
	artistName, albumName := filepath.Base(filepath.Dir(dir)), filepath.Base(dir)
	fileName := strings.TrimSuffix(filepath.Base(songPath), filepath.Ext(songPath))

	var curArtist, curTitle string
	tag, err := id3v2.Open(songPath, id3v2.Options{Parse: true})
	if err == nil {
		if m.cfg.UpdateMetadata {
			if tag.Artist() == "" && artistName != "Music" {
				tag.SetArtist(artistName)
			}
			if tag.Album() == "" {
				tag.SetAlbum(albumName)
			}
			if tag.Title() == "" {
				tag.SetTitle(fileName)
			}
			_ = tag.Save()
		}
		curArtist, curTitle = tag.Artist(), tag.Title()
		_ = tag.Close()
	}

	if curArtist == "" && artistName != "Music" {
		curArtist = artistName
	}
	if curTitle == "" {
		curTitle = fileName
	}

	var cmds []tea.Cmd
	lrcPath := strings.TrimSuffix(songPath, filepath.Ext(songPath)) + ".lrc"
	hasSynced := false
	if _, err := os.Stat(lrcPath); err == nil {
		content, _ := os.ReadFile(lrcPath)
		m.currentLyrics, hasSynced = m.parseLyrics(string(content))
	}

	if (!hasSynced || len(m.currentLyrics) == 0) && m.cfg.AutoDownloadLyrics && curArtist != "" && curTitle != "" {
		cmds = append(cmds, m.downloadLyricsCmd(curArtist, curTitle, lrcPath))
	} else if len(m.currentLyrics) == 0 {
		m.currentLyrics = []lrcLine{}
	}

	hasArt := false
	for _, n := range []string{"cover.jpg", "folder.jpg", "album.jpg", "band.jpg", "artist.jpg"} {
		if _, err := os.Stat(filepath.Join(dir, n)); err == nil {
			hasArt = true
			break
		}
	}
	if !hasArt && m.cfg.AutoDownloadArt {
		cmds = append(cmds, m.downloadArtCmd(dir))
	}

	return tea.Batch(cmds...)
}

func (m *Model) parseLyrics(content string) ([]lrcLine, bool) {
	c := strings.ReplaceAll(content, "\r\n", "\n")
	c = strings.ReplaceAll(c, "\r", "\n")
	lines := strings.Split(c, "\n")

	var res []lrcLine
	re := regexp.MustCompile(`\[(\d+):(\d+(?:\.\d+)?)\](.*)`)

	hasTimestamps := false
	for _, line := range lines {
		if re.MatchString(line) {
			hasTimestamps = true
			break
		}
	}

	lastTime := time.Duration(0)
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		matches := re.FindStringSubmatch(line)
		if len(matches) == 4 {
			min, _ := time.ParseDuration(matches[1] + "m")
			sec, _ := time.ParseDuration(matches[2] + "s")
			lastTime = min + sec
			res = append(res, lrcLine{time: lastTime, text: strings.TrimSpace(matches[3])})
		} else if !strings.HasPrefix(line, "[") {
			if !hasTimestamps {
				res = append(res, lrcLine{time: time.Duration(len(res)) * 5 * time.Second, text: line})
			} else {
				res = append(res, lrcLine{time: lastTime, text: line})
			}
		}
	}
	return res, hasTimestamps
}

func (m *Model) addPathToPlaylist(path string) {
	info, _ := os.Stat(path)
	if info != nil && info.IsDir() {
		_ = filepath.Walk(path, func(p string, f os.FileInfo, err error) error {
			if err == nil && !f.IsDir() && utils.IsSupportedAudioFile(f.Name()) {
				track := m.getTrackMetadata(p)
				m.sendCommand(ipc.Command{Type: ipc.CmdAddTrack, Payload: track})

				if m.conn == nil {
					m.playlist = append(m.playlist, track)
					if m.playingIdx == -1 {
						m.playingIdx = 0
						_ = m.player.PlayTrack(&m.playlist[0])
					}
				}
			}
			return nil
		})
	} else if info != nil {
		track := m.getTrackMetadata(path)
		m.sendCommand(ipc.Command{Type: ipc.CmdAddTrack, Payload: track})

		if m.conn == nil {
			m.playlist = append(m.playlist, track)
			if m.playingIdx == -1 {
				m.playingIdx = 0
				_ = m.player.PlayTrack(&m.playlist[0])
			}
		}
	}
}

// --- IPC Helpers ---

func (m *Model) listenToServer() tea.Cmd {
	return func() tea.Msg {
		if m.dec == nil {
			return nil
		}
		var state ipc.PlayerState
		if err := m.dec.Decode(&state); err != nil {
			return errMsg(err)
		}
		return serverStateMsg(state)
	}
}

func (m *Model) skipTrack(dir int) {
	if len(m.playlist) == 0 {
		return
	}
	m.playingIdx = (m.playingIdx + dir + len(m.playlist)) % len(m.playlist)
	_ = m.player.PlayTrack(&m.playlist[m.playingIdx])
	m.currentLyrics = []lrcLine{}
	m.syncPlaylist()
}

func (m *Model) cycleBGMode() {
	for i := 0; i < 5; i++ {
		m.bgMode = (m.bgMode + 1) % 5
		if m.bgMode == bgImage && m.cfg.ImagePath == "" {
			continue
		}
		if m.bgMode == bgKaraoke && len(m.currentLyrics) == 0 {
			continue
		}
		break
	}
}

// --- Formatting Helpers ---

func formatDuration(d time.Duration) string {
	mins, secs := int(d.Minutes()), int(d.Seconds())%60
	return fmt.Sprintf("%d:%02d", mins, secs)
}

func cleanString(s string) string {
	s = strings.ToLower(s)
	reNum := regexp.MustCompile(`^\d+[\s.\-_]*`)
	s = reNum.ReplaceAllString(s, "")
	reSuffix := regexp.MustCompile(`[\(\[].*?[\)\]]`)
	s = reSuffix.ReplaceAllString(s, "")
	reNoise := regexp.MustCompile(`\b(official|video|audio|lyrics|hd|4k|remastered|remaster|edit|radio|live)\b`)
	s = reNoise.ReplaceAllString(s, "")
	s = regexp.MustCompile(`[ 	]+`).ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
