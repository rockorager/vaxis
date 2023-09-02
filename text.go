package vaxis

import "github.com/rivo/uniseg"

// Text represents the value of a [Cell], or a text segment suitable for
// use in several [Window] methods. When used as a [Cell], the value
// of Content should be one extended-grapheme-cluster ("character"). A helper
// function "[Characters]" is provided to turn any string into a slice of
// extended-grapheme-clusters
type Text struct {
	// Content is the value of the cell or text segment
	Content string
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

// Character is a single extended-grapheme-cluster. It also contains the width
// of the EGC
type Character struct {
	Grapheme string
	Width    int
}

// Converts a string into a slice of Characters suitable to assign to terminal cells
func Characters(s string) []Character {
	egcs := make([]Character, 0, len(s))
	state := -1
	cluster := ""
	w := 0
	for s != "" {
		cluster, s, w, state = uniseg.FirstGraphemeClusterInString(s, state)
		if cluster == "\t" {
			for i := 0; i < 8; i += 1 {
				egcs = append(egcs, Character{" ", 1})
			}
			continue
		}
		egcs = append(egcs, Character{cluster, w})
	}
	return egcs
}
