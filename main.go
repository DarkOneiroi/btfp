package main

import (
	"btfp/ipc"
	"btfp/server"
	"btfp/tui"
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	daemonFlag := flag.Bool("daemon", false, "Start as background server")
	viewFlag := flag.String("view", "", "Start in specific view (library, playlist, player, viz)")
	allFlag := flag.Bool("all", false, "Start all 4 views in separate terminal windows")
	waybarFlag := flag.String("waybar", "none", "Output Waybar JSON status (all, prev, status, next, mute, song)")
	remoteFlag := flag.String("remote", "none", "Send remote command (play, pause, next, prev, mute)")
	flag.Parse()

	// Check for subcommands in positional arguments
	args := flag.Args()
	if len(args) > 0 {
		switch args[0] {
		case "waybar":
			comp := "all"
			if len(args) > 1 {
				comp = args[1]
			}
			outputWaybar(comp)
			return
		case "remote":
			cmd := "pause"
			if len(args) > 1 {
				cmd = args[1]
			}
			sendRemote(cmd)
			return
		}
	}

	// Handle flags
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
		time.Sleep(200 * time.Millisecond)
		conn, err = net.Dial("unix", ipc.SocketPath)
		if err != nil {
			fmt.Println("Starting standalone mode...")
			startTUI(*viewFlag, nil)
			return
		}
	}
	defer conn.Close()

	runClient(conn, *viewFlag)
}

func startAll() {
	views := []string{"playlist", "player", "viz"}
	for _, v := range views {
		exec.Command("wezterm", "start", os.Args[0], "--view", v).Start()
	}
}

func startTUI(view string, conn net.Conn) {
	m := tui.NewModel(view)
	if conn != nil {
		m.SetConn(conn)
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

func outputWaybar(component string) {
	conn, err := net.Dial("unix", ipc.SocketPath)
	if err != nil {
		// Return empty output if server not running
		fmt.Println("{}")
		return
	}
	defer conn.Close()

	// Request state
	dec := gob.NewDecoder(conn)
	
	var state ipc.PlayerState
	conn.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
	if err := dec.Decode(&state); err != nil {
		fmt.Println("{}")
		return
	}

	if state.ActiveClients <= 0 {
		fmt.Println("{}")
		return
	}

	status := "󰐊" // Play icon
	if state.IsPlaying {
		status = "󰏤" // Pause icon
	}
	mute := "󰕾"
	if state.IsMuted {
		mute = "󰝟"
	}

	song := "No track"
	if state.CurrentTrack != nil {
		song = fmt.Sprintf("%s - %s", state.CurrentTrack.Artist, state.CurrentTrack.Title)
	}

	// Use play/pause icons for the status indicator
	icon := "󰏤" // Pause icon (showing it is paused)
	if state.IsPlaying {
		icon = "󰐊" // Play icon (showing it is playing)
	}

	var text string
	switch component {
	case "prev":
		text = "󰒮"
	case "status":
		text = status
	case "next":
		text = "󰒭"
	case "mute":
		text = mute
	case "song":
		text = song
	default: // "all" or anything else
		text = fmt.Sprintf("%s %s", icon, song)
	}

	output := map[string]string{
		"text":    text,
		"tooltip": fmt.Sprintf("BehindTheForestPlayer (BTFP)\nTrack: %s\nVolume: %d%%", song, int(state.Volume*100)),
		"class":   "custom-btfp",
	}
	
	if state.IsPlaying {
		output["class"] = "custom-btfp playing"
	}

	json.NewEncoder(os.Stdout).Encode(output)
}

func sendRemote(cmd string) {
	conn, err := net.Dial("unix", ipc.SocketPath)
	if err != nil {
		fmt.Printf("Error: no BTFP server running\n")
		return
	}
	defer conn.Close()

	enc := gob.NewEncoder(conn)
	var c ipc.Command
	switch strings.ToLower(cmd) {
	case "play", "pause":
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
	enc.Encode(c)
}
