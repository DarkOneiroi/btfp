// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package main

import (
	"btfp/internal/ipc-shared"
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	session := flag.String("session", "music", "Session name")
	flag.Parse()

	socketPath := ipc.GetSocketPath("fetcher", *session)
	_ = os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		fmt.Printf("Failed to start fetcher service: %v\n", err)
		return
	}
	defer func() { _ = listener.Close() }()

	fmt.Printf("Fetcher Service [%s] started on %s\n", *session, socketPath)

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
			var artist, album, dir string
			
			// Try []string first
			if p, ok := cmd.Payload.([]string); ok && len(p) == 3 {
				artist, album, dir = p[0], p[1], p[2]
			} else if p, ok := cmd.Payload.([]interface{}); ok && len(p) == 3 {
				// Fallback for generic slices
				artist, _ = p[0].(string)
				album, _ = p[1].(string)
				dir, _ = p[2].(string)
			}

			if artist == "" || dir == "" {
				fmt.Printf("FetchArt Error: Invalid payload: %v\n", cmd.Payload)
				continue
			}

			fmt.Printf("FetchArt Request: artist=%q, album=%q, dir=%q\n", artist, album, dir)
			path := downloadArt(artist, album, dir)
			if path != "" {
				fmt.Printf("SENDING: %+v\n", ipc.MsgFetchResult{Type: "art", Path: path})
				_ = enc.Encode(ipc.MsgFetchResult{Type: "art", Path: path})
			} else {
				fmt.Println("FetchArt Failed")
			}
		}
	}
}

func downloadLyrics(artist, title string) string {
	fmt.Printf("Lyrics Request: artist=%q, title=%q\n", artist, title)
	query := url.QueryEscape(fmt.Sprintf("%s %s", artist, title))
	apiURL := fmt.Sprintf("https://lrclib.net/api/search?q=%s", query)
	resp, err := http.Get(apiURL)
	if err != nil {
		fmt.Printf("Lyrics Error: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	var results []struct {
		SyncedLyrics string `json:"syncedLyrics"`
		PlainLyrics  string `json:"plainLyrics"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		fmt.Printf("Lyrics JSON Error: %v\n", err)
		return ""
	}

	if len(results) > 0 {
		if results[0].SyncedLyrics != "" {
			fmt.Println("Lyrics Success: Synced found")
			return results[0].SyncedLyrics
		}
		if results[0].PlainLyrics != "" {
			fmt.Println("Lyrics Success: Plain found")
			return results[0].PlainLyrics
		}
	}
	fmt.Println("Lyrics Failed: No results")
	return ""
}

func downloadArt(artist, album, dir string) string {
	fmt.Printf("downloadArt: artist=%q, album=%q, dir=%q\n", artist, album, dir)
	query := url.QueryEscape(fmt.Sprintf("%s %s", artist, album))
	apiURL := fmt.Sprintf("https://itunes.apple.com/search?term=%s&entity=album&limit=1", query)
	fmt.Printf("downloadArt: API URL=%s\n", apiURL)
	resp, err := http.Get(apiURL)
	if err != nil {
		fmt.Printf("downloadArt HTTP Error: %v\n", err)
		return ""
	}
	defer resp.Body.Close()

	var results struct {
		Results []struct {
			ArtworkUrl100 string `json:"artworkUrl100"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		fmt.Printf("downloadArt JSON Error: %v\n", err)
		return ""
	}

	if len(results.Results) == 0 {
		fmt.Println("downloadArt: No results from iTunes")
		return ""
	}

	// Get high res version
	imgURL := strings.Replace(results.Results[0].ArtworkUrl100, "100x100bb", "500x500bb", 1)
	fmt.Printf("downloadArt: Downloading image from %s\n", imgURL)
	imgResp, err := http.Get(imgURL)
	if err != nil {
		fmt.Printf("downloadArt Image HTTP Error: %v\n", err)
		return ""
	}
	defer imgResp.Body.Close()

	artPath := filepath.Join(dir, "cover.jpg")
	fmt.Printf("downloadArt: Saving to %s\n", artPath)
	out, err := os.Create(artPath)
	if err != nil {
		fmt.Printf("downloadArt Create Error: %v\n", err)
		return ""
	}
	defer out.Close()

	n, err := io.Copy(out, imgResp.Body)
	if err != nil {
		fmt.Printf("downloadArt Copy Error: %v\n", err)
		return ""
	}
	fmt.Printf("downloadArt: Saved %d bytes to %s\n", n, artPath)
	return artPath
}
