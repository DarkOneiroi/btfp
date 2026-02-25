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
	os.Setenv("HOME", tmpHome)

	// Create a mock music folder
	musicDir := filepath.Join(tmpHome, "Music")
	os.MkdirAll(musicDir, 0755)
	os.WriteFile(filepath.Join(musicDir, "test.mp3"), []byte("mock audio data"), 0644)

	// 1. Initialize Model
	m := tui.NewModel("library")

	// 2. Mock Window Size
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// 3. Verify initial state
	if m.View() == "" {
		t.Error("View rendered empty string")
	}
}

func TestLayoutOverflow(t *testing.T) {
	m := tui.NewModel("library")
	// Simulate terminal size
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	
	view := m.View()
	if view == "" {
		t.Error("View rendered empty string")
	}
	
	// Basic height check: the output should be contained within terminal height
	// lipgloss.Place might add some vertical centering but we use lipgloss.Top now.
	lines := strings.Count(view, "\n")
	if lines > 25 {
		t.Errorf("View has too many lines: %d, might overflow terminal", lines)
	}
}
