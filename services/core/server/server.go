// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package server

import (
	"btfp/internal/config"
	"btfp/internal/ipc-shared"
	"btfp/services/core/player"
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

// Client represents a connected IPC client
type Client struct {
	conn net.Conn
	enc  *gob.Encoder
}

// Server represents the main IPC server managing the player and clients
type Server struct {
	player     player.Player
	playlist   []player.Track
	playingIdx int
	clients    map[net.Conn]*Client
	mu         sync.Mutex
	shouldQuit bool
	listener   net.Listener
	handlers   map[ipc.CommandType]commandHandler
}

type commandHandler func(*Server, interface{})

// Start initializes and runs the IPC server
func Start() {
	_ = os.Remove(ipc.SocketPath)

	cfg, _ := config.LoadConfig()

	// Register types for interface{} encoding
	gob.Register(time.Duration(0))
	gob.Register(0)
	gob.Register(player.Track{})
	gob.Register([]player.Track{})

	listener, err := net.Listen("unix", ipc.SocketPath)
	if err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
		return
	}
	defer func() { _ = listener.Close() }()

	s := &Server{
		player:     player.NewMusicPlayer(cfg),
		clients:    make(map[net.Conn]*Client),
		playingIdx: -1,
		listener:   listener,
	}
	s.registerHandlers()

	// Server Update Loop
	go func() {
		for range time.Tick(time.Second / 10) {
			s.player.Update()
			s.mu.Lock()
			status := s.player.GetStatus()
			if status.IsDone && len(s.playlist) > 0 {
				s.playingIdx = (s.playingIdx + 1) % len(s.playlist)
				_ = s.player.PlayTrack(&s.playlist[s.playingIdx])
			}
			shouldQuit := s.shouldQuit
			s.mu.Unlock()
			s.broadcastState()
			if shouldQuit {
				time.Sleep(100 * time.Millisecond) // Give clients time to receive quit signal
				_ = s.listener.Close()
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

func (s *Server) registerHandlers() {
	s.handlers = map[ipc.CommandType]commandHandler{
		ipc.CmdPlay: func(srv *Server, p interface{}) {
			if idx, ok := p.(int); ok && idx >= 0 && idx < len(srv.playlist) {
				srv.playingIdx = idx
				_ = srv.player.PlayTrack(&srv.playlist[srv.playingIdx])
			} else {
				srv.player.TogglePause()
			}
		},
		ipc.CmdPause: func(srv *Server, _ interface{}) {
			srv.player.TogglePause()
		},
		ipc.CmdNext: func(srv *Server, _ interface{}) {
			if len(srv.playlist) > 0 {
				srv.playingIdx = (srv.playingIdx + 1) % len(srv.playlist)
				_ = srv.player.PlayTrack(&srv.playlist[srv.playingIdx])
			}
		},
		ipc.CmdPrev: func(srv *Server, _ interface{}) {
			if len(srv.playlist) > 0 {
				srv.playingIdx = (srv.playingIdx - 1 + len(srv.playlist)) % len(srv.playlist)
				_ = srv.player.PlayTrack(&srv.playlist[srv.playingIdx])
			}
		},
		ipc.CmdPlaylistAdd: func(srv *Server, p interface{}) {
			if t, ok := p.(player.Track); ok {
				srv.playlist = append(srv.playlist, t)
				if srv.playingIdx == -1 {
					srv.playingIdx = 0
					_ = srv.player.PlayTrack(&srv.playlist[0])
				}
			}
		},
		ipc.CmdPlayTrack: func(srv *Server, p interface{}) {
			if t, ok := p.(player.Track); ok {
				srv.playlist = append(srv.playlist, t)
				srv.playingIdx = len(srv.playlist) - 1
				_ = srv.player.PlayTrack(&srv.playlist[srv.playingIdx])
			}
		},
		ipc.CmdVolume: func(srv *Server, p interface{}) {
			if v, ok := p.(float64); ok {
				srv.player.SetVolume(v)
			}
		},
		ipc.CmdSeek: func(srv *Server, p interface{}) {
			if d, ok := p.(time.Duration); ok {
				srv.player.Seek(d)
			}
		},
		ipc.CmdMute: func(srv *Server, _ interface{}) {
			srv.player.ToggleMute()
		},
		ipc.CmdQuit: func(srv *Server, _ interface{}) {
			srv.shouldQuit = true
		},
		ipc.CmdTTSLanguage: func(srv *Server, p interface{}) {
			if lang, ok := p.(string); ok {
				cfg, _ := config.LoadConfig()
				cfg.TTSLanguage = lang
				pitch := 0
				if cfg.TTSPitch != 0 {
					pitch = int(cfg.TTSPitch)
				}
				srv.player.SetTTSParams(lang, pitch) // Temp usage of SetTTSParams
			}
		},
		ipc.CmdTTSSpeaker: func(srv *Server, p interface{}) {
			if speaker, ok := p.(int); ok {
				cfg, _ := config.LoadConfig()
				srv.player.SetTTSParams(cfg.TTSLanguage, speaker)
			}
		},
	}
}

func (s *Server) handleClient(c *Client) {
	defer func() {
		s.mu.Lock()
		delete(s.clients, c.conn)
		s.mu.Unlock()
		_ = c.conn.Close()
	}()

	dec := gob.NewDecoder(c.conn)
	for {
		var cmd ipc.Command
		if err := dec.Decode(&cmd); err != nil {
			return
		}
		s.mu.Lock()
		if handler, ok := s.handlers[cmd.Type]; ok {
			handler(s, cmd.Payload)
		}
		s.mu.Unlock()
		s.broadcastState()
	}
}

func (s *Server) broadcastState() {
	s.mu.Lock()
	defer s.mu.Unlock()

	status := s.player.GetStatus()
	var current *ipc.TrackInfo
	if status.CurrentTrack != nil {
		current = &ipc.TrackInfo{
			Title:  status.CurrentTrack.Title,
			Artist: status.CurrentTrack.Artist,
			Path:   status.CurrentTrack.Path,
			Length: status.CurrentTrack.Length,
		}
	}

	playlist := make([]ipc.TrackInfo, len(s.playlist))
	for i, t := range s.playlist {
		playlist[i] = ipc.TrackInfo{Title: t.Title, Artist: t.Artist, Path: t.Path, Length: t.Length}
	}

	cfg, _ := config.LoadConfig() // Should ideally be cached or retrieved from player if player tracks it properly

	state := ipc.PlayerState{
		CurrentTrack:  current,
		IsPlaying:     status.IsPlaying,
		IsMuted:       status.IsMuted,
		Volume:        status.Volume,
		Elapsed:       status.Elapsed,
		Playlist:      playlist,
		PlayingIdx:    s.playingIdx,
		ShouldQuit:    s.shouldQuit,
		ActiveClients: len(s.clients),
		TTSLanguage:   cfg.TTSLanguage,
		TTSSpeaker:    int(cfg.TTSSpeed), // Using speed as speaker ID for now as temp hack
	}

	for _, c := range s.clients {
		_ = c.conn.SetWriteDeadline(time.Now().Add(20 * time.Millisecond))
		if err := c.enc.Encode(state); err != nil {
			// Connection error, will be handled by handleClient return
			continue
		}
	}
}
