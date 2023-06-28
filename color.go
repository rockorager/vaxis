package vaxis

// Color is a terminal color. The zero value represents the default foreground
// or background color
type Color uint32

const (
	indexed Color = 1 << 24
	rgb     Color = 1 << 25
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

func RGBColor(r uint8, g uint8, b uint8) Color {
	color := Color(int(r)<<16 | int(g)<<8 | int(b))
	return color | rgb
}

func IndexColor(index uint8) Color {
	color := Color(index)
	return color | indexed
}
