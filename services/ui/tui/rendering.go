// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package tui

import (
	"btfp/internal/utils"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bogem/id3v2/v2"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// --- View Rendering ---

func (m *Model) renderLibraryView() string {
	tree, content, right := m.renderTree(), m.renderListView(m.libList), m.renderRightPanel()
	l := lipgloss.NewStyle().Width(m.width/3).MaxHeight(m.height).Padding(0, 1).Render(tree)
	mid := lipgloss.NewStyle().Width(m.width/3).MaxHeight(m.height).Padding(0, 1).Render(content)
	r := lipgloss.NewStyle().Width(m.width/3).MaxHeight(m.height).Padding(0, 1).Render(right)
	return lipgloss.JoinHorizontal(lipgloss.Top, l, mid, r)
}

func (m *Model) renderPlaylistView() string {
	content, right := m.renderListView(m.playList), m.renderRightPanel()
	l := lipgloss.NewStyle().Width(m.width/2-2).MaxHeight(m.height).Padding(0, 1).Render(content)
	r := lipgloss.NewStyle().Width(m.width/2-2).MaxHeight(m.height).Padding(0, 1).Render(right)
	return lipgloss.JoinHorizontal(lipgloss.Top, l, r)
}

func (m *Model) renderPlayerView() string {
	var trackTitle string
	var trackLength time.Duration
	var elapsed time.Duration
	var volume float64
	var isPlaying bool
	var isMuted bool

	// In a real microservices setup, these would come from m.lastServerState
	// For now we assume they are synced into the model fields if we had them.
	// Since we refactored handleServerState to not update player, we need to store them in Model.

	if trackTitle == "" {
		return lipgloss.NewStyle().Width(40).Align(lipgloss.Center).Render("No track playing.\nPress [tab] for Library.")
	}
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Title)).Bold(true)
	barW := 40
	pct := 0.0
	if trackLength > 0 {
		pct = float64(elapsed) / float64(trackLength)
	}
	fill := int(float64(barW) * pct)
	if fill > barW {
		fill = barW
	}
	bar := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Highlight)).Render(strings.Repeat("━", fill)) + lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Subtext)).Render(strings.Repeat("━", barW-fill))
	stateText := "PLAYING"
	if !isPlaying {
		stateText = "PAUSED"
	}
	volStr := fmt.Sprintf("VOL: %d%%", int(volume*100))
	if isMuted {
		volStr = "VOL: MUTE"
	}

	ttsInfo := fmt.Sprintf("TTS: %s | Voice: %d", strings.ToUpper(m.cfg.TTSLanguage), int(m.cfg.TTSSpeed))

	ui := lipgloss.JoinVertical(lipgloss.Center, lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Accent)).Bold(true).Render("BTFP PLAYER"), "", titleStyle.Render(trackTitle), fmt.Sprintf("%s / %s", formatDuration(elapsed), formatDuration(trackLength)), volStr, ttsInfo, "", bar, "", lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Highlight)).Render(stateText), "\n[h] Toggle Help")
	return lipgloss.NewStyle().Width(80).Height(20).Align(lipgloss.Center, lipgloss.Center).Render(ui)
}

func (m *Model) renderRightPanel() string {
	var songPath string
	if m.view == viewLibrary {
		if sel, ok := m.libList.SelectedItem().(item); ok {
			songPath = sel.path
		}
	} else if m.view == viewPlaylist {
		if sel, ok := m.playList.SelectedItem().(item); ok {
			songPath = sel.path
		}
	} else if m.playingIdx >= 0 {
		songPath = m.playlist[m.playingIdx].Path
	}

	if songPath == "" {
		return lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Subtext)).Render("\n\n   (No Selection)")
	}
	return lipgloss.JoinVertical(lipgloss.Left, m.renderArt(songPath), m.renderMetadata(songPath), lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.theme.Accent)).Render(fmt.Sprintf("\nQUEUE: %d tracks", len(m.playlist))))
}

