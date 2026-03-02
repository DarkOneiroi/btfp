// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package tests

import (
	"btfp/tui"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestAppLifecycle(t *testing.T) {
	// Setup environment
	tmpHome := t.TempDir()
	_ = os.Setenv("HOME", tmpHome)

	// Create a mock music folder
	musicDir := filepath.Join(tmpHome, "Music")
	_ = os.MkdirAll(musicDir, 0755)
	_ = os.WriteFile(filepath.Join(musicDir, "test.mp3"), []byte("mock audio data"), 0644)

	// 1. Initialize Model
	m := tui.NewModel("library")

	// 2. Mock Window Size
	_, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// 3. Verify initial state
	if m.View() == "" {
		t.Error("View rendered empty string")
	}
}

func TestLayoutOverflow(t *testing.T) {
	m := tui.NewModel("library")
	// Simulate terminal size
	_, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	view := m.View()
	if view == "" {
		t.Error("View rendered empty string")
	}

	// Basic height check: the output should be contained within terminal height
	lines := strings.Count(view, "
")
	if lines > 25 {
		t.Errorf("View has too many lines: %d, might overflow terminal", lines)
	}
}
