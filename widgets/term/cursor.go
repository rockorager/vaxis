package term

import (
	"git.sr.ht/~rockorager/rtk"
)

type cursor struct {
	fg      rtk.Color
	bg      rtk.Color
	ul      rtk.Color
	ulStyle rtk.UnderlineStyle
	attrs   rtk.AttributeMask
	style   rtk.CursorStyle

	// position
	row row    // 0-indexed
	col column // 0-indexed
}
