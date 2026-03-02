// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package main

import (
	"btfp/internal/ipc-shared"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

func main() {
	_ = os.Remove(ipc.FetcherSocketPath)
	listener, err := net.Listen("unix", ipc.FetcherSocketPath)
	if err != nil {
		fmt.Printf("Failed to start fetcher service: %v\n", err)
		return
	}
	defer func() { _ = listener.Close() }()

	fmt.Println("Fetcher Service started on", ipc.FetcherSocketPath)

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

	for {
		var cmd ipc.Command
		if err := dec.Decode(&cmd); err != nil {
			return
		}

		switch cmd.Type {
		case ipc.CmdFetchLyrics:
			payload, _ := cmd.Payload.([]string) // [artist, title, lrcPath]
			if len(payload) == 3 {
				content := downloadLyrics(payload[0], payload[1])
				if content != "" {
					_ = os.WriteFile(payload[2], []byte(content), 0644)
					_ = enc.Encode(ipc.MsgFetchResult{Type: "lyrics", Path: payload[2], Content: content})
				}
			}

		case ipc.CmdFetchArt:
			dir, _ := cmd.Payload.(string)
			path := downloadArt(dir)
			if path != "" {
				_ = enc.Encode(ipc.MsgFetchResult{Type: "art", Path: path})
			}
		}
	}
}

func downloadLyrics(artist, title string) string {
	query := url.QueryEscape(fmt.Sprintf("%s %s", artist, title))
	apiURL := fmt.Sprintf("https://lrclib.net/api/search?q=%s", query)
	resp, err := http.Get(apiURL)
	if err != nil {
		return ""
	}
	defer func() { _ = resp.Body.Close() }()

	var results []struct {
		SyncedLyrics string `json:"syncedLyrics"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return ""
	}

	if len(results) > 0 {
		return results[0].SyncedLyrics
	}
	return ""
}

func downloadArt(dir string) string {
	resp, err := http.Get("https://picsum.photos/500")
	if err != nil {
		return ""
	}
	defer func() { _ = resp.Body.Close() }()

	artPath := filepath.Join(dir, "cover.jpg")
	out, err := os.Create(artPath)
	if err != nil {
		return ""
	}
	defer func() { _ = out.Close() }()

	return artPath
}
