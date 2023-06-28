package rtk

type Cell struct {
	Character      string // Extended Grapheme Cluster
	Foreground     Color
	Background     Color
	Underline      Color
	UnderlineStyle UnderlineStyle
	Attribute      AttributeMask
	Hyperlink      string
	HyperlinkID    string
}

// AttributeMask represents a bitmask of boolean attributes to style a cell
type AttributeMask uint8

const (
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
