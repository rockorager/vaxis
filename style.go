package vaxis

// Style contains all the data required to style a [Cell] or [Segment]
type Style struct {
	// Hyperlink is used for adding OSC 8 information to the cell or
	// segment.
	Hyperlink string
	// HyperlinkID is used to signal to the terminal that non-contiguous
	// hyperlinks are part of the same link, and any hints the terminal may
	// show should apply to all cells with the same HyperlinkID
	HyperlinkID string
	// WidthHint is used to signal to the renderer how wide this cell is.
	// This value is only used as an optimization, the renderer will
	// calculate if the value is 0
	WidthHint int
	// Foreground is the color to apply to the foreground of this cell
	Foreground Color
	// Background is the color to apply to the background of this cell
	Background Color
	// UnderlineColor is the color to apply to the underline of this cell,
	// if supported
	UnderlineColor Color
	// UnderlineStyle is the type of underline to apply (single, double,
	// curly, etc). If a particular style is not supported, Vaxis will
	// fallback to single underlines
	UnderlineStyle UnderlineStyle
	// Attribute represents all other style information for this cell (bold,
	// dim, italic, etc)
	Attribute AttributeMask
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

// UnderlineStyle represents the style of underline to apply
type UnderlineStyle uint8

const (
	UnderlineOff UnderlineStyle = iota
	UnderlineSingle
	UnderlineDouble
	UnderlineCurly
	UnderlineDotted
	UnderlineDashed
)
