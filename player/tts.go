// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package player

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gopxl/beep"
	sherpa "github.com/k2-fsa/sherpa-onnx-go/sherpa_onnx"
)

// TTSPlayer handles text-to-speech conversion and playback
type TTSPlayer struct {
	tts    *sherpa.OfflineTts
	config *sherpa.OfflineTtsConfig
}

// NewTTSPlayer creates a new TTS player with the specified model paths
func NewTTSPlayer(modelPath, lexiconPath, tokensPath string) (*TTSPlayer, error) {
	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("TTS model not found at %s. Please see documentation for setup.", modelPath)
	}

	config := &sherpa.OfflineTtsConfig{
		Model: sherpa.OfflineTtsModelConfig{
			Vits: sherpa.OfflineTtsVitsModelConfig{
				Model:   modelPath,
				Lexicon: lexiconPath,
				Tokens:  tokensPath,
			},
			NumThreads: 4,
			Debug:      0,
			Provider:   "cpu",
		},
	}

	tts := sherpa.NewOfflineTts(config)
	if tts == nil {
		return nil, fmt.Errorf("failed to create offline TTS engine")
	}

	return &TTSPlayer{
		tts:    tts,
		config: config,
	}, nil
}

// Close releases the TTS engine resources
func (p *TTSPlayer) Close() {
	if p.tts != nil {
		sherpa.DeleteOfflineTts(p.tts)
	}
}

// GenerateAudio converts text to a beep.Streamer
func (p *TTSPlayer) GenerateAudio(text string, speakerID int) (beep.StreamSeekCloser, beep.Format, error) {
	// Generating audio with speed 1.0
	audio := p.tts.Generate(text, speakerID, 1.0)
	if audio.Samples == nil {
		return nil, beep.Format{}, fmt.Errorf("failed to generate audio")
	}

	sampleRate := beep.SampleRate(audio.SampleRate)

	// Convert float32 samples to beep.Streamer
	// sherpa-onnx returns mono audio as float32 slice
	streamer := &ttsStreamer{
		samples: audio.Samples,
		pos:     0,
		rate:    sampleRate,
	}

	return streamer, beep.Format{
		SampleRate:  sampleRate,
		NumChannels: 1,
		Precision:   4,
	}, nil
}

type ttsStreamer struct {
	samples []float32
	pos     int
	rate    beep.SampleRate
}

func (s *ttsStreamer) Stream(samples [][2]float64) (n int, ok bool) {
	for i := range samples {
		if s.pos >= len(s.samples) {
			return i, false
		}
		val := float64(s.samples[s.pos])
		samples[i][0] = val
		samples[i][1] = val
		s.pos++
	}
	return len(samples), true
}

func (s *ttsStreamer) Err() error    { return nil }
func (s *ttsStreamer) Len() int      { return len(s.samples) }
func (s *ttsStreamer) Position() int { return s.pos }
func (s *ttsStreamer) Seek(p int) error {
	if p < 0 || p >= len(s.samples) {
		return fmt.Errorf("invalid seek position")
	}
	s.pos = p
	return nil
}
func (s *ttsStreamer) Close() error { return nil }

// ReadTextFile reads the content of a text file
func ReadTextFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// GetTTSModelPaths returns the paths to the required TTS model files for a language
func GetTTSModelPaths(lang string) (model, lexicon, tokens string) {
	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, ".config", "btfp", "models", lang)

	// Example structure:
	// ~/.config/btfp/models/en/model.onnx
	// ~/.config/btfp/models/en/lexicon.txt
	// ~/.config/btfp/models/en/tokens.txt

	return filepath.Join(baseDir, "model.onnx"),
		filepath.Join(baseDir, "lexicon.txt"),
		filepath.Join(baseDir, "tokens.txt")
}
