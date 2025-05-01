package vaxis

import "math"

// Color is a terminal color. The zero value represents the default foreground
// or background color
type Color uint32

const (
	indexed Color = 1 << 24
	rgb     Color = 1 << 25
)

const (
	// Indexed terminal color constants
	ColorBlack Color = iota | indexed
	ColorMaroon
	ColorGreen
	ColorOlive
	ColorNavy
	ColorPurple
	ColorTeal
	ColorSilver
	ColorGray
	ColorRed
	ColorLime
	ColorYellow
	ColorBlue
	ColorFuschia
	ColorAqua
	ColorWhite

	ColorDefault Color = 0
)

// Params returns the TParm parameters for the color, or an empty slice if the
// color is the default color
func (c Color) Params() []uint8 {
	switch {
	case c&indexed != 0:
		return []uint8{uint8(c)}
	case c&rgb != 0:
		r := uint8(c >> 16)
		g := uint8(c >> 8)
		b := uint8(c)
		return []uint8{r, g, b}
	}
	return []uint8{}
}

// asIndex returns an 8bit color index for a given color. If the color is the
// default color or already an index color, this returns itself. RGB colors will
// be converted to their closest 256-index match, excluding indexes 0-15 as
// these are typically user altered
func (c Color) asIndex() Color {
	if c&rgb == 0 {
		return c
	}
	// Convert to 256 palette
	oR := uint8(c >> 16)
	oG := uint8(c >> 8)
	oB := uint8(c)
	dist := math.Inf(1)
	match := -1
	for i, v := range colorIndex {
		dR := uint8(v >> 16)
		dG := uint8(v >> 8)
		dB := uint8(v)
		// weighted, thanks stackoverflow. We skip the sqrt
		// because we don't care about the absolute value of the
		// distance, only the comparisons
		trial := sq(float64(dR-oR)*.3) + sq(float64(dG-oG)*.59) + sq(float64(dB-oB)*.11)
		if trial < dist {
			match = i
			dist = trial
		}
		if dist == 0 {
			// return early when we have an exact match
			return IndexColor(uint8(i + 16))
		}
	}
	if match < 0 {
		return Color(0)
	}
	return IndexColor(uint8(match + 16))
}

func sq(v float64) float64 {
	return v * v
}

// RGBColor creates a new Color based on the supplied RGB values
func RGBColor(r uint8, g uint8, b uint8) Color {
	color := Color(int(r)<<16 | int(g)<<8 | int(b))
	return color | rgb
}

// HexColor creates a new Color based on the supplied 24-bit hex value
func HexColor(v uint32) Color {
	return Color(v) | rgb
}

// IndexColor creates a new Color from the supplied 8 bit value. Values 0-255
// are valid
func IndexColor(index uint8) Color {
	color := Color(index)
	return color | indexed
}

