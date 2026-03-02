// Copyright (c) 2026 DarkOneiroi
// All rights reserved.
// This source code is proprietary and confidential.
// Unauthorized copying of this file, via any medium, is strictly prohibited.

package visualizations

// PaletteType defines the set of characters used for rendering
type PaletteType int

const (
	// PaletteStandard uses a set of standard density characters
	PaletteStandard PaletteType = iota
	// PaletteBlocks uses block-style characters
	PaletteBlocks
	// PaletteBraille uses braille-dot characters
	PaletteBraille
	// PaletteASCII uses only basic ASCII characters
	PaletteASCII
	// PaletteSimple uses a minimal set of characters
	PaletteSimple
	// PaletteDetailed uses a wide range of characters for high detail
	PaletteDetailed
	// PaletteBinary uses only 0 and 1
	PaletteBinary
	// PaletteHex uses hexadecimal characters
	PaletteHex
	// PaletteDots uses only dot characters
	PaletteDots
	// PaletteLines uses line-drawing characters
	PaletteLines
	// PaletteMath uses mathematical symbols
	PaletteMath
	// PaletteShades uses grayscale shade blocks
	PaletteShades
	// PaletteGradient uses a smooth character gradient
	PaletteGradient
	// PaletteMatrix uses characters reminiscent of the Matrix movie
	PaletteMatrix
	// PaletteMinimal uses only two characters
	PaletteMinimal
	// PaletteTypeCount is the total number of palettes
	PaletteTypeCount
)

// GetCharacters returns the slice of runes associated with a PaletteType
func GetCharacters(p PaletteType) []rune {
	switch p {
	case PaletteBlocks:
		return []rune(" ░▒▓█")
	case PaletteBraille:
		return []rune(" ⠁⠂⠃⠄⠅⠆⠇⡀⡁⡂⡃⡄⡅⡆⡇")
	case PaletteASCII:
		return []rune(" .:-=+*#%@")
	case PaletteSimple:
		return []rune(" .oO@")
	case PaletteDetailed:
		return []rune(" .'`^,:;Il!i><~+_-?][}{1)(|\/tfjrxnuvczMWQH8JPK6A4d25gbp9qwmkLO0UYXZE")
	case PaletteBinary:
		return []rune(" 01")
	case PaletteHex:
		return []rune(" 0123456789ABCDEF")
	case PaletteDots:
		return []rune(" .·•●")
	case PaletteLines:
		return []rune(" ─│┌┐└┘├┤┬┴┼")
	case PaletteMath:
		return []rune(" +-×÷=≠≈∞∑∏")
	case PaletteShades:
		return []rune(" ░▒▓█")
	case PaletteGradient:
		return []rune(" .:-=+*#%@")
	case PaletteMatrix:
		return []rune(" 0123456789$+-*/=%\"'#&_(),.;:?!\|{}<>[]^~")
	case PaletteMinimal:
		return []rune(" #")
	case PaletteStandard:
		fallthrough
	default:
		return []rune(" .:-=+*#%@")
	}
}
