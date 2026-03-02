// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package player

import (
	"io"
	"os/exec"

	"github.com/gopxl/beep"
	"github.com/gopxl/beep/wav"
)

// ffmpegStreamer wraps an FFmpeg process and its output pipe
// It implements beep.StreamSeekCloser
type ffmpegStreamer struct {
	beep.StreamSeekCloser
	io.Closer
	cmd *exec.Cmd
}

// Close ensures the pipe is closed and the FFmpeg process is terminated
func (fs *ffmpegStreamer) Close() error {
	err1 := fs.StreamSeekCloser.Close()
	err2 := fs.Closer.Close()
	if fs.cmd.Process != nil {
		_ = fs.cmd.Process.Kill()
	}
	if err1 != nil {
		return err1
	}
	return err2
}

// decodeWithFFmpeg uses FFmpeg to decode any format into a WAV stream
func (p *MusicPlayer) decodeWithFFmpeg(path string) (beep.StreamSeekCloser, beep.Format, error) {
	// Command: ffmpeg -i <input> -f wav pipe:1
	// This pipes the decoded audio in WAV format to stdout
	cmd := exec.Command("ffmpeg", "-i", path, "-f", "wav", "pipe:1")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, beep.Format{}, err
	}

	if err = cmd.Start(); err != nil {
		return nil, beep.Format{}, err
	}

	// Since FFmpeg is outputting WAV, we can use the native WAV decoder to read the pipe
	streamer, format, err := wav.Decode(stdout)
	if err != nil {
		_ = stdout.Close()
		_ = cmd.Process.Kill()
		return nil, beep.Format{}, err
	}

	wrapped := &ffmpegStreamer{
		StreamSeekCloser: streamer,
		Closer:           stdout,
		cmd:              cmd,
	}

	return wrapped, format, nil
}
