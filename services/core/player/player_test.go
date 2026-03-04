// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package player

import (
	"btfp/internal/config"
	"btfp/internal/models"
	"testing"
	"time"

	"github.com/gopxl/beep"
)

func TestMusicPlayerInitialization(t *testing.T) {
	p := NewMusicPlayer(config.Config{})
	status := p.GetStatus()
	if status.Volume != 1.0 {
		t.Errorf("Expected volume 1.0, got %f", status.Volume)
	}
	if status.IsPlaying {
		t.Error("Player should not be playing on initialization")
	}
}

func TestVolumeControls(t *testing.T) {
	p := NewMusicPlayer(config.Config{})

	p.SetVolume(0.5)
	status := p.GetStatus()
	if status.Volume != 0.5 {
		t.Errorf("Expected volume 0.5, got %f", status.Volume)
	}

	p.ToggleMute()
	status = p.GetStatus()
	if !status.IsMuted {
		t.Error("Player should be muted")
	}
	if status.Volume != 0 {
		t.Errorf("Expected volume 0 when muted, got %f", status.Volume)
	}

	p.ToggleMute()
	status = p.GetStatus()
	if status.IsMuted {
		t.Error("Player should be unmuted")
	}
	if status.Volume != 0.5 {
		t.Errorf("Expected volume restored to 0.5, got %f", status.Volume)
	}
}

func TestTrackMetadata(t *testing.T) {
	track := models.Track{
		Title:  "King Nothing",
		Artist: "Metallica",
		Path:   "/path/to/song.mp3",
		Length: 300 * time.Second,
	}

	if track.Title != "King Nothing" {
		t.Errorf("Incorrect title: %s", track.Title)
	}
}
