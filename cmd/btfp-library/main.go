// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package main

import (
	"btfp/internal/ipc-shared"
	"btfp/internal/utils"
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"

	"github.com/bogem/id3v2/v2"
)

func main() {
	_ = os.Remove(ipc.LibrarySocketPath)
	listener, err := net.Listen("unix", ipc.LibrarySocketPath)
	if err != nil {
		fmt.Printf("Failed to start library service: %v\n", err)
		return
	}
	defer func() { _ = listener.Close() }()

	fmt.Println("Library Service started on", ipc.LibrarySocketPath)

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
		case ipc.CmdLibScan:
			path, _ := cmd.Payload.(string)
			entries := scanDirectory(path)
			_ = enc.Encode(ipc.MsgLibEntries{Path: path, Entries: entries})

		case ipc.CmdLibGetMetadata:
			path, _ := cmd.Payload.(string)
			meta := getTrackMetadata(path)
			_ = enc.Encode(meta)
		}
	}
}

func scanDirectory(path string) []ipc.LibEntry {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}

	var res []ipc.LibEntry
	for _, entry := range entries {
		info, _ := entry.Info()
		fullPath := filepath.Join(path, entry.Name())
		if entry.IsDir() || utils.IsSupportedAudioFile(entry.Name()) {
			desc := "Dir"
			if !entry.IsDir() {
				desc = fmt.Sprintf("%.1f MB", float64(info.Size())/1024/1024)
			}
			res = append(res, ipc.LibEntry{
				Title: entry.Name(),
				Desc:  desc,
				Path:  fullPath,
				IsDir: entry.IsDir(),
			})
		}
	}
	return res
}

func getTrackMetadata(path string) ipc.TrackInfo {
	fileName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	dir := filepath.Dir(path)
	artistName := filepath.Base(filepath.Dir(dir))

	track := ipc.TrackInfo{
		Title: fileName,
		Path:  path,
	}

	tag, err := id3v2.Open(path, id3v2.Options{Parse: true})
	if err == nil {
		defer func() { _ = tag.Close() }()
		artist := tag.GetTextFrame("TPE1").Text
		title := tag.GetTextFrame("TIT2").Text

		if artist != "" {
			track.Artist = artist
		}
		if title != "" {
			track.Title = title
		}
	}

	if track.Artist == "" && artistName != "Music" && artistName != "." {
		track.Artist = artistName
	}

	if track.Artist == "" {
		track.Artist = "Unknown Artist"
	}

	return track
}
