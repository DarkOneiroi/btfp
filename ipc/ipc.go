// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package ipc

import (
	"btfp/player"
	"encoding/gob"
	"net"
	"time"
)

// SocketPath is the default path for the IPC unix socket
const SocketPath = "/tmp/btfp.sock"

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
	// CmdAddTrack adds a track to the playlist
	CmdAddTrack
	// CmdPlayTrack adds and immediately plays a track
	CmdPlayTrack
	// CmdQuit terminates the server and all clients
	CmdQuit
	// CmdTTSLanguage changes the TTS language ("en", "cs")
	CmdTTSLanguage
	// CmdTTSSpeaker changes the TTS speaker ID
	CmdTTSSpeaker
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

func init() {
	// Register types for gob encoding of interface{}
	gob.Register(time.Duration(0))
	gob.Register(0)
	gob.Register(player.Track{})
	gob.Register(TrackInfo{})
	gob.Register([]TrackInfo{})
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
