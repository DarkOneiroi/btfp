// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package main

import (
	"btfp/internal/ipc-shared"
	"btfp/services/core/player"
	"encoding/gob"
	"fmt"
	"net"
	"os"
)

func main() {
	_ = os.Remove(ipc.TTSSocketPath)
	listener, err := net.Listen("unix", ipc.TTSSocketPath)
	if err != nil {
		fmt.Printf("Failed to start TTS service: %v\n", err)
		return
	}
	defer func() { _ = listener.Close() }()

	fmt.Println("TTS Service started on", ipc.TTSSocketPath)

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

	var currentTTS *player.TTSPlayer

	for {
		var cmd ipc.Command
		if err := dec.Decode(&cmd); err != nil {
			return
		}

		switch cmd.Type {
		case ipc.CmdTTSGenerate:
			payload, _ := cmd.Payload.(map[string]interface{})
			text, _ := payload["text"].(string)
			lang, _ := payload["lang"].(string)
			speaker, _ := payload["speaker"].(int)

			model, lexicon, tokens := player.GetTTSModelPaths(lang)
			if currentTTS == nil {
				var err error
				currentTTS, err = player.NewTTSPlayer(model, lexicon, tokens)
				if err != nil {
					fmt.Printf("TTS Init Error: %v\n", err)
					continue
				}
			}

			streamer, format, err := currentTTS.GenerateAudio(text, speaker)
			if err == nil {
				samples := make([]float32, streamer.Len())
				_ = enc.Encode(ipc.MsgTTSResult{Samples: samples, Rate: int(format.SampleRate)})
			}
		}
	}
}
