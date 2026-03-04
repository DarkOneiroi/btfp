// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package player

import (
	"btfp/internal/config"
	"btfp/internal/models"
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

// Global fixed sample rate to avoid hardware re-init panics
const HardwareSampleRate = beep.SampleRate(44100)

type Player interface {
	PlayTrack(t *models.Track) error
	TogglePause()
	SetVolume(v float64)
	ToggleMute()
	Seek(d time.Duration)
	Update()
	GetStatus() models.Status
	SetStatus(s models.Status)
}

// MusicPlayer handles the audio playback lifecycle
type MusicPlayer struct {
	status      models.Status
	prevVolume  float64
	ctrl        *beep.Ctrl
	volumeCtrl  *effects.Volume
	resampler   *beep.Resampler
	streamer    beep.StreamSeekCloser
	format      beep.Format
	cfg         config.Config
	initialized bool
	session     string
}

// NewMusicPlayer creates and initializes a new music player instance
func NewMusicPlayer(cfg config.Config) *MusicPlayer {
	return &MusicPlayer{
		status: models.Status{
			Volume: 1.0,
		},
		cfg: cfg,
	}
}

func (p *MusicPlayer) SetSession(s string) {
	p.session = s
}

func (p *MusicPlayer) GetStatus() models.Status { return p.status }
func (p *MusicPlayer) SetStatus(s models.Status) { p.status = s }

func (p *MusicPlayer) PlayTrack(t *models.Track) error {
	if p.streamer != nil {
		_ = p.streamer.Close()
		p.streamer = nil
	}

	var streamer beep.StreamSeekCloser
	var format beep.Format
	var err error
	var f *os.File

	ext := strings.ToLower(filepath.Ext(t.Path))

	switch ext {
	case ".mp3":
		f, err = os.Open(t.Path)
		if err == nil { streamer, format, err = mp3.Decode(f) }
	case ".wav":
		f, err = os.Open(t.Path)
		if err == nil { streamer, format, err = wav.Decode(f) }
	case ".flac":
		f, err = os.Open(t.Path)
		if err == nil { streamer, format, err = flac.Decode(f) }
	case ".ogg", ".vorbis":
		f, err = os.Open(t.Path)
		if err == nil { streamer, format, err = vorbis.Decode(f) }
	default:
		streamer, format, err = p.decodeWithFFmpeg(t.Path)
	}

	if err != nil {
		if f != nil { _ = f.Close() }
		streamer, format, err = p.decodeWithFFmpeg(t.Path)
	}

	if err != nil || streamer == nil {
		if err == nil { err = fmt.Errorf("decoding failed") }
		return err
	}

	if !p.initialized {
		err = speaker.Init(HardwareSampleRate, HardwareSampleRate.N(time.Second/10))
		if err != nil {
			return fmt.Errorf("failed to init speaker: %w", err)
		}
		p.initialized = true
	}

	p.format = format
	p.status.IsDone = false
	if streamer.Len() > 0 {
		t.Length = format.SampleRate.D(streamer.Len()).Round(time.Second)
	}

	// Wrap with done detection
	ds := &doneStreamer{
		StreamSeekCloser: streamer,
		onDone: func() {
			p.status.IsDone = true
			p.status.IsPlaying = false
		},
	}

	p.streamer = ds
	p.resampler = beep.Resample(4, format.SampleRate, HardwareSampleRate, ds)
	p.ctrl = &beep.Ctrl{Streamer: p.resampler, Paused: false}
	p.volumeCtrl = &effects.Volume{
		Streamer: p.ctrl,
		Base:     2,
		Volume:   0,
	}
	
	p.status.CurrentTrack = t
	p.status.IsPlaying = true
	p.status.Elapsed = 0
	
	p.applyVolume()
	speaker.Clear()
	speaker.Play(p.volumeCtrl)

	return nil
}

func (p *MusicPlayer) TogglePause() {
	if p.ctrl != nil {
		p.ctrl.Paused = !p.ctrl.Paused
		p.status.IsPlaying = !p.ctrl.Paused
	}
}

func (p *MusicPlayer) SetVolume(v float64) {
	if v < 0 { v = 0 }
	if v > 1 { v = 1 }
	p.status.Volume = v
	if !p.status.IsMuted { p.applyVolume() }
}

func (p *MusicPlayer) ToggleMute() {
	if p.status.IsMuted {
		p.status.IsMuted = false
		p.status.Volume = p.prevVolume
	} else {
		p.status.IsMuted = true
		p.prevVolume = p.status.Volume
		p.status.Volume = 0
	}
	p.applyVolume()
}

func (p *MusicPlayer) applyVolume() {
	if p.volumeCtrl != nil {
		if p.status.Volume <= 0.01 {
			p.volumeCtrl.Volume = -10 
		} else {
			p.volumeCtrl.Volume = math.Log2(p.status.Volume)
		}
	}
}

func (p *MusicPlayer) Seek(d time.Duration) {
	if p.streamer == nil { return }
	newPos := p.streamer.Position() + HardwareSampleRate.N(d)
	if newPos < 0 { newPos = 0 }
	if newPos >= p.streamer.Len() { newPos = p.streamer.Len() - 1 }
	speaker.Lock()
	_ = p.streamer.Seek(newPos)
	speaker.Unlock()
}

func (p *MusicPlayer) Update() {
	if p.status.IsPlaying && p.streamer != nil {
		pos := p.streamer.Position()
		p.status.Elapsed = p.format.SampleRate.D(pos)
		
		// If Len() is known, use it as a safety check, but otherwise rely on callback
		if p.streamer.Len() > 0 && pos >= p.streamer.Len() {
			p.status.IsDone = true
			p.status.IsPlaying = false
		}
	}
}

type doneStreamer struct {
	beep.StreamSeekCloser
	onDone func()
}

func (s *doneStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	n, ok = s.StreamSeekCloser.Stream(samples)
	if !ok {
		s.onDone()
	}
	return n, ok
}
