package vaxis

// Cell represents a single cell in a terminal window. It contains a [Character]
// and a [Style], which fully defines the value. The zero value is rendered as
// an empty space
type Cell struct {
	Character
	Style
	// sixel marks if this cell has had a sixel graphic drawn on it.
	// If true, it won't be drawn in the render cycle.
	sixel bool
}
