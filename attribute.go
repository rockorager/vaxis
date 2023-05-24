package rtk

// AttributeMask represents a bitmask of boolean attributes to style a cell
type AttributeMask uint32

const (
	AttrBold AttributeMask = 1 << iota
	AttrDim
	AttrItalic
	AttrUnderline
	AttrBlink
	AttrReverse
	AttrInvisible
	AttrStrikethrough
)

// TODO Smulx (crazy underlines) support \x1B[4:%p1%dm. Wezterm and kitty
// support this feature