func (m *Model) renderArt(path string) string {
	info, _ := os.Stat(path)
	dir := path
	if info != nil && !info.IsDir() {
		dir = filepath.Dir(path)
	}

	if art, ok := m.artCache[dir]; ok {
		return art
	}

	var artPath string
	for _, n := range []string{"cover.jpg", "folder.jpg", "album.jpg", "band.jpg", "artist.jpg"} {
		p := filepath.Join(dir, n)
		if _, err := os.Stat(p); err == nil {
			artPath = p
			break
		}
	}

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.theme.Accent)).Render("COVER ART")
	if artPath == "" {
		return fmt.Sprintf("\n%s\n\n   (No Art)", title)
	}

	art := fmt.Sprintf("\n%s\n%s", title, utils.ImageToASCII(artPath, m.width/5))
	m.artCache[dir] = art
	return art
}

func (m *Model) renderMetadata(path string) string {
	info, _ := os.Stat(path)
	if info != nil && info.IsDir() {
		return ""
	}

	if meta, ok := m.metadataCache[path]; ok {
		return meta
	}

	tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err == nil {
		defer func() { _ = tag.Close() }()
	}

	artist := "Unknown"
	album := "Unknown"
	title := filepath.Base(path)
	genre := "Unknown"
	year := "Unknown"

	if tag != nil {
		artist = tag.GetTextFrame("TPE1").Text
		album = tag.GetTextFrame("TALB").Text
		title = tag.GetTextFrame("TIT2").Text
		genre = tag.GetTextFrame("TCON").Text
		year = tag.GetTextFrame("TYER").Text
	}

	accent := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.theme.Accent))
	lyricsStatus := "Missing"
	if _, err := os.Stat(strings.TrimSuffix(path, filepath.Ext(path)) + ".lrc"); err == nil {
		lyricsStatus = "Available"
	}

	meta := fmt.Sprintf("\n%s\n Artist: %s\n Album:  %s\n Title:  %s\n Genre:  %s\n Year:   %s\n Lyrics: %s",
		accent.Render("METADATA"), artist, album, title, genre, year, lyricsStatus)

	m.metadataCache[path] = meta
	return meta
}

func (m *Model) renderTree() string {
	home, _ := os.UserHomeDir()
	musicDir := filepath.Join(home, "Music")
	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.theme.Accent)).Render("FOLDER TREE") + "\n\n")
	rel, _ := filepath.Rel(musicDir, m.currentDir)
	parts := strings.Split(rel, string(filepath.Separator))
	curPath := musicDir
	for i, p := range parts {
		pref := "📁 "
		if p == "." {
			p = "Music"
		} else {
			sb.WriteString(strings.Repeat("  ", i))
			pref = "└─ 📁 "
		}

		if len(parts) > 5 && i < len(parts)-5 {
			continue
		}

		cnt := m.playlistCounts[curPath]
		line := pref + p
		if cnt > 0 {
			line += lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Highlight)).Render(fmt.Sprintf(" (%d)", cnt))
		}
		sb.WriteString(line + "\n")
		if p != "Music" {
			curPath = filepath.Join(curPath, p)
		}
	}
	return sb.String()
}

