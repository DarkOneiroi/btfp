package player

import (
	"io"
	"os"
	"os/exec"
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

type Track struct {
	Title  string
	Artist string
	Path   string
	Length time.Duration
}

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

func NewMusicPlayer() *MusicPlayer {
	return &MusicPlayer{
		Volume: 1.0,
	}
}

func (p *MusicPlayer) PlayTrack(t *Track) error {
	if p.streamer != nil {
		p.streamer.Close()
	}

	var streamer beep.StreamSeekCloser
	var format beep.Format
	var err error
	var f *os.File

	ext := strings.ToLower(filepath.Ext(t.Path))
	
	// Native decoders
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
		// Fallback to ffmpeg for other formats
		streamer, format, err = p.decodeWithFFmpeg(t.Path)
	}

	if err != nil {
		if f != nil {
			f.Close()
		}
		// If native failed, try ffmpeg as ultimate fallback
		if ext == ".mp3" || ext == ".wav" || ext == ".flac" || ext == ".ogg" {
			streamer, format, err = p.decodeWithFFmpeg(t.Path)
		}
		
		if err != nil {
			return err
		}
	}

	p.sampleRate = format.SampleRate
	p.streamer = streamer
	p.IsDone = false

	if streamer.Len() > 0 {
		t.Length = format.SampleRate.D(streamer.Len()).Round(time.Second)
	}

	if p.ctrl == nil {
		speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	}

	p.ctrl = &beep.Ctrl{Streamer: streamer, Paused: false}

	// Set up volume control
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

type ffmpegStreamer struct {
	beep.StreamSeekCloser
	io.Closer
	cmd *exec.Cmd
}

func (fs *ffmpegStreamer) Close() error {
	err1 := fs.StreamSeekCloser.Close()
	err2 := fs.Closer.Close()
	if fs.cmd.Process != nil {
		fs.cmd.Process.Kill()
	}
	if err1 != nil {
		return err1
	}
	return err2
}

func (p *MusicPlayer) decodeWithFFmpeg(path string) (beep.StreamSeekCloser, beep.Format, error) {
	// ffmpeg -i input -f wav -
	cmd := exec.Command("ffmpeg", "-i", path, "-f", "wav", "pipe:1")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, beep.Format{}, err
	}

	if err := cmd.Start(); err != nil {
		return nil, beep.Format{}, err
	}

	// We use wav.Decode because we are piping wav from ffmpeg
	streamer, format, err := wav.Decode(stdout)
	if err != nil {
		stdout.Close()
		cmd.Process.Kill()
		return nil, beep.Format{}, err
	}

	// Wrap the streamer to close the pipe and kill the process when done
	wrapped := &ffmpegStreamer{
		StreamSeekCloser: streamer,
		Closer:           stdout,
		cmd:              cmd,
	}

	return wrapped, format, nil
}

func (p *MusicPlayer) TogglePause() {
	if p.ctrl != nil {
		p.ctrl.Paused = !p.ctrl.Paused
		p.IsPlaying = !p.ctrl.Paused
	}
}

func (p *MusicPlayer) Seek(offset time.Duration) {
	if p.streamer == nil {
		return
	}

	newPos := p.streamer.Position() + p.sampleRate.N(offset)
	if newPos < 0 {
		newPos = 0
	}
	if newPos >= p.streamer.Len() {
		p.IsDone = true
		return
	}

	speaker.Lock()
	p.streamer.Seek(newPos)
	speaker.Unlock()
}

func (p *MusicPlayer) SetVolume(v float64) {
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	p.Volume = v
	p.IsMuted = false
	p.applyVolume()
}

func (p *MusicPlayer) ToggleMute() {
	if p.IsMuted {
		p.Volume = p.prevVolume
		p.IsMuted = false
	} else {
		p.prevVolume = p.Volume
		p.Volume = 0
		p.IsMuted = true
	}
	p.applyVolume()
}

func (p *MusicPlayer) applyVolume() {
	if p.volumeCtrl == nil {
		return
	}
	speaker.Lock()
	// Beep volume is logarithmic. Volume 0 is multiplier 1.
	// We'll map 0.0-1.0 to a useful range.
	if p.Volume <= 0 {
		p.volumeCtrl.Volume = -10 // Effectively silent
	} else {
		// Map 0.1-1.0 to -3.0 to 0.0
		p.volumeCtrl.Volume = (p.Volume - 1.0) * 3.0
	}
	speaker.Unlock()
}

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
