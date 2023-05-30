package align

import "git.sr.ht/~rockorager/rtk"

// Center returns a Surface centered vertically and horizontally within the
// parent surface.
func Center(parent rtk.Surface, cols int, rows int) rtk.Surface {
	pCols, pRows := parent.Size()
	row := (pRows / 2) - (rows / 2)
	col := (pCols / 2) - (cols / 2)
	return rtk.NewSubSurface(parent, col, row, cols, rows)
}

func TopLeft(parent rtk.Surface, cols int, rows int) rtk.Surface {
	return rtk.NewSubSurface(parent, 0, 0, cols, rows)
}

func TopMiddle(parent rtk.Surface, cols int, rows int) rtk.Surface {
	pCols, _ := parent.Size()
	col := (pCols / 2) - (cols / 2)
	return rtk.NewSubSurface(parent, col, 0, cols, rows)
}

func TopRight(parent rtk.Surface, cols int, rows int) rtk.Surface {
	pCols, _ := parent.Size()
	col := pCols - cols
	return rtk.NewSubSurface(parent, col, 0, cols, rows)
}

func BottomLeft(parent rtk.Surface, cols int, rows int) rtk.Surface {
	_, pRows := parent.Size()
	row := pRows - rows
	return rtk.NewSubSurface(parent, 0, row, cols, rows)
}

func BottomMiddle(parent rtk.Surface, cols int, rows int) rtk.Surface {
	pCols, pRows := parent.Size()
	row := pRows - rows
	col := (pCols / 2) - (cols / 2)
	return rtk.NewSubSurface(parent, col, row, cols, rows)
}

func BottomRight(parent rtk.Surface, cols int, rows int) rtk.Surface {
	pCols, pRows := parent.Size()
	row := pRows - rows
	col := pCols - cols
	return rtk.NewSubSurface(parent, col, row, cols, rows)
}
