// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package ipc

import (
	"btfp/services/core/player"
	"encoding/gob"
	"net"
	"time"
)

// Socket paths for different services
const (
	CoreSocketPath     = "/tmp/btfp-core.sock"
	LibrarySocketPath  = "/tmp/btfp-library.sock"
	FetcherSocketPath  = "/tmp/btfp-fetcher.sock"
	TTSSocketPath      = "/tmp/btfp-tts.sock"
	VizSocketPath      = "/tmp/btfp-viz.sock"
	PlaylistSocketPath = "/tmp/btfp-playlist.sock"
)

// SocketPath is maintained for backward compatibility (points to Core)
const SocketPath = CoreSocketPath

// CommandType defines the available IPC commands
type CommandType int

const (
	// CmdPlay starts or resumes playback
	CmdPlay CommandType = iota
	// CmdPause pauses playback
	CmdPause
	// CmdNext skips to the next track
	CmdNext
	// CmdPrev skips to the previous track
	CmdPrev
	// CmdStop stops playback
	CmdStop
	// CmdSeek seeks within the current track
	CmdSeek
	// CmdVolume sets the playback volume
	CmdVolume
	// CmdMute toggles mute state
	CmdMute
	// CmdPlayTrack adds and immediately plays a track
	CmdPlayTrack
	// CmdQuit terminates the server and all clients
	CmdQuit

	// CmdLibScan requests a directory scan from the Library service
	CmdLibScan
	// CmdLibGetMetadata requests track metadata from the Library service
	CmdLibGetMetadata

	// CmdFetchLyrics requests lyrics from the Fetcher service
	CmdFetchLyrics
	// CmdFetchArt requests album art from the Fetcher service
	CmdFetchArt

	// CmdTTSGenerate requests audio generation from the TTS service
	CmdTTSGenerate
	// CmdTTSLanguage changes the TTS language
	CmdTTSLanguage
	// CmdTTSSpeaker changes the TTS speaker ID
	CmdTTSSpeaker

	// CmdVizGenerate requests a visualization frame from the Viz service
	CmdVizGenerate

	// CmdPlaylistAdd adds a track to the playlist managed by the Playlist service
	CmdPlaylistAdd
	// CmdPlaylistRemove removes a track from the playlist
	CmdPlaylistRemove
	// CmdPlaylistGet retrieves the current playlist
	CmdPlaylistGet
	// CmdPlaylistClear clears the entire playlist
	CmdPlaylistClear
)

// Command represents a message sent from client to server
type Command struct {
	Type    CommandType
	Payload interface{}
}

// TrackInfo represents metadata for a single track in the IPC state
type TrackInfo struct {
	Title  string
	Artist string
	Path   string
	Length time.Duration
}

// PlayerState represents the complete state broadcast by the server
type PlayerState struct {
	CurrentTrack  *TrackInfo
	IsPlaying     bool
	IsMuted       bool
	Volume        float64
	Elapsed       time.Duration
	Playlist      []TrackInfo
	PlayingIdx    int
	ShouldQuit    bool
	ActiveClients int

	// TTS State
	TTSLanguage string
	TTSSpeaker  int
}

// MsgLibEntries is the response from Library service
type MsgLibEntries struct {
	Path    string
	Entries []LibEntry
}

// LibEntry represents a single file or directory in the library
type LibEntry struct {
	Title, Desc, Path string
	IsDir             bool
}

// MsgFetchResult is the response from Fetcher service
type MsgFetchResult struct {
	Type    string // "lyrics" or "art"
	Path    string
	Content string // lyrics text or art path
}

// MsgTTSResult is the response from TTS service
type MsgTTSResult struct {
	Path    string
	Samples []float32
	Rate    int
}

func init() {
	// Register types for gob encoding of interface{}
	gob.Register(time.Duration(0))
	gob.Register(0)
	gob.Register(player.Track{})
	gob.Register(TrackInfo{})
	gob.Register([]TrackInfo{})
	gob.Register(MsgLibEntries{})
	gob.Register(MsgFetchResult{})
	gob.Register(MsgTTSResult{})
	gob.Register([]LibEntry{})
}

// SendCommand sends a command over the given connection
func SendCommand(conn net.Conn, cmd Command) error {
	enc := gob.NewEncoder(conn)
	return enc.Encode(cmd)
}

// ReceiveState waits for and decodes a PlayerState from the connection
func ReceiveState(conn net.Conn) (PlayerState, error) {
	dec := gob.NewDecoder(conn)
	var state PlayerState
	err := dec.Decode(&state)
	return state, err
}
