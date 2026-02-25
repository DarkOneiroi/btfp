package server

import (
	"encoding/gob"
	"fmt"
	"btfp/ipc"
	"btfp/player"
	"net"
	"os"
	"sync"
	"time"
)

type Client struct {
	conn net.Conn
	enc  *gob.Encoder
}

type Server struct {
	player     *player.MusicPlayer
	playlist   []player.Track
	playingIdx int
	clients    map[net.Conn]*Client
	mu         sync.Mutex
	shouldQuit bool
	listener   net.Listener
}

func Start() {
	os.Remove(ipc.SocketPath)
	listener, err := net.Listen("unix", ipc.SocketPath)
	if err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		return
	}
	defer listener.Close()

	s := &Server{
		player:     player.NewMusicPlayer(),
		clients:    make(map[net.Conn]*Client),
		playingIdx: -1,
		listener:   listener,
	}

	// Server Update Loop
	go func() {
		for range time.Tick(time.Second / 10) {
			s.player.Update()
			s.mu.Lock()
			if s.player.IsDone && len(s.playlist) > 0 {
				s.playingIdx = (s.playingIdx + 1) % len(s.playlist)
				s.player.PlayTrack(&s.playlist[s.playingIdx])
			}
			shouldQuit := s.shouldQuit
			s.mu.Unlock()
			s.broadcastState()
			if shouldQuit {
				s.listener.Close()
				os.Exit(0)
			}
		}
	}()

	fmt.Println("BTFP (BehindTheForestPlayer) Server started on", ipc.SocketPath)

	for {
		conn, err := listener.Accept()
		if err != nil {
			if s.shouldQuit {
				return
			}
			continue
		}
		
		s.mu.Lock()
		client := &Client{
			conn: conn,
			enc:  gob.NewEncoder(conn),
		}
		s.clients[conn] = client
		s.mu.Unlock()
		
		go s.handleClient(client)
	}
}

func (s *Server) handleClient(c *Client) {
	defer func() {
		s.mu.Lock()
		delete(s.clients, c.conn)
		s.mu.Unlock()
		c.conn.Close()
	}()

	dec := gob.NewDecoder(c.conn)
	for {
		var cmd ipc.Command
		if err := dec.Decode(&cmd); err != nil {
			return
		}
		s.mu.Lock()
		s.processCommand(cmd)
		s.mu.Unlock()
		s.broadcastState()
	}
}

func (s *Server) processCommand(cmd ipc.Command) {
	switch cmd.Type {
	case ipc.CmdPlay:
		if p, ok := cmd.Payload.(int); ok && p >= 0 && p < len(s.playlist) {
			s.playingIdx = p
			s.player.PlayTrack(&s.playlist[s.playingIdx])
		} else {
			s.player.TogglePause()
		}
	case ipc.CmdPause:
		s.player.TogglePause()
	case ipc.CmdNext:
		if len(s.playlist) > 0 {
			s.playingIdx = (s.playingIdx + 1) % len(s.playlist)
			s.player.PlayTrack(&s.playlist[s.playingIdx])
		}
	case ipc.CmdPrev:
		if len(s.playlist) > 0 {
			s.playingIdx = (s.playingIdx - 1 + len(s.playlist)) % len(s.playlist)
			s.player.PlayTrack(&s.playlist[s.playingIdx])
		}
	case ipc.CmdAddTrack:
		if t, ok := cmd.Payload.(player.Track); ok {
			s.playlist = append(s.playlist, t)
			if s.playingIdx == -1 {
				s.playingIdx = 0
				s.player.PlayTrack(&s.playlist[0])
			}
		}
	case ipc.CmdVolume:
		if v, ok := cmd.Payload.(float64); ok {
			s.player.SetVolume(v)
		}
	case ipc.CmdSeek:
		if d, ok := cmd.Payload.(time.Duration); ok {
			s.player.Seek(d)
		}
	case ipc.CmdMute:
		s.player.ToggleMute()
	case ipc.CmdQuit:
		s.shouldQuit = true
	}
}

func (s *Server) broadcastState() {
	s.mu.Lock()
	defer s.mu.Unlock()

	var current *ipc.TrackInfo
	if s.player.CurrentTrack != nil {
		current = &ipc.TrackInfo{
			Title:  s.player.CurrentTrack.Title,
			Artist: s.player.CurrentTrack.Artist,
			Path:   s.player.CurrentTrack.Path,
			Length: s.player.CurrentTrack.Length,
		}
	}

	playlist := make([]ipc.TrackInfo, len(s.playlist))
	for i, t := range s.playlist {
		playlist[i] = ipc.TrackInfo{Title: t.Title, Artist: t.Artist, Path: t.Path, Length: t.Length}
	}

	state := ipc.PlayerState{
		CurrentTrack: current,
		IsPlaying:    s.player.IsPlaying,
		IsMuted:      s.player.IsMuted,
		Volume:       s.player.Volume,
		Elapsed:      s.player.Elapsed,
		Playlist:     playlist,
		PlayingIdx:   s.playingIdx,
		ShouldQuit:   s.shouldQuit,
		ActiveClients: len(s.clients),
	}

	for _, c := range s.clients {
		c.conn.SetWriteDeadline(time.Now().Add(20 * time.Millisecond))
		if err := c.enc.Encode(state); err != nil {
			// Connection error, will be handled by handleClient return
		}
	}
}
