package term

import (
	"git.sr.ht/~rockorager/vaxis"
)

type cell struct {
	combining []rune
	width     int
	content   rune
	fg        vaxis.Color
	bg        vaxis.Color
	attrs     vaxis.AttributeMask
	wrapped   bool
}

func (c *cell) rune() rune {
	if c.content == rune(0) {
		return ' '
	}
	return c.content
}

// Erasing removes characters from the screen without affecting other characters
// on the screen. Erased characters are lost. The cursor position does not
// change when erasing characters or lines. Erasing resets the attributes, but
// applies the background color of the passed style
func (c *cell) erase(bg vaxis.Color) {
	c.content = 0
	c.attrs = 0
	c.bg = bg
}

// selectiveErase removes the cell content, but keeps the attributes
func (c *cell) selectiveErase() {
	c.content = 0
}
