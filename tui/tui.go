package tui

import (
	"encoding/gob"
	"encoding/json"
	"fmt"
	"btfp/config"
	"btfp/ipc"
	"btfp/player"
	"btfp/utils"
	"btfp/visualizations"
	"io"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/bogem/id3v2/v2"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

type viewState int

const (
	viewLibrary viewState = iota
	viewPlaylist
	viewPlayer
	viewViz
)

type backgroundMode int

const (
	bgVisualization backgroundMode = iota
	bgEQBars
	bgKaraoke
	bgEmpty
	bgImage
)

type lrcLine struct {
	time time.Duration
	text string
}

type item struct {
	title, desc, path string
	isDir             bool
	selected          bool
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.desc }
func (i item) FilterValue() string { return i.title }

type Model struct {
	width, height  int
	view           viewState
	lockedView     bool
	bgMode         backgroundMode
	libList        list.Model
	playList       list.Model
	playlist       []player.Track
	playingIdx     int
	currentDir     string
	selectedPaths  map[string]bool
	vizFrame       *visualizations.Frame
	err            error
	preset         int
	colorMode      int
	palette        int
	showLegend     bool
	lastResize     time.Time
	player         *player.MusicPlayer
	cfg            config.Config
	theme          config.Theme
	currentLyrics  []lrcLine
	startTime      time.Time
	metadataCache  map[string]string
	artCache       map[string]string
	playlistCounts map[string]int
	conn           net.Conn
	enc            *gob.Encoder
	dec            *gob.Decoder
}

type vizTickMsg time.Time
type errMsg error
type artDownloadedMsg string
type lyricsDownloadedMsg struct {
	path   string
	lyrics []lrcLine
}
type tickMsg time.Time
type libraryMsg []list.Item
type serverStateMsg ipc.PlayerState

func NewModel(initialView string) *Model {
	cfg, theme := config.LoadConfig()
	home, _ := os.UserHomeDir()
	musicDir := filepath.Join(home, "Music")

	startView := viewState(cfg.DefaultView)
	if startView > viewViz {
		startView = viewLibrary
	}
	locked := false
	if initialView != "" {
		locked = true
		switch initialView {
		case "library":
			startView = viewLibrary
		case "playlist":
			startView = viewPlaylist
		case "player":
			startView = viewPlayer
		case "viz":
			startView = viewViz
		}
	}

	m := &Model{
		view:           startView,
		lockedView:     locked,
		bgMode:         backgroundMode(cfg.BGMode),
		currentDir:     musicDir,
		selectedPaths:  make(map[string]bool),
		preset:         cfg.Pattern,
		colorMode:      cfg.ColorMode,
		palette:        cfg.Palette,
		showLegend:     cfg.ShowLegend,
		player:         player.NewMusicPlayer(),
		playingIdx:     -1,
		cfg:            cfg,
		theme:          theme,
		startTime:      time.Now(),
		metadataCache:  make(map[string]string),
		artCache:       make(map[string]string),
		playlistCounts: make(map[string]int),
	}

	if m.lockedView {
		if m.view == viewViz {
			m.bgMode = bgVisualization
		} else if m.view == viewPlayer {
			m.bgMode = bgEmpty
		}
	}

	m.libList = list.New([]list.Item{}, itemDelegate{m}, 0, 0)
	m.libList.Title = "MUSIC LIBRARY"
	m.libList.SetShowStatusBar(false)
	m.libList.Styles.Title = lipgloss.NewStyle().Background(lipgloss.Color(m.theme.Title)).Foreground(lipgloss.Color(m.theme.Text)).Padding(0, 1)

	m.playList = list.New([]list.Item{}, itemDelegate{m}, 0, 0)
	m.playList.Title = "PLAYLIST"
	m.playList.SetShowStatusBar(false)
	m.playList.Styles.Title = lipgloss.NewStyle().Background(lipgloss.Color(m.theme.Highlight)).Foreground(lipgloss.Color(m.theme.Text)).Padding(0, 1)

	return m
}

func (m *Model) SetConn(conn net.Conn) {
	m.conn = conn
	m.enc = gob.NewEncoder(conn)
	m.dec = gob.NewDecoder(conn)
}

func (m *Model) syncPlaylist() {
	m.updatePlaylistCounts()
	m.updatePlaylistItems()
}

func (m *Model) updatePlaylistCounts() {
	m.playlistCounts = make(map[string]int)
	home, _ := os.UserHomeDir()
	musicBase := filepath.Join(home, "Music")
	for _, t := range m.playlist {
		m.playlistCounts[t.Path]++
		dir := filepath.Dir(t.Path)
		for {
			m.playlistCounts[dir]++
			if dir == "/" || dir == "." || dir == musicBase {
				break
			}
			dir = filepath.Dir(dir)
		}
	}
}

func (m *Model) updatePlaylistItems() {
	var items []list.Item
	for i, t := range m.playlist {
		pref := "  "
		if i == m.playingIdx {
			pref = "▶ "
		}
		items = append(items, item{title: pref + t.Title, desc: t.Path, path: t.Path})
	}
	m.playList.SetItems(items)
}

type itemDelegate struct {
	m *Model
}

func (d itemDelegate) Height() int                               { return 1 }
func (d itemDelegate) Spacing() int                              { return 0 }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d itemDelegate) Render(w io.Writer, l list.Model, index int, listItem list.Item) {
	i, ok := listItem.(item)
	if !ok {
		return
	}

	style := lipgloss.NewStyle().PaddingLeft(2)
	if index == l.Index() {
		style = style.Foreground(lipgloss.Color(d.m.theme.Highlight)).Bold(true)
	} else {
		style = style.Foreground(lipgloss.Color(d.m.theme.Text))
	}

	prefix := ""
	if i.selected {
		prefix = "✔ "
	} else if i.isDir {
		prefix = "📁 "
	} else {
		prefix = "🎵 "
	}

	title := i.title
	if count := d.m.playlistCounts[i.path]; count > 0 {
		title += fmt.Sprintf(" (%d)", count)
	}

	fmt.Fprint(w, style.Render(prefix+title))
}

