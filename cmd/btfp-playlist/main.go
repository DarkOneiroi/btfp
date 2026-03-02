// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package main

import (
	"btfp/internal/ipc-shared"
	"encoding/gob"
	"fmt"
	"net"
	"os"
)

func main() {
	_ = os.Remove(ipc.PlaylistSocketPath)
	listener, err := net.Listen("unix", ipc.PlaylistSocketPath)
	if err != nil {
		fmt.Printf("Failed to start playlist service: %v\n", err)
		return
	}
	defer func() { _ = listener.Close() }()

	fmt.Println("Playlist Service started on", ipc.PlaylistSocketPath)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	defer func() { _ = conn.Close() }()
	dec := gob.NewDecoder(conn)
	enc := gob.NewEncoder(conn)

	var playlist []ipc.TrackInfo

	for {
		var cmd ipc.Command
		if err := dec.Decode(&cmd); err != nil {
			return
		}

		switch cmd.Type {
		case ipc.CmdPlaylistAdd:
			if t, ok := cmd.Payload.(ipc.TrackInfo); ok {
				playlist = append(playlist, t)
			}
		case ipc.CmdPlaylistRemove:
			if idx, ok := cmd.Payload.(int); ok && idx >= 0 && idx < len(playlist) {
				playlist = append(playlist[:idx], playlist[idx+1:]...)
			}
		case ipc.CmdPlaylistGet:
			_ = enc.Encode(playlist)
		case ipc.CmdPlaylistClear:
			playlist = []ipc.TrackInfo{}
		}
	}
}
