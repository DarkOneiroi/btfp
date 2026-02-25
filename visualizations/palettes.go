package visualizations

type PaletteType int

const (
	PaletteStandard PaletteType = iota
	PaletteBlocks
	PaletteCircles
	PaletteSmooth
	PaletteBraille
	PaletteGeometric
	PaletteMixed
	PaletteDots
	PaletteExtended
	PaletteSimple
	PaletteShades
	PaletteLines
	PaletteTriangles
	PaletteArrows
	PalettePowerline
	PaletteBoxDraw
	PaletteTypeCount
)

func (p PaletteType) String() string {
	switch p {
	case PaletteStandard:
		return "Standard"
	case PaletteBlocks:
		return "Blocks"
	case PaletteCircles:
		return "Circles"
	case PaletteSmooth:
		return "Smooth"
	case PaletteBraille:
		return "Braille"
	case PaletteGeometric:
		return "Geometric"
	case PaletteMixed:
		return "Mixed"
	case PaletteDots:
		return "Dots"
	case PaletteExtended:
		return "Extended"
	case PaletteSimple:
		return "Simple"
	case PaletteShades:
		return "Shades"
	case PaletteLines:
		return "Lines"
	case PaletteTriangles:
		return "Triangles"
	case PaletteArrows:
		return "Arrows"
	case PalettePowerline:
		return "Powerline"
	case PaletteBoxDraw:
		return "BoxDraw"
	default:
		return "Unknown"
	}
}

func GetCharacters(p PaletteType) []rune {
	switch p {
	case PaletteStandard:
		return []rune{' ', '.', ':', '-', '=', '+', '*', '#', '%', '@'}
	case PaletteBlocks:
		return []rune{' ', '░', '▒', '▓', '█'}
	case PaletteCircles:
		return []rune{' ', '·', '∘', '○', '◌', '◍', '◎', '◉', '●', '█'}
	case PaletteSmooth:
		return []rune{' ', '·', '∘', '○', '◌', '◍', '◎', '◉', '●', '█'}
	case PaletteBraille:
		return []rune{' ', '⠁', '⠃', '⠇', '⠏', '⠟', '⠿', '⡿', '⣿'}
	case PaletteGeometric:
		return []rune{' ', '▪', '▫', '▬', '▭', '▮', '▯', '■', '█'}
	case PaletteMixed:
		return []rune{' ', '·', '∘', '░', '▒', '▓', '●', '◉', '■', '█'}
	case PaletteDots:
		return []rune{' ', '⡀', '⡄', '⡆', '⡇', '⣇', '⣧', '⣷', '⣿'}
	case PaletteExtended:
		return []rune{' ', '.', '\'', '`', '^', '"', ',', ':', ';', 'I', 'l', '!', 'i', '>', '<', '~', '+', '_', '-', '?', ']', '[', '}', '{', '1', ')', '(', '|', '\\', '/', 't', 'f', 'j', 'r', 'x', 'n', 'u', 'v', 'c', 'z', 'X', 'Y', 'U', 'J', 'C', 'L', 'Q', '0', 'O', 'Z', 'm', 'w', 'q', 'p', 'd', 'b', 'k', 'h', 'a', 'o', '*', '#', 'M', 'W', '&', '8', '%', 'B', '@', '$'}
	case PaletteSimple:
		return []rune{' ', '.', 'o', 'O', '@'}
	case PaletteShades:
		return []rune{' ', '░', '░', '▒', '▒', '▓', '▓', '█', '█'}
	case PaletteLines:
		return []rune{' ', '╌', '╍', '┄', '┅', '┈', '┉', '━', '█'}
	case PaletteTriangles:
		return []rune{' ', '▵', '▴', '▿', '▾', '◂', '◃', '▸', '▹'}
	case PaletteArrows:
		return []rune{' ', '›', '»', '⟩', '→', '⇒', '⟹', '⟾', '▶'}
	case PalettePowerline:
		return []rune{' ', '\ue0b0', '\ue0b1', '\ue0b2', '\ue0b3', '\ue0b4', '\ue0b5', '\ue0b6', '█'}
	case PaletteBoxDraw:
		return []rune{' ', '─', '━', '│', '┃', '┼', '╋', '╬', '█'}
	default:
		return []rune{' ', '@'}
	}
}
