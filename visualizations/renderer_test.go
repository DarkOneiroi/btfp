// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package visualizations

import (
	"testing"
)

func TestNewFrame(t *testing.T) {
	width, height := 80, 24
	frame := NewFrame(width, height, PatternPlasma)

	if frame.Width != width {
		t.Errorf("expected width %d, got %d", width, frame.Width)
	}
	if len(frame.Data) != width*height {
		t.Errorf("expected data length %d, got %d", width*height, len(frame.Data))
	}
}

func TestGeneratePattern(t *testing.T) {
	frame := NewFrame(10, 10, PatternPlasma)
	frame.GeneratePattern(0.5)

	// Verify data is populated
	hasNonZero := false
	for _, v := range frame.Data {
		if v > 0 {
			hasNonZero = true
		}
		if v < 0 || v > 1 {
			t.Errorf("value %f out of bounds [0, 1]", v)
		}
	}
	if !hasNonZero {
		t.Error("pattern generation resulted in empty frame")
	}
}

func TestPatternStrings(t *testing.T) {
	if PatternPlasma.String() != "Plasma" {
		t.Errorf("expected Plasma, got %s", PatternPlasma.String())
	}
	if PatternEQ.String() != "EQ" {
		t.Errorf("expected EQ, got %s", PatternEQ.String())
	}
}
