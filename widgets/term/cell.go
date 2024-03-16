package term

import (
	"git.sr.ht/~rockorager/vaxis"
)

type cell struct {
	vaxis.Cell
	wrapped bool
}

func (c *cell) rune() string {
	if c.Grapheme == "" {
		return " "
	}
	return c.Grapheme
}

// Erasing removes characters from the screen without affecting other characters
// on the screen. Erased characters are lost. The cursor position does not
// change when erasing characters or lines. Erasing resets the attributes, but
// applies the background color of the passed style
func (c *cell) erase(bg vaxis.Color) {
	c.Grapheme = ""
	c.Attribute = 0
	c.UnderlineStyle = vaxis.UnderlineOff
	c.Background = bg
	c.Hyperlink = ""
	c.HyperlinkParams = ""
}

// selectiveErase removes the cell content, but keeps the attributes
func (c *cell) selectiveErase() {
	c.Grapheme = " "
}
