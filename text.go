package vaxis

import "github.com/rivo/uniseg"

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
