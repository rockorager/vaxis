package vaxis

type Text struct {
	Content        string
	Hyperlink      string
	HyperlinkID    string
	WidthHint      int
	Foreground     Color
	Background     Color
	UnderlineColor Color
	UnderlineStyle UnderlineStyle
	Attribute      AttributeMask
	// sixel marks if this cell has had a sixel graphic drawn on it.
	// If true, it won't be drawn in the render cycle.
	sixel bool
}

// AttributeMask represents a bitmask of boolean attributes to style a cell
type AttributeMask uint8

const (
	AttrNone               = 0
	AttrBold AttributeMask = 1 << iota
	AttrDim
	AttrItalic
	AttrBlink
	AttrReverse
	AttrInvisible
	AttrStrikethrough
)

type UnderlineStyle uint8

const (
	UnderlineOff UnderlineStyle = iota
	UnderlineSingle
	UnderlineDouble
	UnderlineCurly
	UnderlineDotted
	UnderlineDashed
)
