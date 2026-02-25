package player

import (
	"testing"
	"time"
)

func TestMusicPlayerInitialization(t *testing.T) {
	p := NewMusicPlayer()
	if p.Volume != 1.0 {
		t.Errorf("Expected volume 1.0, got %f", p.Volume)
	}
	if p.IsPlaying {
		t.Error("Player should not be playing on initialization")
	}
}

func TestVolumeControls(t *testing.T) {
	p := NewMusicPlayer()

	p.SetVolume(0.5)
	if p.Volume != 0.5 {
		t.Errorf("Expected volume 0.5, got %f", p.Volume)
	}

	p.ToggleMute()
	if !p.IsMuted {
		t.Error("Player should be muted")
	}
	if p.Volume != 0 {
		t.Errorf("Expected volume 0 when muted, got %f", p.Volume)
	}

	p.ToggleMute()
	if p.IsMuted {
		t.Error("Player should be unmuted")
	}
	if p.Volume != 0.5 {
		t.Errorf("Expected volume restored to 0.5, got %f", p.Volume)
	}
}

func TestTrackMetadata(t *testing.T) {
	track := Track{
		Title:  "King Nothing",
		Artist: "Metallica",
		Path:   "/path/to/song.mp3",
		Length: 300 * time.Second,
	}

	if track.Title != "King Nothing" {
		t.Errorf("Incorrect title: %s", track.Title)
	}
}
