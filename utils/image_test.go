// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package utils

import (
	"image/color"
	"testing"
)

func TestImageToASCIIInvalidPath(t *testing.T) {
	res := ImageToASCII("non-existent.jpg", 10)
	if res != "" {
		t.Error("expected empty string for invalid path")
	}
}

func TestContrastEnhance(t *testing.T) {
	c := color.RGBA{100, 100, 100, 255}
	// (100-128)*1.5 + 128 = -28*1.5 + 128 = -42 + 128 = 86
	enhanced := ContrastEnhance(c, 1.5)

	if enhanced.R != 86 {
		t.Errorf("expected enhanced red 86, got %d", enhanced.R)
	}
}
