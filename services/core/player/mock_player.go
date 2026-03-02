// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package player

import (
	"time"
)

// MockPlayer is a mock implementation of the Player interface for testing.
type MockPlayer struct {
	status Status
}

// NewMockPlayer creates a new mock player instance.
func NewMockPlayer() *MockPlayer {
	return &MockPlayer{
		status: Status{
			Volume: 1.0,
		},
	}
}

// PlayTrack is a mock implementation
func (m *MockPlayer) PlayTrack(t *Track) error {
	m.status.CurrentTrack = t
	m.status.IsPlaying = true
	return nil
}

// TogglePause is a mock implementation
func (m *MockPlayer) TogglePause() {
	m.status.IsPlaying = !m.status.IsPlaying
}

// SetVolume is a mock implementation
func (m *MockPlayer) SetVolume(v float64) {
	m.status.Volume = v
}

// ToggleMute is a mock implementation
func (m *MockPlayer) ToggleMute() {
	m.status.IsMuted = !m.status.IsMuted
}

// Seek is a mock implementation
func (m *MockPlayer) Seek(d time.Duration) {
	m.status.Elapsed += d
}

// Update is a mock implementation
func (m *MockPlayer) Update() {
	if m.status.IsPlaying {
		m.status.Elapsed += time.Second / 10
	}
}

// GetStatus is a mock implementation
func (m *MockPlayer) GetStatus() Status {
	return m.status
}

// SetStatus is a mock implementation
func (m *MockPlayer) SetStatus(s Status) {
	m.status = s
}

// SetTTSParams is a mock implementation
func (m *MockPlayer) SetTTSParams(lang string, speaker int) {
	// No-op for mock
}
