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
	return func() tea.Msg {
		conn, err := net.Dial("unix", ipc.LibrarySocketPath)
		if err != nil {
			return errMsg(err)
		}
		defer func() { _ = conn.Close() }()

		enc := gob.NewEncoder(conn)
		dec := gob.NewDecoder(conn)

		_ = enc.Encode(ipc.Command{Type: ipc.CmdLibScan, Payload: path})

		var result ipc.MsgLibEntries
		if err := dec.Decode(&result); err != nil {
			return errMsg(err)
		}

		var items []list.Item
		home, _ := os.UserHomeDir()
		if path != filepath.Join(home, "Music") && path != "/" {
			items = append(items, item{title: "..", desc: "Go up", path: filepath.Dir(path), isDir: true})
		}

		for _, entry := range result.Entries {
			inPlaylist := false
			if !entry.IsDir {
				if m.playlistCounts[entry.Path] > 0 {
					inPlaylist = true
				}
			} else {
				if cnt := m.playlistCounts[entry.Path]; cnt > 0 {
					inPlaylist = true
				}
			}
			items = append(items, item{
				title:      entry.Title,
				desc:       entry.Desc,
				path:       entry.Path,
				isDir:      entry.IsDir,
				selected:   m.selectedPaths[entry.Path],
				inPlaylist: inPlaylist,
			})
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

func (m *Model) getTrackMetadata(path string) ipc.TrackInfo {
	conn, err := net.Dial("unix", ipc.LibrarySocketPath)
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
	conn, err := net.Dial("unix", ipc.PlaylistSocketPath)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	enc := gob.NewEncoder(conn)

	info, _ := os.Stat(path)
	if info != nil && info.IsDir() {
		_ = filepath.Walk(path, func(p string, f os.FileInfo, err error) error {
			if err == nil && !f.IsDir() && utils.IsSupportedAudioFile(f.Name()) {
				track := m.getTrackMetadata(p)
				_ = enc.Encode(ipc.Command{Type: ipc.CmdPlaylistAdd, Payload: track})
				m.sendCommand(ipc.Command{Type: ipc.CmdPlaylistAdd, Payload: track})
			}
			return nil
		})
	} else if info != nil {
		track := m.getTrackMetadata(path)
		_ = enc.Encode(ipc.Command{Type: ipc.CmdPlaylistAdd, Payload: track})
		m.sendCommand(ipc.Command{Type: ipc.CmdPlaylistAdd, Payload: track})
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