func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.loadDirectory(m.currentDir), tick(), vizTick()}
	if m.conn != nil {
		cmds = append(cmds, m.listenToServer())
	}
	return tea.Batch(cmds...)
}

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

func (m *Model) sendCommand(cmd ipc.Command) {
	if m.enc != nil {
		_ = m.enc.Encode(cmd)
	}
}

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
				if !entry.IsDir() {
					desc = fmt.Sprintf("%.1f MB", float64(info.Size())/1024/1024)
				}
				items = append(items, item{title: entry.Name(), desc: desc, path: fullPath, isDir: entry.IsDir(), selected: m.selectedPaths[fullPath]})
			}
		}
		return libraryMsg(items)
	}
}

func (m *Model) downloadArtCmd(dir string) tea.Cmd {
	return func() tea.Msg {
		resp, err := http.Get("https://picsum.photos/500")
		if err == nil {
			defer resp.Body.Close()
			artPath := filepath.Join(dir, "cover.jpg")
			out, _ := os.Create(artPath)
			io.Copy(out, resp.Body)
			out.Close()
			return artDownloadedMsg(artPath)
		}
		return nil
	}
}

func (m *Model) downloadLyricsCmd(artist, title, lrcPath string) tea.Cmd {
	return func() tea.Msg {
		var duration int
		if m.player != nil && m.player.CurrentTrack != nil {
			duration = int(m.player.CurrentTrack.Length.Seconds())
		}

		cleanArtist, cleanTitle := cleanString(artist), cleanString(title)

		apiURL := fmt.Sprintf("https://lrclib.net/api/get?artist_name=%s&track_name=%s", url.QueryEscape(cleanArtist), url.QueryEscape(cleanTitle))
		if duration > 0 {
			apiURL += fmt.Sprintf("&duration=%d", duration)
		}

		var synced string
		resp, err := http.Get(apiURL)
		if err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				var result struct {
					SyncedLyrics string `json:"syncedLyrics"`
				}
				if err := json.NewDecoder(resp.Body).Decode(&result); err == nil {
					if regexp.MustCompile(`\[\d+:\d+`).MatchString(result.SyncedLyrics) {
						synced = result.SyncedLyrics
					}
				}
			}
		}

		if synced == "" {
			searchURL := fmt.Sprintf("https://lrclib.net/api/search?q=%s", url.QueryEscape(cleanArtist+" "+cleanTitle))
			resp, err := http.Get(searchURL)
			if err == nil {
				defer resp.Body.Close()
				if resp.StatusCode == http.StatusOK {
					var results []struct {
						SyncedLyrics string  `json:"syncedLyrics"`
						Duration     float64 `json:"duration"`
					}
					if err := json.NewDecoder(resp.Body).Decode(&results); err == nil {
						for _, r := range results {
							if r.SyncedLyrics != "" && regexp.MustCompile(`\[\d+:\d+`).MatchString(r.SyncedLyrics) {
								if duration > 0 && math.Abs(r.Duration-float64(duration)) < 10 {
									synced = r.SyncedLyrics
									break
								}
								if synced == "" {
									synced = r.SyncedLyrics
								}
							}
						}
					}
				}
			}
		}

		if synced != "" && regexp.MustCompile(`\[\d+:\d+`).MatchString(synced) {
			os.WriteFile(lrcPath, []byte(synced), 0644)
			lyrics, _ := m.parseLyrics(synced)
			return lyricsDownloadedMsg{path: lrcPath, lyrics: lyrics}
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
			tag.Save()
		}
		curArtist, curTitle = tag.Artist(), tag.Title()
		tag.Close()
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

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case serverStateMsg:
		if msg.ShouldQuit {
			return m, tea.Quit
		}
		m.playingIdx = msg.PlayingIdx
		m.playlist = make([]player.Track, len(msg.Playlist))
		for i, t := range msg.Playlist {
			m.playlist[i] = player.Track{Title: t.Title, Artist: t.Artist, Path: t.Path, Length: t.Length}
		}
		if msg.CurrentTrack != nil {
			if m.player.CurrentTrack == nil || m.player.CurrentTrack.Path != msg.CurrentTrack.Path {
				m.player.CurrentTrack = &player.Track{
					Title:  msg.CurrentTrack.Title,
					Artist: msg.CurrentTrack.Artist,
					Path:   msg.CurrentTrack.Path,
					Length: msg.CurrentTrack.Length,
				}
				cmds = append(cmds, m.syncMetadataAndArt(m.player.CurrentTrack.Path))
			}
		} else {
			m.player.CurrentTrack = nil
		}
		m.player.IsPlaying = msg.IsPlaying
		m.player.IsMuted = msg.IsMuted
		m.player.Volume = msg.Volume
		m.player.Elapsed = msg.Elapsed
		m.syncPlaylist()
		cmds = append(cmds, m.listenToServer())

	case libraryMsg:
		m.libList.SetItems(msg)
	case artDownloadedMsg:
		delete(m.artCache, filepath.Dir(string(msg)))
		if m.view == viewLibrary {
			cmds = append(cmds, m.loadDirectory(m.currentDir))
		}
	case lyricsDownloadedMsg:
		if m.player.CurrentTrack != nil {
			lrcPath := strings.TrimSuffix(m.player.CurrentTrack.Path, filepath.Ext(m.player.CurrentTrack.Path)) + ".lrc"
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
			cmds = append(cmds, m.loadDirectory(m.currentDir))
		}
	case vizTickMsg:
		if (m.view == viewPlayer || m.view == viewViz) && (m.bgMode == bgVisualization || m.bgMode == bgEQBars) {
			if m.vizFrame == nil {
				m.vizFrame = visualizations.NewFrame(m.width, m.height, visualizations.PatternType(m.preset))
				m.vizFrame.ColorMode = visualizations.ColorMode(m.colorMode)
				m.vizFrame.PaletteType = visualizations.PaletteType(m.palette)
			}
			m.vizFrame.Time = time.Since(m.startTime).Seconds()
			levels := make([]float64, 32)
			if m.player.IsPlaying && !m.player.IsMuted {
				for i := range levels {
					levels[i] = (math.Sin(m.vizFrame.Time*float64(i+1)*0.5)+1.0)*0.3 + rand.Float64()*0.2
					levels[i] *= m.player.Volume
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
		cmds = append(cmds, vizTick())
	case tea.KeyMsg:
		if m.libList.FilterState() == list.Filtering || m.playList.FilterState() == list.Filtering {
			break
		}
		key := msg.String()
		switch key {
		case "q", "ctrl+c":
			m.sendCommand(ipc.Command{Type: ipc.CmdQuit})
			return m, tea.Quit
		case "backspace":
			if m.view == viewLibrary {
				home, _ := os.UserHomeDir()
				if m.currentDir != filepath.Join(home, "Music") {
					m.currentDir = filepath.Dir(m.currentDir)
					cmds = append(cmds, m.loadDirectory(m.currentDir))
				}
			}
		case "tab":
			if !m.lockedView {
				m.view = (m.view + 1) % 3
				if m.view == viewPlaylist {
					m.updatePlaylistItems()
				}
			}
		case " ":
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
		case "a":
			if m.view == viewLibrary {
				for p, s := range m.selectedPaths {
					if s {
						m.addPathToPlaylist(p)
					}
				}
				m.selectedPaths = make(map[string]bool)
				m.syncPlaylist()
				cmds = append(cmds, m.loadDirectory(m.currentDir))
			}
		case "enter":
			if m.view == viewLibrary {
				if sel, ok := m.libList.SelectedItem().(item); ok {
					if sel.isDir {
						m.currentDir = sel.path
						cmds = append(cmds, m.loadDirectory(m.currentDir))
					} else {
						track := player.Track{Title: sel.title, Path: sel.path}
						m.sendCommand(ipc.Command{Type: ipc.CmdAddTrack, Payload: track})
						if m.conn == nil {
							m.playlist = append(m.playlist, track)
							m.playingIdx = len(m.playlist) - 1
							m.player.PlayTrack(&m.playlist[m.playingIdx])
							m.syncPlaylist()
							m.view = viewPlayer
							cmds = append(cmds, m.syncMetadataAndArt(sel.path))
						}
					}
				}
			} else if m.view == viewPlaylist {
				if m.playList.Index() < len(m.playlist) {
					idx := m.playList.Index()
					m.sendCommand(ipc.Command{Type: ipc.CmdPlay, Payload: idx})
					if m.conn == nil {
						m.playingIdx = idx
						m.player.PlayTrack(&m.playlist[m.playingIdx])
						m.syncPlaylist()
						m.view = viewPlayer
						cmds = append(cmds, m.syncMetadataAndArt(m.playlist[m.playingIdx].Path))
					}
				}
			}
		case "c", "i", "p":
			if (m.view == viewPlayer || m.view == viewViz) && m.vizFrame != nil {
				if key == "c" {
					m.preset = (m.preset + 1) % 7
					m.vizFrame.PatternType = visualizations.PatternType(m.preset)
				}
				if key == "i" {
					m.colorMode = (m.colorMode + 1) % 7
					m.vizFrame.ColorMode = visualizations.ColorMode(m.colorMode)
				}
				if key == "p" {
					m.palette = (m.palette + 1) % 16
					m.vizFrame.PaletteType = visualizations.PaletteType(m.palette)
				}
			}
		case "v":
			if m.view == viewPlayer || m.view == viewViz {
				m.cycleBGMode()
			}
		case "h", "?":
			m.showLegend = !m.showLegend
		case "+", "=":
			vol := m.player.Volume + 0.1
			m.sendCommand(ipc.Command{Type: ipc.CmdVolume, Payload: vol})
			if m.conn == nil {
				m.player.SetVolume(vol)
			}
		case "-", "_":
			vol := m.player.Volume - 0.1
			m.sendCommand(ipc.Command{Type: ipc.CmdVolume, Payload: vol})
			if m.conn == nil {
				m.player.SetVolume(vol)
			}
		case "m":
			m.sendCommand(ipc.Command{Type: ipc.CmdMute})
			if m.conn == nil {
				m.player.ToggleMute()
			}
		case "l", "right":
			m.sendCommand(ipc.Command{Type: ipc.CmdSeek, Payload: 5 * time.Second})
			if m.conn == nil {
				m.player.Seek(5 * time.Second)
			}
		case "left":
			m.sendCommand(ipc.Command{Type: ipc.CmdSeek, Payload: -5 * time.Second})
			if m.conn == nil {
				m.player.Seek(-5 * time.Second)
			}
		case "n":
			m.sendCommand(ipc.Command{Type: ipc.CmdNext})
			if m.conn == nil {
				m.skipTrack(1)
				cmds = append(cmds, m.syncMetadataAndArt(m.playlist[m.playingIdx].Path))
			}
		case "b":
			m.sendCommand(ipc.Command{Type: ipc.CmdPrev})
			if m.conn == nil {
				m.skipTrack(-1)
				cmds = append(cmds, m.syncMetadataAndArt(m.playlist[m.playingIdx].Path))
			}
		}
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.libList.SetSize(m.width/3-2, m.height-2)
		m.playList.SetSize(m.width/2-2, m.height-2)
		if m.vizFrame != nil {
			m.vizFrame.Width, m.vizFrame.Height = m.width, m.height
			m.vizFrame.Data = make([]float64, m.width*m.height)
		}
	case tickMsg:
		if m.conn == nil {
			m.player.Update()
			if m.player.IsDone {
				m.skipTrack(1)
				cmds = append(cmds, m.syncMetadataAndArt(m.playlist[m.playingIdx].Path))
			}
		}
		cmds = append(cmds, tick())
	case errMsg:
		if m.conn != nil {
			return m, tea.Quit
		}
		m.err = msg
	}
	if m.view == viewLibrary {
		m.libList, _ = m.libList.Update(msg)
	}
	if m.view == viewPlaylist {
		m.playList, _ = m.playList.Update(msg)
	}
	return m, tea.Batch(cmds...)
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

func (m *Model) skipTrack(dir int) {
	if len(m.playlist) == 0 {
		return
	}
	m.playingIdx = (m.playingIdx + dir + len(m.playlist)) % len(m.playlist)
	m.player.PlayTrack(&m.playlist[m.playingIdx])
	m.currentLyrics = []lrcLine{}
	m.syncPlaylist()
}

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
		defer tag.Close()
		if tag.Artist() != "" {
			track.Artist = tag.Artist()
		}
		if tag.Title() != "" {
			track.Title = tag.Title()
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

func (m *Model) addPathToPlaylist(path string) {
	info, _ := os.Stat(path)
	if info != nil && info.IsDir() {
		filepath.Walk(path, func(p string, f os.FileInfo, err error) error {
			if err == nil && !f.IsDir() && utils.IsSupportedAudioFile(f.Name()) {
				track := m.getTrackMetadata(p)
				m.sendCommand(ipc.Command{Type: ipc.CmdAddTrack, Payload: track})
				if m.conn == nil {
					m.playlist = append(m.playlist, track)
				}
			}
			return nil
		})
	} else if info != nil {
		track := m.getTrackMetadata(path)
		m.sendCommand(ipc.Command{Type: ipc.CmdAddTrack, Payload: track})
		if m.conn == nil {
			m.playlist = append(m.playlist, track)
		}
	}
}

func (m *Model) View() string {
	if m.err != nil {
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, fmt.Sprintf("Error: %v", m.err))
	}
	var res string
	switch m.view {
	case viewLibrary:
		res = m.renderLibraryView()
	case viewPlaylist:
		res = m.renderPlaylistView()
	case viewPlayer:
		fg, bg := m.renderPlayerView(), m.getBackground()
		if m.lockedView {
			bg = ""
		}
		res = m.overlayUI(bg, fg)
	case viewViz:
		res = m.getBackground()
	}

	if m.showLegend {
		res = m.overlayUI(res, m.renderLegend())
	}
	return res
}

func (m *Model) getBackground() string {
	switch m.bgMode {
	case bgVisualization, bgEQBars:
		if m.vizFrame == nil {
			return ""
		}
		return m.vizFrame.Render(false)
	case bgKaraoke:
		return m.renderKaraoke()
	case bgImage:
		return m.renderImage()
	default:
		return ""
	}
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

	activeIdx := -1
	for i, line := range m.currentLyrics {
		if m.player.Elapsed >= line.time {
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

func (m *Model) renderListView(l list.Model) string {
	view := l.View()
	lines := strings.Split(view, "\n")
	h := l.Height()
	if h <= 0 {
		return view
	}

	totalItems := len(l.Items())
	if totalItems <= h {
		return view
	}

	cursor := l.Index()
	scrollPos := int(float64(cursor) / float64(totalItems) * float64(h))

	var sb strings.Builder
	for i, line := range lines {
		if i >= h {
			break
		}
		indicator := " "
		if i == scrollPos {
			indicator = "┃"
		} else if i > 0 && i < h-1 {
			indicator = "│"
		}
		sb.WriteString(line + indicator + "\n")
	}
	return sb.String()
}

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
	if err != nil {
		return ""
	}
	defer tag.Close()

	accent := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(m.theme.Accent))
	lyricsStatus := "Missing"
	if _, err := os.Stat(strings.TrimSuffix(path, filepath.Ext(path)) + ".lrc"); err == nil {
		lyricsStatus = "Available"
	}

	meta := fmt.Sprintf("\n%s\n Artist: %s\n Album:  %s\n Title:  %s\n Genre:  %s\n Year:   %s\n Lyrics: %s",
		accent.Render("METADATA"), tag.Artist(), tag.Album(), tag.Title(), tag.Genre(), tag.Year(), lyricsStatus)

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

func (m *Model) renderPlayerView() string {
	var trackTitle string
	var trackLength time.Duration
	if m.player.CurrentTrack != nil {
		trackTitle = m.player.CurrentTrack.Title
		trackLength = m.player.CurrentTrack.Length
	}

	if trackTitle == "" {
		return lipgloss.NewStyle().Width(40).Align(lipgloss.Center).Render("No track playing.\nPress [tab] for Library.")
	}
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Title)).Bold(true)
	barW := 40
	pct := 0.0
	if trackLength > 0 {
		pct = float64(m.player.Elapsed) / float64(trackLength)
	}
	fill := int(float64(barW) * pct)
	if fill > barW {
		fill = barW
	}
	bar := lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Highlight)).Render(strings.Repeat("━", fill)) + lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Subtext)).Render(strings.Repeat("━", barW-fill))
	status := "PLAYING"
	if !m.player.IsPlaying {
		status = "PAUSED"
	}
	volStr := fmt.Sprintf("VOL: %d%%", int(m.player.Volume*100))
	if m.player.IsMuted {
		volStr = "VOL: MUTE"
	}

	ui := lipgloss.JoinVertical(lipgloss.Center, lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Accent)).Bold(true).Render("BTFP PLAYER"), "", titleStyle.Render(trackTitle), fmt.Sprintf("%s / %s", formatDuration(m.player.Elapsed), formatDuration(trackLength)), volStr, "", bar, "", lipgloss.NewStyle().Foreground(lipgloss.Color(m.theme.Highlight)).Render(status), "\n[h] Toggle Help")
	return lipgloss.NewStyle().Width(80).Height(20).Align(lipgloss.Center, lipgloss.Center).Render(ui)
}

func (m *Model) overlayUI(bg string, fg string) string {
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
		if r == '\x1b' {
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
	s = regexp.MustCompile(`[ \t]+`).ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

func tick() tea.Cmd { return tea.Tick(time.Second/10, func(t time.Time) tea.Msg { return tickMsg(t) }) }
func vizTick() tea.Cmd {
	return tea.Tick(time.Second/30, func(t time.Time) tea.Msg { return vizTickMsg(t) })
}
