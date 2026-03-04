// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package tui

import (
	"btfp/internal/ipc-shared"
	"btfp/internal/utils"
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

// --- Library & Navigation Helpers ---

func (m *Model) loadDirectory(path string) tea.Cmd {
	selected := make(map[string]bool)
	for k, v := range m.selectedPaths { selected[k] = v }
	counts := make(map[string]int)
	for k, v := range m.playlistCounts { counts[k] = v }
	session := m.session

	return func() tea.Msg {
		socketPath := ipc.GetSocketPath("library", session)
		conn, err := net.DialTimeout("unix", socketPath, 500*time.Millisecond)
		if err != nil {
			return nil 
		}
		defer func() { _ = conn.Close() }()

		enc := gob.NewEncoder(conn)
		dec := gob.NewDecoder(conn)

		_ = enc.Encode(ipc.Command{Type: ipc.CmdLibScan, Payload: path})

		var result ipc.MsgLibEntries
		if err := dec.Decode(&result); err != nil {
			return nil
		}

		var items []list.Item
		home, _ := os.UserHomeDir()
		if path != filepath.Join(home, "Music") && path != "/" {
			items = append(items, item{title: "..", desc: "Go up", path: filepath.Dir(path), isDir: true})
		}

		for _, entry := range result.Entries {
			items = append(items, item{
				title:      entry.Title,
				desc:       entry.Desc,
				path:       entry.Path,
				isDir:      entry.IsDir,
				selected:   selected[entry.Path],
				inPlaylist: counts[entry.Path] > 0,
			})
		}
		return libraryMsg(items)
	}
}

func (m *Model) syncPlaylist() {
	m.updatePlaylistCounts()
	m.updateLibraryIcons()
	if m.view == viewPlaylist {
		m.updatePlaylistItems()
	}
}

func (m *Model) updateLibraryIcons() {
	items := m.libList.Items()
	for i, it := range items {
		itm := it.(item)
		inPl := m.playlistCounts[itm.path] > 0
		if itm.inPlaylist != inPl {
			itm.inPlaylist = inPl
			m.libList.SetItem(i, itm)
		}
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
			if parent == dir { break }
			dir = parent
		}
	}
}

// --- Metadata & Lyrics Helpers ---

func (m *Model) getTrackMetadata(path string) ipc.TrackInfo {
	socketPath := ipc.GetSocketPath("library", m.session)
	conn, err := net.DialTimeout("unix", socketPath, 500*time.Millisecond)
	if err != nil {
		return ipc.TrackInfo{Title: filepath.Base(path), Path: path}
	}
	defer func() { _ = conn.Close() }()

	enc := gob.NewEncoder(conn)
	dec := gob.NewDecoder(conn)

	_ = enc.Encode(ipc.Command{Type: ipc.CmdLibGetMetadata, Payload: path})

	var info ipc.TrackInfo
	_ = dec.Decode(&info)
	return info
}

func (m *Model) loadLyrics(path string) tea.Cmd {
	return func() tea.Msg {
		lrcPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".lrc"
		if _, err := os.Stat(lrcPath); err == nil {
			content, _ := os.ReadFile(lrcPath)
			lyrics, _ := m.parseLyrics(string(content))
			return lyricsDownloadedMsg{path: lrcPath, lyrics: lyrics}
		}
		return nil
	}
}

