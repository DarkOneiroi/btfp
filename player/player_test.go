package player

import (
	"testing"
)

func TestMusicPlayerInitialization(t *testing.T) {
	p := NewMusicPlayer()
	if p.Volume != 1.0 {
		t.Errorf("expected default volume 1.0, got %f", p.Volume)
	}
	if p.IsPlaying {
		t.Error("expected player to be stopped on init")
	}
}

func TestVolumeControls(t *testing.T) {
	p := NewMusicPlayer()

	p.SetVolume(0.5)
	if p.Volume != 0.5 {
		t.Errorf("expected volume 0.5, got %f", p.Volume)
	}

	p.SetVolume(1.5) // Test clamping
	if p.Volume != 1.0 {
		t.Errorf("expected clamped volume 1.0, got %f", p.Volume)
	}

	p.ToggleMute()
	if !p.IsMuted || p.Volume != 0 {
		t.Error("mute toggle failed")
	}

	p.ToggleMute()
	if p.IsMuted || p.Volume != 1.0 {
		t.Error("unmute toggle failed")
	}
}
