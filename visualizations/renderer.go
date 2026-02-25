package visualizations

import (
	"fmt"
	"math"
	"strings"
)

type PatternType int

const (
	PatternPlasma PatternType = iota
	PatternWaves
	PatternRipples
	PatternVortex
	PatternGeometric
	PatternSpiral
	PatternGrid
	PatternTypeCount
	PatternEQ
)

func (p PatternType) String() string {
	switch p {
	case PatternPlasma:
		return "Plasma"
	case PatternWaves:
		return "Waves"
	case PatternRipples:
		return "Ripples"
	case PatternVortex:
		return "Vortex"
	case PatternGeometric:
		return "Geometric"
	case PatternSpiral:
		return "Spiral"
	case PatternGrid:
		return "Grid"
	case PatternEQ:
		return "EQ"
	default:
		return "Unknown"
	}
}

type ColorMode int

const (
	ColorRainbow ColorMode = iota
	ColorMonochrome
	ColorFire
	ColorOcean
	ColorNeon
	ColorCool
	ColorChromatic
	ColorModeCount
)

func (c ColorMode) String() string {
	switch c {
	case ColorRainbow:
		return "Rainbow"
	case ColorMonochrome:
		return "Monochrome"
	case ColorFire:
		return "Fire"
	case ColorOcean:
		return "Ocean"
	case ColorNeon:
		return "Neon"
	case ColorCool:
		return "Cool"
	case ColorChromatic:
		return "Chromatic"
	default:
		return "Unknown"
	}
}

type Frame struct {
	Width         int
	Height        int
	Data          []float64
	PatternType   PatternType
	ColorMode     ColorMode
	PaletteType   PaletteType
	Time          float64
	IsTransparent bool
	AudioLevels   []float64
}

func NewFrame(width, height int, pattern PatternType) *Frame {
	return &Frame{
		Width:       width,
		Height:      height,
		Data:        make([]float64, width*height),
		PatternType: pattern,
		ColorMode:   ColorRainbow,
		PaletteType: PaletteStandard,
		AudioLevels: make([]float64, 32),
	}
}

func (f *Frame) Update(dt float64) {
	f.Time += dt
}

func (f *Frame) GeneratePattern(audioLevel float64) {
	t := f.Time
	for y := 0; y < f.Height; y++ {
		for x := 0; x < f.Width; x++ {
			u := float64(x) / float64(f.Width)
			v := float64(y) / float64(f.Height)
			var val float64

			switch f.PatternType {
			case PatternPlasma:
				val = f.genPlasma(u, v, t, audioLevel)
			case PatternWaves:
				val = f.genWaves(u, v, t, audioLevel)
			case PatternRipples:
				val = f.genRipples(u, v, t, audioLevel)
			case PatternVortex:
				val = f.genVortex(u, v, t, audioLevel)
			case PatternGeometric:
				val = f.genGeometric(u, v, t, audioLevel)
			case PatternSpiral:
				val = f.genSpiral(u, v, t, audioLevel)
			case PatternGrid:
				val = f.genGrid(u, v, t, audioLevel)
			case PatternEQ:
				val = f.genEQ(u, v, t, audioLevel)
			default:
				val = f.genPlasma(u, v, t, audioLevel)
			}

			if val < 0 {
				val = 0
			}
			if val > 1 {
				val = 1
			}
			f.Data[y*f.Width+x] = val
		}
	}
}

func (f *Frame) genPlasma(u, v, t, audioLevel float64) float64 {
	val := math.Sin(u*10.0+t) + math.Cos(v*10.0+t) + math.Sin(math.Sqrt(u*u+v*v)*10.0+t) + audioLevel*2.0
	return (val + 3.0) / 6.0
}

func (f *Frame) genWaves(u, v, t, audioLevel float64) float64 {
	val := math.Sin(u*15.0+t) * math.Sin(v*15.0+t*0.5)
	val += audioLevel * 1.5
	return (val + 1.0) / 2.5
}

func (f *Frame) genRipples(u, v, t, audioLevel float64) float64 {
	dist := math.Sqrt((u-0.5)*(u-0.5) + (v-0.5)*(v-0.5))
	val := math.Sin(dist*30.0 - t*8.0)
	val *= (1.0 - dist)
	val += audioLevel
	return (val + 1.0) / 2.0
}

func (f *Frame) genVortex(u, v, t, audioLevel float64) float64 {
	dx, dy := u-0.5, v-0.5
	angle := math.Atan2(dy, dx)
	dist := math.Sqrt(dx*dx + dy*dy)
	val := math.Sin(angle*5.0 + dist*20.0 - t*5.0)
	val += audioLevel
	return (val + 1.0) / 2.0
}

func (f *Frame) genGeometric(u, v, t, audioLevel float64) float64 {
	val := math.Mod(u*10.0+t, 1.0) * math.Mod(v*10.0+t, 1.0)
	val += audioLevel
	return val
}

func (f *Frame) genSpiral(u, v, t, audioLevel float64) float64 {
	dx, dy := u-0.5, v-0.5
	dist := math.Sqrt(dx*dx + dy*dy)
	angle := math.Atan2(dy, dx)
	val := math.Sin(10.0*dist - t*10.0 + angle)
	val += audioLevel
	return (val + 1.0) / 2.0
}

func (f *Frame) genGrid(u, v, t, audioLevel float64) float64 {
	gv := math.Sin(u*50.0) * math.Sin(v*50.0)
	val := gv + math.Sin(t+audioLevel*10.0)
	return (val + 1.0) / 2.0
}

func (f *Frame) genEQ(u, v, t, audioLevel float64) float64 {
	numLevels := len(f.AudioLevels)
	if numLevels == 0 {
		return 0.0
	}
	bandIdx := int(u * float64(numLevels))
	if bandIdx >= numLevels {
		bandIdx = numLevels - 1
	}
	h := f.AudioLevels[bandIdx]
	if 1.0-v <= h {
		return 1.0 - ((1.0 - v) / (h + 0.001) * 0.5)
	}
	return 0.0
}

func (f *Frame) Render(streamMode bool) string {
	var sb strings.Builder
	chars := GetCharacters(f.PaletteType)

	for y := 0; y < f.Height; y++ {
		for x := 0; x < f.Width; x++ {
			idx := y*f.Width + x
			val := f.Data[idx]
			if f.IsTransparent && val < 0.05 {
				sb.WriteString(" ")
				continue
			}
			charIdx := int(val * float64(len(chars)-1))
			if charIdx < 0 {
				charIdx = 0
			}
			if charIdx >= len(chars) {
				charIdx = len(chars) - 1
			}
			char := chars[charIdx]
			color := f.getColor(val, x, y)
			sb.WriteString(fmt.Sprintf("\x1b[38;5;%dm%c\x1b[0m", color, char))
		}
		if y < f.Height-1 || streamMode {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func (f *Frame) getColor(val float64, x, y int) int {
	switch f.ColorMode {
	case ColorMonochrome:
		return 232 + int(val*23)
	case ColorFire:
		if val < 0.25 {
			return 16 + int(val*4*3)
		}
		if val < 0.5 {
			return 160 + int((val-0.25)*4*6)
		}
		return 226
	case ColorOcean:
		return 16 + int(val*30)
	case ColorNeon:
		return 129 + int(val*40)
	case ColorCool:
		return 23 + int(val*20)
	case ColorChromatic:
		return 16 + int(val*215)
	case ColorRainbow:
		fallthrough
	default:
		return 16 + int(val*215)
	}
}