func (m *Model) syncMetadataAndArt(songPath string) tea.Cmd {
	session := m.session
	return func() tea.Msg {
		socketPath := ipc.GetSocketPath("fetcher", session)
		conn, err := net.DialTimeout("unix", socketPath, 200*time.Millisecond)
		if err != nil {
			return nil
		}
		defer func() { _ = conn.Close() }()

		enc := gob.NewEncoder(conn)
		dec := gob.NewDecoder(conn)

		info, _ := os.Stat(songPath)
		isDir := info != nil && info.IsDir()

		dir := songPath
		if !isDir {
			dir = filepath.Dir(songPath)
		}

		track := m.getTrackMetadata(songPath)
		
		// For directories, getTrackMetadata might not be as useful, 
		// use directory name as potential album name if needed
		artist := track.Artist
		album := track.Album
		if isDir && album == "" {
			album = filepath.Base(songPath)
		}

		// Request Lyrics (only for files)
		if !isDir {
			lrcPath := strings.TrimSuffix(songPath, filepath.Ext(songPath)) + ".lrc"
			if _, err := os.Stat(lrcPath); os.IsNotExist(err) && m.cfg.AutoDownloadLyrics {
				_ = enc.Encode(ipc.Command{Type: ipc.CmdFetchLyrics, Payload: []string{artist, track.Title, lrcPath}})
				var res ipc.MsgFetchResult
				if err := dec.Decode(&res); err == nil && res.Type == "lyrics" {
					lyrics, _ := m.parseLyrics(res.Content)
					return lyricsDownloadedMsg{path: lrcPath, lyrics: lyrics}
				}
			}
		}

		// Request Art
		if m.cfg.AutoDownloadArt {
			// Check if already has local art
			hasArt := false
			for _, n := range []string{"cover.jpg", "folder.jpg", "album.jpg", "band.jpg", "artist.jpg"} {
				if _, err := os.Stat(filepath.Join(dir, n)); err == nil {
					hasArt = true
					break
				}
			}

			if !hasArt {
				_ = enc.Encode(ipc.Command{Type: ipc.CmdFetchArt, Payload: []string{artist, album, dir}})
				var res ipc.MsgFetchResult
				if err := dec.Decode(&res); err == nil && res.Type == "art" {
					return artDownloadedMsg(res.Path)
				}
			}
		}

		return nil
	}
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
		if line == "" { continue }
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

func (m *Model) AddPathToPlaylist(path string) {
	info, err := os.Stat(path)
	if err != nil {
		return
	}

	if info.IsDir() {
		_ = filepath.Walk(path, func(p string, i os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !i.IsDir() && utils.IsSupportedAudioFile(p) {
				if strings.ToLower(filepath.Ext(p)) == ".txt" {
					return nil
				}
				track := m.getTrackMetadata(p)
				m.sendCommand(ipc.Command{Type: ipc.CmdPlaylistAdd, Payload: track})
			}
			return nil
		})
	} else {
		track := m.getTrackMetadata(path)
		m.sendCommand(ipc.Command{Type: ipc.CmdPlaylistAdd, Payload: track})
	}
}

// --- IPC Helpers ---

func (m *Model) listenToServer() tea.Cmd {
	session := m.session
	return func() tea.Msg {
		if m.dec == nil {
			socketPath := ipc.GetSocketPath("core", session)
			conn, err := net.DialTimeout("unix", socketPath, 200*time.Millisecond)
			if err == nil {
				m.conn = conn
				m.enc = gob.NewEncoder(conn)
				m.dec = gob.NewDecoder(conn)
			} else {
				time.Sleep(500 * time.Millisecond)
				return tickMsg(time.Now())
			}
		}
		
		var state ipc.PlayerState
		if err := m.dec.Decode(&state); err != nil {
			m.dec = nil // Mark for reconnection
			time.Sleep(500 * time.Millisecond)
			return tickMsg(time.Now())
		}
		return serverStateMsg(state)
	}
}

func (m *Model) requestVizFrame(w, h int, isPlaying, isMuted bool, vol float64, preset, colorMode, palette int, bg backgroundMode, startTime time.Time) tea.Cmd {
	session := m.session
	return func() tea.Msg {
		if bg != bgVisualization { return nil }

		socketPath := ipc.GetSocketPath("viz", session)
		conn, err := net.DialTimeout("unix", socketPath, 100*time.Millisecond)
		if err != nil { return nil }
		defer func() { _ = conn.Close() }()

		enc := gob.NewEncoder(conn)
		dec := gob.NewDecoder(conn)

		payload := map[string]interface{}{
			"width":     w,
			"height":    h,
			"isPlaying": isPlaying && !isMuted,
			"volume":    vol,
			"pattern":   preset,
			"colorMode": colorMode,
			"palette":   palette,
			"time":      time.Since(startTime).Seconds(),
		}

		_ = enc.Encode(ipc.Command{Type: ipc.CmdVizGenerate, Payload: payload})
		var rendered string
		if err := dec.Decode(&rendered); err != nil { return nil }
		return vizFrameMsg(rendered)
	}
}

func (m *Model) cycleBGMode() {
	for i := 0; i < 4; i++ {
		m.bgMode = (m.bgMode + 1) % 4
		if m.bgMode == bgImage && m.cfg.ImagePath == "" { continue }
		if m.bgMode == bgKaraoke && len(m.currentLyrics) == 0 { continue }
		break
	}
}

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