func (m *Model) renderLegend() string {
	var keys [][]string
	switch m.view {
	case viewLibrary:
		keys = [][]string{
			{"[enter]", "Play / Open"},
			{"[space]", "Select"},
			{"[a]", "Add Selected"},
			{"[back]", "Go Up"},
			{"[/]", "Filter"},
		}
	case viewPlaylist:
		keys = [][]string{
			{"[enter]", "Play Selected"},
		}
	case viewPlayer:
		keys = [][]string{
			{"[space]", "Pause/Play"},
			{"[n/b]", "Next/Prev"},
			{"[l/r]", "Seek 5s"},
			{"[+/-]", "Volume"},
			{"[m]", "Mute"},
			{"[v]", "BG Mode"},
			{"[t]", "TTS Lang"},
			{"[s]", "TTS Voice"},
		}
	case viewViz:
		keys = [][]string{
			{"[v]", "BG Mode"},
			{"[c/i/p]", "Viz Patterns"},
		}
	}

	if !m.lockedView {
		keys = append(keys, []string{"[tab]", "Cycle View"})
	}
	keys = append(keys, []string{"[q]", "Quit"})

	var rows []string
	for _, k := range keys {
		rows = append(rows, fmt.Sprintf("%-10s %s",
			lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Highlight)).Render(k[0]),
			lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Text)).Render(k[1])))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, rows...)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m *Model) renderKaraoke() string {
	if len(m.currentLyrics) == 0 {
		return ""
	}

	// We'd need current elapsed from Model state
	elapsed := time.Duration(0)

	activeIdx := -1
	for i, line := range m.currentLyrics {
		if elapsed >= line.time {
			activeIdx = i
		} else {
			break
		}
	}

	var sb strings.Builder
	boxBottom := m.height/2 + 10
	if boxBottom >= m.height {
		boxBottom = m.height - 1
	}

	for i := 0; i < boxBottom+1; i++ {
		sb.WriteString("\n")
	}

	if activeIdx == -1 {
		return sb.String() + lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center).Italic(true).Foreground(lipgloss.Color(m.theme.Subtext)).Render("( Instrumental )")
	}

	for i := 0; i < 3; i++ {
		idx := activeIdx + i
		if idx < len(m.currentLyrics) {
			style := lipgloss.NewStyle().Width(m.width).Align(lipgloss.Center)
			if idx == activeIdx {
				style = style.Bold(true).Foreground(lipgloss.Color(m.theme.Accent))
			} else {
				style = style.Foreground(lipgloss.Color(m.theme.Subtext))
			}
			sb.WriteString(style.Render(m.currentLyrics[idx].text) + "\n")
		} else {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func (m *Model) renderImage() string { data, _ := os.ReadFile(m.cfg.ImagePath); return string(data) }

func (m *Model) overlayUI(bg, fg string) string {
	bgLines := strings.Split(strings.TrimSuffix(bg, "\n"), "\n")
	fgLines := strings.Split(fg, "\n")
	fgW, fgH := 0, len(fgLines)
	for _, l := range fgLines {
		if w := lipgloss.Width(l); w > fgW {
			fgW = w
		}
	}
	top, left := m.height/2-fgH/2, m.width/2-fgW/2
	if top < 0 {
		top = 0
	}
	if left < 0 {
		left = 0
	}
	var res strings.Builder
	for y := 0; y < m.height; y++ {
		bgL := strings.Repeat(" ", m.width)
		if y < len(bgLines) {
			bgL = bgLines[y]
		}
		if y >= top && y < top+fgH {
			fL := fgLines[y-top]
			pL := sliceANSI(bgL, 0, left)
			pR := sliceANSI(bgL, left+lipgloss.Width(fL), m.width)
			res.WriteString(pL)
			res.WriteString(fL)
			res.WriteString(pR)
		} else {
			res.WriteString(bgL)
		}
		if y < m.height-1 {
			res.WriteString("\n")
		}
	}
	return res.String()
}

func sliceANSI(s string, start, end int) string {
	var res strings.Builder
	visualPos := 0
	inEsc := false
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		r := runes[i]
		if r == '\033' {
			inEsc = true
			res.WriteRune(r)
			continue
		}
		if inEsc {
			res.WriteRune(r)
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		w := runewidth.RuneWidth(r)
		if visualPos >= start && visualPos+w <= end {
			res.WriteRune(r)
		} else if visualPos < start && visualPos+w > start {
			res.WriteRune(' ')
		}
		visualPos += w
	}
	for visualPos < end {
		res.WriteRune(' ')
		visualPos++
	}
	return res.String()
}
