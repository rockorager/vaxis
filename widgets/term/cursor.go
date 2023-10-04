package term

import (
	"git.sr.ht/~rockorager/vaxis"
)

type cursor struct {
	fg      vaxis.Color
	bg      vaxis.Color
	ul      vaxis.Color
	ulStyle vaxis.UnderlineStyle
	attrs   vaxis.AttributeMask
	style   vaxis.CursorStyle
	url     string
	urlId   string

	// position
	row row    // 0-indexed
	col column // 0-indexed
}
