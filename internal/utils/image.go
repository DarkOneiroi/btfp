// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package utils

import (
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"strings"

	"github.com/nfnt/resize"
)

// ImageToASCII converts an image to high-fidelity ASCII or uses terminal-native protocols if available.
func ImageToASCII(path string, width int) string {
	if os.Getenv("TERM_PROGRAM") == "WezTerm" || os.Getenv("TERM_PROGRAM") == "iTerm.app" {
		return imageToProtocol(path, width)
	}

	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()

	img, _, err := image.Decode(f)
	if err != nil {
		return ""
	}

	bounds := img.Bounds()
	ratio := float64(bounds.Dy()) / float64(bounds.Dx())

	// Aspect ratio: terminal cells are roughly 2:1 height:width.
	// Since 1 half-block char = 2 vertical pixels, the effective cell is 1:1.
	// We adjust height to maintain the image's original aspect ratio.
	height := int(float64(width) * ratio * 2.0)
	if height%2 != 0 {
		height++
	}

	// High quality Lanczos3 resampling
	img = resize.Resize(uint(width), uint(height), img, resize.Lanczos3)

	var sb strings.Builder
	for y := 0; y < height; y += 2 {
		for x := 0; x < width; x++ {
			c1 := img.At(x, y)
			c2 := img.At(x, y+1)

			tr, tg, tb, _ := c1.RGBA()
			br, bg, bb, _ := c2.RGBA()

			// Convert to 8-bit
			r1, g1, b1 := uint8(tr>>8), uint8(tg>>8), uint8(tb>>8)
			r2, g2, b2 := uint8(br>>8), uint8(bg>>8), uint8(bb>>8)

			// [38;2;R;G;Bm -> Foreground (Top pixel)
			// [48;2;R;G;Bm -> Background (Bottom pixel)
			// ▀ -> Half block
			sb.WriteString(fmt.Sprintf("[38;2;%d;%d;%dm[48;2;%d;%d;%dm▀[0m", r1, g1, b1, r2, g2, b2))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func imageToProtocol(path string, width int) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	encoded := base64.StdEncoding.EncodeToString(data)
	// OSC 1337 ; File = [args] : [base64] ST
	return fmt.Sprintf("\033]1337;File=width=%d;preserveAspectRatio=1;inline=1:%s\033\\", width, encoded)
}

// ContrastEnhance improves image visibility for terminal rendering (unused currently, but available)
func ContrastEnhance(c color.Color, factor float64) color.RGBA {
	r, g, b, a := c.RGBA()
	rf, gf, bf := float64(r>>8), float64(g>>8), float64(b>>8)

	rf = (rf-128)*factor + 128
	gf = (gf-128)*factor + 128
	bf = (bf-128)*factor + 128

	clamp := func(v float64) uint8 {
		if v < 0 {
			return 0
		}
		if v > 255 {
			return 255
		}
		return uint8(v)
	}

	return color.RGBA{clamp(rf), clamp(gf), clamp(bf), uint8(a >> 8)}
}
