package vaxis

// Style contains all the data required to style a [Cell] or [Segment]
type Style struct {
	// Hyperlink is used for adding OSC 8 information to the cell or
	// segment.
	Hyperlink string
	// HyperlinkParams is passed as the param string for OSC 8 sequences.
	// Typically this will be something like "id=<some-id>" to signal
	// non-contiguous links which are the same (IE when a link may be
	// wrapped on lines)
	HyperlinkParams string
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
