// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package main

import (
	"btfp/internal/ipc-shared"
	"btfp/services/visualization/visualizations"
	"encoding/gob"
	"flag"
	"fmt"
	"math"
	"math/rand"
	"net"
	"os"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	session := flag.String("session", "music", "Session name")
	flag.Parse()

	socketPath := ipc.GetSocketPath("viz", *session)
	_ = os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		fmt.Printf("Failed to start viz service: %v\n", err)
		return
	}
	defer func() { _ = listener.Close() }()

	fmt.Printf("Visualization Service [%s] started on %s\n", *session, socketPath)

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

	var frame *visualizations.Frame

	for {
		var cmd ipc.Command
		if err := dec.Decode(&cmd); err != nil {
			return
		}

		if cmd.Type == ipc.CmdVizGenerate {
			payload, _ := cmd.Payload.(map[string]interface{})
			w, _ := payload["width"].(int)
			h, _ := payload["height"].(int)
			isPlaying, _ := payload["isPlaying"].(bool)
			volume, _ := payload["volume"].(float64)
			pattern, _ := payload["pattern"].(int)
			colorMode, _ := payload["colorMode"].(int)
			palette, _ := payload["palette"].(int)
			currTime, _ := payload["time"].(float64)

			if w <= 0 { w = 80 }
			if h <= 0 { h = 20 }

			// FORCE RESET on pattern change or if dimensions changed significantly
			if frame == nil || frame.Width != w || frame.Height != h || int(frame.PatternType) != pattern {
				frame = visualizations.NewFrame(w, h, visualizations.PatternType(pattern))
			}

			levels := make([]float64, 32)
			for i := range levels {
				if isPlaying {
					// Use Sin + Random for a nice dynamic EQ effect
					levels[i] = (math.Sin(currTime*float64(i+1)*0.5)+1.0)*0.3 + (rand.Float64() * 0.4)
					levels[i] *= volume
				} else {
					// Low-key breathing animation when paused
					levels[i] = (math.Sin(currTime*0.5+float64(i)*0.2)+1.0)*0.05
				}
				if levels[i] > 1.0 { levels[i] = 1.0 }
			}

			frame.PatternType = visualizations.PatternType(pattern)
			frame.ColorMode = visualizations.ColorMode(colorMode)
			frame.PaletteType = visualizations.PaletteType(palette)
			frame.AudioLevels = levels
			frame.Time = currTime
			frame.GeneratePattern(levels[0])
			rendered := frame.Render(false)
			_ = enc.Encode(rendered)
		} else {
			_ = enc.Encode("")
		}
	}
}
