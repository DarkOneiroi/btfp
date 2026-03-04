// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package server

import (
	"btfp/internal/config"
	"btfp/internal/ipc-shared"
	"btfp/internal/models"
	"btfp/services/core/player"
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"sync"
	"time"
)

type MusicServer struct {
	player     *player.MusicPlayer
	playlist   []models.Track
	playingIdx int
	clients    map[net.Conn]*Client
	mu         sync.Mutex
	shouldQuit bool
	listener   net.Listener
	cmdChan    chan commandPacket
	session    string
}

func NewMusicServer(session string) *MusicServer {
	cfg, _ := config.LoadConfig()
	p := player.NewMusicPlayer(cfg)
	p.SetSession(session)
	return &MusicServer{
		player:     p,
		clients:    make(map[net.Conn]*Client),
		playingIdx: -1,
		cmdChan:    make(chan commandPacket, 100),
		session:    session,
	}
}

func (s *MusicServer) Start() {
	socketPath := ipc.GetSocketPath("core", s.session)
	_ = os.Remove(socketPath)

	var err error
	s.listener, err = net.Listen("unix", socketPath)
	if err != nil {
		fmt.Printf("Failed to start music server: %v\n", err)
		return
	}
	defer func() { _ = s.listener.Close() }()

	go s.processCommands()
	go s.updateLoop()

	fmt.Printf("BTFP Music Server [%s] started on %s\n", s.session, socketPath)

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.shouldQuit { return }
			continue
		}

		s.mu.Lock()
		client := &Client{conn: conn, enc: gob.NewEncoder(conn)}
		s.clients[conn] = client
		s.mu.Unlock()

		go s.handleClient(client)
	}
}

func (s *MusicServer) processCommands() {
	for p := range s.cmdChan {
		s.mu.Lock()
		s.execute(p.cmd)
		s.broadcast()
		s.mu.Unlock()
	}
}

func (s *MusicServer) updateLoop() {
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
		s.broadcast()
		if shouldQuit {
			time.Sleep(100 * time.Millisecond)
			_ = s.listener.Close()
			os.Exit(0)
		}
	}
}

func (s *MusicServer) execute(cmd ipc.Command) {
	switch cmd.Type {
	case ipc.CmdPlay:
		if idx, ok := cmd.Payload.(int); ok && idx >= 0 && idx < len(s.playlist) {
			s.playingIdx = idx
			_ = s.player.PlayTrack(&s.playlist[s.playingIdx])
		} else {
			s.player.TogglePause()
		}
	case ipc.CmdPause:
		s.player.TogglePause()
	case ipc.CmdNext:
		if len(s.playlist) > 0 {
			s.playingIdx = (s.playingIdx + 1) % len(s.playlist)
			_ = s.player.PlayTrack(&s.playlist[s.playingIdx])
		}
	case ipc.CmdPrev:
		if len(s.playlist) > 0 {
			s.playingIdx = (s.playingIdx - 1 + len(s.playlist)) % len(s.playlist)
			_ = s.player.PlayTrack(&s.playlist[s.playingIdx])
		}
	case ipc.CmdPlaylistAdd:
		if t, ok := cmd.Payload.(ipc.TrackInfo); ok {
			s.playlist = append(s.playlist, models.Track{
				Title: t.Title, Artist: t.Artist, Path: t.Path, Length: t.Length,
			})
			if s.playingIdx == -1 {
				s.playingIdx = 0
				_ = s.player.PlayTrack(&s.playlist[0])
			}
		}
	case ipc.CmdPlayTrack:
		if t, ok := cmd.Payload.(ipc.TrackInfo); ok {
			newTrack := models.Track{Title: t.Title, Artist: t.Artist, Path: t.Path, Length: t.Length}
			s.playlist = append(s.playlist, newTrack)
			s.playingIdx = len(s.playlist) - 1
			_ = s.player.PlayTrack(&newTrack)
		}
	case ipc.CmdVolume:
		if v, ok := cmd.Payload.(float64); ok { s.player.SetVolume(v) }
	case ipc.CmdSeek:
		if d, ok := cmd.Payload.(time.Duration); ok { s.player.Seek(d) }
	case ipc.CmdMute:
		s.player.ToggleMute()
	case ipc.CmdQuit:
		s.shouldQuit = true
	}
}

func (s *MusicServer) handleClient(c *Client) {
	defer func() {
		s.mu.Lock()
		delete(s.clients, c.conn)
		s.mu.Unlock()
		_ = c.conn.Close()
	}()
	dec := gob.NewDecoder(c.conn)
	for {
		var cmd ipc.Command
		if err := dec.Decode(&cmd); err != nil { return }
		s.cmdChan <- commandPacket{cmd: cmd, c: c}
	}
}

func (s *MusicServer) broadcast() {
	status := s.player.GetStatus()
	var current *ipc.TrackInfo
	if status.CurrentTrack != nil {
		current = &ipc.TrackInfo{
			Title: status.CurrentTrack.Title, Artist: status.CurrentTrack.Artist,
			Path: status.CurrentTrack.Path, Length: status.CurrentTrack.Length,
		}
	}
	playlist := make([]ipc.TrackInfo, len(s.playlist))
	for i, t := range s.playlist {
		playlist[i] = ipc.TrackInfo{Title: t.Title, Artist: t.Artist, Path: t.Path, Length: t.Length}
	}
	state := ipc.PlayerState{
		CurrentTrack: current, IsPlaying: status.IsPlaying, IsMuted: status.IsMuted,
		Volume: status.Volume, Elapsed: status.Elapsed, Playlist: playlist,
		PlayingIdx: s.playingIdx, ShouldQuit: s.shouldQuit, ActiveClients: len(s.clients),
	}
	for _, c := range s.clients {
		_ = c.conn.SetWriteDeadline(time.Now().Add(50 * time.Millisecond))
		_ = c.enc.Encode(state)
	}
}
