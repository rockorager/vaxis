package term

import (
	"git.sr.ht/~rockorager/vaxis"
)

type cursor struct {
	vaxis.Cell
	style vaxis.CursorStyle

	// position
	row row    // 0-indexed
	col column // 0-indexed
}