// index 0 is IndexColor(16)
var colorIndex = []uint32{
	0x000000,
	0x00005F,
	0x000087,
	0x0000AF,
	0x0000D7,
	0x0000FF,
	0x005F00,
	0x005F5F,
	0x005F87,
	0x005FAF,
	0x005FD7,
	0x005FFF,
	0x008700,
	0x00875F,
	0x008787,
	0x0087Af,
	0x0087D7,
	0x0087FF,
	0x00AF00,
	0x00AF5F,
	0x00AF87,
	0x00AFAF,
	0x00AFD7,
	0x00AFFF,
	0x00D700,
	0x00D75F,
	0x00D787,
	0x00D7AF,
	0x00D7D7,
	0x00D7FF,
	0x00FF00,
	0x00FF5F,
	0x00FF87,
	0x00FFAF,
	0x00FFd7,
	0x00FFFF,
	0x5F0000,
	0x5F005F,
	0x5F0087,
	0x5F00AF,
	0x5F00D7,
	0x5F00FF,
	0x5F5F00,
	0x5F5F5F,
	0x5F5F87,
	0x5F5FAF,
	0x5F5FD7,
	0x5F5FFF,
	0x5F8700,
	0x5F875F,
	0x5F8787,
	0x5F87AF,
	0x5F87D7,
	0x5F87FF,
	0x5FAF00,
	0x5FAF5F,
	0x5FAF87,
	0x5FAFAF,
	0x5FAFD7,
	0x5FAFFF,
	0x5FD700,
	0x5FD75F,
	0x5FD787,
	0x5FD7AF,
	0x5FD7D7,
	0x5FD7FF,
	0x5FFF00,
	0x5FFF5F,
	0x5FFF87,
	0x5FFFAF,
	0x5FFFD7,
	0x5FFFFF,
	0x870000,
	0x87005F,
	0x870087,
	0x8700AF,
	0x8700D7,
	0x8700FF,
	0x875F00,
	0x875F5F,
	0x875F87,
	0x875FAF,
	0x875FD7,
	0x875FFF,
	0x878700,
	0x87875F,
	0x878787,
	0x8787AF,
	0x8787D7,
	0x8787FF,
	0x87AF00,
	0x87AF5F,
	0x87AF87,
	0x87AFAF,
	0x87AFD7,
	0x87AFFF,
	0x87D700,
	0x87D75F,
	0x87D787,
	0x87D7AF,
	0x87D7D7,
	0x87D7FF,
	0x87FF00,
	0x87FF5F,
	0x87FF87,
	0x87FFAF,
	0x87FFD7,
	0x87FFFF,
	0xAF0000,
	0xAF005F,
	0xAF0087,
	0xAF00AF,
	0xAF00D7,
	0xAF00FF,
	0xAF5F00,
	0xAF5F5F,
	0xAF5F87,
	0xAF5FAF,
	0xAF5FD7,
	0xAF5FFF,
	0xAF8700,
	0xAF875F,
	0xAF8787,
	0xAF87AF,
	0xAF87D7,
	0xAF87FF,
	0xAFAF00,
	0xAFAF5F,
	0xAFAF87,
	0xAFAFAF,
	0xAFAFD7,
	0xAFAFFF,
	0xAFD700,
	0xAFD75F,
	0xAFD787,
	0xAFD7AF,
	0xAFD7D7,
	0xAFD7FF,
	0xAFFF00,
	0xAFFF5F,
	0xAFFF87,
	0xAFFFAF,
	0xAFFFD7,
	0xAFFFFF,
	0xD70000,
	0xD7005F,
	0xD70087,
	0xD700AF,
	0xD700D7,
	0xD700FF,
	0xD75F00,
	0xD75F5F,
	0xD75F87,
	0xD75FAF,
	0xD75FD7,
	0xD75FFF,
	0xD78700,
	0xD7875F,
	0xD78787,
	0xD787AF,
	0xD787D7,
	0xD787FF,
	0xD7AF00,
	0xD7AF5F,
	0xD7AF87,
	0xD7AFAF,
	0xD7AFD7,
	0xD7AFFF,
	0xD7D700,
	0xD7D75F,
	0xD7D787,
	0xD7D7AF,
	0xD7D7D7,
	0xD7D7FF,
	0xD7FF00,
	0xD7FF5F,
	0xD7FF87,
	0xD7FFAF,
	0xD7FFD7,
	0xD7FFFF,
	0xFF0000,
	0xFF005F,
	0xFF0087,
	0xFF00AF,
	0xFF00D7,
	0xFF00FF,
	0xFF5F00,
	0xFF5F5F,
	0xFF5F87,
	0xFF5FAF,
	0xFF5FD7,
	0xFF5FFF,
	0xFF8700,
	0xFF875F,
	0xFF8787,
	0xFF87AF,
	0xFF87D7,
	0xFF87FF,
	0xFFAF00,
	0xFFAF5F,
	0xFFAF87,
	0xFFAFAF,
	0xFFAFD7,
	0xFFAFFF,
	0xFFD700,
	0xFFD75F,
	0xFFD787,
	0xFFD7AF,
	0xFFD7D7,
	0xFFD7FF,
	0xFFFF00,
	0xFFFF5F,
	0xFFFF87,
	0xFFFFAF,
	0xFFFFD7,
	0xFFFFFF,
	0x080808,
	0x121212,
	0x1C1C1C,
	0x262626,
	0x303030,
	0x3A3A3A,
	0x444444,
	0x4E4E4E,
	0x585858,
	0x626262,
	0x6C6C6C,
	0x767676,
	0x808080,
	0x8A8A8A,
	0x949494,
	0x9E9E9E,
	0xA8A8A8,
	0xB2B2B2,
	0xBCBCBC,
	0xC6C6C6,
	0xD0D0D0,
	0xDADADA,
	0xE4E4E4,
	0xEEEEEE,
}
