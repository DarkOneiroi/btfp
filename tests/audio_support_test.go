// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package tests

import (
	"btfp/internal/utils"
	"btfp/services/ui/tui"
	"os"
	"path/filepath"
	"testing"
)

func TestAudioFormatSupport(t *testing.T) {
	// Setup environment
	tmpHome := t.TempDir()
	_ = os.Setenv("HOME", tmpHome)

	// Create a mock music folder
	musicDir := filepath.Join(tmpHome, "Music")
	_ = os.MkdirAll(musicDir, 0755)

	formats := []string{"test.mp3", "test.wav", "test.flac", "test.ogg", "test.m4a", "test.aac"}
	for _, f := range formats {
		_ = os.WriteFile(filepath.Join(musicDir, f), []byte("mock audio data"), 0644)
	}

	// 1. Initialize Model
	_ = tui.NewModel("library", "music")

	// 2. Check utils for support
	for _, f := range formats {
		if !utils.IsSupportedAudioFile(f) {
			t.Errorf("Format %s should be supported", f)
		}
	}
}
