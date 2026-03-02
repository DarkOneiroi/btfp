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

func (m *MockPlayer) PlayTrack(t *Track) error {
	m.status.CurrentTrack = t
	m.status.IsPlaying = true
	m.status.IsDone = false
	return nil
}

func (m *MockPlayer) TogglePause() {
	m.status.IsPlaying = !m.status.IsPlaying
}

func (m *MockPlayer) SetVolume(v float64) {
	m.status.Volume = v
}

func (m *MockPlayer) ToggleMute() {
	m.status.IsMuted = !m.status.IsMuted
}

func (m *MockPlayer) Seek(d time.Duration) {
	m.status.Elapsed += d
}

func (m *MockPlayer) Update() {
	if m.status.IsPlaying {
		m.status.Elapsed += time.Second
	}
}

func (m *MockPlayer) GetStatus() Status {
	return m.status
}

func (m *MockPlayer) SetStatus(s Status) {
	m.status = s
}
