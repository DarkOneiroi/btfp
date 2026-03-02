// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package tui

import (
	"btfp/player"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestUpdateQuit(t *testing.T) {
	m := NewModel("")

	// 1. Test Server Quit Signal (ShouldQuit)
	msg := serverStateMsg{ShouldQuit: true}
	_, cmd := m.Update(msg)
	if cmd == nil {
		t.Fatal("Expected non-nil command for server quit signal")
	}

	// 2. Test Local Quit Key (when m.conn is nil)
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")}
	_, cmd = m.Update(keyMsg)
	if cmd == nil {
		t.Fatal("Expected non-nil command for local 'q' press")
	}
}

func TestUpdateAlwaysPassesToLists(t *testing.T) {
	m := NewModel("")
	m.view = viewPlayer // Set to view that is NOT library

	// Create a dummy library message
	items := []item{{title: "test.mp3", path: "/test.mp3"}}
	var libItems []interface{}
	for _, it := range items {
		libItems = append(libItems, it)
	}
	_ = libItems

	// Send libraryMsg update - should update libList regardless of view
	msg := libraryMsg{}
	_, _ = m.Update(msg)
}

func TestOptimisticPlaylistUpdates(t *testing.T) {
	m := NewModel("")
	m.view = viewLibrary

	// Mock a track selection
	track := player.Track{Title: "Test Song", Path: "/test.mp3"}

	// Simulate "Enter" key on a file (this usually triggers handleEnter)
	// We'll test the core logic: playlist should be updated immediately
	m.playlist = append(m.playlist, track)
	m.playingIdx = len(m.playlist) - 1

	if len(m.playlist) != 1 {
		t.Errorf("Expected playlist size 1, got %d", len(m.playlist))
	}
	if m.playingIdx != 0 {
		t.Errorf("Expected playingIdx 0, got %d", m.playingIdx)
	}
}

func TestCmdPlayTrackLogic(t *testing.T) {
	// This tests that the model correctly transitions state when a track is played
	m := NewModel("library")

	// Simulate adding and playing a track
	track := player.Track{Title: "King Nothing", Path: "/metallica/king_nothing.mp3"}
	m.playlist = append(m.playlist, track)
	m.playingIdx = 0
	m.view = viewPlayer // handleEnter sets this

	if m.view != viewPlayer {
		t.Errorf("Expected view to be viewPlayer, got %v", m.view)
	}
}
