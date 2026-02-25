package player

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/effects"
	"github.com/gopxl/beep/flac"
	"github.com/gopxl/beep/mp3"
	"github.com/gopxl/beep/speaker"
	"github.com/gopxl/beep/vorbis"
	"github.com/gopxl/beep/wav"
)

// Track represents a single audio track and its metadata
type Track struct {
	Title  string
	Artist string
	Path   string
	Length time.Duration
}

// MusicPlayer handles the audio playback lifecycle
type MusicPlayer struct {
	CurrentTrack *Track
	IsPlaying    bool
	IsDone       bool
	IsMuted      bool
	Volume       float64 // 0.0 to 1.0
	prevVolume   float64
	Elapsed      time.Duration
	ctrl         *beep.Ctrl
	volumeCtrl   *effects.Volume
	streamer     beep.StreamSeekCloser
	sampleRate   beep.SampleRate
}

// NewMusicPlayer creates and initializes a new music player instance
func NewMusicPlayer() *MusicPlayer {
	return &MusicPlayer{
		Volume: 1.0,
	}
}

// PlayTrack starts playback of the given track, selecting the appropriate decoder
func (p *MusicPlayer) PlayTrack(t *Track) error {
	if p.streamer != nil {
		_ = p.streamer.Close()
	}

	var streamer beep.StreamSeekCloser
	var format beep.Format
	var err error
	var f *os.File

	ext := strings.ToLower(filepath.Ext(t.Path))

	// Strategy: Attempt native decoding first for efficiency
	switch ext {
	case ".mp3":
		f, err = os.Open(t.Path)
		if err == nil {
			streamer, format, err = mp3.Decode(f)
		}
	case ".wav":
		f, err = os.Open(t.Path)
		if err == nil {
			streamer, format, err = wav.Decode(f)
		}
	case ".flac":
		f, err = os.Open(t.Path)
		if err == nil {
			streamer, format, err = flac.Decode(f)
		}
	case ".ogg", ".vorbis":
		f, err = os.Open(t.Path)
		if err == nil {
			streamer, format, err = vorbis.Decode(f)
		}
	default:
		// Fallback to ffmpeg for universal compatibility (M4A, AAC, etc.)
		streamer, format, err = p.decodeWithFFmpeg(t.Path)
	}

	// Recovery Strategy: If native decoding fails, try FFmpeg as a last resort
	if err != nil {
		if f != nil {
			_ = f.Close()
		}
		if ext == ".mp3" || ext == ".wav" || ext == ".flac" || ext == ".ogg" {
			streamer, format, err = p.decodeWithFFmpeg(t.Path)
		}

		if err != nil {
			return fmt.Errorf("failed to decode track %s: %w", t.Path, err)
		}
	}

	p.sampleRate = format.SampleRate
	p.streamer = streamer
	p.IsDone = false

	if streamer.Len() > 0 {
		t.Length = format.SampleRate.D(streamer.Len()).Round(time.Second)
	}

	// Initialize speaker if this is the first track
	if p.ctrl == nil {
		err = speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
		if err != nil {
			return fmt.Errorf("failed to initialize speaker: %w", err)
		}
	}

	p.ctrl = &beep.Ctrl{Streamer: streamer, Paused: false}

	// Set up volume control effect
	p.volumeCtrl = &effects.Volume{
		Streamer: p.ctrl,
		Base:     2,
		Volume:   0, // 0 means no change (multiplier = 1.0)
	}
	p.applyVolume()

	p.CurrentTrack = t
	p.IsPlaying = true
	p.Elapsed = 0

	speaker.Clear()
	speaker.Play(p.volumeCtrl)

	return nil
}

// TogglePause toggles the play/pause state of the player
func (p *MusicPlayer) TogglePause() {
	if p.ctrl != nil {
		p.ctrl.Paused = !p.ctrl.Paused
		p.IsPlaying = !p.ctrl.Paused
	}
}

// SetVolume sets the player volume (0.0 to 1.0)
func (p *MusicPlayer) SetVolume(v float64) {
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	p.Volume = v
	if !p.IsMuted {
		p.applyVolume()
	}
}

// ToggleMute toggles the mute state
func (p *MusicPlayer) ToggleMute() {
	if p.IsMuted {
		p.IsMuted = false
		p.Volume = p.prevVolume
	} else {
		p.IsMuted = true
		p.prevVolume = p.Volume
		p.Volume = 0
	}
	p.applyVolume()
}

func (p *MusicPlayer) applyVolume() {
	if p.volumeCtrl != nil {
		// Beep's volume is logarithmic. Volume 0 is 1.0x, -1 is 0.5x, etc.
		// We map 0.0-1.0 to something reasonable.
		if p.Volume == 0 {
			p.volumeCtrl.Volume = -10 // Close to silent
		} else {
			p.volumeCtrl.Volume = math.Log2(p.Volume)
		}
	}
}

// Seek moves the playback position by the given duration
func (p *MusicPlayer) Seek(d time.Duration) {
	if p.streamer == nil {
		return
	}

	newPos := p.streamer.Position() + p.sampleRate.N(d)
	if newPos < 0 {
		newPos = 0
	}
	if newPos >= p.streamer.Len() {
		newPos = p.streamer.Len() - 1
	}

	speaker.Lock()
	_ = p.streamer.Seek(newPos)
	speaker.Unlock()
}

// Update refreshes the player's internal state (elapsed time, completion)
func (p *MusicPlayer) Update() {
	if p.IsPlaying && p.streamer != nil {
		pos := p.streamer.Position()
		p.Elapsed = p.sampleRate.D(pos)

		if pos >= p.streamer.Len() {
			p.IsDone = true
			p.IsPlaying = false
		}
	}
}
