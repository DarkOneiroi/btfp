package ipc

import (
	"encoding/gob"
	"btfp/player"
	"net"
	"time"
)

const SocketPath = "/tmp/btfp.sock"

type CommandType int

const (
	CmdPlay CommandType = iota
	CmdPause
	CmdNext
	CmdPrev
	CmdSeek
	CmdVolume
	CmdMute
	CmdAddTrack
	CmdGetState
	CmdSetVizMode
	CmdQuit
)

type Command struct {
	Type    CommandType
	Payload interface{}
}

type TrackInfo struct {
	Title  string
	Artist string
	Path   string
	Length time.Duration
}

type PlayerState struct {
	CurrentTrack *TrackInfo
	IsPlaying    bool
	IsMuted      bool
	Volume       float64
	Elapsed      time.Duration
	Playlist     []TrackInfo
	PlayingIdx   int
	ShouldQuit   bool
	ActiveClients int
}

func init() {
	gob.Register(player.Track{})
	gob.Register(TrackInfo{})
	gob.Register([]TrackInfo{})
	gob.Register(time.Duration(0))
}

// Helpers for encoding/decoding
func SendCommand(conn net.Conn, cmd Command) error {
	enc := gob.NewEncoder(conn)
	return enc.Encode(cmd)
}

func ReceiveState(conn net.Conn) (PlayerState, error) {
	var state PlayerState
	dec := gob.NewDecoder(conn)
	err := dec.Decode(&state)
	return state, err
}
