// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package ipc

import (
	"btfp/internal/models"
	"encoding/gob"
	"fmt"
	"net"
	"time"
)

// Instance types
const (
	ModeMusic = "music"
)

// GetSocketPath returns a unique socket path for a given service and session
func GetSocketPath(service, session string) string {
	return fmt.Sprintf("/tmp/btfp-%s-%s.sock", session, service)
}

// CommandType defines the available IPC commands
type CommandType int

const (
	// Core Commands
	CmdPlay CommandType = iota
	CmdPause
	CmdNext
	CmdPrev
	CmdStop
	CmdSeek
	CmdVolume
	CmdMute
	CmdPlayTrack
	CmdQuit

	// Library Commands
	CmdLibScan
	CmdLibGetMetadata

	// Fetcher Commands
	CmdFetchLyrics
	CmdFetchArt

	// Visualization Commands
	CmdVizGenerate

	// Playlist Commands
	CmdPlaylistAdd
	CmdPlaylistRemove
	CmdPlaylistGet
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
	Album  string
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

func init() {
	gob.Register(time.Duration(0))
	gob.Register(0)
	gob.Register(models.Track{})
	gob.Register(TrackInfo{})
	gob.Register([]TrackInfo{})
	gob.Register(MsgLibEntries{})
	gob.Register(MsgFetchResult{})
	gob.Register([]LibEntry{})
	gob.Register(map[string]interface{}{})
	gob.Register([]string{})
	gob.Register([]float64{})
}

func SendCommand(conn net.Conn, cmd Command) error {
	enc := gob.NewEncoder(conn)
	return enc.Encode(cmd)
}

func ReceiveState(conn net.Conn) (PlayerState, error) {
	dec := gob.NewDecoder(conn)
	var state PlayerState
	err := dec.Decode(&state)
	return state, err
}
