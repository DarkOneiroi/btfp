// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package models

import "time"

// Track represents a single audio track and its metadata
type Track struct {
	Title  string
	Artist string
	Album  string
	Path   string
	Length time.Duration
}

// Status represents the current state of the player
type Status struct {
	CurrentTrack *Track
	IsPlaying    bool
	IsDone       bool
	IsMuted      bool
	Volume       float64
	Elapsed      time.Duration
}
