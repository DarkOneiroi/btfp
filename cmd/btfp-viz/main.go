// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package main

import (
	"btfp/internal/ipc-shared"
	"btfp/services/visualization/visualizations"
	"encoding/gob"
	"fmt"
	"net"
	"os"
)

func main() {
	_ = os.Remove(ipc.VizSocketPath)
	listener, err := net.Listen("unix", ipc.VizSocketPath)
	if err != nil {
		fmt.Printf("Failed to start visualization service: %v\n", err)
		return
	}
	defer func() { _ = listener.Close() }()

	fmt.Println("Visualization Service started on", ipc.VizSocketPath)

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
			levels, _ := payload["levels"].([]float64)
			pattern, _ := payload["pattern"].(int)

			if frame == nil || frame.Width != w || frame.Height != h {
				frame = visualizations.NewFrame(w, h, visualizations.PatternType(pattern))
			}

			frame.AudioLevels = levels
			frame.GeneratePattern(levels[0])
			rendered := frame.Render(false)
			_ = enc.Encode(rendered)
		}
	}
}
