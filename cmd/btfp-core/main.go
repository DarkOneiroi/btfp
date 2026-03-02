// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package main

import (
	"btfp/internal/ipc-shared"
	"btfp/services/core/player"
	"btfp/services/core/server"
	"btfp/services/ui/tui"
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Register types for interface{} encoding
	gob.Register(time.Duration(0))
	gob.Register(0)
	gob.Register(player.Track{})
	gob.Register([]player.Track{})

	daemonFlag := flag.Bool("daemon", false, "Start as server daemon")
	viewFlag := flag.String("view", "", "Start in specific view (library, playlist, player, viz)")
	allFlag := flag.Bool("all", false, "Start all 4 views in separate terminal windows")
	waybarFlag := flag.String("waybar", "none", "Output Waybar JSON status (all, prev, status, next, mute, song)")
	remoteFlag := flag.String("remote", "none", "Send remote command (play, pause, next, prev, mute)")
	flag.Parse()

	if *daemonFlag {
		server.Start()
		return
	}

	if *allFlag {
		startAll()
		*viewFlag = "library"
	}

	if *waybarFlag != "none" {
		outputWaybar(*waybarFlag)
		return
	}

	if *remoteFlag != "none" {
		sendRemote(*remoteFlag)
		return
	}

	// Default: Try to connect to existing server or start TUI
	conn, err := net.Dial("unix", ipc.SocketPath)
	if err != nil {
		// No server, start one and retry
		go server.Start()
		time.Sleep(500 * time.Millisecond)
		conn, err = net.Dial("unix", ipc.SocketPath)
		if err != nil {
			fmt.Println("Starting standalone mode...")
			startTUI(*viewFlag, nil)
			return
		}
	}
	defer func() { _ = conn.Close() }()

	runClient(conn, *viewFlag)
}

func startAll() {
	views := []string{"playlist", "player", "viz"}
	for _, v := range views {
		_ = exec.Command("wezterm", "start", os.Args[0], "--view", v).Start()
	}
}

func startTUI(view string, conn net.Conn) {
	m := tui.NewModel(view)
	if conn != nil {
		m.SetConn(conn, gob.NewEncoder(conn), gob.NewDecoder(conn))
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func runClient(conn net.Conn, view string) {
	startTUI(view, conn)
}

func outputWaybar(mode string) {
	conn, err := net.Dial("unix", ipc.SocketPath)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	_ = conn.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
	dec := gob.NewDecoder(conn)
	var state ipc.PlayerState
	if err := dec.Decode(&state); err != nil {
		return
	}

	output := make(map[string]interface{})
	status := ""
	if state.IsPlaying {
		status = ""
	}
	if state.IsMuted {
		status = "󰝟"
	}

	song := "Stopped"
	if state.CurrentTrack != nil {
		song = fmt.Sprintf("%s - %s", state.CurrentTrack.Artist, state.CurrentTrack.Title)
	}

	switch mode {
	case "status":
		fmt.Println(status)
	case "song":
		fmt.Println(song)
	case "all":
		output["text"] = fmt.Sprintf("%s %s", status, song)
		output["tooltip"] = song
		output["class"] = "custom-btfp"
		_ = json.NewEncoder(os.Stdout).Encode(output)
	}
}

func sendRemote(cmd string) {
	conn, err := net.Dial("unix", ipc.SocketPath)
	if err != nil {
		return
	}
	defer func() { _ = conn.Close() }()

	enc := gob.NewEncoder(conn)
	var c ipc.Command
	switch cmd {
	case "play":
		c = ipc.Command{Type: ipc.CmdPlay}
	case "pause":
		c = ipc.Command{Type: ipc.CmdPause}
	case "next":
		c = ipc.Command{Type: ipc.CmdNext}
	case "prev":
		c = ipc.Command{Type: ipc.CmdPrev}
	case "mute":
		c = ipc.Command{Type: ipc.CmdMute}
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		return
	}
	_ = enc.Encode(c)
}
